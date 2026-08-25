package httpapi

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"

	"unhoused/internal/config"
)

// clientStatuses are the Nomad allocation ClientStatus values tracked in
// versionGroup.statusCounts, per specs/api.md.
var clientStatuses = []string{"running", "pending", "failed", "complete", "lost"}

func newStatusCounts() map[string]int {
	counts := make(map[string]int, len(clientStatuses))
	for _, s := range clientStatuses {
		counts[s] = 0
	}
	return counts
}

func stringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// deriveJobStatus maps a Nomad job to the running/pending/stopped/dead
// indicator shown on the Job Status Page header.
func deriveJobStatus(job *nomadapi.Job) string {
	if job.Stop != nil && *job.Stop {
		return "stopped"
	}
	if job.Status != nil {
		return *job.Status
	}
	return ""
}

// versionSubmitTimes builds a version -> SubmitTime lookup from the job's
// version history, used as the uptime reference point for allocations in
// that version.
func versionSubmitTimes(versions []*nomadapi.Job) map[uint64]time.Time {
	times := make(map[uint64]time.Time, len(versions))
	for _, v := range versions {
		if v == nil || v.Version == nil || v.SubmitTime == nil {
			continue
		}
		times[*v.Version] = time.Unix(0, *v.SubmitTime)
	}
	return times
}

func uptimeSeconds(submitTime, now time.Time) int64 {
	d := now.Sub(submitTime)
	if d < 0 {
		return 0
	}
	return int64(d.Seconds())
}

// groupByVersion groups allocation stubs by job version, sorted newest
// version first.
func groupByVersion(allocs []*nomadapi.AllocationListStub, submitTimes map[uint64]time.Time, now time.Time) []versionGroupDTO {
	groups := make(map[uint64]*versionGroupDTO)

	for _, a := range allocs {
		group, ok := groups[a.JobVersion]
		if !ok {
			group = &versionGroupDTO{
				Version:                       a.JobVersion,
				NewestAllocationUptimeSeconds: uptimeSeconds(submitTimes[a.JobVersion], now),
				StatusCounts:                  newStatusCounts(),
			}
			groups[a.JobVersion] = group
		}

		_, known := group.StatusCounts[a.ClientStatus]
		if known {
			group.StatusCounts[a.ClientStatus]++
		}
	}

	result := make([]versionGroupDTO, 0, len(groups))
	for _, group := range groups {
		result = append(result, *group)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Version > result[j].Version
	})

	return result
}

// allocationFilters holds the Job Status Page's per-field allocation table
// filters, parsed from query parameters. An empty field means "no filter" on
// that field.
type allocationFilters struct {
	Search    string
	TaskGroup string
	Version   string
	Node      string
	Status    string
	Desired   string
}

// filterAllocationStubs returns only the stubs matching every non-empty
// field in f. nodeIPs maps node ID to node IP (see nodeIPsByNodeID) and is
// consulted for the search field alongside allocation ID and node name; pass
// nil when f.Search is empty, since it's unused in that case.
func filterAllocationStubs(stubs []*nomadapi.AllocationListStub, f allocationFilters, nodeIPs map[string]string) []*nomadapi.AllocationListStub {
	filtered := make([]*nomadapi.AllocationListStub, 0, len(stubs))
	search := strings.ToLower(f.Search)

	for _, s := range stubs {
		if search != "" &&
			!strings.Contains(strings.ToLower(s.ID), search) &&
			!strings.Contains(strings.ToLower(s.NodeName), search) &&
			!strings.Contains(strings.ToLower(nodeIPs[s.NodeID]), search) {
			continue
		}
		if f.TaskGroup != "" && s.TaskGroup != f.TaskGroup {
			continue
		}
		if f.Version != "" && strconv.FormatUint(s.JobVersion, 10) != f.Version {
			continue
		}
		if f.Node != "" && s.NodeName != f.Node {
			continue
		}
		if f.Status != "" && s.ClientStatus != f.Status {
			continue
		}
		if f.Desired != "" && s.DesiredStatus != f.Desired {
			continue
		}
		filtered = append(filtered, s)
	}

	return filtered
}

// nodeIPsByNodeID builds a node ID -> node IP lookup from a node list, for
// resolving the Job Status Page search's node-IP match without an
// AllocationInfo call per allocation.
func nodeIPsByNodeID(nodes []*nomadapi.NodeListStub) map[string]string {
	ips := make(map[string]string, len(nodes))
	for _, n := range nodes {
		ips[n.ID] = n.Address
	}
	return ips
}

// allocationFilterOptions lists the distinct task group, version, and node
// values across stubs, for the Job Status Page's filter dropdowns. Callers
// pass the full unfiltered stub set so option lists don't narrow as other
// filters are applied.
func allocationFilterOptions(stubs []*nomadapi.AllocationListStub) filterOptionsDTO {
	taskGroups := make(map[string]struct{})
	versions := make(map[uint64]struct{})
	nodes := make(map[string]struct{})

	for _, s := range stubs {
		taskGroups[s.TaskGroup] = struct{}{}
		versions[s.JobVersion] = struct{}{}
		nodes[s.NodeName] = struct{}{}
	}

	opts := filterOptionsDTO{
		TaskGroups: make([]string, 0, len(taskGroups)),
		Versions:   make([]uint64, 0, len(versions)),
		Nodes:      make([]string, 0, len(nodes)),
	}
	for tg := range taskGroups {
		opts.TaskGroups = append(opts.TaskGroups, tg)
	}
	for v := range versions {
		opts.Versions = append(opts.Versions, v)
	}
	for n := range nodes {
		opts.Nodes = append(opts.Nodes, n)
	}

	sort.Strings(opts.TaskGroups)
	sort.Slice(opts.Versions, func(i, j int) bool { return opts.Versions[i] > opts.Versions[j] })
	sort.Strings(opts.Nodes)

	return opts
}

// allocationSortColumns are the Job Status Page allocation table's sortable
// columns, matching the column-header click targets in the frontend.
var allocationSortColumns = []string{"id", "node", "status", "desired", "taskGroup", "version", "uptime"}

func isAllocationSortColumn(column string) bool {
	for _, c := range allocationSortColumns {
		if c == column {
			return true
		}
	}
	return false
}

// allocationSort holds the Job Status Page allocation table's sort state,
// parsed from the `sort`/`dir` query parameters. A zero-value/invalid
// Direction (anything but "asc"/"desc") means unsorted — the stubs' original
// (Nomad-returned) order is preserved.
type allocationSort struct {
	Column    string
	Direction string
}

// sortAllocationStubs returns stubs sorted per s, or stubs unchanged if s
// isn't a valid, active sort. submitTimes resolves the "uptime" column,
// since uptime isn't a stub field (see uptimeSeconds/versionSubmitTimes).
func sortAllocationStubs(stubs []*nomadapi.AllocationListStub, s allocationSort, submitTimes map[uint64]time.Time) []*nomadapi.AllocationListStub {
	if !isAllocationSortColumn(s.Column) || (s.Direction != "asc" && s.Direction != "desc") {
		return stubs
	}

	sorted := make([]*nomadapi.AllocationListStub, len(stubs))
	copy(sorted, stubs)

	less := func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		switch s.Column {
		case "node":
			return a.NodeName < b.NodeName
		case "status":
			return a.ClientStatus < b.ClientStatus
		case "desired":
			return a.DesiredStatus < b.DesiredStatus
		case "taskGroup":
			return a.TaskGroup < b.TaskGroup
		case "version":
			return a.JobVersion < b.JobVersion
		case "uptime":
			// Smaller uptime = more recently submitted, i.e. a later SubmitTime.
			return submitTimes[a.JobVersion].After(submitTimes[b.JobVersion])
		default: // "id"
			return a.ID < b.ID
		}
	}

	sort.SliceStable(sorted, func(i, j int) bool {
		if s.Direction == "desc" {
			return less(j, i)
		}
		return less(i, j)
	})

	return sorted
}

// paginate computes the effective page and the [offset, offset+limit) slice
// bounds for a page of size pageSize over totalItems items. Non-positive
// page/pageSize fall back to 1/50; pageSize is capped at 500. If page lands
// beyond the last available page, it's clamped to the last page instead of
// returning an empty slice.
func paginate(page, pageSize, totalItems int) (effectivePage, effectivePageSize, totalPages, offset, limit int) {
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}
	if page <= 0 {
		page = 1
	}

	totalPages = (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset = (page - 1) * pageSize
	limit = pageSize

	return page, pageSize, totalPages, offset, limit
}

func portURL(ip string, port int) string {
	return fmt.Sprintf("http://%s:%d", ip, port)
}

// nodeURL derives the environment/region-specific hostname link for an
// allocation's http-labeled port, per specs/functional_requirements.md.
func nodeURL(env config.Environment, region config.Region, nodeName string, port int) (string, error) {
	switch env {
	case config.EnvironmentStaging:
		return fmt.Sprintf("http://%s.node.%s.staging.mailforce:%d", nodeName, region, port), nil
	case config.EnvironmentProduction:
		shortRegion, err := config.ShortRegion(region)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("http://%s.c.mailforce-production-%s.internal:%d", nodeName, shortRegion, port), nil
	default:
		return "", fmt.Errorf("unknown environment %q", env)
	}
}

// extractPorts builds the ports[] DTO for an allocation, one entry per
// network port, per specs/api.md.
func extractPorts(mappings []nomadapi.PortMapping, nodeName string, profile config.Profile) ([]portDTO, error) {
	ports := make([]portDTO, 0, len(mappings))

	for _, p := range mappings {
		dto := portDTO{
			Label: p.Label,
			IP:    p.HostIP,
			Port:  p.Value,
			URL:   portURL(p.HostIP, p.Value),
		}

		if p.Label == "http" {
			url, err := nodeURL(profile.Environment, profile.Region, nodeName, p.Value)
			if err != nil {
				return nil, err
			}
			dto.NodeURL = url
		}

		ports = append(ports, dto)
	}

	return ports, nil
}

package httpapi

import (
	"fmt"
	"sort"
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

// allocationNodeIP returns the host IP an allocation is running on, derived
// from its allocated network resources. Assumes standard Nomad host
// networking, where the allocation's network IP is the node's own IP.
func allocationNodeIP(alloc *nomadapi.Allocation) string {
	if alloc.AllocatedResources == nil {
		return ""
	}

	networks := alloc.AllocatedResources.Shared.Networks
	if len(networks) > 0 && networks[0] != nil {
		return networks[0].IP
	}

	ports := alloc.AllocatedResources.Shared.Ports
	if len(ports) > 0 {
		return ports[0].HostIP
	}

	return ""
}

// extractPorts builds the ports[] DTO for an allocation, one entry per
// network port, per specs/api.md.
func extractPorts(alloc *nomadapi.Allocation, profile config.Profile) ([]portDTO, error) {
	if alloc.AllocatedResources == nil {
		return nil, nil
	}

	mappings := alloc.AllocatedResources.Shared.Ports
	ports := make([]portDTO, 0, len(mappings))

	for _, p := range mappings {
		dto := portDTO{
			Label: p.Label,
			IP:    p.HostIP,
			Port:  p.Value,
			URL:   portURL(p.HostIP, p.Value),
		}

		if p.Label == "http" {
			url, err := nodeURL(profile.Environment, profile.Region, alloc.NodeName, p.Value)
			if err != nil {
				return nil, err
			}
			dto.NodeURL = url
		}

		ports = append(ports, dto)
	}

	return ports, nil
}

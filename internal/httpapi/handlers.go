package httpapi

import (
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"

	"unhoused/internal/syncutil"
)

// allocationFetchConcurrency bounds how many allocations' details are fetched
// from Nomad concurrently per job-status request.
const allocationFetchConcurrency = 8

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles := make([]profileDTO, 0, len(s.cfg.Profiles))
	for _, p := range s.cfg.Profiles {
		profiles = append(profiles, profileDTO{
			Name:        p.Name,
			Environment: string(p.Environment),
			Region:      string(p.Region),
		})
	}

	writeJSON(w, http.StatusOK, profilesResponse{
		RefreshIntervalSeconds: s.cfg.RefreshIntervalSeconds,
		Profiles:               profiles,
	})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	profileName := r.PathValue("profile")

	_, client, ok := s.profile(profileName)
	if !ok {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}

	stubs, err := client.ListJobs(r.Context())
	if err != nil {
		status, message := classifyNomadErr(err, "job not found")
		writeError(w, status, message)
		return
	}

	jobs := make([]jobListItemDTO, 0, len(stubs))
	for _, stub := range stubs {
		jobs = append(jobs, jobListItemDTO{
			ID:         stub.ID,
			Name:       stub.Name,
			SubmitTime: time.Unix(0, stub.SubmitTime),
		})
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].SubmitTime.After(jobs[j].SubmitTime)
	})

	writeJSON(w, http.StatusOK, jobsResponse{Jobs: jobs})
}

func (s *Server) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	profileName := r.PathValue("profile")
	jobID := r.PathValue("jobId")

	profile, client, ok := s.profile(profileName)
	if !ok {
		writeError(w, http.StatusNotFound, "profile not found")
		return
	}

	ctx := r.Context()

	// job, versions, and allocStubs are independent Nomad calls, fetched
	// concurrently to cut this request's latency roughly to a third.
	var (
		job           *nomadapi.Job
		versions      []*nomadapi.Job
		allocStubs    []*nomadapi.AllocationListStub
		jobErr        error
		versionsErr   error
		allocationErr error
		wg            sync.WaitGroup
	)

	wg.Go(func() {
		job, jobErr = client.JobInfo(ctx, jobID)
	})
	wg.Go(func() {
		versions, versionsErr = client.JobVersions(ctx, jobID)
	})
	wg.Go(func() {
		allocStubs, allocationErr = client.JobAllocations(ctx, jobID)
	})
	wg.Wait()

	if jobErr != nil {
		status, message := classifyNomadErr(jobErr, "job not found")
		writeError(w, status, message)
		return
	}
	if versionsErr != nil {
		status, message := classifyNomadErr(versionsErr, "job not found")
		writeError(w, status, message)
		return
	}
	if allocationErr != nil {
		status, message := classifyNomadErr(allocationErr, "job not found")
		writeError(w, status, message)
		return
	}

	now := s.now()
	submitTimes := versionSubmitTimes(versions)
	// versionGroups and filterOptions reflect ALL of the job's allocations,
	// unaffected by the table filters/pagination below — they represent
	// overall job health and the full set of possible filter values.
	versionGroups := groupByVersion(allocStubs, submitTimes, now)
	filterOptions := allocationFilterOptions(allocStubs)

	query := r.URL.Query()
	filters := allocationFilters{
		Search:    query.Get("q"),
		TaskGroup: query.Get("taskGroup"),
		Version:   query.Get("version"),
		Node:      query.Get("node"),
		Status:    query.Get("status"),
		Desired:   query.Get("desired"),
	}
	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("pageSize"))
	allocSort := allocationSort{
		Column:    query.Get("sort"),
		Direction: query.Get("dir"),
	}

	// Node IPs are only needed to match the search field, and resolving them
	// costs one extra Nomad call (cheap — one call regardless of allocation
	// count — but still skipped unless a search is actually in play).
	var nodeIPs map[string]string
	if filters.Search != "" {
		nodes, err := client.ListNodes(ctx)
		if err != nil {
			status, message := classifyNomadErr(err, "job not found")
			writeError(w, status, message)
			return
		}
		nodeIPs = nodeIPsByNodeID(nodes)
	}

	filteredStubs := filterAllocationStubs(allocStubs, filters, nodeIPs)
	sortedStubs := sortAllocationStubs(filteredStubs, allocSort, submitTimes)
	effectivePage, effectivePageSize, totalPages, offset, limit := paginate(page, pageSize, len(sortedStubs))
	end := offset + limit
	if end > len(sortedStubs) {
		end = len(sortedStubs)
	}
	pageStubs := sortedStubs[offset:end]

	pagination := paginationDTO{
		Page:       effectivePage,
		PageSize:   effectivePageSize,
		TotalItems: len(filteredStubs),
		TotalPages: totalPages,
	}

	// Allocation details are fetched concurrently (bounded by
	// allocationFetchConcurrency) since each is an independent Nomad call.
	// Only the current page's stubs are enriched, not every allocation on the
	// job, so this stays bounded regardless of how many allocations the job
	// has in total. Each closure writes only to its own index of allocations,
	// so no synchronization is needed for those writes; errors are collected
	// by fanOut.Wait() and reported after every fetch has completed, rather
	// than short-circuiting the others (there's no cancellation hook to do
	// otherwise once goroutines are already running).
	allocations := make([]allocationDTO, len(pageStubs))
	fanOut := syncutil.NewFanOut(allocationFetchConcurrency)

	for i, stub := range pageStubs {
		fanOut.Run(func(any) error {
			allocPorts, err := client.GetAllocationPorts(ctx, stub.ID)
			if err != nil {
				return err
			}

			ports, err := extractPorts(allocPorts.Ports, stub.NodeName, profile)
			if err != nil {
				return err
			}

			allocations[i] = allocationDTO{
				ID:                  stub.ID,
				NodeName:            stub.NodeName,
				NodeIP:              allocPorts.NodeIP,
				ClientStatus:        stub.ClientStatus,
				DesiredStatus:       stub.DesiredStatus,
				TaskGroup:           stub.TaskGroup,
				Version:             stub.JobVersion,
				LastModifiedSeconds: lastModifiedSeconds(submitTimes[stub.JobVersion], now),
				Ports:               ports,
			}

			return nil
		}, nil)
	}

	errs := fanOut.Wait()
	if len(errs) > 0 {
		status, message := classifyNomadErr(errs[0], "allocation not found")
		writeError(w, status, message)
		return
	}

	writeJSON(w, http.StatusOK, jobStatusResponse{
		Job: jobDTO{
			ID:     stringVal(job.ID),
			Name:   stringVal(job.Name),
			Status: deriveJobStatus(job),
		},
		VersionGroups: versionGroups,
		Pagination:    pagination,
		FilterOptions: filterOptions,
		Allocations:   allocations,
	})
}

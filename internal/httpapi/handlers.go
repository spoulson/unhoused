package httpapi

import (
	"net/http"
	"sort"
	"time"

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

	job, err := client.JobInfo(ctx, jobID)
	if err != nil {
		status, message := classifyNomadErr(err, "job not found")
		writeError(w, status, message)
		return
	}

	versions, err := client.JobVersions(ctx, jobID)
	if err != nil {
		status, message := classifyNomadErr(err, "job not found")
		writeError(w, status, message)
		return
	}

	allocStubs, err := client.JobAllocations(ctx, jobID)
	if err != nil {
		status, message := classifyNomadErr(err, "job not found")
		writeError(w, status, message)
		return
	}

	now := s.now()
	submitTimes := versionSubmitTimes(versions)
	versionGroups := groupByVersion(allocStubs, submitTimes, now)

	// Allocation details are fetched concurrently (bounded by
	// allocationFetchConcurrency) since each is an independent Nomad call.
	// Each closure writes only to its own index of allocations, so no
	// synchronization is needed for those writes; errors are collected by
	// fanOut.Wait() and reported after every fetch has completed, rather than
	// short-circuiting the others (there's no cancellation hook to do
	// otherwise once goroutines are already running).
	allocations := make([]allocationDTO, len(allocStubs))
	fanOut := syncutil.NewFanOut(allocationFetchConcurrency)

	for i, stub := range allocStubs {
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
				ID:            stub.ID,
				NodeName:      stub.NodeName,
				NodeIP:        allocPorts.NodeIP,
				ClientStatus:  stub.ClientStatus,
				DesiredStatus: stub.DesiredStatus,
				TaskGroup:     stub.TaskGroup,
				Version:       stub.JobVersion,
				UptimeSeconds: uptimeSeconds(submitTimes[stub.JobVersion], now),
				Ports:         ports,
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
		Allocations:   allocations,
	})
}

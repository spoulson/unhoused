package httpapi

import (
	"net/http"
	"sort"
	"time"
)

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

	allocations := make([]allocationDTO, 0, len(allocStubs))
	for _, stub := range allocStubs {
		alloc, err := client.AllocationInfo(ctx, stub.ID)
		if err != nil {
			status, message := classifyNomadErr(err, "allocation not found")
			writeError(w, status, message)
			return
		}

		ports, err := extractPorts(alloc, profile)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}

		allocations = append(allocations, allocationDTO{
			ID:            stub.ID,
			NodeName:      stub.NodeName,
			NodeIP:        allocationNodeIP(alloc),
			ClientStatus:  stub.ClientStatus,
			DesiredStatus: stub.DesiredStatus,
			TaskGroup:     stub.TaskGroup,
			Version:       stub.JobVersion,
			UptimeSeconds: uptimeSeconds(submitTimes[stub.JobVersion], now),
			Ports:         ports,
		})
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

package nomaddemo

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
)

// Server serves the subset of the Nomad HTTP API that unhoused's
// internal/nomadclient.Client calls, backed by fabricated data. It's a pure
// function of wall-clock time: every request recomputes the current state
// from the jobs' definitions rather than reading mutable state, so no
// background goroutine or locking is needed to keep it "running".
type Server struct {
	jobs     []jobDef
	jobsByID map[string]jobDef
	now      func() time.Time
}

// NewServer builds a Server. Job ages ("submitted 3 days ago", etc.) and the
// continuously-redeploying job's cycle are seeded relative to now(), which
// defaults to time.Now if not overridden — tests can inject a fixed clock.
func NewServer(now func() time.Time) *Server {
	if now == nil {
		now = time.Now
	}

	epoch := now()
	jobs := buildJobs(epoch)

	jobsByID := make(map[string]jobDef, len(jobs))
	for _, j := range jobs {
		jobsByID[j.id] = j
	}

	return &Server{jobs: jobs, jobsByID: jobsByID, now: now}
}

// Handler returns the http.Handler serving the simulated Nomad API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/jobs", s.handleListJobs)
	mux.HandleFunc("GET /v1/job/{jobId}", s.handleJobInfo)
	mux.HandleFunc("GET /v1/job/{jobId}/versions", s.handleJobVersions)
	mux.HandleFunc("GET /v1/job/{jobId}/allocations", s.handleJobAllocations)
	mux.HandleFunc("GET /v1/allocation/{allocId}", s.handleAllocationInfo)
	mux.HandleFunc("GET /v1/nodes", s.handleListNodes)
	mux.HandleFunc("GET /v1/deployments", s.handleListDeployments)
	return mux
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	now := s.now()

	stubs := make([]*nomadapi.JobListStub, len(s.jobs))
	for i, j := range s.jobs {
		stubs[i] = &nomadapi.JobListStub{
			ID:         j.id,
			Name:       j.name,
			Status:     j.status,
			Stop:       j.stop,
			SubmitTime: j.submitTime(now).UnixNano(),
		}
	}

	writeJSON(w, stubs)
}

func (s *Server) handleJobInfo(w http.ResponseWriter, r *http.Request) {
	job, ok := s.jobsByID[r.PathValue("jobId")]
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	now := s.now()
	writeJSON(w, &nomadapi.Job{
		ID:         &job.id,
		Name:       &job.name,
		Status:     &job.status,
		Stop:       &job.stop,
		SubmitTime: ptr(job.submitTime(now).UnixNano()),
	})
}

func (s *Server) handleJobVersions(w http.ResponseWriter, r *http.Request) {
	job, ok := s.jobsByID[r.PathValue("jobId")]
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	writeJSON(w, nomadapi.JobVersionsResponse{Versions: job.versions(s.now())})
}

func (s *Server) handleJobAllocations(w http.ResponseWriter, r *http.Request) {
	job, ok := s.jobsByID[r.PathValue("jobId")]
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	allocs := job.allocations(s.now())
	stubs := make([]*nomadapi.AllocationListStub, len(allocs))
	for i, a := range allocs {
		stubs[i] = a.listStub()
	}

	writeJSON(w, stubs)
}

// handleAllocationInfo looks up an allocation by ID across every job, since
// the Nomad API's single-allocation endpoint isn't scoped by job.
func (s *Server) handleAllocationInfo(w http.ResponseWriter, r *http.Request) {
	allocID := r.PathValue("allocId")
	now := s.now()

	for _, job := range s.jobs {
		for _, a := range job.allocations(now) {
			if a.id == allocID {
				writeJSON(w, a.info())
				return
			}
		}
	}

	http.Error(w, "allocation not found", http.StatusNotFound)
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, nodeListStubs())
}

// handleListDeployments returns every job's current deployment (see
// jobDef.deployments), matching Nomad's GET /v1/deployments returning
// deployments across all jobs rather than being scoped to one.
func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	now := s.now()

	var deployments []*nomadapi.Deployment
	for _, j := range s.jobs {
		if j.deployments == nil {
			continue
		}
		deployments = append(deployments, j.deployments(now)...)
	}

	writeJSON(w, deployments)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		log.Printf("nomaddemo: failed to write response: %v", err)
	}
}

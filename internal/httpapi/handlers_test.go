package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"

	"unhoused/internal/config"
	"unhoused/internal/nomadclient"
)

// fakeNomad is a hand-written nomadclient.API fake, per specs/conventions.md
// (no mocking library).
type fakeNomad struct {
	jobs    []*nomadapi.JobListStub
	jobsErr error

	job    *nomadapi.Job
	jobErr error

	versions    []*nomadapi.Job
	versionsErr error

	allocs    []*nomadapi.AllocationListStub
	allocsErr error

	allocInfo    map[string]*nomadapi.Allocation
	allocInfoErr error
}

var _ nomadclient.API = (*fakeNomad)(nil)

func (f *fakeNomad) ListJobs(context.Context) ([]*nomadapi.JobListStub, error) {
	return f.jobs, f.jobsErr
}

func (f *fakeNomad) JobInfo(context.Context, string) (*nomadapi.Job, error) {
	return f.job, f.jobErr
}

func (f *fakeNomad) JobVersions(context.Context, string) ([]*nomadapi.Job, error) {
	return f.versions, f.versionsErr
}

func (f *fakeNomad) JobAllocations(context.Context, string) ([]*nomadapi.AllocationListStub, error) {
	return f.allocs, f.allocsErr
}

func (f *fakeNomad) AllocationInfo(_ context.Context, allocID string) (*nomadapi.Allocation, error) {
	if f.allocInfoErr != nil {
		return nil, f.allocInfoErr
	}
	return f.allocInfo[allocID], nil
}

// realNotFoundErr round-trips a request through the actual Nomad SDK against
// a test server that returns 404, so tests exercise the real
// api.UnexpectedResponseError type classifyNomadErr type-asserts on, rather
// than a hand-rolled stand-in.
func realNotFoundErr(t *testing.T) error {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client, err := nomadapi.NewClient(&nomadapi.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("building nomad client: %v", err)
	}

	_, _, err = client.Jobs().Info("missing", nil)
	if err == nil {
		t.Fatal("expected an error from a 404 response, got nil")
	}

	return err
}

func testConfig() *config.Config {
	return &config.Config{
		RefreshIntervalSeconds: 5,
		Profiles: []config.Profile{
			{Name: "prod-usw1", Environment: config.EnvironmentProduction, Region: config.RegionUSWest1},
			{Name: "staging-euw1", Environment: config.EnvironmentStaging, Region: config.RegionEuropeWest1},
		},
	}
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var v T
	err := json.Unmarshal(rec.Body.Bytes(), &v)
	if err != nil {
		t.Fatalf("decoding response body %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestHandleListProfiles(t *testing.T) {
	srv := NewServer(testConfig(), map[string]nomadclient.API{})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	got := decodeJSON[profilesResponse](t, rec)
	if got.RefreshIntervalSeconds != 5 {
		t.Errorf("RefreshIntervalSeconds = %d, want 5", got.RefreshIntervalSeconds)
	}
	if len(got.Profiles) != 2 {
		t.Fatalf("len(Profiles) = %d, want 2", len(got.Profiles))
	}
	if got.Profiles[0].Name != "prod-usw1" || got.Profiles[0].Environment != "production" || got.Profiles[0].Region != "us-west1" {
		t.Errorf("Profiles[0] = %+v", got.Profiles[0])
	}
}

func TestHandleListJobsUnknownProfile(t *testing.T) {
	srv := NewServer(testConfig(), map[string]nomadclient.API{})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/nope/jobs", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	got := decodeJSON[errorEnvelope](t, rec)
	if got.Error.Message != "profile not found" {
		t.Errorf("error message = %q", got.Error.Message)
	}
}

func TestHandleListJobsSortedNewestFirst(t *testing.T) {
	fake := &fakeNomad{
		jobs: []*nomadapi.JobListStub{
			{ID: "old", Name: "old", SubmitTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()},
			{ID: "new", Name: "new", SubmitTime: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC).UnixNano()},
		},
	}
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	got := decodeJSON[jobsResponse](t, rec)
	if len(got.Jobs) != 2 {
		t.Fatalf("len(Jobs) = %d, want 2", len(got.Jobs))
	}
	if got.Jobs[0].ID != "new" || got.Jobs[1].ID != "old" {
		t.Errorf("Jobs order = %q, %q, want new, old", got.Jobs[0].ID, got.Jobs[1].ID)
	}
}

func TestHandleListJobsNomadError(t *testing.T) {
	fake := &fakeNomad{jobsErr: errors.New("connection refused")}
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}

func TestHandleJobStatusUnknownJob(t *testing.T) {
	fake := &fakeNomad{jobErr: realNotFoundErr(t)}
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/missing", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", rec.Code, rec.Body.String())
	}

	got := decodeJSON[errorEnvelope](t, rec)
	if got.Error.Message != "job not found" {
		t.Errorf("error message = %q", got.Error.Message)
	}
}

func TestHandleJobStatusHappyPath(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	submitTime := now.Add(-1234 * time.Second)

	fake := &fakeNomad{
		job: &nomadapi.Job{
			ID:     ptr("web"),
			Name:   ptr("web"),
			Status: ptr("running"),
			Stop:   ptr(false),
		},
		versions: []*nomadapi.Job{
			{Version: ptr(uint64(3)), SubmitTime: ptr(submitTime.UnixNano())},
		},
		allocs: []*nomadapi.AllocationListStub{
			{
				ID:            "alloc-1",
				NodeName:      "node1",
				JobVersion:    3,
				ClientStatus:  "running",
				DesiredStatus: "run",
				TaskGroup:     "web",
			},
		},
		allocInfo: map[string]*nomadapi.Allocation{
			"alloc-1": {
				NodeName: "node1",
				AllocatedResources: &nomadapi.AllocatedResources{
					Shared: nomadapi.AllocatedSharedResources{
						Networks: []*nomadapi.NetworkResource{{IP: "10.0.0.5"}},
						Ports:    []nomadapi.PortMapping{{Label: "http", Value: 8080, HostIP: "10.0.0.5"}},
					},
				},
			},
		},
	}

	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})
	srv.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	got := decodeJSON[jobStatusResponse](t, rec)

	if got.Job.ID != "web" || got.Job.Status != "running" {
		t.Errorf("Job = %+v", got.Job)
	}

	if len(got.VersionGroups) != 1 {
		t.Fatalf("len(VersionGroups) = %d, want 1", len(got.VersionGroups))
	}
	vg := got.VersionGroups[0]
	if vg.Version != 3 || vg.NewestAllocationUptimeSeconds != 1234 || vg.StatusCounts["running"] != 1 {
		t.Errorf("VersionGroups[0] = %+v", vg)
	}

	if len(got.Allocations) != 1 {
		t.Fatalf("len(Allocations) = %d, want 1", len(got.Allocations))
	}
	alloc := got.Allocations[0]
	if alloc.NodeIP != "10.0.0.5" || alloc.UptimeSeconds != 1234 || alloc.Version != 3 {
		t.Errorf("Allocations[0] = %+v", alloc)
	}
	if len(alloc.Ports) != 1 || alloc.Ports[0].URL != "http://10.0.0.5:8080" {
		t.Errorf("Allocations[0].Ports = %+v", alloc.Ports)
	}
	if alloc.Ports[0].NodeURL != "http://node1.c.mailforce-production-usw1.internal:8080" {
		t.Errorf("Allocations[0].Ports[0].NodeURL = %q", alloc.Ports[0].NodeURL)
	}
}

func TestHandleJobStatusStoppedJob(t *testing.T) {
	fake := &fakeNomad{
		job: &nomadapi.Job{
			ID:     ptr("web"),
			Name:   ptr("web"),
			Status: ptr("running"),
			Stop:   ptr(true),
		},
		versions: []*nomadapi.Job{},
		allocs:   []*nomadapi.AllocationListStub{},
	}

	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	got := decodeJSON[jobStatusResponse](t, rec)
	if got.Job.Status != "stopped" {
		t.Errorf("Job.Status = %q, want stopped", got.Job.Status)
	}
}

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func (f *fakeNomad) GetAllocationPorts(_ context.Context, allocID string) (nomadclient.AllocationPorts, error) {
	if f.allocInfoErr != nil {
		return nomadclient.AllocationPorts{}, f.allocInfoErr
	}

	alloc := f.allocInfo[allocID]
	if alloc == nil || alloc.AllocatedResources == nil {
		return nomadclient.AllocationPorts{}, nil
	}

	result := nomadclient.AllocationPorts{
		Ports: alloc.AllocatedResources.Shared.Ports,
	}

	networks := alloc.AllocatedResources.Shared.Networks
	if len(networks) > 0 && networks[0] != nil {
		result.NodeIP = networks[0].IP
		return result, nil
	}

	if len(result.Ports) > 0 {
		result.NodeIP = result.Ports[0].HostIP
	}

	return result, nil
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
	require.NoError(t, err, "building nomad client")

	_, _, err = client.Jobs().Info("missing", nil)
	require.Error(t, err, "expected an error from a 404 response")

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
	require.NoError(t, err, "decoding response body %q", rec.Body.String())
	return v
}

func TestHandleListProfiles(t *testing.T) {
	srv := NewServer(testConfig(), map[string]nomadclient.API{})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	got := decodeJSON[profilesResponse](t, rec)
	assert.Equal(t, 5, got.RefreshIntervalSeconds)
	require.Len(t, got.Profiles, 2)
	assert.Equal(t, "prod-usw1", got.Profiles[0].Name)
	assert.Equal(t, "production", got.Profiles[0].Environment)
	assert.Equal(t, "us-west1", got.Profiles[0].Region)
}

func TestHandleListJobsUnknownProfile(t *testing.T) {
	srv := NewServer(testConfig(), map[string]nomadclient.API{})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/nope/jobs", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	got := decodeJSON[errorEnvelope](t, rec)
	assert.Equal(t, "profile not found", got.Error.Message)
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

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	got := decodeJSON[jobsResponse](t, rec)
	require.Len(t, got.Jobs, 2)
	assert.Equal(t, "new", got.Jobs[0].ID)
	assert.Equal(t, "old", got.Jobs[1].ID)
}

func TestHandleListJobsNomadError(t *testing.T) {
	fake := &fakeNomad{jobsErr: errors.New("connection refused")}
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestHandleJobStatusUnknownJob(t *testing.T) {
	fake := &fakeNomad{jobErr: realNotFoundErr(t)}
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/missing", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())

	got := decodeJSON[errorEnvelope](t, rec)
	assert.Equal(t, "job not found", got.Error.Message)
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

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	got := decodeJSON[jobStatusResponse](t, rec)

	assert.Equal(t, "web", got.Job.ID)
	assert.Equal(t, "running", got.Job.Status)

	require.Len(t, got.VersionGroups, 1)
	vg := got.VersionGroups[0]
	assert.EqualValues(t, 3, vg.Version)
	assert.Equal(t, int64(1234), vg.NewestAllocationUptimeSeconds)
	assert.Equal(t, 1, vg.StatusCounts["running"])

	require.Len(t, got.Allocations, 1)
	alloc := got.Allocations[0]
	assert.Equal(t, "10.0.0.5", alloc.NodeIP)
	assert.Equal(t, int64(1234), alloc.UptimeSeconds)
	assert.EqualValues(t, 3, alloc.Version)

	require.Len(t, alloc.Ports, 1)
	assert.Equal(t, "http://10.0.0.5:8080", alloc.Ports[0].URL)
	assert.Equal(t, "http://node1.c.mailforce-production-usw1.internal:8080", alloc.Ports[0].NodeURL)
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
	assert.Equal(t, "stopped", got.Job.Status)
}

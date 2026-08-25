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

	nodes    []*nomadapi.NodeListStub
	nodesErr error
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

func (f *fakeNomad) ListNodes(context.Context) ([]*nomadapi.NodeListStub, error) {
	return f.nodes, f.nodesErr
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
	assert.Equal(t, int64(1234), vg.NewestAllocationLastModifiedSeconds)
	assert.Equal(t, 1, vg.StatusCounts["running"])

	require.Len(t, got.Allocations, 1)
	alloc := got.Allocations[0]
	assert.Equal(t, "10.0.0.5", alloc.NodeIP)
	assert.Equal(t, int64(1234), alloc.LastModifiedSeconds)
	assert.EqualValues(t, 3, alloc.Version)

	require.Len(t, alloc.Ports, 1)
	assert.Equal(t, "http://10.0.0.5:8080", alloc.Ports[0].URL)
	assert.Equal(t, "http://node1.c.mailforce-production-usw1.internal:8080", alloc.Ports[0].NodeURL)

	assert.Equal(t, 1, got.Pagination.Page)
	assert.Equal(t, 50, got.Pagination.PageSize)
	assert.Equal(t, 1, got.Pagination.TotalItems)
	assert.Equal(t, 1, got.Pagination.TotalPages)

	assert.Equal(t, []string{"web"}, got.FilterOptions.TaskGroups)
	assert.EqualValues(t, []uint64{3}, got.FilterOptions.Versions)
	assert.Equal(t, []string{"node1"}, got.FilterOptions.Nodes)
}

func multiAllocFake(now time.Time) *fakeNomad {
	submitTimes := map[uint64]int64{
		3: now.Add(-1234 * time.Second).UnixNano(),
		2: now.Add(-5000 * time.Second).UnixNano(),
	}

	nodeIPs := map[string]string{"node1": "10.1.1.1", "node2": "10.1.1.2"}

	allocInfo := make(map[string]*nomadapi.Allocation)
	stubs := make([]*nomadapi.AllocationListStub, 0, 5)
	add := func(id, taskGroup string, version uint64, node, clientStatus string) {
		stubs = append(stubs, &nomadapi.AllocationListStub{
			ID: id, TaskGroup: taskGroup, JobVersion: version, NodeID: "id-" + node, NodeName: node,
			ClientStatus: clientStatus, DesiredStatus: "run",
		})
		allocInfo[id] = &nomadapi.Allocation{
			NodeName: node,
			AllocatedResources: &nomadapi.AllocatedResources{
				Shared: nomadapi.AllocatedSharedResources{
					Networks: []*nomadapi.NetworkResource{{IP: nodeIPs[node]}},
				},
			},
		}
	}

	add("a1", "web", 3, "node1", "running")
	add("a2", "web", 3, "node2", "pending")
	add("a3", "worker", 3, "node1", "running")
	add("a4", "worker", 2, "node2", "failed")
	add("a5", "web", 2, "node1", "running")

	return &fakeNomad{
		job: &nomadapi.Job{ID: ptr("web"), Name: ptr("web"), Status: ptr("running"), Stop: ptr(false)},
		versions: []*nomadapi.Job{
			{Version: ptr(uint64(3)), SubmitTime: ptr(submitTimes[3])},
			{Version: ptr(uint64(2)), SubmitTime: ptr(submitTimes[2])},
		},
		allocs:    stubs,
		allocInfo: allocInfo,
		nodes: []*nomadapi.NodeListStub{
			{ID: "id-node1", Address: nodeIPs["node1"]},
			{ID: "id-node2", Address: nodeIPs["node2"]},
		},
	}
}

func TestHandleJobStatusFiltering(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	fake := multiAllocFake(now)
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})
	srv.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web?taskGroup=web&node=node1", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	got := decodeJSON[jobStatusResponse](t, rec)

	require.Len(t, got.Allocations, 2, "a1 and a5 match taskGroup=web,node=node1")
	ids := []string{got.Allocations[0].ID, got.Allocations[1].ID}
	assert.ElementsMatch(t, []string{"a1", "a5"}, ids)
	assert.Equal(t, 2, got.Pagination.TotalItems)

	// versionGroups and filterOptions stay unfiltered.
	assert.Len(t, got.VersionGroups, 2)
	assert.Equal(t, []string{"web", "worker"}, got.FilterOptions.TaskGroups)
	assert.Equal(t, []string{"node1", "node2"}, got.FilterOptions.Nodes)
}

func TestHandleJobStatusSearch(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	fake := multiAllocFake(now)
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})
	srv.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web?q=NODE1", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	got := decodeJSON[jobStatusResponse](t, rec)

	require.Len(t, got.Allocations, 3, "a1, a3, a5 all run on node1")
	ids := make([]string, len(got.Allocations))
	for i, a := range got.Allocations {
		ids[i] = a.ID
	}
	assert.ElementsMatch(t, []string{"a1", "a3", "a5"}, ids)
	assert.Equal(t, 3, got.Pagination.TotalItems)
}

func TestHandleJobStatusSearchByNodeIP(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	fake := multiAllocFake(now)
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})
	srv.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web?q=10.1.1.2", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	got := decodeJSON[jobStatusResponse](t, rec)

	require.Len(t, got.Allocations, 2, "a2 and a4 both run on node2 (10.1.1.2)")
	ids := []string{got.Allocations[0].ID, got.Allocations[1].ID}
	assert.ElementsMatch(t, []string{"a2", "a4"}, ids)
}

func TestHandleJobStatusSort(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	fake := multiAllocFake(now)
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})
	srv.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web?sort=id&dir=desc&pageSize=200", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	got := decodeJSON[jobStatusResponse](t, rec)

	require.Len(t, got.Allocations, 5)
	ids := make([]string, len(got.Allocations))
	for i, a := range got.Allocations {
		ids[i] = a.ID
	}
	assert.Equal(t, []string{"a5", "a4", "a3", "a2", "a1"}, ids)
}

func TestHandleJobStatusSortInvalidIgnored(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	fake := multiAllocFake(now)
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})
	srv.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web?sort=bogus&dir=desc&pageSize=200", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	got := decodeJSON[jobStatusResponse](t, rec)

	require.Len(t, got.Allocations, 5)
	ids := make([]string, len(got.Allocations))
	for i, a := range got.Allocations {
		ids[i] = a.ID
	}
	assert.Equal(t, []string{"a1", "a2", "a3", "a4", "a5"}, ids, "invalid column falls back to original stub order")
}

func TestHandleJobStatusPagination(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	fake := multiAllocFake(now)
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})
	srv.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web?page=2&pageSize=2", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	got := decodeJSON[jobStatusResponse](t, rec)

	require.Len(t, got.Allocations, 2)
	assert.Equal(t, 2, got.Pagination.Page)
	assert.Equal(t, 2, got.Pagination.PageSize)
	assert.Equal(t, 5, got.Pagination.TotalItems)
	assert.Equal(t, 3, got.Pagination.TotalPages)
}

func TestHandleJobStatusPageBeyondRangeClamps(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	fake := multiAllocFake(now)
	srv := NewServer(testConfig(), map[string]nomadclient.API{"prod-usw1": fake})
	srv.now = func() time.Time { return now }

	req := httptest.NewRequest(http.MethodGet, "/api/profiles/prod-usw1/jobs/web?page=99&pageSize=2", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	got := decodeJSON[jobStatusResponse](t, rec)

	assert.Equal(t, 3, got.Pagination.Page, "clamps to last valid page")
	require.Len(t, got.Allocations, 1, "last page has the remaining 1 of 5 allocations")
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

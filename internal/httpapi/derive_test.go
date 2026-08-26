package httpapi

import (
	"testing"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"unhoused/internal/config"
)

func ptr[T any](v T) *T { return &v }

func TestDeriveJobStatus(t *testing.T) {
	tests := []struct {
		name string
		job  *nomadapi.Job
		want string
	}{
		{"stopped overrides status", &nomadapi.Job{Stop: ptr(true), Status: ptr("running")}, "stopped"},
		{"running passthrough", &nomadapi.Job{Stop: ptr(false), Status: ptr("running")}, "running"},
		{"pending passthrough", &nomadapi.Job{Stop: ptr(false), Status: ptr("pending")}, "pending"},
		{"dead passthrough", &nomadapi.Job{Stop: ptr(false), Status: ptr("dead")}, "dead"},
		{"nil stop, running", &nomadapi.Job{Status: ptr("running")}, "running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveJobStatus(tt.job)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLastModifiedSeconds(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		submitTime time.Time
		want       int64
	}{
		{"1234 seconds ago", now.Add(-1234 * time.Second), 1234},
		{"exactly now", now, 0},
		{"future submit time clamps to 0", now.Add(1 * time.Hour), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastModifiedSeconds(tt.submitTime, now)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersionSubmitTimes(t *testing.T) {
	versions := []*nomadapi.Job{
		{Version: ptr(uint64(3)), SubmitTime: ptr(int64(3_000_000_000))},
		{Version: ptr(uint64(2)), SubmitTime: ptr(int64(2_000_000_000))},
		nil,
		{Version: nil, SubmitTime: ptr(int64(9_000_000_000))},
	}

	got := versionSubmitTimes(versions)

	require.Len(t, got, 2)
	assert.True(t, got[3].Equal(time.Unix(0, 3_000_000_000)), "versionSubmitTimes()[3] = %v", got[3])
	assert.True(t, got[2].Equal(time.Unix(0, 2_000_000_000)), "versionSubmitTimes()[2] = %v", got[2])
}

func TestGroupByVersion(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	submitTimes := map[uint64]time.Time{
		3: now.Add(-1234 * time.Second),
		2: now.Add(-5000 * time.Second),
	}

	allocs := []*nomadapi.AllocationListStub{
		{JobVersion: 3, ClientStatus: "running"},
		{JobVersion: 3, ClientStatus: "running"},
		{JobVersion: 3, ClientStatus: "pending"},
		{JobVersion: 2, ClientStatus: "lost"},
		{JobVersion: 2, ClientStatus: "unknown-status"},
	}

	got := groupByVersion(allocs, submitTimes, now)

	require.Len(t, got, 2)

	// Sorted newest-first.
	require.Equal(t, uint64(3), got[0].Version)
	require.Equal(t, uint64(2), got[1].Version)

	v3 := got[0]
	assert.Equal(t, int64(1234), v3.NewestAllocationLastModifiedSeconds)
	assert.Equal(t, 2, v3.StatusCounts["running"])
	assert.Equal(t, 1, v3.StatusCounts["pending"])
	assert.Equal(t, 0, v3.StatusCounts["failed"], "present as zero key")

	v2 := got[1]
	assert.Equal(t, 1, v2.StatusCounts["lost"])
	// Unknown client statuses are dropped rather than growing the map.
	assert.Len(t, v2.StatusCounts, len(clientStatuses))
}

func TestPortAddress(t *testing.T) {
	got := portAddress("10.0.0.5", 8080)
	assert.Equal(t, "10.0.0.5:8080", got)
}

func TestNodeAddress(t *testing.T) {
	tests := []struct {
		name     string
		env      config.Environment
		region   config.Region
		nodeName string
		port     int
		want     string
		wantErr  bool
	}{
		{
			name: "staging", env: config.EnvironmentStaging, region: config.RegionUSWest1,
			nodeName: "node1", port: 8080,
			want: "node1.node.us-west1.staging.mailforce:8080",
		},
		{
			name: "production", env: config.EnvironmentProduction, region: config.RegionUSWest1,
			nodeName: "node1", port: 8080,
			want: "node1.c.mailforce-production-usw1.internal:8080",
		},
		{
			name: "production europe-west1", env: config.EnvironmentProduction, region: config.RegionEuropeWest1,
			nodeName: "node2", port: 443,
			want: "node2.c.mailforce-production-euw1.internal:443",
		},
		{
			name: "unknown environment", env: config.Environment("dev"), region: config.RegionUSWest1,
			nodeName: "node1", port: 8080,
			wantErr: true,
		},
		{
			name: "unknown region", env: config.EnvironmentProduction, region: config.Region("mars"),
			nodeName: "node1", port: 8080,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nodeAddress(tt.env, tt.region, tt.nodeName, tt.port)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractPorts(t *testing.T) {
	profile := config.Profile{
		Environment: config.EnvironmentStaging,
		Region:      config.RegionUSWest1,
	}

	ports := []nomadapi.PortMapping{
		{Label: "app", Value: 8080, HostIP: "10.0.0.5"},
		{Label: "metrics", Value: 9090, HostIP: "10.0.0.5"},
	}

	got, err := extractPorts(ports, "node1", profile)
	require.NoError(t, err)
	require.Len(t, got, 2)

	appPort := got[0]
	assert.Equal(t, "10.0.0.5:8080", appPort.Address)
	assert.Equal(t, "node1.node.us-west1.staging.mailforce:8080", appPort.NodeAddress)

	metricsPort := got[1]
	assert.Equal(t, "10.0.0.5:9090", metricsPort.Address)
	assert.Equal(t, "node1.node.us-west1.staging.mailforce:9090", metricsPort.NodeAddress)
}

func TestExtractPortsNoPorts(t *testing.T) {
	got, err := extractPorts(nil, "node1", config.Profile{})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func testStubs() []*nomadapi.AllocationListStub {
	return []*nomadapi.AllocationListStub{
		{ID: "a1", TaskGroup: "web", JobVersion: 3, NodeID: "n1", NodeName: "node1", ClientStatus: "running", DesiredStatus: "run"},
		{ID: "a2", TaskGroup: "web", JobVersion: 3, NodeID: "n2", NodeName: "node2", ClientStatus: "pending", DesiredStatus: "run"},
		{ID: "a3", TaskGroup: "worker", JobVersion: 2, NodeID: "n1", NodeName: "node1", ClientStatus: "failed", DesiredStatus: "stop"},
	}
}

func testNodeIPs() map[string]string {
	return map[string]string{"n1": "10.0.0.1", "n2": "10.0.0.2"}
}

func TestFilterAllocationStubs(t *testing.T) {
	stubs := testStubs()
	nodeIPs := testNodeIPs()

	tests := []struct {
		name    string
		filters allocationFilters
		wantIDs []string
	}{
		{"no filters", allocationFilters{}, []string{"a1", "a2", "a3"}},
		{"task group", allocationFilters{TaskGroup: "web"}, []string{"a1", "a2"}},
		{"version", allocationFilters{Version: "2"}, []string{"a3"}},
		{"node", allocationFilters{Node: "node1"}, []string{"a1", "a3"}},
		{"status", allocationFilters{Status: "pending"}, []string{"a2"}},
		{"desired", allocationFilters{Desired: "stop"}, []string{"a3"}},
		{"combined, no match", allocationFilters{TaskGroup: "web", Node: "node1", Status: "pending"}, []string{}},
		{"combined, match", allocationFilters{TaskGroup: "web", Node: "node1"}, []string{"a1"}},
		{"search matches allocation ID substring", allocationFilters{Search: "a1"}, []string{"a1"}},
		{"search matches node name substring, case-insensitive", allocationFilters{Search: "NODE1"}, []string{"a1", "a3"}},
		{"search matches node IP substring", allocationFilters{Search: "10.0.0.2"}, []string{"a2"}},
		{"search matches across all fields", allocationFilters{Search: "a"}, []string{"a1", "a2", "a3"}},
		{"search combined with other filter", allocationFilters{Search: "node1", Status: "failed"}, []string{"a3"}},
		{"search, no match", allocationFilters{Search: "nope"}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterAllocationStubs(stubs, tt.filters, nodeIPs)

			gotIDs := make([]string, len(got))
			for i, s := range got {
				gotIDs[i] = s.ID
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestNodeIPsByNodeID(t *testing.T) {
	nodes := []*nomadapi.NodeListStub{
		{ID: "n1", Address: "10.0.0.1"},
		{ID: "n2", Address: "10.0.0.2"},
	}

	got := nodeIPsByNodeID(nodes)

	assert.Equal(t, map[string]string{"n1": "10.0.0.1", "n2": "10.0.0.2"}, got)
}

func TestAllocationFilterOptions(t *testing.T) {
	got := allocationFilterOptions(testStubs())

	assert.Equal(t, []string{"web", "worker"}, got.TaskGroups)
	assert.Equal(t, []uint64{3, 2}, got.Versions, "versions sorted numerically descending")
	assert.Equal(t, []string{"node1", "node2"}, got.Nodes)
}

func TestAllocationFilterOptionsEmpty(t *testing.T) {
	got := allocationFilterOptions(nil)

	assert.Empty(t, got.TaskGroups)
	assert.Empty(t, got.Versions)
	assert.Empty(t, got.Nodes)
}

func TestSortAllocationStubs(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	submitTimes := map[uint64]time.Time{
		1: now.Add(-5000 * time.Second), // older submit -> bigger lastModified
		2: now.Add(-1234 * time.Second), // newer submit -> smaller lastModified
	}

	s1 := &nomadapi.AllocationListStub{ID: "b", NodeName: "nodeB", ClientStatus: "pending", DesiredStatus: "stop", TaskGroup: "web", JobVersion: 1}
	s2 := &nomadapi.AllocationListStub{ID: "a", NodeName: "nodeA", ClientStatus: "running", DesiredStatus: "run", TaskGroup: "worker", JobVersion: 2}
	stubs := []*nomadapi.AllocationListStub{s1, s2}

	tests := []struct {
		name    string
		sort    allocationSort
		wantIDs []string
	}{
		{"id asc", allocationSort{Column: "id", Direction: "asc"}, []string{"a", "b"}},
		{"id desc", allocationSort{Column: "id", Direction: "desc"}, []string{"b", "a"}},
		{"node asc", allocationSort{Column: "node", Direction: "asc"}, []string{"a", "b"}},
		{"status asc", allocationSort{Column: "status", Direction: "asc"}, []string{"b", "a"}},
		{"desired asc", allocationSort{Column: "desired", Direction: "asc"}, []string{"a", "b"}},
		{"taskGroup asc", allocationSort{Column: "taskGroup", Direction: "asc"}, []string{"b", "a"}},
		{"version asc", allocationSort{Column: "version", Direction: "asc"}, []string{"b", "a"}},
		{"version desc", allocationSort{Column: "version", Direction: "desc"}, []string{"a", "b"}},
		{"lastModified asc (smallest lastModified = most recent submit first)", allocationSort{Column: "lastModified", Direction: "asc"}, []string{"a", "b"}},
		{"lastModified desc", allocationSort{Column: "lastModified", Direction: "desc"}, []string{"b", "a"}},
		{"invalid column, unsorted", allocationSort{Column: "bogus", Direction: "asc"}, []string{"b", "a"}},
		{"invalid direction, unsorted", allocationSort{Column: "id", Direction: "sideways"}, []string{"b", "a"}},
		{"empty sort, unsorted", allocationSort{}, []string{"b", "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortAllocationStubs(stubs, tt.sort, submitTimes)

			gotIDs := make([]string, len(got))
			for i, s := range got {
				gotIDs[i] = s.ID
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestPaginate(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		totalItems   int
		wantPage     int
		wantPageSize int
		wantTotal    int
		wantOffset   int
		wantLimit    int
	}{
		{"first page", 1, 10, 25, 1, 10, 3, 0, 10},
		{"middle page", 2, 10, 25, 2, 10, 3, 10, 10},
		{"last page", 3, 10, 25, 3, 10, 3, 20, 10},
		{"page beyond range clamps to last", 99, 10, 25, 3, 10, 3, 20, 10},
		{"zero page defaults to 1", 0, 10, 25, 1, 10, 3, 0, 10},
		{"negative page defaults to 1", -5, 10, 25, 1, 10, 3, 0, 10},
		{"zero pageSize defaults to 50", 1, 0, 25, 1, 50, 1, 0, 50},
		{"pageSize clamps to 500 max", 1, 10000, 600, 1, 500, 2, 0, 500},
		{"no items still reports 1 total page", 1, 10, 0, 1, 10, 1, 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPage, gotPageSize, gotTotalPages, gotOffset, gotLimit := paginate(tt.page, tt.pageSize, tt.totalItems)

			assert.Equal(t, tt.wantPage, gotPage, "page")
			assert.Equal(t, tt.wantPageSize, gotPageSize, "pageSize")
			assert.Equal(t, tt.wantTotal, gotTotalPages, "totalPages")
			assert.Equal(t, tt.wantOffset, gotOffset, "offset")
			assert.Equal(t, tt.wantLimit, gotLimit, "limit")
		})
	}
}

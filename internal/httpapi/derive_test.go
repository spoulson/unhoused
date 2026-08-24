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

func TestUptimeSeconds(t *testing.T) {
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
			got := uptimeSeconds(tt.submitTime, now)
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
	assert.Equal(t, int64(1234), v3.NewestAllocationUptimeSeconds)
	assert.Equal(t, 2, v3.StatusCounts["running"])
	assert.Equal(t, 1, v3.StatusCounts["pending"])
	assert.Equal(t, 0, v3.StatusCounts["failed"], "present as zero key")

	v2 := got[1]
	assert.Equal(t, 1, v2.StatusCounts["lost"])
	// Unknown client statuses are dropped rather than growing the map.
	assert.Len(t, v2.StatusCounts, len(clientStatuses))
}

func TestPortURL(t *testing.T) {
	got := portURL("10.0.0.5", 8080)
	assert.Equal(t, "http://10.0.0.5:8080", got)
}

func TestNodeURL(t *testing.T) {
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
			want: "http://node1.node.us-west1.staging.mailforce:8080",
		},
		{
			name: "production", env: config.EnvironmentProduction, region: config.RegionUSWest1,
			nodeName: "node1", port: 8080,
			want: "http://node1.c.mailforce-production-usw1.internal:8080",
		},
		{
			name: "production europe-west1", env: config.EnvironmentProduction, region: config.RegionEuropeWest1,
			nodeName: "node2", port: 443,
			want: "http://node2.c.mailforce-production-euw1.internal:443",
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
			got, err := nodeURL(tt.env, tt.region, tt.nodeName, tt.port)

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
		{Label: "http", Value: 8080, HostIP: "10.0.0.5"},
		{Label: "metrics", Value: 9090, HostIP: "10.0.0.5"},
	}

	got, err := extractPorts(ports, "node1", profile)
	require.NoError(t, err)
	require.Len(t, got, 2)

	httpPort := got[0]
	assert.Equal(t, "http://10.0.0.5:8080", httpPort.URL)
	assert.Equal(t, "http://node1.node.us-west1.staging.mailforce:8080", httpPort.NodeURL)

	metricsPort := got[1]
	assert.Empty(t, metricsPort.NodeURL)
}

func TestExtractPortsNoPorts(t *testing.T) {
	got, err := extractPorts(nil, "node1", config.Profile{})
	require.NoError(t, err)
	assert.Empty(t, got)
}

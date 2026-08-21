package httpapi

import (
	"testing"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"

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
			if got != tt.want {
				t.Errorf("deriveJobStatus() = %q, want %q", got, tt.want)
			}
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
			if got != tt.want {
				t.Errorf("uptimeSeconds() = %d, want %d", got, tt.want)
			}
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

	if len(got) != 2 {
		t.Fatalf("versionSubmitTimes() returned %d entries, want 2", len(got))
	}
	if !got[3].Equal(time.Unix(0, 3_000_000_000)) {
		t.Errorf("versionSubmitTimes()[3] = %v", got[3])
	}
	if !got[2].Equal(time.Unix(0, 2_000_000_000)) {
		t.Errorf("versionSubmitTimes()[2] = %v", got[2])
	}
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

	if len(got) != 2 {
		t.Fatalf("groupByVersion() returned %d groups, want 2", len(got))
	}

	// Sorted newest-first.
	if got[0].Version != 3 || got[1].Version != 2 {
		t.Fatalf("groupByVersion() order = %d, %d, want 3, 2", got[0].Version, got[1].Version)
	}

	v3 := got[0]
	if v3.NewestAllocationUptimeSeconds != 1234 {
		t.Errorf("v3 uptime = %d, want 1234", v3.NewestAllocationUptimeSeconds)
	}
	if v3.StatusCounts["running"] != 2 {
		t.Errorf("v3 running count = %d, want 2", v3.StatusCounts["running"])
	}
	if v3.StatusCounts["pending"] != 1 {
		t.Errorf("v3 pending count = %d, want 1", v3.StatusCounts["pending"])
	}
	if v3.StatusCounts["failed"] != 0 {
		t.Errorf("v3 failed count = %d, want 0 (present as zero key)", v3.StatusCounts["failed"])
	}

	v2 := got[1]
	if v2.StatusCounts["lost"] != 1 {
		t.Errorf("v2 lost count = %d, want 1", v2.StatusCounts["lost"])
	}
	// Unknown client statuses are dropped rather than growing the map.
	if len(v2.StatusCounts) != len(clientStatuses) {
		t.Errorf("v2 StatusCounts has %d keys, want %d", len(v2.StatusCounts), len(clientStatuses))
	}
}

func TestPortURL(t *testing.T) {
	got := portURL("10.0.0.5", 8080)
	want := "http://10.0.0.5:8080"
	if got != want {
		t.Errorf("portURL() = %q, want %q", got, want)
	}
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
				if err == nil {
					t.Fatalf("nodeURL() returned no error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("nodeURL() returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("nodeURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAllocationNodeIP(t *testing.T) {
	tests := []struct {
		name  string
		alloc *nomadapi.Allocation
		want  string
	}{
		{
			name:  "no allocated resources",
			alloc: &nomadapi.Allocation{},
			want:  "",
		},
		{
			name: "from network resource",
			alloc: &nomadapi.Allocation{
				AllocatedResources: &nomadapi.AllocatedResources{
					Shared: nomadapi.AllocatedSharedResources{
						Networks: []*nomadapi.NetworkResource{{IP: "10.0.0.5"}},
					},
				},
			},
			want: "10.0.0.5",
		},
		{
			name: "falls back to port host ip",
			alloc: &nomadapi.Allocation{
				AllocatedResources: &nomadapi.AllocatedResources{
					Shared: nomadapi.AllocatedSharedResources{
						Ports: []nomadapi.PortMapping{{HostIP: "10.0.0.9"}},
					},
				},
			},
			want: "10.0.0.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := allocationNodeIP(tt.alloc)
			if got != tt.want {
				t.Errorf("allocationNodeIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractPorts(t *testing.T) {
	profile := config.Profile{
		Environment: config.EnvironmentStaging,
		Region:      config.RegionUSWest1,
	}

	alloc := &nomadapi.Allocation{
		NodeName: "node1",
		AllocatedResources: &nomadapi.AllocatedResources{
			Shared: nomadapi.AllocatedSharedResources{
				Ports: []nomadapi.PortMapping{
					{Label: "http", Value: 8080, HostIP: "10.0.0.5"},
					{Label: "metrics", Value: 9090, HostIP: "10.0.0.5"},
				},
			},
		},
	}

	got, err := extractPorts(alloc, profile)
	if err != nil {
		t.Fatalf("extractPorts() returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("extractPorts() returned %d ports, want 2", len(got))
	}

	httpPort := got[0]
	if httpPort.URL != "http://10.0.0.5:8080" {
		t.Errorf("http port URL = %q", httpPort.URL)
	}
	if httpPort.NodeURL != "http://node1.node.us-west1.staging.mailforce:8080" {
		t.Errorf("http port NodeURL = %q", httpPort.NodeURL)
	}

	metricsPort := got[1]
	if metricsPort.NodeURL != "" {
		t.Errorf("non-http port NodeURL = %q, want empty", metricsPort.NodeURL)
	}
}

func TestExtractPortsNoAllocatedResources(t *testing.T) {
	got, err := extractPorts(&nomadapi.Allocation{}, config.Profile{})
	if err != nil {
		t.Fatalf("extractPorts() returned error: %v", err)
	}
	if got != nil {
		t.Errorf("extractPorts() = %v, want nil", got)
	}
}

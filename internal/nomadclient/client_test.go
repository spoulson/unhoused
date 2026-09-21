package nomadclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, ports []nomadapi.PortMapping, nodeIP string) (*httptest.Server, *int32) {
	t.Helper()

	var requestCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		alloc := nomadapi.Allocation{
			ID: "alloc-1",
			AllocatedResources: &nomadapi.AllocatedResources{
				Shared: nomadapi.AllocatedSharedResources{
					Networks: []*nomadapi.NetworkResource{{IP: nodeIP}},
					Ports:    ports,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(alloc)
		assert.NoError(t, err, "encoding response")
	}))
	t.Cleanup(server.Close)

	return server, &requestCount
}

func TestGetAllocationPortsFetchesAndExtractsPorts(t *testing.T) {
	wantPorts := []nomadapi.PortMapping{{Label: "http", Value: 8080, HostIP: "10.0.0.5"}}
	server, _ := newTestServer(t, wantPorts, "10.0.0.5")

	client, err := New(server.URL, "")
	require.NoError(t, err)

	got, err := client.GetAllocationPorts(context.Background(), "alloc-1")
	require.NoError(t, err)

	assert.Equal(t, wantPorts, got.Ports)
	assert.Equal(t, "10.0.0.5", got.NodeIP)
}

func TestGetAllocationPortsCachesRepeatedCalls(t *testing.T) {
	wantPorts := []nomadapi.PortMapping{{Label: "http", Value: 8080, HostIP: "10.0.0.5"}}
	server, requestCount := newTestServer(t, wantPorts, "10.0.0.5")

	client, err := New(server.URL, "")
	require.NoError(t, err)

	ctx := context.Background()

	_, err = client.GetAllocationPorts(ctx, "alloc-1")
	require.NoError(t, err, "first GetAllocationPorts call")

	_, err = client.GetAllocationPorts(ctx, "alloc-1")
	require.NoError(t, err, "second GetAllocationPorts call")

	got := atomic.LoadInt32(requestCount)
	assert.Equal(t, int32(1), got, "second call should be served from cache")
}

func TestDockerImageFromJob(t *testing.T) {
	tests := []struct {
		name string
		job  *nomadapi.Job
		want string
	}{
		{"nil job", nil, ""},
		{"no task groups", &nomadapi.Job{}, ""},
		{
			"docker task",
			&nomadapi.Job{TaskGroups: []*nomadapi.TaskGroup{
				{Tasks: []*nomadapi.Task{{Driver: "docker", Config: map[string]any{"image": "myrepo/web:1.2.3"}}}},
			}},
			"myrepo/web:1.2.3",
		},
		{
			"non-docker driver only",
			&nomadapi.Job{TaskGroups: []*nomadapi.TaskGroup{
				{Tasks: []*nomadapi.Task{{Driver: "exec", Config: map[string]any{"command": "/bin/true"}}}},
			}},
			"",
		},
		{
			"first match across multiple task groups",
			&nomadapi.Job{TaskGroups: []*nomadapi.TaskGroup{
				{Tasks: []*nomadapi.Task{{Driver: "exec"}}},
				{Tasks: []*nomadapi.Task{
					{Driver: "docker", Config: map[string]any{"image": "myrepo/first:1"}},
					{Driver: "docker", Config: map[string]any{"image": "myrepo/second:1"}},
				}},
			}},
			"myrepo/first:1",
		},
		{
			"nil task group and task tolerated",
			&nomadapi.Job{TaskGroups: []*nomadapi.TaskGroup{
				nil,
				{Tasks: []*nomadapi.Task{nil, {Driver: "docker", Config: map[string]any{"image": "myrepo/web:1"}}}},
			}},
			"myrepo/web:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DockerImageFromJob(tt.job)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVersionTaggedTime(t *testing.T) {
	tests := []struct {
		name     string
		job      *nomadapi.Job
		wantTime time.Time
		wantOK   bool
	}{
		{"nil job", nil, time.Time{}, false},
		{"no version tag", &nomadapi.Job{}, time.Time{}, false},
		{
			"tagged",
			&nomadapi.Job{VersionTag: &nomadapi.JobVersionTag{Name: "release", TaggedTime: 3_000_000_000}},
			time.Unix(0, 3_000_000_000),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotOK := VersionTaggedTime(tt.job)
			assert.Equal(t, tt.wantOK, gotOK)
			assert.True(t, tt.wantTime.Equal(gotTime), "VersionTaggedTime() = %v, want %v", gotTime, tt.wantTime)
		})
	}
}

func TestJobVersionDockerImageCachesRepeatedCalls(t *testing.T) {
	client, err := New("http://127.0.0.1:0", "")
	require.NoError(t, err)

	job := &nomadapi.Job{TaskGroups: []*nomadapi.TaskGroup{
		{Tasks: []*nomadapi.Task{{Driver: "docker", Config: map[string]any{"image": "myrepo/web:1.2.3"}}}},
	}}

	got := client.JobVersionDockerImage("web", 3, job)
	assert.Equal(t, "myrepo/web:1.2.3", got)

	// Second call passes a different job spec for the same jobID/version; the
	// cached value should win, since it's ignored on a cache hit.
	staleJob := &nomadapi.Job{TaskGroups: []*nomadapi.TaskGroup{
		{Tasks: []*nomadapi.Task{{Driver: "docker", Config: map[string]any{"image": "should-not-be-seen:1"}}}},
	}}
	got = client.JobVersionDockerImage("web", 3, staleJob)
	assert.Equal(t, "myrepo/web:1.2.3", got, "second call should be served from cache")
}

func TestGetAllocationPortsNoAllocatedResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		alloc := nomadapi.Allocation{ID: "alloc-1"}
		w.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(w).Encode(alloc)
		assert.NoError(t, err, "encoding response")
	}))
	t.Cleanup(server.Close)

	client, err := New(server.URL, "")
	require.NoError(t, err)

	got, err := client.GetAllocationPorts(context.Background(), "alloc-1")
	require.NoError(t, err)

	assert.Nil(t, got.Ports)
	assert.Empty(t, got.NodeIP)
}

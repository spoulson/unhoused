package nomadclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

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

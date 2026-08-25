// Package nomadclient wraps the subset of the Nomad HTTP API used by
// unhoused behind a small interface, so callers can be tested against a
// hand-written fake instead of a real Nomad server.
package nomadclient

import (
	"context"
	"net/http"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
)

// API is the Nomad functionality unhoused depends on.
type API interface {
	ListJobs(ctx context.Context) ([]*nomadapi.JobListStub, error)
	JobInfo(ctx context.Context, jobID string) (*nomadapi.Job, error)
	JobVersions(ctx context.Context, jobID string) ([]*nomadapi.Job, error)
	JobAllocations(ctx context.Context, jobID string) ([]*nomadapi.AllocationListStub, error)
	AllocationInfo(ctx context.Context, allocID string) (*nomadapi.Allocation, error)
	GetAllocationPorts(ctx context.Context, allocID string) (AllocationPorts, error)
	ListNodes(ctx context.Context) ([]*nomadapi.NodeListStub, error)
}

const (
	allocationPortsCacheCapacity = 512
	allocationPortsCacheTTL      = 5 * time.Minute
)

// AllocationPorts is the network-reachability info for an allocation: its
// assigned ports and the IP of the node it's running on.
type AllocationPorts struct {
	Ports  []nomadapi.PortMapping
	NodeIP string
}

// Client is the real API implementation backed by the Nomad SDK.
type Client struct {
	nomad           *nomadapi.Client
	allocationPorts *lruCache[string, AllocationPorts]
}

var _ API = (*Client)(nil)

// New creates a Client scoped to a single Nomad cluster at addr, authenticating with token.
//
// A custom HttpClient is supplied so every request/response can be logged (see transport.go).
// This bypasses Nomad SDK's own TLS auto-configuration (api.ConfigureTLS), which only matters for
// https Nomad addresses using custom certificates — not used by any profile in
// specs/configuration.md today. If that's needed later, TLS config would need to be threaded
// through here alongside the logging transport.
func New(addr, token string) (*Client, error) {
	httpClient := &http.Client{
		Transport: &loggingTransport{},
	}

	nomad, err := nomadapi.NewClient(&nomadapi.Config{
		Address:    addr,
		SecretID:   token,
		HttpClient: httpClient,
	})
	if err != nil {
		return nil, err
	}

	client := &Client{
		nomad:           nomad,
		allocationPorts: newLRUCache[string, AllocationPorts](allocationPortsCacheCapacity, allocationPortsCacheTTL),
	}

	return client, nil
}

func (c *Client) ListJobs(ctx context.Context) ([]*nomadapi.JobListStub, error) {
	q := (&nomadapi.QueryOptions{}).WithContext(ctx)

	jobs, _, err := c.nomad.Jobs().List(q)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func (c *Client) JobInfo(ctx context.Context, jobID string) (*nomadapi.Job, error) {
	q := (&nomadapi.QueryOptions{}).WithContext(ctx)

	job, _, err := c.nomad.Jobs().Info(jobID, q)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (c *Client) JobVersions(ctx context.Context, jobID string) ([]*nomadapi.Job, error) {
	q := (&nomadapi.QueryOptions{}).WithContext(ctx)

	versions, _, _, err := c.nomad.Jobs().Versions(jobID, false, q)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

func (c *Client) JobAllocations(ctx context.Context, jobID string) ([]*nomadapi.AllocationListStub, error) {
	q := (&nomadapi.QueryOptions{}).WithContext(ctx)

	allocs, _, err := c.nomad.Jobs().Allocations(jobID, false, q)
	if err != nil {
		return nil, err
	}

	return allocs, nil
}

func (c *Client) AllocationInfo(ctx context.Context, allocID string) (*nomadapi.Allocation, error) {
	q := (&nomadapi.QueryOptions{}).WithContext(ctx)

	alloc, _, err := c.nomad.Allocations().Info(allocID, q)
	if err != nil {
		return nil, err
	}

	return alloc, nil
}

// GetAllocationPorts returns the network ports assigned to an allocation and
// the IP of the node it's running on, fetched via AllocationInfo. Results are
// cached per allocation ID for allocationPortsCacheTTL, since this
// information doesn't change for an allocation's lifetime.
func (c *Client) GetAllocationPorts(ctx context.Context, allocID string) (AllocationPorts, error) {
	cached, ok := c.allocationPorts.Get(allocID)
	if ok {
		return cached, nil
	}

	alloc, err := c.AllocationInfo(ctx, allocID)
	if err != nil {
		return AllocationPorts{}, err
	}

	result := AllocationPorts{
		NodeIP: nodeIPFromAllocation(alloc),
	}
	if alloc.AllocatedResources != nil {
		result.Ports = alloc.AllocatedResources.Shared.Ports
	}

	c.allocationPorts.Set(allocID, result)

	return result, nil
}

// ListNodes returns the cluster's nodes. Used to resolve node IPs for the
// Job Status Page's search, without an AllocationInfo call per allocation:
// this is a single Nomad call regardless of how many allocations the job has.
func (c *Client) ListNodes(ctx context.Context) ([]*nomadapi.NodeListStub, error) {
	q := (&nomadapi.QueryOptions{}).WithContext(ctx)

	nodes, _, err := c.nomad.Nodes().List(q)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// nodeIPFromAllocation returns the host IP an allocation is running on,
// derived from its allocated network resources. Assumes standard Nomad host
// networking, where the allocation's network IP is the node's own IP.
func nodeIPFromAllocation(alloc *nomadapi.Allocation) string {
	if alloc.AllocatedResources == nil {
		return ""
	}

	networks := alloc.AllocatedResources.Shared.Networks
	if len(networks) > 0 && networks[0] != nil {
		return networks[0].IP
	}

	ports := alloc.AllocatedResources.Shared.Ports
	if len(ports) > 0 {
		return ports[0].HostIP
	}

	return ""
}

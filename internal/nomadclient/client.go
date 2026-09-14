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
	JobVersionDockerImage(jobID string, version uint64, job *nomadapi.Job) string
	ListNodes(ctx context.Context) ([]*nomadapi.NodeListStub, error)
}

const (
	allocationPortsCacheCapacity = 512
	// allocationPortsCacheTTL is 0 (never expire): an allocation's ports and
	// node IP never change once assigned, so cached entries stay valid until
	// evicted by LRU capacity overflow rather than on a timer.
	allocationPortsCacheTTL = 0

	jobVersionImageCacheCapacity = 2048
	// jobVersionImageCacheTTL is 0 (never expire): a job version's task
	// config, and thus its Docker image, is immutable once that version
	// exists, so cached entries stay valid until evicted by LRU capacity
	// overflow rather than on a timer.
	jobVersionImageCacheTTL = 0
)

// jobVersionKey identifies a single job version for jobVersionImages caching.
type jobVersionKey struct {
	jobID   string
	version uint64
}

// AllocationPorts is the network-reachability info for an allocation: its
// assigned ports and the IP of the node it's running on.
type AllocationPorts struct {
	Ports  []nomadapi.PortMapping
	NodeIP string
}

// Client is the real API implementation backed by the Nomad SDK.
type Client struct {
	nomad            *nomadapi.Client
	allocationPorts  *lruCache[string, AllocationPorts]
	jobVersionImages *lruCache[jobVersionKey, string]
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
		nomad:            nomad,
		allocationPorts:  newLRUCache[string, AllocationPorts](allocationPortsCacheCapacity, allocationPortsCacheTTL),
		jobVersionImages: newLRUCache[jobVersionKey, string](jobVersionImageCacheCapacity, jobVersionImageCacheTTL),
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
// cached per allocation ID indefinitely (see allocationPortsCacheTTL), since
// this information doesn't change for an allocation's lifetime.
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

// DockerImageFromJob returns the Docker image (with tag) configured on the
// job's first docker-driver task, walking task groups then tasks in order. A
// job can have multiple docker tasks with different images; this returns
// only the first one found. Returns "" if the job has no docker-driver task.
func DockerImageFromJob(job *nomadapi.Job) string {
	if job == nil {
		return ""
	}

	for _, tg := range job.TaskGroups {
		if tg == nil {
			continue
		}
		for _, task := range tg.Tasks {
			if task == nil || task.Driver != "docker" {
				continue
			}
			image, ok := task.Config["image"].(string)
			if ok && image != "" {
				return image
			}
		}
	}

	return ""
}

// VersionTaggedTime returns the time job's Nomad version tag was applied,
// and whether the version has a tag at all (via `nomad job tag apply`).
// Unlike a version's task config, a tag can be added, changed, or removed
// after the version itself is created — so, unlike DockerImageFromJob, this
// must always be read fresh rather than cached.
func VersionTaggedTime(job *nomadapi.Job) (time.Time, bool) {
	if job == nil || job.VersionTag == nil {
		return time.Time{}, false
	}
	return time.Unix(0, job.VersionTag.TaggedTime), true
}

// JobVersionDockerImage returns the Docker image (with tag) for jobID at
// version, derived from that version's job spec (job) and cached
// indefinitely afterward: a job version's task config never changes once
// that version exists, so this never needs to be recomputed for a version
// already seen (see jobVersionImageCacheTTL).
func (c *Client) JobVersionDockerImage(jobID string, version uint64, job *nomadapi.Job) string {
	key := jobVersionKey{jobID: jobID, version: version}

	cached, ok := c.jobVersionImages.Get(key)
	if ok {
		return cached
	}

	image := DockerImageFromJob(job)
	c.jobVersionImages.Set(key, image)

	return image
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

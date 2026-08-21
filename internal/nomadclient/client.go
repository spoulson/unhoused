// Package nomadclient wraps the subset of the Nomad HTTP API used by
// unhoused behind a small interface, so callers can be tested against a
// hand-written fake instead of a real Nomad server.
package nomadclient

import (
	"context"

	nomadapi "github.com/hashicorp/nomad/api"
)

// API is the Nomad functionality unhoused depends on.
type API interface {
	ListJobs(ctx context.Context) ([]*nomadapi.JobListStub, error)
	JobInfo(ctx context.Context, jobID string) (*nomadapi.Job, error)
	JobVersions(ctx context.Context, jobID string) ([]*nomadapi.Job, error)
	JobAllocations(ctx context.Context, jobID string) ([]*nomadapi.AllocationListStub, error)
	AllocationInfo(ctx context.Context, allocID string) (*nomadapi.Allocation, error)
}

// Client is the real API implementation backed by the Nomad SDK.
type Client struct {
	nomad *nomadapi.Client
}

var _ API = (*Client)(nil)

// New creates a Client scoped to a single Nomad cluster at addr, authenticating with token.
func New(addr, token string) (*Client, error) {
	nomad, err := nomadapi.NewClient(&nomadapi.Config{
		Address:  addr,
		SecretID: token,
	})
	if err != nil {
		return nil, err
	}

	return &Client{nomad: nomad}, nil
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

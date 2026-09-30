// Package nomaddemo simulates a subset of the Nomad HTTP API (job listing,
// job/version/allocation lookups, node listing) with fabricated but
// internally-consistent data, so unhoused can be pointed at it as an
// ordinary profile without needing a real Nomad cluster.
//
// It's meant for repo maintainers to demo or manually test unhoused against
// (e.g. the continuously-redeploying job below exercises the Job Status
// Page's live-polling and version-proportion-bar behavior), but has no
// dependency on the rest of the codebase and is safe for any dev to run.
package nomaddemo

import (
	"fmt"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
)

// simAllocation is one simulated Nomad allocation, in the shape needed to
// answer both the per-job allocation list and the single-allocation lookup
// endpoints.
type simAllocation struct {
	id            string
	jobID         string
	jobVersion    uint64
	taskGroup     string
	node          demoNode
	clientStatus  string
	desiredStatus string
	ports         []nomadapi.PortMapping
}

func (a simAllocation) listStub() *nomadapi.AllocationListStub {
	return &nomadapi.AllocationListStub{
		ID:            a.id,
		Name:          fmt.Sprintf("%s.%s[0]", a.jobID, a.taskGroup),
		Namespace:     "default",
		NodeID:        a.node.id,
		NodeName:      a.node.name,
		JobID:         a.jobID,
		JobVersion:    a.jobVersion,
		TaskGroup:     a.taskGroup,
		DesiredStatus: a.desiredStatus,
		ClientStatus:  a.clientStatus,
	}
}

func (a simAllocation) info() *nomadapi.Allocation {
	return &nomadapi.Allocation{
		ID:            a.id,
		Name:          fmt.Sprintf("%s.%s[0]", a.jobID, a.taskGroup),
		Namespace:     "default",
		NodeID:        a.node.id,
		NodeName:      a.node.name,
		JobID:         a.jobID,
		TaskGroup:     a.taskGroup,
		DesiredStatus: a.desiredStatus,
		ClientStatus:  a.clientStatus,
		AllocatedResources: &nomadapi.AllocatedResources{
			Shared: nomadapi.AllocatedSharedResources{
				Networks: []*nomadapi.NetworkResource{{IP: a.node.address}},
				Ports:    a.ports,
			},
		},
	}
}

// portMapping builds a single simulated port assignment for an allocation,
// with a stable port number derived from the allocation ID and label.
// HostIP is set to the node's address, since unhoused's port address display
// (specs/api.md's ports[].address) reads it directly off the port mapping
// rather than re-deriving it from the allocation's node.
func portMapping(allocID, label, nodeAddress string) nomadapi.PortMapping {
	return nomadapi.PortMapping{
		Label:  label,
		Value:  deterministicPort("port", allocID, label),
		HostIP: nodeAddress,
	}
}

// buildPorts assigns one port mapping per label to allocID, running on node.
// A nil/empty labels slice (e.g. a worker task group with no exposed ports)
// yields no ports, same as a real allocation with no network resources.
func buildPorts(allocID string, node demoNode, labels []string) []nomadapi.PortMapping {
	if len(labels) == 0 {
		return nil
	}

	ports := make([]nomadapi.PortMapping, len(labels))
	for i, label := range labels {
		ports[i] = portMapping(allocID, label, node.address)
	}
	return ports
}

func ptr[T any](v T) *T {
	return &v
}

// simDeployment builds a minimal *nomadapi.Deployment carrying just the
// fields internal/httpapi/derive.go's latestDeploymentInfo reads
// (JobID/Status/CreateIndex/CreateTime/ModifyTime).
func simDeployment(jobID, status string, createIndex uint64, createTime, modifyTime time.Time) *nomadapi.Deployment {
	return &nomadapi.Deployment{
		JobID:       jobID,
		Status:      status,
		CreateIndex: createIndex,
		CreateTime:  createTime.UnixNano(),
		ModifyTime:  modifyTime.UnixNano(),
	}
}

// jobVersion builds the nomadapi.Job returned for one entry of a job's
// version history — only the fields unhoused actually reads (see
// internal/nomadclient/client.go and internal/httpapi/derive.go) are
// populated: ID/Name/Status/Stop/Version/SubmitTime/VersionTag, plus enough
// of TaskGroups/Tasks for DockerImageFromJob to find the image.
func jobVersion(jobID, jobName string, version uint64, submitTime time.Time, taskGroups []string, dockerImage string, versionTag *nomadapi.JobVersionTag) *nomadapi.Job {
	job := &nomadapi.Job{
		ID:         &jobID,
		Name:       &jobName,
		Status:     ptr("running"),
		Stop:       ptr(false),
		Version:    &version,
		SubmitTime: ptr(submitTime.UnixNano()),
		VersionTag: versionTag,
	}

	for _, tg := range taskGroups {
		job.TaskGroups = append(job.TaskGroups, &nomadapi.TaskGroup{
			Name: ptr(tg),
			Tasks: []*nomadapi.Task{{
				Name:   tg,
				Driver: "docker",
				Config: map[string]any{"image": dockerImage},
			}},
		})
	}

	return job
}

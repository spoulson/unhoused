package nomaddemo

import (
	"fmt"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
)

// jobDef is one simulated job. status/stop are constant for the demo's
// lifetime; submitTime/versions/allocations are functions of the current
// time so the continuously-redeploying job (see newDeployingJob) can be
// computed fresh on every request instead of mutated by a background
// goroutine — the whole demo server is a pure function of time.Now(), so
// there's no shared state to lock.
type jobDef struct {
	id, name string
	stop     bool
	status   string

	submitTime  func(now time.Time) time.Time
	versions    func(now time.Time) []*nomadapi.Job
	allocations func(now time.Time) []simAllocation
}

// newStaticJob builds a jobDef whose versions/allocations never change —
// most demo jobs represent a Nomad job at rest, long after its last
// deployment finished.
func newStaticJob(id, name string, stop bool, status string, latestSubmit time.Time, versions []*nomadapi.Job, allocs []simAllocation) jobDef {
	return jobDef{
		id:          id,
		name:        name,
		stop:        stop,
		status:      status,
		submitTime:  func(time.Time) time.Time { return latestSubmit },
		versions:    func(time.Time) []*nomadapi.Job { return versions },
		allocations: func(time.Time) []simAllocation { return allocs },
	}
}

// stableAllocs builds count identical, evenly-spread, long-running
// allocations for a static job's task group, starting at nodeOffset in the
// shared node pool.
func stableAllocs(jobID, taskGroup string, version uint64, count, nodeOffset int, clientStatus, desiredStatus string, portLabels []string) []simAllocation {
	allocs := make([]simAllocation, count)
	for i := range count {
		node := nodeAt(nodeOffset + i)
		id := deterministicID("alloc", jobID, taskGroup, fmt.Sprint(version), fmt.Sprint(i))
		allocs[i] = simAllocation{
			id:            id,
			jobID:         jobID,
			jobVersion:    version,
			taskGroup:     taskGroup,
			node:          node,
			clientStatus:  clientStatus,
			desiredStatus: desiredStatus,
			ports:         buildPorts(id, node, portLabels),
		}
	}
	return allocs
}

// buildJobs returns every job the demo server serves, seeded relative to
// epoch (normally the server's start time — see Server.epoch).
func buildJobs(epoch time.Time) []jobDef {
	return []jobDef{
		webFrontendJob(epoch),
		apiGatewayJob(epoch),
		batchWorkerJob(epoch),
		cronSchedulerJob(epoch),
		legacyCheckoutJob(epoch),
		newDeployingJob(deployJobSpec{
			id:          "payments-api",
			name:        "payments-api",
			dockerRepo:  "ghcr.io/example/payments-api",
			epoch:       epoch,
			baseVersion: 12,
			taskGroups: []deployTaskGroup{
				{name: "api", replicas: 6, portLabels: []string{"http"}},
				{name: "worker", replicas: 3, portLabels: nil},
			},
		}),
	}
}

// webFrontendJob is a stable, single-version job spread across several
// nodes — the simplest possible steady state.
func webFrontendJob(epoch time.Time) jobDef {
	const id = "web-frontend"
	const version = uint64(3)
	submitTime := epoch.Add(-72 * time.Hour)
	dockerImage := "ghcr.io/example/web-frontend:1.4.2"

	versions := []*nomadapi.Job{
		jobVersion(id, id, version, submitTime, []string{"web"}, dockerImage, nil),
	}
	allocs := stableAllocs(id, "web", version, 8, 0, "running", "run", []string{"http", "https"})

	return newStaticJob(id, id, false, "running", submitTime, versions, allocs)
}

// apiGatewayJob has two versions present at once with no active rollout —
// a canary or a rollout that stalled partway, both common in a real
// cluster — to show off the version-proportion bars without any polling
// needed.
func apiGatewayJob(epoch time.Time) jobDef {
	const id = "api-gateway"
	const oldVersion = uint64(4)
	const newVersion = uint64(5)
	oldSubmit := epoch.Add(-9 * 24 * time.Hour)
	newSubmit := epoch.Add(-6 * time.Hour)
	oldImage := "ghcr.io/example/api-gateway:2.0.9"
	newImage := "ghcr.io/example/api-gateway:2.1.0"

	versions := []*nomadapi.Job{
		jobVersion(id, id, oldVersion, oldSubmit, []string{"api"}, oldImage, nil),
		jobVersion(id, id, newVersion, newSubmit, []string{"api"}, newImage, &nomadapi.JobVersionTag{
			Name:       "stable",
			TaggedTime: newSubmit.Add(30 * time.Minute).UnixNano(),
		}),
	}

	allocs := append(
		stableAllocs(id, "api", newVersion, 5, 8, "running", "run", []string{"http"}),
		stableAllocs(id, "api", oldVersion, 1, 13, "running", "run", []string{"http"})...,
	)

	return newStaticJob(id, id, false, "running", newSubmit, versions, allocs)
}

// batchWorkerJob includes a failed allocation among otherwise-healthy ones,
// to exercise the "failed" status color/filter.
func batchWorkerJob(epoch time.Time) jobDef {
	const id = "batch-worker"
	const version = uint64(7)
	submitTime := epoch.Add(-30 * time.Hour)
	dockerImage := "ghcr.io/example/batch-worker:0.9.1"

	versions := []*nomadapi.Job{
		jobVersion(id, id, version, submitTime, []string{"worker"}, dockerImage, nil),
	}
	allocs := append(
		stableAllocs(id, "worker", version, 3, 2, "running", "run", nil),
		stableAllocs(id, "worker", version, 1, 5, "failed", "stop", nil)...,
	)

	return newStaticJob(id, id, false, "running", submitTime, versions, allocs)
}

// cronSchedulerJob is a minimal single-allocation job.
func cronSchedulerJob(epoch time.Time) jobDef {
	const id = "cron-scheduler"
	const version = uint64(1)
	submitTime := epoch.Add(-200 * time.Hour)
	dockerImage := "ghcr.io/example/cron-scheduler:1.0.0"

	versions := []*nomadapi.Job{
		jobVersion(id, id, version, submitTime, []string{"cron"}, dockerImage, nil),
	}
	allocs := stableAllocs(id, "cron", version, 1, 6, "running", "run", nil)

	return newStaticJob(id, id, false, "running", submitTime, versions, allocs)
}

// legacyCheckoutJob is stopped (job.Stop=true), with only terminal
// allocations left over from before it was stopped — exercises the
// "stopped" Job Status Page indicator.
func legacyCheckoutJob(epoch time.Time) jobDef {
	const id = "legacy-checkout"
	const version = uint64(11)
	submitTime := epoch.Add(-400 * time.Hour)
	dockerImage := "ghcr.io/example/legacy-checkout:3.2.1"

	versions := []*nomadapi.Job{
		jobVersion(id, id, version, submitTime, []string{"checkout"}, dockerImage, nil),
	}
	allocs := stableAllocs(id, "checkout", version, 2, 7, "complete", "stop", nil)

	return newStaticJob(id, id, true, "dead", submitTime, versions, allocs)
}

package nomaddemo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeploySpec(epoch time.Time) deployJobSpec {
	return deployJobSpec{
		id:          "test-job",
		name:        "test-job",
		dockerRepo:  "example/test-job",
		epoch:       epoch,
		baseVersion: 5,
		taskGroups: []deployTaskGroup{
			{name: "web", replicas: 2, portLabels: []string{"http"}},
		},
	}
}

func TestDeployCycleState(t *testing.T) {
	epoch := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	spec := testDeploySpec(epoch)

	cases := []struct {
		name               string
		at                 time.Time
		wantOld, wantNew   uint64
		wantRolloutElapsed time.Duration
	}{
		{"at epoch, first rollout just starting", epoch, 5, 6, 0},
		{"mid first rollout", epoch.Add(2 * time.Minute), 5, 6, 2 * time.Minute},
		{"just before the cycle repeats", epoch.Add(deployCycleLength - time.Second), 5, 6, deployCycleLength - time.Second},
		{"exactly at the next cycle boundary", epoch.Add(deployCycleLength), 6, 7, 0},
		{"mid second rollout", epoch.Add(deployCycleLength + time.Minute), 6, 7, time.Minute},
		{"before epoch (clamped)", epoch.Add(-time.Hour), 5, 6, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			oldVersion, newVersion, rolloutElapsed := deployCycleState(spec, c.at)
			assert.Equal(t, c.wantOld, oldVersion)
			assert.Equal(t, c.wantNew, newVersion)
			assert.Equal(t, c.wantRolloutElapsed, rolloutElapsed)
		})
	}
}

func TestDeployAllocations(t *testing.T) {
	epoch := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	spec := testDeploySpec(epoch)
	// One task group, 2 replicas, rolloutDuration=5m => slotDuration=2m30s,
	// pendingPhase=20% of that = 30s.

	byVersion := func(allocs []simAllocation, version uint64) []simAllocation {
		var matched []simAllocation
		for _, a := range allocs {
			if a.jobVersion == version {
				matched = append(matched, a)
			}
		}
		return matched
	}

	t.Run("rollout just started: slot 0 mid-replacement, slot 1 untouched", func(t *testing.T) {
		allocs := deployAllocations(spec, epoch)
		assert.Len(t, allocs, 3) // slot 0: old+new, slot 1: old only

		oldAllocs := byVersion(allocs, 5)
		newAllocs := byVersion(allocs, 6)
		assert.Len(t, oldAllocs, 2)
		assert.Len(t, newAllocs, 1)

		for _, a := range oldAllocs {
			assert.Equal(t, "running", a.clientStatus)
		}
		assert.Equal(t, "pending", newAllocs[0].clientStatus)
		assert.Equal(t, "run", newAllocs[0].desiredStatus)
	})

	t.Run("slot 0 finished replacing, slot 1 not started", func(t *testing.T) {
		allocs := deployAllocations(spec, epoch.Add(45*time.Second))
		assert.Len(t, allocs, 3)

		oldAllocs := byVersion(allocs, 5)
		newAllocs := byVersion(allocs, 6)
		assert.Len(t, oldAllocs, 2)
		assert.Len(t, newAllocs, 1)
	})

	t.Run("both slots fully replaced by the end of the rollout", func(t *testing.T) {
		allocs := deployAllocations(spec, epoch.Add(deployRolloutDuration))
		newAllocs := byVersion(allocs, 6)
		oldAllocs := byVersion(allocs, 5)

		assert.Len(t, newAllocs, 2)
		for _, a := range newAllocs {
			assert.Equal(t, "running", a.clientStatus)
			assert.Equal(t, "run", a.desiredStatus)
		}
		for _, a := range oldAllocs {
			assert.Equal(t, "complete", a.clientStatus)
			assert.Equal(t, "stop", a.desiredStatus)
		}
	})

	t.Run("allocation IDs are stable across repeated calls at the same time", func(t *testing.T) {
		at := epoch.Add(90 * time.Second)
		first := deployAllocations(spec, at)
		second := deployAllocations(spec, at)
		assert.Equal(t, first, second)
	})
}

func TestNewDeployingJobDeployments(t *testing.T) {
	epoch := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	spec := testDeploySpec(epoch)
	job := newDeployingJob(spec)

	t.Run("mid rollout reports a running deployment", func(t *testing.T) {
		now := epoch.Add(2 * time.Minute)
		deployments := job.deployments(now)
		require.Len(t, deployments, 1)
		assert.Equal(t, spec.id, deployments[0].JobID)
		assert.Equal(t, "running", deployments[0].Status)
		assert.Equal(t, epoch.UnixNano(), deployments[0].CreateTime, "CreateTime is when the current rollout began")
		assert.Equal(t, now.UnixNano(), deployments[0].ModifyTime, "still in progress, so ModifyTime tracks now")
	})

	t.Run("idle period reports a successful deployment", func(t *testing.T) {
		deployments := job.deployments(epoch.Add(deployRolloutDuration + 30*time.Second))
		require.Len(t, deployments, 1)
		assert.Equal(t, spec.id, deployments[0].JobID)
		assert.Equal(t, "successful", deployments[0].Status)
		assert.Equal(t, epoch.UnixNano(), deployments[0].CreateTime)
		assert.Equal(t, epoch.Add(deployRolloutDuration).UnixNano(), deployments[0].ModifyTime, "ModifyTime is when the rollout finished")
	})
}

func TestDeployTaskGroupStates(t *testing.T) {
	spec := testDeploySpec(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	// 2 replicas => slotDuration=2m30s, pendingPhase=30s.

	cases := []struct {
		name                    string
		rolloutElapsed          time.Duration
		wantPlaced, wantHealthy int
	}{
		{"rollout just started: slot 0 replacing", 0, 1, 0},
		{"slot 0 healthy, slot 1 not started", 1 * time.Minute, 1, 1},
		{"slot 1 replacing", 2*time.Minute + 30*time.Second, 2, 1},
		{"rollout finished", deployRolloutDuration, 2, 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := deployTaskGroupStates(spec, c.rolloutElapsed)["web"]
			require.NotNil(t, state)
			assert.Equal(t, 2, state.DesiredTotal)
			assert.Equal(t, c.wantPlaced, state.PlacedAllocs)
			assert.Equal(t, c.wantHealthy, state.HealthyAllocs)
		})
	}
}

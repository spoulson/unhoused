package nomaddemo

import (
	"fmt"
	"time"

	nomadapi "github.com/hashicorp/nomad/api"
)

// The continuously-redeploying demo job simulates a rolling deployment:
// every deployRolloutDuration, all of its allocations are replaced with a
// new version, one task-group "slot" at a time (mirroring Nomad's rolling
// update), followed by deployIdleDuration of quiet steady-state before the
// next rollout begins.
const (
	deployRolloutDuration = 5 * time.Minute
	deployIdleDuration    = 1 * time.Minute
	deployCycleLength     = deployRolloutDuration + deployIdleDuration

	// deployPendingFraction is the portion of a slot's replacement window
	// during which the new allocation is "pending" (starting up) and the
	// old one is draining, before the new one is considered healthy.
	deployPendingFraction = 0.2
)

// deployTaskGroup is one task group of a continuously-redeploying job.
type deployTaskGroup struct {
	name       string
	replicas   int
	portLabels []string
}

// deployJobSpec describes a continuously-redeploying job. Everything about
// its current state (version, allocations) is derived purely from how much
// time has elapsed since epoch — see deployCycleState.
type deployJobSpec struct {
	id, name    string
	dockerRepo  string
	epoch       time.Time
	baseVersion uint64
	taskGroups  []deployTaskGroup
}

// newDeployingJob builds the jobDef for a continuously-redeploying job.
func newDeployingJob(spec deployJobSpec) jobDef {
	return jobDef{
		id:     spec.id,
		name:   spec.name,
		stop:   false,
		status: "running",
		submitTime: func(now time.Time) time.Time {
			_, newVersion, _ := deployCycleState(spec, now)
			return deployVersionSubmitTime(spec, newVersion)
		},
		versions: func(now time.Time) []*nomadapi.Job {
			oldVersion, newVersion, _ := deployCycleState(spec, now)
			return []*nomadapi.Job{
				deployJobVersionEntry(spec, oldVersion),
				deployJobVersionEntry(spec, newVersion),
			}
		},
		allocations: func(now time.Time) []simAllocation {
			return deployAllocations(spec, now)
		},
		deployments: func(now time.Time) []*nomadapi.Deployment {
			_, newVersion, rolloutElapsed := deployCycleState(spec, now)
			createTime := deployVersionSubmitTime(spec, newVersion)

			status := "successful"
			modifyTime := createTime.Add(deployRolloutDuration)
			if rolloutElapsed < deployRolloutDuration {
				status = "running"
				modifyTime = now
			}

			return []*nomadapi.Deployment{simDeployment(spec.id, status, newVersion, createTime, modifyTime, deployTaskGroupStates(spec, rolloutElapsed))}
		},
	}
}

// deployCycleState returns the two job versions currently in play (the one
// being replaced and the one replacing it — identical when a rollout just
// started) and how far into the current rollout/idle cycle now is.
func deployCycleState(spec deployJobSpec, now time.Time) (oldVersion, newVersion uint64, rolloutElapsed time.Duration) {
	// Clamp now itself (not just the elapsed duration used below) so
	// rolloutElapsed, derived from now.Sub(cycleStart), can't go negative.
	if now.Before(spec.epoch) {
		now = spec.epoch
	}

	cycleIndex := uint64(now.Sub(spec.epoch) / deployCycleLength)
	cycleStart := spec.epoch.Add(time.Duration(cycleIndex) * deployCycleLength)

	rolloutElapsed = now.Sub(cycleStart)
	newVersion = spec.baseVersion + cycleIndex + 1
	oldVersion = newVersion - 1
	return oldVersion, newVersion, rolloutElapsed
}

// deployVersionSubmitTime returns when version v of spec was registered.
// Versions at or before baseVersion predate the demo server itself (the
// job's pristine seed state); later versions line up with deployCycleState.
func deployVersionSubmitTime(spec deployJobSpec, v uint64) time.Time {
	if v <= spec.baseVersion {
		return spec.epoch.Add(-1 * time.Hour)
	}
	cycleIndex := v - spec.baseVersion - 1
	return spec.epoch.Add(time.Duration(cycleIndex) * deployCycleLength)
}

func deployVersionDockerImage(spec deployJobSpec, v uint64) string {
	return fmt.Sprintf("%s:1.0.%d", spec.dockerRepo, v)
}

func deployJobVersionEntry(spec deployJobSpec, v uint64) *nomadapi.Job {
	taskGroups := make([]string, len(spec.taskGroups))
	for i, tg := range spec.taskGroups {
		taskGroups[i] = tg.name
	}
	return jobVersion(spec.id, spec.name, v, deployVersionSubmitTime(spec, v), taskGroups, deployVersionDockerImage(spec, v), nil)
}

// deploySlotPhase is where one task-group slot is in its replacement.
type deploySlotPhase int

const (
	slotNotStarted deploySlotPhase = iota // still fully on the old version
	slotReplacing                         // new allocation pending, old one draining
	slotReplaced                          // new allocation healthy, old one complete
)

// deploySlotPhaseAt classifies a slot from how long ago its replacement
// window opened (negative = hasn't opened yet) and the pending sub-window.
func deploySlotPhaseAt(slotElapsed, pendingPhase time.Duration) deploySlotPhase {
	switch {
	case slotElapsed < 0:
		return slotNotStarted
	case slotElapsed < pendingPhase:
		return slotReplacing
	default:
		return slotReplaced
	}
}

// deployTaskGroupStates reports each task group's rollout progress the way
// Nomad's Deployment.TaskGroups does: every replica is desired, those whose
// replacement has begun are placed, and those whose replacement has
// finished are healthy. Mirrors the slot timing in deployAllocations.
func deployTaskGroupStates(spec deployJobSpec, rolloutElapsed time.Duration) map[string]*nomadapi.DeploymentState {
	states := make(map[string]*nomadapi.DeploymentState, len(spec.taskGroups))
	for _, tg := range spec.taskGroups {
		slotDuration := deployRolloutDuration / time.Duration(tg.replicas)
		pendingPhase := time.Duration(float64(slotDuration) * deployPendingFraction)

		state := &nomadapi.DeploymentState{DesiredTotal: tg.replicas}
		for slot := range tg.replicas {
			switch deploySlotPhaseAt(rolloutElapsed-time.Duration(slot)*slotDuration, pendingPhase) {
			case slotReplacing:
				state.PlacedAllocs++
			case slotReplaced:
				state.PlacedAllocs++
				state.HealthyAllocs++
			}
		}
		states[tg.name] = state
	}
	return states
}

// deployAllocations computes every allocation currently in play for spec at
// now, across all of its task groups. Each task group replaces its replicas
// one "slot" at a time over deployRolloutDuration; a slot that hasn't
// started its replacement yet reports only its old-version allocation
// (matching Nomad not having scheduled the replacement yet), a slot mid
// replacement reports both, and a finished slot reports only the new one.
func deployAllocations(spec deployJobSpec, now time.Time) []simAllocation {
	oldVersion, newVersion, rolloutElapsed := deployCycleState(spec, now)

	var allocs []simAllocation
	nodeOffset := 0

	for _, tg := range spec.taskGroups {
		slotDuration := deployRolloutDuration / time.Duration(tg.replicas)
		pendingPhase := time.Duration(float64(slotDuration) * deployPendingFraction)

		for slot := range tg.replicas {
			node := nodeAt(nodeOffset + slot)
			slotElapsed := rolloutElapsed - time.Duration(slot)*slotDuration

			oldID := deterministicID("alloc", spec.id, tg.name, fmt.Sprint(slot), fmt.Sprint(oldVersion))
			newID := deterministicID("alloc", spec.id, tg.name, fmt.Sprint(slot), fmt.Sprint(newVersion))

			oldAlloc := simAllocation{
				id: oldID, jobID: spec.id, jobVersion: oldVersion, taskGroup: tg.name, node: node,
				ports: buildPorts(oldID, node, tg.portLabels),
			}
			newAlloc := simAllocation{
				id: newID, jobID: spec.id, jobVersion: newVersion, taskGroup: tg.name, node: node,
				ports: buildPorts(newID, node, tg.portLabels),
			}

			switch deploySlotPhaseAt(slotElapsed, pendingPhase) {
			case slotNotStarted:
				// Not this slot's turn yet: still fully on the old version.
				oldAlloc.clientStatus, oldAlloc.desiredStatus = "running", "run"
				allocs = append(allocs, oldAlloc)
			case slotReplacing:
				// Mid-replacement: new allocation starting up, old one draining.
				oldAlloc.clientStatus, oldAlloc.desiredStatus = "running", "stop"
				newAlloc.clientStatus, newAlloc.desiredStatus = "pending", "run"
				allocs = append(allocs, oldAlloc, newAlloc)
			case slotReplaced:
				// Replacement finished: new allocation healthy, old one complete.
				oldAlloc.clientStatus, oldAlloc.desiredStatus = "complete", "stop"
				newAlloc.clientStatus, newAlloc.desiredStatus = "running", "run"
				allocs = append(allocs, oldAlloc, newAlloc)
			}
		}

		nodeOffset += tg.replicas
	}

	return allocs
}

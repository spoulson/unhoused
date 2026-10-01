import { useEffect, useRef } from 'react'
import type { DeploymentStatus } from '../api/types'
import { DEPLOYMENT_STATUS_EMOJIS, DEPLOYMENT_STATUS_LABELS } from './deploymentStatus'

/**
 * Fires a browser Notification whenever deploymentStatus changes, while enabled is true and the browser
 * has granted Notification permission — including a change into "deployed", since a rollout finishing is
 * exactly what's worth notifying about (DeploymentBadge itself stays silent for that state).
 *
 * The first defined deploymentStatus seen only seeds the tracked value rather than notifying, so the
 * initial load of a job never fires a spurious notification. JobStatusPage fully remounts per job (see
 * Layout's <Outlet key={...}>), so switching jobs naturally resets this without extra bookkeeping.
 */
export function useDeploymentNotifications(
  enabled: boolean,
  jobId: string,
  jobName: string,
  deploymentStatus: DeploymentStatus | undefined,
) {
  const previous = useRef<DeploymentStatus | undefined>(undefined)

  useEffect(() => {
    if (deploymentStatus === undefined) {
      return
    }

    const prev = previous.current
    previous.current = deploymentStatus

    if (prev === undefined || prev === deploymentStatus || deploymentStatus === '') {
      return
    }
    if (!enabled || typeof Notification === 'undefined' || Notification.permission !== 'granted') {
      return
    }

    new Notification(`Deployment ${jobName}`, {
      body: `${DEPLOYMENT_STATUS_EMOJIS[deploymentStatus]} ${jobName} ${DEPLOYMENT_STATUS_LABELS[deploymentStatus]}`,
      icon: '/logo.svg',
      tag: `unhoused-deployment-${jobId}`,
    })
  }, [enabled, jobId, jobName, deploymentStatus])
}

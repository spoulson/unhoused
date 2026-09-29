import type { DeploymentStatus } from '../api/types'
import styles from './DeploymentBadge.module.css'

// "deployed" (fully rolled out) is the default, implied state and never renders a badge — only the
// exceptions ("deploying" and "failed") are called out.
type NoteworthyStatus = Extract<DeploymentStatus, 'deploying' | 'failed'>

const LABELS: Record<NoteworthyStatus, string> = {
  deploying: 'deploying',
  failed: 'deploy failed',
}

const ICONS: Record<NoteworthyStatus, string> = {
  deploying: '⟳',
  failed: '✗',
}

function isNoteworthy(status: DeploymentStatus): status is NoteworthyStatus {
  return status === 'deploying' || status === 'failed'
}

interface DeploymentBadgeProps {
  status: DeploymentStatus
}

// Shown next to the job status badge on the Profile Page's jobs table, indicating whether the job's
// most recent deployment is still rolling out or failed/was cancelled. Renders nothing when the
// deployment is fully rolled out ("deployed") or the job has no deployment at all (batch/system jobs,
// or service jobs without an `update` block) — see specs/api.md.
export function DeploymentBadge({ status }: DeploymentBadgeProps) {
  if (!isNoteworthy(status)) {
    return null
  }

  return (
    <span className={`${styles.badge} ${styles[status]}`} title={`Deployment: ${LABELS[status]}`}>
      <span aria-hidden="true">{ICONS[status]}</span>
      {LABELS[status]}
    </span>
  )
}

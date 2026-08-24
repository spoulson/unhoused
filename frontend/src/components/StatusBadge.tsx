import type { ClientStatus, JobStatus } from '../api/types'
import { statusColor } from '../lib/statusColors'
import { statusIcon } from '../lib/statusIcons'
import styles from './StatusBadge.module.css'

interface StatusBadgeProps {
  status: JobStatus | ClientStatus
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span className={`${styles.badge} ${styles[statusColor(status)]}`}>
      <span aria-hidden="true">{statusIcon(status)}</span>
      {status}
    </span>
  )
}

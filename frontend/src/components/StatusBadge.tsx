import type { ClientStatus, JobStatus } from '../api/types'
import { statusColor } from '../lib/statusColors'
import { statusIcon } from '../lib/statusIcons'
import { BadgeCount } from './BadgeCount'
import styles from './StatusBadge.module.css'

interface StatusBadgeProps {
  status: JobStatus | ClientStatus
  /** When set, shown inline at the end of the badge. */
  count?: number
}

export function StatusBadge({ status, count }: StatusBadgeProps) {
  return (
    <span className={`${styles.badge} ${styles[statusColor(status)]}`}>
      <span aria-hidden="true">{statusIcon(status)}</span>
      {status}
      {count !== undefined && <BadgeCount count={count} />}
    </span>
  )
}

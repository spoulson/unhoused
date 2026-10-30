import type { ClientStatus, JobStatus } from '../api/types'
import { statusColor } from '../lib/statusColors'
import { statusIcon } from '../lib/statusIcons'
import { CountBubble } from './CountBubble'
import styles from './StatusBadge.module.css'

interface StatusBadgeProps {
  status: JobStatus | ClientStatus
  /** When set, shown as a red bubble on the badge's upper-right corner. */
  count?: number
}

export function StatusBadge({ status, count }: StatusBadgeProps) {
  return (
    <span className={`${styles.badge} ${styles[statusColor(status)]}`}>
      <span aria-hidden="true">{statusIcon(status)}</span>
      {status}
      {count !== undefined && <CountBubble count={count} />}
    </span>
  )
}

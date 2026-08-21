import type { ClientStatus, JobStatus } from '../api/types'

type StatusColorToken = 'green' | 'yellow' | 'gray' | 'red' | 'aqua' | 'orange'

const STATUS_COLORS: Record<JobStatus | ClientStatus, StatusColorToken> = {
  running: 'green',
  pending: 'yellow',
  stopped: 'gray',
  dead: 'red',
  failed: 'red',
  complete: 'aqua',
  lost: 'orange',
}

export function statusColor(status: JobStatus | ClientStatus): StatusColorToken {
  return STATUS_COLORS[status]
}

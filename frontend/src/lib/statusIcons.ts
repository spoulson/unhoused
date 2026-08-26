import type { ClientStatus, JobStatus } from '../api/types'

const STATUS_ICONS: Record<JobStatus | ClientStatus, string> = {
  running: '✓',
  pending: '●',
  stopped: '■',
  dead: '✗',
  failed: '✗',
  complete: '■',
  lost: '▲',
}

export function statusIcon(status: JobStatus | ClientStatus): string {
  return STATUS_ICONS[status]
}

/** Formats an RFC3339 timestamp in the viewer's local time, e.g. "9/30/2026 10:54:04am". */
export function formatLocalTimestamp(iso: string): string {
  const d = new Date(iso)
  const month = d.getMonth() + 1
  const day = d.getDate()
  const year = d.getFullYear()
  const minutes = String(d.getMinutes()).padStart(2, '0')
  const seconds = String(d.getSeconds()).padStart(2, '0')
  const period = d.getHours() < 12 ? 'am' : 'pm'
  const hours = d.getHours() % 12 || 12
  return `${month}/${day}/${year} ${hours}:${minutes}:${seconds}${period}`
}

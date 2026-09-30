const UNITS: [label: string, seconds: number][] = [
  ['d', 86400],
  ['h', 3600],
  ['m', 60],
  ['s', 1],
]

// Durations of 10 minutes or more drop the seconds unit entirely, so a multi-hour duration doesn't show
// a meaningless "6h 4s"-style remainder — at that scale the seconds digit isn't useful information.
const HIDE_SECONDS_THRESHOLD_SECONDS = 10 * 60

/** Formats a duration in seconds as the two largest applicable units, e.g. "2h 5m" or "45s" — seconds are
 * omitted once the duration reaches 10 minutes (see HIDE_SECONDS_THRESHOLD_SECONDS). */
export function formatDuration(totalSeconds: number): string {
  if (totalSeconds <= 0) {
    return '0s'
  }

  const units = totalSeconds >= HIDE_SECONDS_THRESHOLD_SECONDS ? UNITS.filter(([label]) => label !== 's') : UNITS

  const parts: string[] = []
  let remaining = totalSeconds

  for (const [label, unitSeconds] of units) {
    if (remaining < unitSeconds) {
      continue
    }
    const value = Math.floor(remaining / unitSeconds)
    remaining -= value * unitSeconds
    parts.push(`${value}${label}`)
    if (parts.length === 2) {
      break
    }
  }

  return parts.join(' ')
}

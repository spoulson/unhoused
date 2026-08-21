const UNITS: [label: string, seconds: number][] = [
  ['d', 86400],
  ['h', 3600],
  ['m', 60],
  ['s', 1],
]

/** Formats a duration in seconds as the two largest applicable units, e.g. "2h 5m" or "45s". */
export function formatDuration(totalSeconds: number): string {
  if (totalSeconds <= 0) {
    return '0s'
  }

  const parts: string[] = []
  let remaining = totalSeconds

  for (const [label, unitSeconds] of UNITS) {
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

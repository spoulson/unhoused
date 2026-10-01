import type { DeploymentStatus } from '../api/types'
import { formatDuration } from './duration'
import { formatLocalTimestamp } from './timestamp'

/** Human-readable label for each non-empty DeploymentStatus value. See specs/api.md. */
export const DEPLOYMENT_STATUS_LABELS: Record<Exclude<DeploymentStatus, ''>, string> = {
  deployed: 'deployed',
  deploying: 'deploying',
  failed: 'deploy failed',
}

/** Emoji for each non-empty DeploymentStatus value: success / busy / failure. */
export const DEPLOYMENT_STATUS_EMOJIS: Record<Exclude<DeploymentStatus, ''>, string> = {
  deployed: '✅',
  deploying: '🔄',
  failed: '❌',
}

/** Capitalized, bare (no duration) badge text for each non-empty DeploymentStatus value. */
const BADGE_STATUS_WORDS: Record<Exclude<DeploymentStatus, ''>, string> = {
  deployed: 'Deployed',
  deploying: 'Deploying',
  failed: 'Failed',
}

/**
 * Badge text for a deployment status, including elapsed time when known: "Deploying for <duration>",
 * "Deployed <duration> ago", "Failed <duration> ago". Falls back to the bare, capitalized status word
 * ("Deploying"/"Deployed"/"Failed") when elapsedSeconds is null (the underlying Nomad deployment's
 * timestamps aren't set — see specs/api.md).
 */
export function deploymentStatusBadgeText(status: Exclude<DeploymentStatus, ''>, elapsedSeconds: number | null): string {
  if (elapsedSeconds === null) {
    return BADGE_STATUS_WORDS[status]
  }

  const duration = formatDuration(elapsedSeconds)
  if (status === 'deploying') {
    return `Deploying for ${duration}`
  }
  if (status === 'deployed') {
    return `Deployed ${duration} ago`
  }
  return `Failed ${duration} ago`
}

/**
 * Tooltip text for a deployment status badge: the badge text (deploymentStatusBadgeText) plus, when the
 * duration is known, the absolute local timestamp it's measured from in parentheses, e.g.
 * "Deploying for 2m 10s (9/30/2026 10:54:04am)". While deploying with a known progressPercent, the
 * percentage follows the duration: "Deploying for 2m 10s, 60% complete (9/30/2026 10:54:04am)".
 */
export function deploymentStatusTooltip(
  status: Exclude<DeploymentStatus, ''>,
  elapsedSeconds: number | null,
  since: string,
  progressPercent: number | null = null,
): string {
  let text = deploymentStatusBadgeText(status, elapsedSeconds)
  if (status === 'deploying' && progressPercent !== null) {
    text = `${text}, ${progressPercent}% complete`
  }
  if (elapsedSeconds === null || since === '') {
    return text
  }
  return `${text} (${formatLocalTimestamp(since)})`
}

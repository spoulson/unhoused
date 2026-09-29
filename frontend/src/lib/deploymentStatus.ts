import type { DeploymentStatus } from '../api/types'

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

import type { CSSProperties } from 'react'
import type { DeploymentStatus } from '../api/types'
import { deploymentStatusBadgeText, deploymentStatusTooltip } from '../lib/deploymentStatus'
import { useTheme } from '../theme/ThemeContext'
import styles from './DeploymentBadge.module.css'

type KnownStatus = Exclude<DeploymentStatus, ''>

const ICONS: Record<Exclude<KnownStatus, 'deploying'>, string> = {
  deployed: '✓',
  failed: '✗',
}

// "deploying" gets an animated spinner instead of a static glyph — a separate file per theme since
// GIF pixels are baked in (no `currentColor` equivalent), colored to match each theme's --gb-bg0, the
// same color the other badge icons already render in via `color: var(--gb-bg0)` + currentColor.
const DEPLOYING_SPINNER_SRC: Record<'light' | 'dark', string> = {
  light: '/icons/deploying-spinner-light.gif',
  dark: '/icons/deploying-spinner-dark.gif',
}

function isKnownStatus(status: DeploymentStatus): status is KnownStatus {
  return status !== ''
}

interface DeploymentBadgeProps {
  status: DeploymentStatus
  elapsedSeconds: number | null
  since: string
  progressPercent: number | null
}

// Shown next to the job status badge (Profile Page's Deployment column, and the Job Status Page's
// heading): the job's most recent deployment state, plus how long it's been in that state when known
// ("Deploying for 2m", "Deployed 3h ago", "Failed 5m ago"). The tooltip additionally shows the absolute
// local timestamp the duration is measured from, e.g. "Deploying for 2m 10s (9/30/2026 10:54:04am)".
// While deploying, the badge's yellow background doubles as a left-to-right progress bar (completed
// area --gb-yellow, remaining area --gb-yellow-dim), and the tooltip adds "N% complete".
// Renders nothing for jobs with no deployment at all (batch/system jobs, or service jobs without an
// `update` block) — see specs/api.md.
export function DeploymentBadge({ status, elapsedSeconds, since, progressPercent }: DeploymentBadgeProps) {
  const { theme } = useTheme()

  if (!isKnownStatus(status)) {
    return null
  }

  const style =
    status === 'deploying' && progressPercent !== null
      ? ({ '--progress': `${progressPercent}%` } as CSSProperties)
      : undefined

  return (
    <span
      className={`${styles.badge} ${styles[status]}`}
      style={style}
      title={deploymentStatusTooltip(status, elapsedSeconds, since, progressPercent)}
    >
      {status === 'deploying' ? (
        <img className={styles.spinner} src={DEPLOYING_SPINNER_SRC[theme]} alt="" aria-hidden="true" />
      ) : (
        <span aria-hidden="true">{ICONS[status]}</span>
      )}
      {deploymentStatusBadgeText(status, elapsedSeconds)}
    </span>
  )
}

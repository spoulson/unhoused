import styles from './NotificationToggle.module.css'

interface NotificationToggleProps {
  enabled: boolean
  blocked: boolean
  onToggle: () => void
}

// Bell body + clapper, shared by both states: filled (no slash) when notifications are on, outlined with
// a slash through it when they're off — a diagonal slash reads as "muted" far more clearly than a filled
// vs. outlined bell alone.
const BELL_BODY_PATH =
  'M6 1.3c-.35 0-.63.28-.63.63v.3C3.9 2.6 3 3.8 3 5.3v2.1L2 9h8L9 7.4V5.3c0-1.5-.9-2.7-2.37-3.1v-.3c0-.35-.28-.63-.63-.63z'
const BELL_CLAPPER_PATH = 'M4.6 9.6a1.4 1.4 0 0 0 2.8 0'

export function NotificationToggle({ enabled, blocked, onToggle }: NotificationToggleProps) {
  const label = blocked
    ? 'Notifications blocked in browser settings'
    : enabled
      ? 'Disable deployment notifications'
      : 'Enable deployment notifications'

  return (
    <button
      type="button"
      className={`${styles.toggle} ${blocked ? styles.blocked : ''}`}
      onClick={onToggle}
      aria-label={label}
      title={label}
    >
      <svg viewBox="0 0 12 12" aria-hidden="true">
        <path
          d={BELL_BODY_PATH}
          fill={enabled ? 'currentColor' : 'none'}
          stroke="currentColor"
          strokeWidth="0.7"
          strokeLinejoin="round"
        />
        <path d={BELL_CLAPPER_PATH} fill="none" stroke="currentColor" strokeWidth="0.7" strokeLinecap="round" />
        {!enabled && (
          <line x1="1.5" y1="10.5" x2="10.5" y2="1.5" stroke="currentColor" strokeWidth="0.9" strokeLinecap="round" />
        )}
      </svg>
    </button>
  )
}

import styles from './AutoRefreshToggle.module.css'

interface AutoRefreshToggleProps {
  paused: boolean
  onToggle: () => void
}

export function AutoRefreshToggle({ paused, onToggle }: AutoRefreshToggleProps) {
  const label = paused ? 'Resume automatic updates' : 'Pause automatic updates'

  return (
    <button type="button" className={styles.toggle} onClick={onToggle} aria-label={label} title={label}>
      {paused ? (
        <svg viewBox="0 0 12 12" aria-hidden="true">
          <path d="M2 1l9 5-9 5V1z" fill="currentColor" />
        </svg>
      ) : (
        <svg viewBox="0 0 12 12" aria-hidden="true">
          <rect x="2" y="1" width="3" height="10" fill="currentColor" />
          <rect x="7" y="1" width="3" height="10" fill="currentColor" />
        </svg>
      )}
    </button>
  )
}

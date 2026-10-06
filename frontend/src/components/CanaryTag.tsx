import styles from './CanaryTag.module.css'

// Small "Canary" pill marking the canary deployment / the canary version and its allocations.
export function CanaryTag() {
  return <span className={styles.canary}>Canary</span>
}

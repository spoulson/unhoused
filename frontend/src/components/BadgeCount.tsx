import styles from './BadgeCount.module.css'

/** The count associated with its parent badge, shown inline at the badge's end. */
export function BadgeCount({ count }: { count: number }) {
  return <span className={styles.count}>{count}</span>
}

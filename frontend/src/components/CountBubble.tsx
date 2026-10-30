import styles from './CountBubble.module.css'

/** Red bubble pinned to the upper-right corner of its (position: relative) parent. */
export function CountBubble({ count }: { count: number }) {
  return <span className={styles.bubble}>{count}</span>
}

import { CountBubble } from './CountBubble'
import styles from './TaskGroupChip.module.css'

interface Props {
  name: string
  /** When set, shown as a red bubble on the chip's upper-right corner. */
  count?: number
  title?: string
  /** When set, the chip is rendered as a button. */
  onClick?: () => void
}

/** A task group name, optionally with its count. */
export function TaskGroupChip({ name, count, title, onClick }: Props) {
  const className = `${styles.chip} ${count !== undefined ? styles.withCount : ''} ${onClick ? styles.clickable : ''}`
  const content = (
    <>
      <span aria-hidden="true">⊞</span>
      {name}
      {count !== undefined && <CountBubble count={count} />}
    </>
  )
  return onClick ? (
    <button type="button" className={className} title={title} onClick={onClick}>
      {content}
    </button>
  ) : (
    <span className={className} title={title}>
      {content}
    </span>
  )
}

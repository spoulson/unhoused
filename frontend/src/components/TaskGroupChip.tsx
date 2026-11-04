import { BadgeCount } from './BadgeCount'
import styles from './TaskGroupChip.module.css'

interface Props {
  name: string
  /** When set, shown inline at the end of the chip. */
  count?: number
  title?: string
  /** When set, the chip is rendered as a button. */
  onClick?: () => void
}

/** A task group name, optionally with its count. */
export function TaskGroupChip({ name, count, title, onClick }: Props) {
  const className = `${styles.chip} ${onClick ? styles.clickable : ''}`
  const content = (
    <>
      <span aria-hidden="true">⊞</span>
      {name}
      {count !== undefined && <BadgeCount count={count} />}
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

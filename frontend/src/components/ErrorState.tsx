import { ApiError } from '../api/client'
import styles from './ErrorState.module.css'

interface ErrorStateProps {
  error: unknown
}

export function ErrorState({ error }: ErrorStateProps) {
  const message = error instanceof ApiError ? error.message : 'Something went wrong.'

  return (
    <p className={styles.error}>
      {message}
    </p>
  )
}

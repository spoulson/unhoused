import styles from './VersionLabel.module.css'

interface VersionLabelProps {
  version: number
  taggedTime: string
  dockerImage: string
}

/**
 * Renders a job version with syntax-highlighted `tagged-at=`/`image=` segments, mirroring
 * lib/version.ts's formatVersionLabel but as styled JSX instead of a plain string — for contexts that
 * can render markup. Native <option> elements can't, so the Version filter dropdown still uses the plain
 * string form.
 */
export function VersionLabel({ version, taggedTime, dockerImage }: VersionLabelProps) {
  return (
    <span className={`mono ${styles.versionLabel}`}>
      <span className={styles.version}>{version}</span>
      {taggedTime && (
        <>
          {' '}
          <span className={styles.key}>tagged-at=</span>
          <span className={styles.taggedTime}>{taggedTime}</span>
        </>
      )}
      {dockerImage && (
        <>
          {' '}
          <span className={styles.key}>image=</span>
          <span className={styles.image}>{dockerImage}</span>
        </>
      )}
    </span>
  )
}

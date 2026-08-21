import { Link, Outlet, useParams } from 'react-router-dom'
import styles from './Layout.module.css'

export function Layout() {
  const { profileName, jobId } = useParams()

  return (
    <div className={styles.layout}>
      <header className={styles.header}>
        <Link to="/" className={styles.title}>
          unhoused
        </Link>
        <nav className={styles.breadcrumb}>
          <Link to="/">Home</Link>
          {profileName && (
            <>
              {' / '}
              <Link to={`/profiles/${profileName}`}>{profileName}</Link>
            </>
          )}
          {profileName && jobId && (
            <>
              {' / '}
              <span className="mono">{jobId}</span>
            </>
          )}
        </nav>
      </header>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  )
}

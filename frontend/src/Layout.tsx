import { Link, Outlet, useParams } from 'react-router-dom'
import { ThemeToggle } from './components/ThemeToggle'
import styles from './Layout.module.css'

export function Layout() {
  const { profileName, jobId } = useParams()

  return (
    <div className={styles.layout}>
      <header className={styles.header}>
        <Link to="/" className={styles.title}>
          <img src="/logo.svg" alt="" className={styles.logo} />
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
        <div className={styles.themeToggle}>
          <ThemeToggle />
        </div>
      </header>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  )
}

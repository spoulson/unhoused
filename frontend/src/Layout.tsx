import { useState } from 'react'
import { Link, Outlet, useNavigate, useParams } from 'react-router-dom'
import { fetchJSON } from './api/client'
import { useProfiles } from './api/queries'
import type { JobsResponse } from './api/types'
import { ThemeToggle } from './components/ThemeToggle'
import styles from './Layout.module.css'

export function Layout() {
  const { profileName, jobId } = useParams()
  const { data: profilesData } = useProfiles()
  const navigate = useNavigate()
  const [isSwitchingProfile, setIsSwitchingProfile] = useState(false)

  async function handleProfileChange(newProfileName: string) {
    if (!profileName || newProfileName === profileName) {
      return
    }

    setIsSwitchingProfile(true)
    try {
      if (jobId) {
        try {
          const jobs = await fetchJSON<JobsResponse>(`/api/profiles/${encodeURIComponent(newProfileName)}/jobs`)
          if (jobs.jobs.some((job) => job.id === jobId)) {
            navigate(`/profiles/${newProfileName}/jobs/${jobId}`)
            return
          }
        } catch {
          // Fall through to the profile page if the job lookup fails for any reason.
        }
      }

      navigate(`/profiles/${newProfileName}`)
    } finally {
      setIsSwitchingProfile(false)
    }
  }

  // Keeps the currently selected profile as an option even before the profiles list has loaded
  // (or if it's somehow missing from it), so the <select> always has a valid selected value.
  const knownCurrentProfile = profilesData?.profiles.some((p) => p.name === profileName)

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
              <select
                className={styles.profileSelect}
                value={profileName}
                onChange={(e) => handleProfileChange(e.target.value)}
                disabled={isSwitchingProfile}
                aria-label="Switch profile"
              >
                {!knownCurrentProfile && <option value={profileName}>{profileName}</option>}
                {profilesData?.profiles.map((p) => (
                  <option key={p.name} value={p.name}>
                    {p.name}
                  </option>
                ))}
              </select>
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
        {/* Keyed by profile/job so switching either fully remounts the page instead of a stale
            previous profile's data lingering via useJobStatus's keepPreviousData. */}
        <Outlet key={`${profileName ?? ''}/${jobId ?? ''}`} />
      </main>
    </div>
  )
}

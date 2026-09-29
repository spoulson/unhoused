import { useState } from 'react'
import { Link, Outlet, useNavigate, useParams } from 'react-router-dom'
import { fetchJSON } from './api/client'
import { useProfiles } from './api/queries'
import type { JobsResponse } from './api/types'
import { AutoRefreshToggle } from './components/AutoRefreshToggle'
import { ThemeToggle } from './components/ThemeToggle'
import styles from './Layout.module.css'

export interface JobStatusOutletContext {
  autoRefreshPaused: boolean
}

export function Layout() {
  const { profileSlug, jobId } = useParams()
  const { data: profilesData } = useProfiles()
  const profileName = profilesData?.profiles.find((p) => p.slug === profileSlug)?.name ?? profileSlug
  const navigate = useNavigate()
  const [isSwitchingProfile, setIsSwitchingProfile] = useState(false)
  const [autoRefreshPaused, setAutoRefreshPaused] = useState(false)

  // Each job visit starts with automatic updates running, regardless of whether a previous job was paused.
  // Adjusted during render (rather than in an effect) per React's "resetting state on prop change" pattern.
  const [pausedForJobId, setPausedForJobId] = useState(jobId)
  if (jobId !== pausedForJobId) {
    setPausedForJobId(jobId)
    setAutoRefreshPaused(false)
  }

  async function handleProfileChange(newProfileSlug: string) {
    if (!profileSlug || newProfileSlug === profileSlug) {
      return
    }

    setIsSwitchingProfile(true)
    try {
      if (jobId) {
        try {
          const jobs = await fetchJSON<JobsResponse>(`/api/profiles/${encodeURIComponent(newProfileSlug)}/jobs`)
          if (jobs.jobs.some((job) => job.id === jobId)) {
            navigate(`/profiles/${newProfileSlug}/jobs/${jobId}`)
            return
          }
        } catch {
          // Fall through to the profile page if the job lookup fails for any reason.
        }
      }

      navigate(`/profiles/${newProfileSlug}`)
    } finally {
      setIsSwitchingProfile(false)
    }
  }

  // Keeps the currently selected profile as an option even before the profiles list has loaded
  // (or if it's somehow missing from it), so the <select> always has a valid selected value.
  const knownCurrentProfile = profilesData?.profiles.some((p) => p.slug === profileSlug)

  return (
    <div className={styles.layout}>
      <header className={styles.header}>
        <Link to="/" className={styles.title}>
          <img src="/logo.svg" alt="" className={styles.logo} />
          unhoused
        </Link>
        <nav className={styles.breadcrumb}>
          <Link to="/">Home</Link>
          {profileSlug && (
            <>
              {' / '}
              <span className={styles.icon} aria-hidden="true">
                ▣
              </span>
              <span className={styles.profileGroup}>
                <Link to={`/profiles/${profileSlug}`}>{profileName}</Link>
                {/* An inline SVG, not a Unicode glyph, since triangle/chevron characters render
                    inconsistently (sometimes near-invisible) across the app's configured fonts. */}
                <svg className={styles.dropdownArrow} aria-hidden="true" viewBox="0 0 10 6" width="10" height="6">
                  <path
                    d="M1 1l4 4 4-4"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
                <select
                  className={styles.profileSelect}
                  value={profileSlug}
                  onChange={(e) => handleProfileChange(e.target.value)}
                  disabled={isSwitchingProfile}
                  aria-label="Switch profile"
                >
                  {!knownCurrentProfile && <option value={profileSlug}>{profileName}</option>}
                  {profilesData?.profiles.map((p) => (
                    <option key={p.slug} value={p.slug}>
                      {p.name}
                    </option>
                  ))}
                </select>
              </span>
            </>
          )}
          {profileSlug && jobId && (
            <>
              {' / '}
              <span className={styles.icon} aria-hidden="true">
                ⛟
              </span>
              <span className="mono">{jobId}</span>
            </>
          )}
        </nav>
        <div className={styles.headerActions}>
          {jobId && <AutoRefreshToggle paused={autoRefreshPaused} onToggle={() => setAutoRefreshPaused((p) => !p)} />}
          <ThemeToggle />
        </div>
      </header>
      <main className={styles.main}>
        {/* Keyed by profile/job so switching either fully remounts the page instead of a stale
            previous profile's data lingering via useJobStatus's keepPreviousData. */}
        <Outlet key={`${profileSlug ?? ''}/${jobId ?? ''}`} context={{ autoRefreshPaused } satisfies JobStatusOutletContext} />
      </main>
    </div>
  )
}

import { Link } from 'react-router-dom'
import { useProfiles } from '../api/queries'
import { ErrorState } from '../components/ErrorState'
import { LoadingState } from '../components/LoadingState'
import styles from './HomePage.module.css'

export function HomePage() {
  const { data, isLoading, error } = useProfiles()

  if (isLoading) {
    return <LoadingState />
  }

  if (error) {
    return <ErrorState error={error} />
  }

  return (
    <div>
      <h1>Profiles</h1>
      <ul className={styles.list}>
        {data?.profiles.map((profile) => (
          <li key={profile.name}>
            <Link to={`/profiles/${profile.name}`} className={styles.card}>
              <span className={styles.name}>{profile.name}</span>
              <span className={styles.meta}>
                {profile.environment} · {profile.region}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </div>
  )
}

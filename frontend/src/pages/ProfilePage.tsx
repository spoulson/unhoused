import { Link, useParams } from 'react-router-dom'
import { useJobs } from '../api/queries'
import { ErrorState } from '../components/ErrorState'
import { LoadingState } from '../components/LoadingState'
import styles from './ProfilePage.module.css'

export function ProfilePage() {
  const { profileName } = useParams<{ profileName: string }>()
  const { data, isLoading, error } = useJobs(profileName ?? '')

  if (isLoading) {
    return <LoadingState />
  }

  if (error) {
    return <ErrorState error={error} />
  }

  return (
    <div>
      <h1>{profileName}</h1>
      {data?.jobs.length === 0 ? (
        <p>No jobs found.</p>
      ) : (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Job</th>
              <th>Submitted</th>
            </tr>
          </thead>
          <tbody>
            {data?.jobs.map((job) => (
              <tr key={job.id}>
                <td>
                  <Link to={`/profiles/${profileName}/jobs/${job.id}`} className="mono">
                    {job.name}
                  </Link>
                </td>
                <td>{new Date(job.submitTime).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

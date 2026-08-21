import { useParams } from 'react-router-dom'
import { useJobStatus } from '../api/queries'
import type { ClientStatus, Port } from '../api/types'
import { ErrorState } from '../components/ErrorState'
import { LoadingState } from '../components/LoadingState'
import { StatusBadge } from '../components/StatusBadge'
import { formatDuration } from '../lib/duration'
import styles from './JobStatusPage.module.css'

const CLIENT_STATUSES: ClientStatus[] = ['running', 'pending', 'failed', 'complete', 'lost']

function PortLinks({ ports }: { ports: Port[] }) {
  return (
    <div className={styles.ports}>
      {ports.map((port) => (
        <div key={`${port.label}-${port.port}`} className={styles.port}>
          <span className={styles.portLabel}>{port.label}</span>
          <a href={port.url} target="_blank" rel="noreferrer" className="mono">
            {port.url}
          </a>
          {port.nodeUrl && (
            <a href={port.nodeUrl} target="_blank" rel="noreferrer" className="mono">
              {port.nodeUrl}
            </a>
          )}
        </div>
      ))}
    </div>
  )
}

export function JobStatusPage() {
  const { profileName, jobId } = useParams<{ profileName: string; jobId: string }>()
  const { data, isLoading, error } = useJobStatus(profileName ?? '', jobId ?? '')

  if (isLoading) {
    return <LoadingState />
  }

  if (error) {
    return <ErrorState error={error} />
  }

  if (!data) {
    return null
  }

  return (
    <div>
      <h1 className={styles.title}>
        {data.job.name} <StatusBadge status={data.job.status} />
      </h1>

      <h2>Versions</h2>
      <div className={styles.versionGroups}>
        {data.versionGroups.map((group) => (
          <div key={group.version} className={styles.versionGroup}>
            <div className={styles.versionHeader}>
              <span>Version {group.version}</span>
              <span className={styles.uptime}>
                uptime {formatDuration(group.newestAllocationUptimeSeconds)}
              </span>
            </div>
            <div className={styles.statusCounts}>
              {CLIENT_STATUSES.filter((status) => group.statusCounts[status] > 0).map((status) => (
                <span key={status} className={styles.statusCount}>
                  <StatusBadge status={status} /> {group.statusCounts[status]}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>

      <h2>Allocations</h2>
      {data.allocations.length === 0 ? (
        <p>No allocations found.</p>
      ) : (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Allocation</th>
              <th>Node</th>
              <th>Status</th>
              <th>Desired</th>
              <th>Task Group</th>
              <th>Version</th>
              <th>Uptime</th>
              <th>Ports</th>
            </tr>
          </thead>
          <tbody>
            {data.allocations.map((alloc) => (
              <tr key={alloc.id}>
                <td className="mono">{alloc.id}</td>
                <td>
                  <div>{alloc.nodeName}</div>
                  <div className={`${styles.nodeIp} mono`}>{alloc.nodeIp}</div>
                </td>
                <td>
                  <StatusBadge status={alloc.clientStatus} />
                </td>
                <td>{alloc.desiredStatus}</td>
                <td>{alloc.taskGroup}</td>
                <td>{alloc.version}</td>
                <td>{formatDuration(alloc.uptimeSeconds)}</td>
                <td>
                  <PortLinks ports={alloc.ports} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

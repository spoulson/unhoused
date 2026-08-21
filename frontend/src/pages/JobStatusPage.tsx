import { useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useJobStatus } from '../api/queries'
import type { Allocation, ClientStatus, Port } from '../api/types'
import { ErrorState } from '../components/ErrorState'
import { LoadingState } from '../components/LoadingState'
import { StatusBadge } from '../components/StatusBadge'
import { formatDuration } from '../lib/duration'
import styles from './JobStatusPage.module.css'

const CLIENT_STATUSES: ClientStatus[] = ['running', 'pending', 'failed', 'complete', 'lost']

// Nomad's fixed set of allocation DesiredStatus values (github.com/hashicorp/nomad/api: AllocDesiredStatus*).
const DESIRED_STATUSES = ['run', 'stop', 'evict']

interface Filters {
  taskGroup: string
  version: string
  node: string
  status: string
  desired: string
}

const EMPTY_FILTERS: Filters = { taskGroup: '', version: '', node: '', status: '', desired: '' }

function distinctSorted<T>(values: Iterable<T>): T[] {
  return Array.from(new Set(values)).sort()
}

function matchesFilters(alloc: Allocation, filters: Filters): boolean {
  return (
    (filters.taskGroup === '' || alloc.taskGroup === filters.taskGroup) &&
    (filters.version === '' || String(alloc.version) === filters.version) &&
    (filters.node === '' || alloc.nodeName === filters.node) &&
    (filters.status === '' || alloc.clientStatus === filters.status) &&
    (filters.desired === '' || alloc.desiredStatus === filters.desired)
  )
}

interface FilterSelectProps {
  label: string
  value: string
  options: string[]
  onChange: (value: string) => void
}

function FilterSelect({ label, value, options, onChange }: FilterSelectProps) {
  return (
    <label className={styles.filter}>
      <span className={styles.filterLabel}>{label}</span>
      <select value={value} onChange={(e) => onChange(e.target.value)}>
        <option value="">All</option>
        {options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </label>
  )
}

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
  const [filters, setFilters] = useState<Filters>(EMPTY_FILTERS)

  const taskGroupOptions = useMemo(
    () => distinctSorted(data?.allocations.map((a) => a.taskGroup) ?? []),
    [data],
  )
  const versionOptions = useMemo(
    () => distinctSorted(data?.allocations.map((a) => a.version) ?? []).map(String),
    [data],
  )
  const nodeOptions = useMemo(
    () => distinctSorted(data?.allocations.map((a) => a.nodeName) ?? []),
    [data],
  )

  const filteredAllocations = useMemo(
    () => data?.allocations.filter((alloc) => matchesFilters(alloc, filters)) ?? [],
    [data, filters],
  )

  function setFilter(key: keyof Filters, value: string) {
    setFilters((prev) => ({ ...prev, [key]: value }))
  }

  if (isLoading) {
    return <LoadingState />
  }

  if (error) {
    return <ErrorState error={error} />
  }

  if (!data) {
    return null
  }

  const hasActiveFilters = Object.values(filters).some((v) => v !== '')

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
        <>
          <div className={styles.filterBar}>
            <FilterSelect
              label="Task Group"
              value={filters.taskGroup}
              options={taskGroupOptions}
              onChange={(v) => setFilter('taskGroup', v)}
            />
            <FilterSelect
              label="Version"
              value={filters.version}
              options={versionOptions}
              onChange={(v) => setFilter('version', v)}
            />
            <FilterSelect
              label="Node"
              value={filters.node}
              options={nodeOptions}
              onChange={(v) => setFilter('node', v)}
            />
            <FilterSelect
              label="Status"
              value={filters.status}
              options={CLIENT_STATUSES}
              onChange={(v) => setFilter('status', v)}
            />
            <FilterSelect
              label="Desired"
              value={filters.desired}
              options={DESIRED_STATUSES}
              onChange={(v) => setFilter('desired', v)}
            />
            {hasActiveFilters && (
              <button type="button" className={styles.clearFilters} onClick={() => setFilters(EMPTY_FILTERS)}>
                Clear filters
              </button>
            )}
          </div>
          <p className={styles.filterSummary}>
            Showing {filteredAllocations.length} of {data.allocations.length} allocations
          </p>
          {filteredAllocations.length === 0 ? (
            <p>No allocations match the selected filters.</p>
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
                {filteredAllocations.map((alloc) => (
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
        </>
      )}
    </div>
  )
}

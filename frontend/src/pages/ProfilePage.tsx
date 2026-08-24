import { useMemo, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useJobs } from '../api/queries'
import type { JobListItem } from '../api/types'
import { ErrorState } from '../components/ErrorState'
import { LoadingState } from '../components/LoadingState'
import styles from './ProfilePage.module.css'

type SortColumn = 'name' | 'submitTime'
type SortDirection = 'asc' | 'desc' | 'none'

const SORT_COLUMNS: SortColumn[] = ['name', 'submitTime']
const SORT_DIRECTIONS: SortDirection[] = ['asc', 'desc', 'none']

// Clicking a column advances it through this cycle.
const SORT_CYCLE: SortDirection[] = ['asc', 'desc', 'none']

const DEFAULT_SORT: SortState = { column: 'submitTime', direction: 'desc' }

function nextDirection(direction: SortDirection): SortDirection {
  return SORT_CYCLE[(SORT_CYCLE.indexOf(direction) + 1) % SORT_CYCLE.length]
}

function ariaSortValue(direction: SortDirection): 'ascending' | 'descending' | undefined {
  if (direction === 'asc') return 'ascending'
  if (direction === 'desc') return 'descending'
  return undefined
}

interface SortState {
  column: SortColumn
  direction: SortDirection
}

function isSortColumn(value: string | null): value is SortColumn {
  return SORT_COLUMNS.includes(value as SortColumn)
}

function isSortDirection(value: string | null): value is SortDirection {
  return SORT_DIRECTIONS.includes(value as SortDirection)
}

/** Reads sort state from the `sort`/`dir` URL params, falling back to the default when absent or invalid. */
function parseSort(searchParams: URLSearchParams): SortState {
  const column = searchParams.get('sort')
  const direction = searchParams.get('dir')

  if (isSortColumn(column) && isSortDirection(direction)) {
    return { column, direction }
  }

  return DEFAULT_SORT
}

/** Builds the full URL query string for a given (search, sort) state — defaults are omitted. */
function buildParams(search: string, sort: SortState): URLSearchParams {
  const params = new URLSearchParams()

  if (search !== '') {
    params.set('q', search)
  }

  if (sort.column !== DEFAULT_SORT.column || sort.direction !== DEFAULT_SORT.direction) {
    params.set('sort', sort.column)
    params.set('dir', sort.direction)
  }

  return params
}

interface SortableHeaderProps {
  column: SortColumn
  label: string
  sort: SortState
  onClick: (column: SortColumn) => void
}

function SortableHeader({ column, label, sort, onClick }: SortableHeaderProps) {
  const direction = sort.column === column ? sort.direction : 'none'

  return (
    <th aria-sort={ariaSortValue(direction)}>
      <button type="button" className={styles.sortButton} onClick={() => onClick(column)}>
        {label}
        {direction !== 'none' && (
          <span className={styles.sortArrow}>{direction === 'asc' ? '▲' : '▼'}</span>
        )}
      </button>
    </th>
  )
}

function sortJobs(jobs: JobListItem[], sort: SortState): JobListItem[] {
  if (sort.direction === 'none') {
    return jobs
  }

  const sorted = [...jobs].sort((a, b) => {
    if (sort.column === 'name') {
      return a.name.localeCompare(b.name)
    }
    return new Date(a.submitTime).getTime() - new Date(b.submitTime).getTime()
  })

  return sort.direction === 'desc' ? sorted.reverse() : sorted
}

export function ProfilePage() {
  const { profileName } = useParams<{ profileName: string }>()
  const { data, isLoading, error } = useJobs(profileName ?? '')
  const [, setSearchParams] = useSearchParams()

  // search/sort live in local state (initialized once from the URL on mount) rather than being derived
  // fresh from useSearchParams() on every render: React Router's setSearchParams functional updater does
  // not reliably see the result of the previous call across rapid successive calls in the same tick (e.g.
  // fast typing), so merging against its `prev` can silently drop keystrokes. Local state batches
  // correctly; the full URL is rebuilt from it (not merged) on every change.
  const [search, setSearch] = useState(() => new URLSearchParams(window.location.search).get('q') ?? '')
  const [sort, setSort] = useState<SortState>(() => parseSort(new URLSearchParams(window.location.search)))

  const filteredJobs = useMemo(() => {
    const query = search.trim().toLowerCase()
    if (query === '') {
      return data?.jobs ?? []
    }
    return (data?.jobs ?? []).filter((job) => job.name.toLowerCase().includes(query))
  }, [data, search])

  const sortedJobs = useMemo(() => sortJobs(filteredJobs, sort), [filteredJobs, sort])

  function handleSearchChange(value: string) {
    setSearch(value)
    setSearchParams(buildParams(value, sort), { replace: true })
  }

  function handleSortClick(column: SortColumn) {
    const next: SortState =
      sort.column === column ? { column, direction: nextDirection(sort.direction) } : { column, direction: 'asc' }
    setSort(next)
    setSearchParams(buildParams(search, next), { replace: true })
  }

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
        <>
          <input
            type="search"
            className={styles.search}
            placeholder="Search jobs by name…"
            value={search}
            onChange={(e) => handleSearchChange(e.target.value)}
            aria-label="Search jobs by name"
          />
          {sortedJobs.length === 0 ? (
            <p>No jobs match "{search}".</p>
          ) : (
            <table className={styles.table}>
              <thead>
                <tr>
                  <SortableHeader column="name" label="Job" sort={sort} onClick={handleSortClick} />
                  <SortableHeader column="submitTime" label="Submitted" sort={sort} onClick={handleSortClick} />
                </tr>
              </thead>
              <tbody>
                {sortedJobs.map((job) => (
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
        </>
      )}
    </div>
  )
}

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

const PAGE_SIZE_OPTIONS = [25, 50, 100, 200]
const DEFAULT_PAGE_SIZE = 50

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

/** Reads the page number from the URL, falling back to 1 when absent or invalid. */
function parsePage(searchParams: URLSearchParams): number {
  const page = Number(searchParams.get('page'))
  return Number.isInteger(page) && page > 0 ? page : 1
}

/** Reads the page size from the URL, falling back to the default when absent or not one of the offered sizes. */
function parsePageSize(searchParams: URLSearchParams): number {
  const pageSize = Number(searchParams.get('pageSize'))
  return PAGE_SIZE_OPTIONS.includes(pageSize) ? pageSize : DEFAULT_PAGE_SIZE
}

/** Builds the full URL query string for a given (search, sort, page, pageSize) state — defaults are omitted. */
function buildParams(search: string, sort: SortState, page: number, pageSize: number): URLSearchParams {
  const params = new URLSearchParams()

  if (search !== '') {
    params.set('q', search)
  }

  if (sort.column !== DEFAULT_SORT.column || sort.direction !== DEFAULT_SORT.direction) {
    params.set('sort', sort.column)
    params.set('dir', sort.direction)
  }

  if (page !== 1) {
    params.set('page', String(page))
  }
  if (pageSize !== DEFAULT_PAGE_SIZE) {
    params.set('pageSize', String(pageSize))
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

interface PaginationProps {
  page: number
  totalPages: number
  pageSize: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
}

function Pagination({ page, totalPages, pageSize, onPageChange, onPageSizeChange }: PaginationProps) {
  return (
    <div className={styles.pagination}>
      <button type="button" onClick={() => onPageChange(page - 1)} disabled={page <= 1}>
        Previous
      </button>
      <span>
        Page {page} of {totalPages}
      </span>
      <button type="button" onClick={() => onPageChange(page + 1)} disabled={page >= totalPages}>
        Next
      </button>
      <label className={styles.pageSize}>
        <span>Per page</span>
        <select value={pageSize} onChange={(e) => onPageSizeChange(Number(e.target.value))}>
          {PAGE_SIZE_OPTIONS.map((size) => (
            <option key={size} value={size}>
              {size}
            </option>
          ))}
        </select>
      </label>
    </div>
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

  const [search, setSearch] = useState(() => new URLSearchParams(window.location.search).get('q') ?? '')
  const [sort, setSort] = useState<SortState>(() => parseSort(new URLSearchParams(window.location.search)))
  const [page, setPage] = useState(() => parsePage(new URLSearchParams(window.location.search)))
  const [pageSize, setPageSize] = useState(() => parsePageSize(new URLSearchParams(window.location.search)))

  const filteredJobs = useMemo(() => {
    const query = search.trim().toLowerCase()
    if (query === '') {
      return data?.jobs ?? []
    }
    return (data?.jobs ?? []).filter((job) => job.name.toLowerCase().includes(query))
  }, [data, search])

  const sortedJobs = useMemo(() => sortJobs(filteredJobs, sort), [filteredJobs, sort])

  const totalPages = Math.max(1, Math.ceil(sortedJobs.length / pageSize))
  // Clamp rather than reset on every render, so a stale page number (e.g. from a shrunk result set)
  // can't point past the end of what's actually available.
  const effectivePage = Math.min(page, totalPages)
  const paginatedJobs = sortedJobs.slice((effectivePage - 1) * pageSize, effectivePage * pageSize)

  function handleSearchChange(value: string) {
    setSearch(value)
    setPage(1)
    setSearchParams(buildParams(value, sort, 1, pageSize), { replace: true })
  }

  function handleSortClick(column: SortColumn) {
    const next: SortState =
      sort.column === column ? { column, direction: nextDirection(sort.direction) } : { column, direction: 'asc' }
    setSort(next)
    setPage(1)
    setSearchParams(buildParams(search, next, 1, pageSize), { replace: true })
  }

  function handlePageChange(nextPage: number) {
    setPage(nextPage)
    setSearchParams(buildParams(search, sort, nextPage, pageSize), { replace: true })
  }

  function handlePageSizeChange(size: number) {
    setPageSize(size)
    setPage(1)
    setSearchParams(buildParams(search, sort, 1, size), { replace: true })
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
            <>
              <p className={styles.filterSummary}>
                Showing {(effectivePage - 1) * pageSize + 1}–
                {Math.min(effectivePage * pageSize, sortedJobs.length)} of {sortedJobs.length} jobs
              </p>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <SortableHeader column="name" label="Job" sort={sort} onClick={handleSortClick} />
                    <SortableHeader column="submitTime" label="Submitted" sort={sort} onClick={handleSortClick} />
                  </tr>
                </thead>
                <tbody>
                  {paginatedJobs.map((job) => (
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
              <Pagination
                page={effectivePage}
                totalPages={totalPages}
                pageSize={pageSize}
                onPageChange={handlePageChange}
                onPageSizeChange={handlePageSizeChange}
              />
            </>
          )}
        </>
      )}
    </div>
  )
}

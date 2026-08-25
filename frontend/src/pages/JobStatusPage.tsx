import { useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useJobStatus } from '../api/queries'
import type { ClientStatus, Port } from '../api/types'
import { CopyButton } from '../components/CopyButton'
import { ErrorState } from '../components/ErrorState'
import { LoadingState } from '../components/LoadingState'
import { StatusBadge } from '../components/StatusBadge'
import { formatDuration } from '../lib/duration'
import styles from './JobStatusPage.module.css'

// Order shown in the Versions summary badges (running first, as the state that matters most) — not the
// alphabetical order the Status filter dropdown uses, see STATUS_FILTER_OPTIONS below.
const CLIENT_STATUSES: ClientStatus[] = ['running', 'pending', 'failed', 'complete', 'lost']

// Nomad's fixed set of allocation DesiredStatus values (github.com/hashicorp/nomad/api: AllocDesiredStatus*).
const DESIRED_STATUSES = ['run', 'stop', 'evict']

// Filter dropdown options are sorted ascending (Version, from the backend, is the one exception — sorted
// numerically descending). CLIENT_STATUSES/DESIRED_STATUSES keep their own order for other uses above.
const STATUS_FILTER_OPTIONS = [...CLIENT_STATUSES].sort()
const DESIRED_FILTER_OPTIONS = [...DESIRED_STATUSES].sort()

interface Filters {
  taskGroup: string
  version: string
  node: string
  status: string
  desired: string
}

const EMPTY_FILTERS: Filters = { taskGroup: '', version: '', node: '', status: '', desired: '' }
const FILTER_KEYS = Object.keys(EMPTY_FILTERS) as (keyof Filters)[]

const PAGE_SIZE_OPTIONS = [25, 50, 100, 200]
const DEFAULT_PAGE_SIZE = 50

type SortColumn = 'id' | 'node' | 'status' | 'desired' | 'taskGroup' | 'version' | 'uptime'
type SortDirection = 'asc' | 'desc' | 'none'

const SORT_COLUMNS: SortColumn[] = ['id', 'node', 'status', 'desired', 'taskGroup', 'version', 'uptime']
const SORT_DIRECTIONS: SortDirection[] = ['asc', 'desc', 'none']

// Clicking a column advances it through this cycle.
const SORT_CYCLE: SortDirection[] = ['asc', 'desc', 'none']

interface SortState {
  column: SortColumn
  direction: SortDirection
}

// Most recently started allocations first, until the user picks a different column/direction.
const DEFAULT_SORT: SortState = { column: 'uptime', direction: 'asc' }

function nextDirection(direction: SortDirection): SortDirection {
  return SORT_CYCLE[(SORT_CYCLE.indexOf(direction) + 1) % SORT_CYCLE.length]
}

function ariaSortValue(direction: SortDirection): 'ascending' | 'descending' | undefined {
  if (direction === 'asc') return 'ascending'
  if (direction === 'desc') return 'descending'
  return undefined
}

function isSortColumn(value: string | null): value is SortColumn {
  return SORT_COLUMNS.includes(value as SortColumn)
}

function isSortDirection(value: string | null): value is SortDirection {
  return SORT_DIRECTIONS.includes(value as SortDirection)
}

/** Reads filters from the URL query params, defaulting each to '' when absent. */
function parseFilters(searchParams: URLSearchParams): Filters {
  const filters = { ...EMPTY_FILTERS }
  for (const key of FILTER_KEYS) {
    filters[key] = searchParams.get(key) ?? ''
  }
  return filters
}

/** Reads the search text from the `q` URL param, defaulting to '' when absent. */
function parseSearch(searchParams: URLSearchParams): string {
  return searchParams.get('q') ?? ''
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

/** Builds the full URL query string for a given (search, filters, sort, page, pageSize) state — defaults are omitted. */
function buildParams(
  search: string,
  filters: Filters,
  sort: SortState,
  page: number,
  pageSize: number,
): URLSearchParams {
  const params = new URLSearchParams()

  if (search !== '') {
    params.set('q', search)
  }

  for (const key of FILTER_KEYS) {
    if (filters[key] !== '') {
      params.set(key, filters[key])
    }
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

function UrlRow({ url, index, copyLabel }: { url: string; index: number; copyLabel: string }) {
  return (
    <span className={`${styles.copyableField} ${styles.urlRow} ${index % 2 === 1 ? styles.urlRowAlt : ''}`}>
      <a href={url} target="_blank" rel="noreferrer" className="mono">
        {url}
      </a>
      <CopyButton value={url} label={copyLabel} />
    </span>
  )
}

function PortLinks({ ports }: { ports: Port[] }) {
  // Rows are indexed continuously across all ports (not reset per port) so the alternating
  // background gives visual contrast between every URL line, while port name labels stay plain.
  let rowIndex = 0

  return (
    <div className={styles.ports}>
      {ports.map((port) => {
        const urlIndex = rowIndex++
        const nodeUrlIndex = port.nodeUrl ? rowIndex++ : -1
        return (
          <div key={`${port.label}-${port.port}`} className={styles.port}>
            <span className={styles.portLabel}>{port.label}</span>
            <UrlRow url={port.url} index={urlIndex} copyLabel="Copy port URL" />
            {port.nodeUrl && <UrlRow url={port.nodeUrl} index={nodeUrlIndex} copyLabel="Copy node URL" />}
          </div>
        )
      })}
    </div>
  )
}

export function JobStatusPage() {
  const { profileName, jobId } = useParams<{ profileName: string; jobId: string }>()
  const [, setSearchParams] = useSearchParams()

  // filters/page/pageSize live in local state (initialized once from the URL on mount) rather than being
  // derived fresh from useSearchParams() on every render — same reasoning as ProfilePage's search/sort:
  // React Router's setSearchParams functional updater can drop rapid successive updates, so the full URL
  // is rebuilt from local state (not merged) on every change instead.
  const [search, setSearch] = useState(() => parseSearch(new URLSearchParams(window.location.search)))
  const [filters, setFilters] = useState<Filters>(() => parseFilters(new URLSearchParams(window.location.search)))
  const [sort, setSort] = useState<SortState>(() => parseSort(new URLSearchParams(window.location.search)))
  const [page, setPage] = useState(() => parsePage(new URLSearchParams(window.location.search)))
  const [pageSize, setPageSize] = useState(() => parsePageSize(new URLSearchParams(window.location.search)))

  const { data, isLoading, error } = useJobStatus(profileName ?? '', jobId ?? '', {
    q: search,
    taskGroup: filters.taskGroup,
    version: filters.version,
    node: filters.node,
    status: filters.status,
    desired: filters.desired,
    sort: sort.direction !== 'none' ? sort.column : undefined,
    dir: sort.direction !== 'none' ? sort.direction : undefined,
    page,
    pageSize,
  })

  function handleSearchChange(value: string) {
    setSearch(value)
    setPage(1)
    setSearchParams(buildParams(value, filters, sort, 1, pageSize), { replace: true })
  }

  function setFilter(key: keyof Filters, value: string) {
    const nextFilters = { ...filters, [key]: value }
    setFilters(nextFilters)
    setPage(1)
    setSearchParams(buildParams(search, nextFilters, sort, 1, pageSize), { replace: true })
  }

  function handleSortClick(column: SortColumn) {
    const next: SortState =
      sort.column === column ? { column, direction: nextDirection(sort.direction) } : { column, direction: 'asc' }
    setSort(next)
    setPage(1)
    setSearchParams(buildParams(search, filters, next, 1, pageSize), { replace: true })
  }

  function handlePageChange(nextPage: number) {
    setPage(nextPage)
    setSearchParams(buildParams(search, filters, sort, nextPage, pageSize), { replace: true })
  }

  function handlePageSizeChange(size: number) {
    setPageSize(size)
    setPage(1)
    setSearchParams(buildParams(search, filters, sort, 1, size), { replace: true })
  }

  function handleClearFilters() {
    setSearch('')
    setFilters(EMPTY_FILTERS)
    setSort(DEFAULT_SORT)
    setPage(1)
    setSearchParams(buildParams('', EMPTY_FILTERS, DEFAULT_SORT, 1, pageSize), { replace: true })
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

  const hasActiveFilters =
    search !== '' ||
    sort.column !== DEFAULT_SORT.column ||
    sort.direction !== DEFAULT_SORT.direction ||
    Object.values(filters).some((v) => v !== '')
  const noAllocationsAtAll = data.pagination.totalItems === 0 && !hasActiveFilters
  const noAllocationsMatchFilters = data.pagination.totalItems === 0 && hasActiveFilters

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
      {noAllocationsAtAll ? (
        <p>No allocations found.</p>
      ) : (
        <>
          <input
            type="search"
            className={styles.search}
            placeholder="Search allocations by ID or node…"
            value={search}
            onChange={(e) => handleSearchChange(e.target.value)}
            aria-label="Search allocations by allocation ID or node name"
          />
          <div className={styles.filterBar}>
            <FilterSelect
              label="Task Group"
              value={filters.taskGroup}
              options={data.filterOptions.taskGroups}
              onChange={(v) => setFilter('taskGroup', v)}
            />
            <FilterSelect
              label="Version"
              value={filters.version}
              options={data.filterOptions.versions.map(String)}
              onChange={(v) => setFilter('version', v)}
            />
            <FilterSelect
              label="Node"
              value={filters.node}
              options={data.filterOptions.nodes}
              onChange={(v) => setFilter('node', v)}
            />
            <FilterSelect
              label="Status"
              value={filters.status}
              options={STATUS_FILTER_OPTIONS}
              onChange={(v) => setFilter('status', v)}
            />
            <FilterSelect
              label="Desired"
              value={filters.desired}
              options={DESIRED_FILTER_OPTIONS}
              onChange={(v) => setFilter('desired', v)}
            />
            {hasActiveFilters && (
              <button type="button" className={styles.clearFilters} onClick={handleClearFilters}>
                Clear filters
              </button>
            )}
          </div>
          {noAllocationsMatchFilters ? (
            <p>No allocations match the selected filters.</p>
          ) : (
            <>
              <p className={styles.filterSummary}>
                Showing {(data.pagination.page - 1) * data.pagination.pageSize + 1}–
                {Math.min(data.pagination.page * data.pagination.pageSize, data.pagination.totalItems)} of{' '}
                {data.pagination.totalItems} allocations
              </p>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <SortableHeader column="id" label="Allocation" sort={sort} onClick={handleSortClick} />
                    <SortableHeader column="node" label="Node" sort={sort} onClick={handleSortClick} />
                    <SortableHeader column="status" label="Status" sort={sort} onClick={handleSortClick} />
                    <SortableHeader column="desired" label="Desired" sort={sort} onClick={handleSortClick} />
                    <SortableHeader column="taskGroup" label="Task Group" sort={sort} onClick={handleSortClick} />
                    <SortableHeader column="version" label="Version" sort={sort} onClick={handleSortClick} />
                    <SortableHeader column="uptime" label="Uptime" sort={sort} onClick={handleSortClick} />
                    <th>Ports</th>
                  </tr>
                </thead>
                <tbody>
                  {data.allocations.map((alloc) => (
                    <tr key={alloc.id}>
                      <td className="mono">
                        <span className={styles.copyableField}>
                          {alloc.id}
                          <CopyButton value={alloc.id} label="Copy allocation ID" />
                        </span>
                      </td>
                      <td>
                        <div className={styles.stackedRows}>
                          <div className={styles.copyableField}>
                            {alloc.nodeName}
                            <CopyButton value={alloc.nodeName} label="Copy node name" />
                          </div>
                          <div className={`${styles.copyableField} ${styles.nodeIp}`}>
                            <span className="mono">{alloc.nodeIp}</span>
                            <CopyButton value={alloc.nodeIp} label="Copy node IP" />
                          </div>
                        </div>
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
              <Pagination
                page={data.pagination.page}
                totalPages={data.pagination.totalPages}
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

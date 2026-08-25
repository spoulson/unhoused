import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { fetchJSON } from './client'
import type { JobStatusResponse, JobsResponse, ProfilesResponse } from './types'

const DEFAULT_REFRESH_INTERVAL_SECONDS = 5

export interface JobStatusParams {
  q?: string
  taskGroup?: string
  version?: string
  node?: string
  status?: string
  desired?: string
  sort?: string
  dir?: string
  page: number
  pageSize: number
}

function jobStatusQueryString(params: JobStatusParams): string {
  const search = new URLSearchParams()
  if (params.q) search.set('q', params.q)
  if (params.taskGroup) search.set('taskGroup', params.taskGroup)
  if (params.version) search.set('version', params.version)
  if (params.node) search.set('node', params.node)
  if (params.status) search.set('status', params.status)
  if (params.desired) search.set('desired', params.desired)
  if (params.sort) search.set('sort', params.sort)
  if (params.dir) search.set('dir', params.dir)
  search.set('page', String(params.page))
  search.set('pageSize', String(params.pageSize))
  return search.toString()
}

export function useProfiles() {
  return useQuery({
    queryKey: ['profiles'],
    queryFn: () => fetchJSON<ProfilesResponse>('/api/profiles'),
  })
}

export function useJobs(profileName: string) {
  return useQuery({
    queryKey: ['jobs', profileName],
    queryFn: () => fetchJSON<JobsResponse>(`/api/profiles/${encodeURIComponent(profileName)}/jobs`),
  })
}

export function useJobStatus(profileName: string, jobId: string, params: JobStatusParams) {
  const { data: profiles } = useProfiles()
  const refreshIntervalSeconds = profiles?.refreshIntervalSeconds ?? DEFAULT_REFRESH_INTERVAL_SECONDS

  return useQuery({
    queryKey: ['jobStatus', profileName, jobId, params],
    queryFn: () =>
      fetchJSON<JobStatusResponse>(
        `/api/profiles/${encodeURIComponent(profileName)}/jobs/${encodeURIComponent(jobId)}?${jobStatusQueryString(params)}`,
      ),
    refetchInterval: refreshIntervalSeconds * 1000,
    // Changing a filter/page/pageSize changes the query key. Without this, TanStack Query would
    // clear `data` and show the loading state on every such change (not just the initial mount),
    // unmounting the filter bar/table mid-interaction. Keeping the previous page's data visible
    // during the refetch avoids that flicker and focus loss.
    placeholderData: keepPreviousData,
  })
}

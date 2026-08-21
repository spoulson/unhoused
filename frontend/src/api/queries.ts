import { useQuery } from '@tanstack/react-query'
import { fetchJSON } from './client'
import type { JobStatusResponse, JobsResponse, ProfilesResponse } from './types'

const DEFAULT_REFRESH_INTERVAL_SECONDS = 5

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

export function useJobStatus(profileName: string, jobId: string) {
  const { data: profiles } = useProfiles()
  const refreshIntervalSeconds = profiles?.refreshIntervalSeconds ?? DEFAULT_REFRESH_INTERVAL_SECONDS

  return useQuery({
    queryKey: ['jobStatus', profileName, jobId],
    queryFn: () =>
      fetchJSON<JobStatusResponse>(
        `/api/profiles/${encodeURIComponent(profileName)}/jobs/${encodeURIComponent(jobId)}`,
      ),
    refetchInterval: refreshIntervalSeconds * 1000,
  })
}

// Mirrors internal/httpapi/dto.go exactly. See specs/api.md for the contract.

export type JobStatus = 'running' | 'pending' | 'stopped' | 'dead'
export type ClientStatus = 'running' | 'pending' | 'failed' | 'complete' | 'lost'

export interface Profile {
  name: string
}

export interface ProfilesResponse {
  refreshIntervalSeconds: number
  profiles: Profile[]
}

export interface JobListItem {
  id: string
  name: string
  submitTime: string
}

export interface JobsResponse {
  jobs: JobListItem[]
}

export interface Job {
  id: string
  name: string
  status: JobStatus
}

export interface VersionGroup {
  version: number
  newestAllocationLastModifiedSeconds: number
  statusCounts: Record<ClientStatus, number>
}

export interface Port {
  label: string
  ip: string
  port: number
  address: string
  nodeAddress: string
}

export interface Allocation {
  id: string
  nodeName: string
  nodeIp: string
  clientStatus: ClientStatus
  desiredStatus: string
  taskGroup: string
  version: number
  lastModifiedSeconds: number
  ports: Port[]
}

export interface Pagination {
  page: number
  pageSize: number
  totalItems: number
  totalPages: number
}

export interface FilterOptions {
  taskGroups: string[]
  versions: number[]
  nodes: string[]
}

export interface JobStatusResponse {
  job: Job
  versionGroups: VersionGroup[]
  pagination: Pagination
  filterOptions: FilterOptions
  allocations: Allocation[]
}

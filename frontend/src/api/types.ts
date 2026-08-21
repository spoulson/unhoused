// Mirrors internal/httpapi/dto.go exactly. See specs/api.md for the contract.

export type Environment = 'staging' | 'production'
export type Region = 'us-west1' | 'us-east4' | 'europe-west1' | 'ause1'

export type JobStatus = 'running' | 'pending' | 'stopped' | 'dead'
export type ClientStatus = 'running' | 'pending' | 'failed' | 'complete' | 'lost'

export interface Profile {
  name: string
  environment: Environment
  region: Region
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
  newestAllocationUptimeSeconds: number
  statusCounts: Record<ClientStatus, number>
}

export interface Port {
  label: string
  ip: string
  port: number
  url: string
  nodeUrl?: string
}

export interface Allocation {
  id: string
  nodeName: string
  nodeIp: string
  clientStatus: ClientStatus
  desiredStatus: string
  taskGroup: string
  version: number
  uptimeSeconds: number
  ports: Port[]
}

export interface JobStatusResponse {
  job: Job
  versionGroups: VersionGroup[]
  allocations: Allocation[]
}

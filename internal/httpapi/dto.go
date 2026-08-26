package httpapi

import "time"

type errorEnvelope struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Message string `json:"message"`
}

type profileDTO struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Region      string `json:"region"`
}

type profilesResponse struct {
	RefreshIntervalSeconds int          `json:"refreshIntervalSeconds"`
	Profiles               []profileDTO `json:"profiles"`
}

type jobListItemDTO struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	SubmitTime time.Time `json:"submitTime"`
}

type jobsResponse struct {
	Jobs []jobListItemDTO `json:"jobs"`
}

type jobDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type versionGroupDTO struct {
	Version                             uint64         `json:"version"`
	NewestAllocationLastModifiedSeconds int64          `json:"newestAllocationLastModifiedSeconds"`
	StatusCounts                        map[string]int `json:"statusCounts"`
}

type portDTO struct {
	Label       string `json:"label"`
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	Address     string `json:"address"`
	NodeAddress string `json:"nodeAddress"`
}

type allocationDTO struct {
	ID                  string    `json:"id"`
	NodeName            string    `json:"nodeName"`
	NodeIP              string    `json:"nodeIp"`
	ClientStatus        string    `json:"clientStatus"`
	DesiredStatus       string    `json:"desiredStatus"`
	TaskGroup           string    `json:"taskGroup"`
	Version             uint64    `json:"version"`
	LastModifiedSeconds int64     `json:"lastModifiedSeconds"`
	Ports               []portDTO `json:"ports"`
}

type paginationDTO struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

type filterOptionsDTO struct {
	TaskGroups []string `json:"taskGroups"`
	Versions   []uint64 `json:"versions"`
	Nodes      []string `json:"nodes"`
}

type jobStatusResponse struct {
	Job           jobDTO            `json:"job"`
	VersionGroups []versionGroupDTO `json:"versionGroups"`
	Pagination    paginationDTO     `json:"pagination"`
	FilterOptions filterOptionsDTO  `json:"filterOptions"`
	Allocations   []allocationDTO   `json:"allocations"`
}

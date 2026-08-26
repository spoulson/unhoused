# REST API

Contract between the React frontend and the Go backend. Derived from
[functional_requirements.md](functional_requirements.md). The backend is the only component that holds
Nomad service URLs and API tokens (see Configuration) — the frontend never talks to Nomad directly, and
these values are never returned in API responses.

## Conventions

- Base path: `/api`
- JSON request/response bodies, `Content-Type: application/json`
- Field names: `camelCase`
- Errors use a consistent envelope and appropriate HTTP status:

  ```json
  { "error": { "message": "human-readable description" } }
  ```

  - `404` — unknown profile / job / allocation
  - `502` — Nomad API unreachable or returned an error
- Timestamps: RFC 3339 strings. Durations (last modified): whole seconds as integers.
- The frontend polls job status endpoints on an interval (default 5s per functional requirements) rather
  than the backend pushing updates — no websocket/SSE is specified.

## Endpoints

### `GET /api/profiles`

Lists configured profiles for the home page, and the refresh interval the frontend should poll at.

Response `200`:

```json
{
  "refreshIntervalSeconds": 5,
  "profiles": [
    { "name": "prod-usw1" }
  ]
}
```

`name` is the profile identifier used in later routes — Nomad URL and token are intentionally omitted.

### `GET /api/profiles/{profile}/jobs`

Lists jobs for the Environment Page, sorted newest to oldest.

Response `200`:

```json
{
  "jobs": [
    { "id": "web", "name": "web", "submitTime": "2026-08-10T12:00:00Z" }
  ]
}
```

Errors: `404` if `{profile}` doesn't match a configured profile; `502` if the Nomad API call fails.

### `GET /api/profiles/{profile}/jobs/{jobId}`

Job Status Page data: allocation counts grouped by version then status, plus a filtered/paginated page of
the allocation table.

Query parameters (all optional):

| Param      | Meaning                                                              | Default |
|------------|-----------------------------------------------------------------------|---------|
| `q`        | Case-insensitive substring match against allocation ID, node name, or node IP | (none — no filter) |
| `taskGroup`| Exact match against allocation task group                            | (none — no filter) |
| `version`  | Exact match against allocation job version (compared as a string)     | (none — no filter) |
| `node`     | Exact match against allocation node name                              | (none — no filter) |
| `status`   | Exact match against allocation client status                          | (none — no filter) |
| `desired`  | Exact match against allocation desired status                         | (none — no filter) |
| `page`     | 1-based page number. Invalid/non-positive values fall back to the default. | `1` |
| `pageSize` | Rows per page. Invalid/non-positive values fall back to the default; clamped to a maximum of `500`. | `50` |
| `sort`     | Column to sort the allocation table by. One of `id`, `node`, `status`, `desired`, `taskGroup`, `version`, `lastModified`. | (none — Nomad's returned order) |
| `dir`      | Sort direction: `asc` or `desc`. Required alongside `sort` — either missing, or either param invalid, leaves the table unsorted. | (none) |

All parameters combine with AND — e.g. `q=node1&status=running` returns only allocations matching both.

Response `200`:

```json
{
  "job": { "id": "web", "name": "web", "status": "running" },
  "versionGroups": [
    {
      "version": 3,
      "newestAllocationLastModifiedSeconds": 1234,
      "statusCounts": { "running": 5, "pending": 1, "failed": 0, "complete": 0, "lost": 0 }
    }
  ],
  "pagination": { "page": 1, "pageSize": 50, "totalItems": 137, "totalPages": 3 },
  "filterOptions": { "taskGroups": ["web"], "versions": [3, 2], "nodes": ["node1", "node2"] },
  "allocations": [
    {
      "id": "3f9a1e2b-...",
      "nodeName": "node1",
      "nodeIp": "10.0.0.5",
      "clientStatus": "running",
      "desiredStatus": "run",
      "taskGroup": "web",
      "version": 3,
      "lastModifiedSeconds": 1234,
      "ports": [
        {
          "label": "http",
          "ip": "10.0.0.5",
          "port": 8080,
          "address": "10.0.0.5:8080",
          "nodeAddress": "node1.node.us-west1.staging.local:8080"
        }
      ]
    }
  ]
}
```

- `job.status` is the top-level indicator for the Job Status Page header: one of `running`, `pending`,
  `stopped`, or `dead`.
- `versionGroups` is sorted newest version first; `statusCounts` keys are the Nomad client statuses
  (`running`, `pending`, `failed`, `complete`, `lost`). Computed from **all** of the job's allocations,
  unaffected by the filter query parameters above — it represents overall job health, not the filtered
  table view.
- `pagination.page`/`pagination.pageSize` echo the effective values used after applying the defaults/clamps
  above. `totalItems`/`totalPages` are computed from the **filtered** allocation set. If the requested
  `page` is beyond `totalPages` (e.g. a filter change shrank the result set), the response clamps to the
  last valid page instead of returning an empty page — `pagination.page` reflects the page actually
  returned.
- `filterOptions` lists the distinct `taskGroup`/`version`/`nodeName` values across **all** of the job's
  allocations (unfiltered), so filter dropdowns can always offer every possible value regardless of which
  filters are currently active. `status`/`desired` aren't included since those are Nomad's fixed enums.
- `allocations` holds only the current page (`pageSize` items or fewer on the last page) of the filtered,
  sorted allocation list, in the same order as before pagination was introduced.
- `lastModifiedSeconds` (per allocation) and `newestAllocationLastModifiedSeconds` (per version group) are
  both `now - submitTime`, where `submitTime` is the Nomad job version's `SubmitTime` (Nomad's per-version
  `GET /v1/job/{id}/versions` data) matching that allocation's `version` — not each allocation's own
  `CreateTime`. All allocations within a version group therefore share the same last-modified value.
- `ports` includes one entry per network port defined on the allocation. `address` (`<ip>:<port>`) and
  `nodeAddress` (`<host>:<port>`, using the profile's configured node hostname template from the
  functional requirements) are always present — computed server-side so the frontend doesn't need to
  know the per-profile hostname template.

Errors: `404` if `{profile}` or `{jobId}` don't exist; `502` if the Nomad API call fails.

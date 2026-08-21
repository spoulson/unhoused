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
- Timestamps: RFC 3339 strings. Durations (uptime): whole seconds as integers.
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
    { "name": "prod-usw1", "environment": "production", "region": "us-west1" }
  ]
}
```

`name` is the profile identifier used in later routes. `environment` / `region` are only the values listed
in the Configuration section (`staging`/`production`, and the four supported regions) — Nomad URL and token
are intentionally omitted.

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

Job Status Page data: allocation counts grouped by version then status, plus the full allocation table.

Response `200`:

```json
{
  "job": { "id": "web", "name": "web", "status": "running" },
  "versionGroups": [
    {
      "version": 3,
      "newestAllocationUptimeSeconds": 1234,
      "statusCounts": { "running": 5, "pending": 1, "failed": 0, "complete": 0, "lost": 0 }
    }
  ],
  "allocations": [
    {
      "id": "3f9a1e2b-...",
      "nodeName": "node1",
      "nodeIp": "10.0.0.5",
      "clientStatus": "running",
      "desiredStatus": "run",
      "taskGroup": "web",
      "version": 3,
      "uptimeSeconds": 1234,
      "ports": [
        {
          "label": "http",
          "ip": "10.0.0.5",
          "port": 8080,
          "url": "http://10.0.0.5:8080",
          "nodeUrl": "http://node1.node.us-west1.staging.mailforce:8080"
        }
      ]
    }
  ]
}
```

- `job.status` is the top-level indicator for the Job Status Page header: one of `running`, `pending`,
  `stopped`, or `dead`.
- `versionGroups` is sorted newest version first; `statusCounts` keys are the Nomad client statuses
  (`running`, `pending`, `failed`, `complete`, `lost`).
- `uptimeSeconds` (per allocation) and `newestAllocationUptimeSeconds` (per version group) are both
  `now - submitTime`, where `submitTime` is the Nomad job version's `SubmitTime` (Nomad's per-version
  `GET /v1/job/{id}/versions` data) matching that allocation's `version` — not each allocation's own
  `CreateTime`. All allocations within a version group therefore share the same uptime.
- `ports` includes one entry per network port defined on the allocation. `url` (`http://<ip>:<port>`) is
  always present. `nodeUrl` is present only when `label == "http"`, and holds the environment/region-derived
  hostname link from the functional requirements — computed server-side so the frontend doesn't need to
  know the per-env/region hostname rules.

Errors: `404` if `{profile}` or `{jobId}` don't exist; `502` if the Nomad API call fails.

# Architecture

- A web UI that displays Nomad job and allocation status, polling or streaming data from the Nomad
HTTP API via `github.com/hashicorp/nomad/api`.

## Frontend

- Web UI frontend is built on React framework and talks to the backend from the browser via REST API.
- Build tooling (recommended, confirm before/while implementing): Vite + React + TypeScript. React Router
  for the three pages (Home, Profile, Job Status). TanStack Query for data fetching — its built-in
  `refetchInterval` maps directly onto the polling behavior in [api.md](api.md) (`refreshIntervalSeconds`
  from `GET /api/profiles`), so no separate state-management library is needed. Plain CSS / CSS Modules
  rather than a component/design-system library, since [style.md](style.md) specifies a custom look
  (Gruvbox palette, distinct text/monospace fonts) that a generic UI kit would work against.

## Backend

- This web app will have Go module name: `unhoused`.
- Requires a configuration file passed with CLI argument `-c <file>`.

## Reverse Proxy Required

- Both frontend and backend services must be both running and listening on separate TCP ports.
- A reverse proxy is needed to seamlessly integrate both services to be accessed by a common public base URL.
- Technology (recommended, confirm before/while implementing): Caddy. Minimal Caddyfile config to route
  `/api/*` to the backend port and everything else to the frontend port, and a small official Docker image
  that fits cleanly into the `docker-compose.yaml` test environment below.

## Authentication

- No authentication is required to access the web UI or REST API. Access control, if any, is expected to
  be handled outside this app (e.g. network placement).

# Docker Test Environment

- Provide a test configuration in `docker-compose.yaml` to host all services and be accessible from
  a single computer.


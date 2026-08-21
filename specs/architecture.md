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

## Reverse Proxy

- Both frontend and backend services run as separate processes, each listening on its own TCP port
  (backend `3001` by default, see [configuration.md](configuration.md); frontend `3000`, both in the Docker
  test environment below and in local dev via `vite.config.ts`).
- [Caddy](https://caddyserver.com) is the reverse proxy, configured via a single `Caddyfile` at the repo
  root:
  - `/api/*` → reverse-proxied to the backend
  - everything else → reverse-proxied to the frontend
- Caddy is the single public entry point; only its port is published to the host / reachable from a
  browser. The frontend and backend ports are internal-only, not published.

## Authentication

- No authentication is required to access the web UI or REST API. Access control, if any, is expected to
  be handled outside this app (e.g. network placement).

# Docker Test Environment

`docker-compose.yaml` at the repo root defines three services on Docker's default bridge network, so they
reach each other by service name:

| Service    | Built from              | Internal port | Published to host |
|------------|--------------------------|----------------|--------------------|
| `backend`  | `Dockerfile` (repo root) | 3001           | no                 |
| `frontend` | `frontend/Dockerfile`    | 3000           | no                 |
| `caddy`    | `caddy:2-alpine` image   | 8080           | `8080:8080`        |

- `backend`: multi-stage build (`golang:1.25-alpine` → `alpine`), runs the compiled `unhoused` binary
  against a config file mounted read-only at `/etc/unhoused/config.yaml` (the image's default `-c` path).
  The compose file mounts `config.example.yaml` — its placeholder Nomad URLs/tokens are what the test
  environment ships with. Never mount a real config file containing live Nomad tokens; per
  [configuration.md](configuration.md) those are plaintext, and this repo's `.gitignore` deliberately
  excludes any `config*.yaml` other than the example.
- `frontend`: multi-stage build (`node:22-alpine`) — `npm run build` produces the static bundle, served in
  the final stage by [`serve`](https://github.com/vercel/serve) on port 3000. It's a standalone service
  (not served directly by Caddy) to match the Reverse Proxy section's frontend/backend-as-separate-services
  constraint.
- `caddy`: routes per the Reverse Proxy section above. `http://localhost:8080` is the one URL to open in a
  browser to reach the whole app.
- If the backend's mounted config should point `nomadUrl` at a Nomad agent running on the host machine
  (rather than another container), use `http://host.docker.internal:4646` instead of `127.0.0.1:4646` —
  inside a container, `127.0.0.1` refers to the container itself, not the host.


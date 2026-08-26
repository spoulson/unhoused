# unhoused

<img src="frontend/public/logo.svg" alt="unhoused logo" width="120" />

An alternative web UI for monitoring [HashiCorp Nomad](https://www.nomadproject.io/) job and allocation status.

The backend (Go) polls the Nomad HTTP API and serves a REST API; the frontend (React) consumes it. Point `unhoused` at
one or more Nomad environments (profiles) and browse jobs, versions, and allocations without needing Nomad's own UI or
CLI.

## Quick start

First, create a backend configuration by copying `config.example.yaml` to `config.yaml`, then edit to setup your Nomad
profiles.

### Docker (recommended)

```
make docker
docker compose up
```

Open `http://localhost:8080`. This builds and runs the backend, frontend, and a Caddy reverse proxy together, using
`config.yaml` as the backend's config.

For live-reloading local iteration (source edits apply without a rebuild):

```
make dev
```

`docker compose down` stops and removes the containers.

### Running locally without Docker

Backend (repo root) — set up a config file first, see [Configuration](#configuration) below, then:

```
go run . -c config.yaml
```

Frontend (`frontend/`), in a separate terminal:

```
cd frontend
npm install
npm run dev
```

The dev server proxies `/api` to `http://localhost:3001`, the backend's default port, so run both together.

## Configuration

The backend takes a YAML config file via `-c <file>`. See [specs/configuration.md](specs/configuration.md) for the full
reference.

### Setup

1. Copy the example file and edit it:

   ```
   cp config.example.yaml config.yaml
   ```

2. Fill in a Nomad URL and API token for each environment you want to monitor (`profiles` below).
3. Run the backend against it:

   ```
   go run . -c config.yaml
   ```

`config.yaml` is gitignored — the repo only tracks `config.example.yaml`, which ships with placeholder values and no
real token. Never commit a config file containing a live Nomad token.

### Fields

Service settings, top-level:

| Field | Meaning | Default |
|---|---|---|
| `httpPublicUrl` | Public URL used for links generated within the app | `http://localhost` |
| `listenPort` | REST API listen port | `3001` |
| `refreshIntervalSeconds` | Poll interval the frontend uses on the Job Status page | — |

`profiles`: one or more entries, each describing a Nomad environment to monitor:

| Field | Meaning |
|---|---|
| `name` | Profile identifier, shown in the UI |
| `nomadUrl` | Nomad HTTP API URL (usually port `4646`) |
| `nomadToken` | Nomad API token, in plaintext |
| `nodeHostnameTemplate` | Template for deriving each port's node address, with `{node}` replaced by the Nomad node name (e.g. `{node}.node.us-west1.staging.example.com`). Optional, defaults to `{node}` |

Nomad tokens are only ever held by the backend — the frontend talks to the backend's REST API, never to Nomad directly.

## Architecture

See [specs/architecture.md](specs/architecture.md) for the full picture, including the Docker Compose setup. Summary:

- **Frontend**: React + TypeScript + Vite, React Router, TanStack Query. Plain CSS Modules (Gruvbox-based theme, see
  [specs/style.md](specs/style.md)).
- **Backend**: Go (module `unhoused`), talks to Nomad via `github.com/hashicorp/nomad/api`.
- **Reverse proxy**: [Caddy](https://caddyserver.com) is the single public entry point in the Docker environment,
  routing `/api/*` to the backend and everything else to the frontend.
- **Auth**: none — access control, if needed, is expected to be handled outside the app (e.g. network placement).

## Development

### Backend (Go, repo root)

| Command | Purpose |
|---|---|
| `go build ./...` | Build |
| `go run . -c <config-file>` | Run |
| `go test ./...` | Test all |
| `go test ./path/to/pkg -run TestName` | Test one |
| `go vet ./...` | Vet |
| `gofmt -l .` / `go fmt ./...` | Format (list / apply) |
| `make lint` | Lint (installs `golangci-lint` into `./bin` if needed) |

### Frontend (React, `frontend/`)

| Command | Purpose |
|---|---|
| `npm install` | Install dependencies |
| `npm run dev` | Dev server |
| `npm run build` | Build (`tsc -b && vite build`) |
| `npm run lint` | Lint (oxlint) |

### Full stack (Docker)

| Command | Purpose |
|---|---|
| `docker compose up` | Run backend, frontend, and Caddy together |
| `docker compose down` | Stop and remove containers |
| `make dev` | Run backend, frontend, and Caddy together in dev mode — frontend and backend source bind-mounted for live reload |
| `make docker` | Build production images (`unhoused-backend:latest`, `unhoused-frontend:latest`) |

## Documentation

- [specs/architecture.md](specs/architecture.md) — system design, Docker environment
- [specs/configuration.md](specs/configuration.md) — config file reference
- [specs/api.md](specs/api.md) — REST API contract
- [specs/conventions.md](specs/conventions.md) — code conventions
- [specs/style.md](specs/style.md) — visual style
- [specs/functional_requirements.md](specs/functional_requirements.md) — feature requirements

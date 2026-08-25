# unhoused

An alternative web UI for monitoring [HashiCorp Nomad](https://www.nomadproject.io/) job and allocation status.

The backend (Go) polls the Nomad HTTP API and serves a REST API; the frontend (React) consumes it. Point
`unhoused` at one or more Nomad environments (profiles) and browse jobs, versions, and allocations without
needing Nomad's own UI or CLI.

## Quick start

### Docker (recommended)

```
docker compose up --build
```

Open `http://localhost:8080`. This builds and runs the backend, frontend, and a Caddy reverse proxy
together, using `config.example.yaml` (placeholder Nomad URLs/tokens) as the backend's config.

For live-reloading local iteration (source edits apply without a rebuild):

```
make dev
```

`docker compose down` stops and removes the containers.

### Running locally without Docker

Backend (repo root):

```
go run . -c config.example.yaml
```

Copy `config.example.yaml` to `config.yaml` and fill in your own Nomad URL(s) and token(s) first — see
[Configuration](#configuration) below.

Frontend (`frontend/`), in a separate terminal:

```
cd frontend
npm install
npm run dev
```

The dev server proxies `/api` to `http://localhost:3001`, the backend's default port, so run both
together.

## Configuration

The backend takes a YAML config file via `-c <file>`. See [config.example.yaml](config.example.yaml) for
a starting point and [specs/configuration.md](specs/configuration.md) for the full reference. In brief:

- Service settings: public URL, listen port (default `3001`), and the Job Status page's refresh interval.
- One or more **profiles**, each describing a Nomad environment: name, environment (`staging` /
  `production`), region, Nomad URL, and Nomad API token (plaintext).

Nomad tokens are only ever held by the backend — the frontend talks to the backend's REST API, never to
Nomad directly. Don't commit a config file containing a real token; the repo's `.gitignore` excludes any
`config*.yaml` other than `config.example.yaml`.

## Architecture

See [specs/architecture.md](specs/architecture.md) for the full picture, including the Docker Compose
setup. Summary:

- **Frontend**: React + TypeScript + Vite, React Router, TanStack Query. Plain CSS Modules (Gruvbox-based
  theme, see [specs/style.md](specs/style.md)).
- **Backend**: Go (module `unhoused`), talks to Nomad via `github.com/hashicorp/nomad/api`.
- **Reverse proxy**: [Caddy](https://caddyserver.com) is the single public entry point in the Docker
  environment, routing `/api/*` to the backend and everything else to the frontend.
- **Auth**: none — access control, if needed, is expected to be handled outside the app (e.g. network
  placement).

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
| `docker compose up --build` | Run backend, frontend, and Caddy together |
| `docker compose down` | Stop and remove containers |
| `make dev` | Dev mode — frontend source bind-mounted for live reload |
| `make docker` | Build production images (`unhoused-backend:latest`, `unhoused-frontend:latest`) |

## Documentation

- [specs/architecture.md](specs/architecture.md) — system design, Docker environment
- [specs/configuration.md](specs/configuration.md) — config file reference
- [specs/api.md](specs/api.md) — REST API contract
- [specs/conventions.md](specs/conventions.md) — code conventions
- [specs/style.md](specs/style.md) — visual style
- [specs/functional_requirements.md](specs/functional_requirements.md) — feature requirements

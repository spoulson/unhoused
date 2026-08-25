# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`unhoused` is a web UI tool written in Go for monitoring HashiCorp Nomad job and allocation status.

## Commands

### Backend (Go, repo root)

- Build: `go build ./...`
- Run: `go run . -c <config-file>` (see [specs/configuration.md](specs/configuration.md); `config.example.yaml` is a starting point)
- Test all: `go test ./...`
- Test single: `go test ./path/to/pkg -run TestName`
- Vet: `go vet ./...`
- Format: `gofmt -l .` (list) / `go fmt ./...` (apply)
- Lint: `make lint` — installs golangci-lint into `./bin` (scoped to this repo, not system-wide) if not
  already present, then runs `golangci-lint run ./...`

### Frontend (React, `frontend/`)

- Install: `cd frontend && npm install`
- Dev server: `npm run dev` (proxies `/api` to `http://localhost:3001`, the backend's default port — run
  the backend alongside it)
- Build: `npm run build` (`tsc -b && vite build`)
- Lint: `npm run lint` (oxlint)

### Full stack (Docker)

- `docker compose up --build` — builds and runs backend, frontend, and a Caddy reverse proxy together (see
  [specs/architecture.md](specs/architecture.md)'s Docker Test Environment section). Open
  `http://localhost:8080` in a browser once it's up.
- `docker compose down` — stop and remove the containers.
- `make dev` — dev mode: frontend runs Vite's dev server bind-mounted to `frontend/`, so source edits show
  up on a browser reload without a rebuild (see [specs/architecture.md](specs/architecture.md)'s Docker Dev
  Mode section). Equivalent to `docker compose -f docker-compose.yaml -f docker-compose.dev.yaml up --build`.
- `make docker` — builds the production backend and frontend images, tagged `unhoused-backend:latest` and
  `unhoused-frontend:latest` respectively.

## Architecture

See [specs/architecture.md](specs/architecture.md).

## Configuration

See [specs/configuration.md](specs/configuration.md).

## REST API

See [specs/api.md](specs/api.md).

## Conventions

See [specs/conventions.md](specs/conventions.md).

## Style and Appearance

See [specs/style.md](specs/style.md).

## Functional Requirements

See [specs/functional_requirements.md](specs/functional_requirements.md).


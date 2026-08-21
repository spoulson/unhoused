# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`unhoused` is a web UI tool written in Go for monitoring HashiCorp Nomad job and allocation status.

## Status

This repository is currently an empty scaffold — no source files exist yet. The sections below are starting
points based on standard Go conventions and the Nomad API; update them as soon as real structure, tooling,
and architecture decisions are in place.

## Commands

- Build: `go build ./...`
- Run: `go run . -c <config-file>` (see [specs/configuration.md](specs/configuration.md); `config.example.yaml` is a starting point)
- Test all: `go test ./...`
- Test single: `go test ./path/to/pkg -run TestName`
- Vet: `go vet ./...`
- Format: `gofmt -l .` (list) / `go fmt ./...` (apply)
- Lint: `make lint` — installs golangci-lint into `./bin` (scoped to this repo, not system-wide) if not
  already present, then runs `golangci-lint run ./...`

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


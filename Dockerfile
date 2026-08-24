# Backend image. See specs/architecture.md's Docker Test Environment section.

# Dev stage: used by docker-compose.dev.yaml, which bind-mounts the repo root over /src so edits
# are picked up without rebuilding this image. Runs under gow (github.com/mitranim/gow), which
# watches .go files and re-runs `go run` on change. No `COPY . .` here — the bind mount supplies
# the source at runtime; this stage only needs modules pre-downloaded.
FROM golang:1.25-alpine AS dev
WORKDIR /src
RUN apk add --no-cache git && \
	go install github.com/mitranim/gow@latest
COPY go.mod go.sum ./
RUN go mod download
EXPOSE 3001
CMD ["gow", "-v", "-i=frontend", "-i=.git", "-i=bin", "run", ".", "-c", "/etc/unhoused/config.yaml"]

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/unhoused .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /out/unhoused /usr/local/bin/unhoused
EXPOSE 3001
ENTRYPOINT ["unhoused"]
CMD ["-c", "/etc/unhoused/config.yaml"]

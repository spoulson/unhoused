# nomaddemo image: simulates a subset of the Nomad HTTP API so unhoused can be pointed at it as a demo
# profile instead of a real cluster. See internal/nomaddemo. Built from the repo root (needs go.mod/go.sum
# and internal/), so its build context is "." — see docker-compose.demo.yaml.

FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/nomaddemo ./cmd/nomaddemo

FROM alpine:3.20
COPY --from=build /out/nomaddemo /usr/local/bin/nomaddemo
EXPOSE 4646
ENTRYPOINT ["nomaddemo"]
CMD ["-listen", ":4646"]

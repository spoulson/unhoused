# Backend image. See specs/architecture.md's Docker Test Environment section.

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

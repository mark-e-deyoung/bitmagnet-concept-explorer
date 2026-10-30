# Bitmagnet Concept Explorer

A read-only Go application for exploring metadata indexed by Bitmagnet. It provides a web UI, HTTP API, and CLI for bounded keyword and concept exploration. It does not download payloads or write tags or metadata to Bitmagnet.

## Security and network behavior

- Network-facing routes require Basic Auth. Set `BCE_AUTH_USER` and `BCE_AUTH_PASSWORD` together; the password must contain at least 16 bytes.
- Basic Auth does not encrypt traffic. Keep the app unexposed or access it through a TLS-terminating reverse proxy. The TrueNAS package exposes the port only for inter-app access and does not offer direct host-port publishing.
- The optional Bitmagnet GraphQL endpoint must use a literal private or loopback IP address or localhost. The app performs read-only queries.
- SearXNG is an explicit opt-in; selected query terms may be forwarded by its configured engines to third parties.
- The app runs as UID/GID 65532 with a read-only root filesystem in the TrueNAS package.

## Build and test

The project uses the Go standard library. With Go 1.27.1, run:

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly go test ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly go vet ./...
```

Build the container from this directory with `Containerfile`. The builder image is pinned by digest.

## TrueNAS app package

The source package is under `ix-dev/community/bitmagnet-concept-explorer/`. It is a candidate for Foundry lifecycle qualification and official catalog review; it is not represented as accepted or listed in the official catalog.

The current release is v0.0.1. See [CHANGELOG.md](CHANGELOG.md).

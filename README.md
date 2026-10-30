# Bitmagnet Concept Explorer

A read-only Go application for exploring metadata indexed by Bitmagnet. It provides a web UI, HTTP API, and CLI for bounded keyword and concept exploration. It does not download payloads or write tags or metadata to Bitmagnet.

## Security and network behavior

- Network-facing routes require Basic Auth. Set `BCE_AUTH_USER` and `BCE_AUTH_PASSWORD` together; the password must contain at least 16 bytes.
- Basic Auth does not encrypt traffic. Keep the app unexposed or access it through a TLS-terminating reverse proxy. The TrueNAS package exposes the port only for inter-app access and does not offer direct host-port publishing.
- The optional Bitmagnet GraphQL endpoint must use a literal private or loopback IP address or localhost. The app performs read-only queries.
- SearXNG is an explicit opt-in; selected query terms may be forwarded by its configured engines to third parties.
- The app runs as UID/GID 65532 with a read-only root filesystem in the TrueNAS package.

## Concept graph data

The graph loader accepts a **provisional display/query-expansion snapshot**, not policy rules or an accepted CPE decision model. It reads JSON with schema `bce-graph-preview-v1` from `BCE_CPE_GRAPH_FILE` and requires `BCE_CPE_GRAPH_SHA256` to match the file's SHA-256 (64 hexadecimal characters, without the `sha256:` prefix). The file is limited to 2 MiB. The API reports `pinned-preview-not-normative`; it never assesses or promotes concepts. The bundled graph fixtures are synthetic tests only.

For a standalone container, place an approved projection in a read-only mount, compute its digest, then set both values together when starting the container:

```sh
sha256sum ./graph.json
# Set BCE_CPE_GRAPH_FILE=/data/graph.json and BCE_CPE_GRAPH_SHA256=<64-hex digest>
# Mount the directory read-only at /data and restart with both settings.
```

When updating the projection, replace the versioned file and update its digest together, then restart the app. A mismatch fails closed. The current TrueNAS package does not expose a projection file mount or these settings, so it cannot yet load an operator-provided graph from the catalog UI. No graph is bundled as policy data. A usable CPE projection must first be published and accepted by the upstream Concept Policy Engine authority.

## Sweep time and admission

Sweep execution has a 55-second end-to-end server deadline, one active sweep per process, and a 60-second HTTP write allowance. The browser and command-line client share a 65-second timeout. Each sweep is bounded to at most 12 Bitmagnet terms (fanout four, eight seconds per request) and three sequential SearXNG requests (eight seconds per request, existing two-second pacing, process-local quota). The configured deadline covers the conservative 52-second sum of those limits plus scheduling and serialization margin. A saturated server rejects a new sweep with HTTP 503 before backend work starts; clients do not retry automatically. Other Bitmagnet API traffic shares the local backend admission slots, so the deadline may yield an explicit partial or failed result under contention rather than guaranteeing full completion.

If the deadline expires after some source results arrive, the API returns a `partial` result with failures and the explicit deadline. If no source result arrives it returns `failed`. A disconnected or canceled caller receives no final response; cancellation propagates to in-flight source requests and is not retried. None of these outcomes establishes corpus absence.

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

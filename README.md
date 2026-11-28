# Bitmagnet Concept Explorer

A read-only Go application for exploring metadata indexed by Bitmagnet. It provides a web UI, HTTP API, and CLI for bounded keyword and concept exploration. It does not download payloads or write tags or metadata to Bitmagnet.

## Choose an interface

- **Human research:** open the workbench, enter a search phrase, and run a bounded
  search. Optional refinements and concept expansion remain explicit choices.
  Search, sweep, source lookup, and feedback preview each have a cancel action;
  cancellation and failures preserve the inputs for a deliberate retry.
- **Accessible interaction:** use native browser controls, the skip link and
  task navigation. Graph actions have equivalent full-name buttons. Validation
  identifies the field to correct, and request status is available without
  moving focus. System appearance, reduced motion, forced colors, text scaling
  and narrow reflow are part of the [accessibility and platform contract](docs/ux/accessibility-platform-contract.md).
- **CLI and automation:** begin with the [route and audience contract](docs/route-audience-contract.md)
  and `bce -help`. Preserve structured partial/failed outcomes, inspect CLI exit
  codes, and use a new explicit request for recovery. Capability discovery
  describes configuration, not dependency health or semantic acceptance.

External consent applies to the selected scope: changing search inputs or a
source selection clears the corresponding external opt-in. Feedback preview
runs a fresh local search using the current phrase, selected concept, manual
expansions, filters, sort and grouping, with external search disabled. The
structured preview remains available in a disclosure for inspection.

Automated keyboard, browser semantics, reflow and contrast checks are evidence
for the exact revision tested. They do not establish native screen-reader
acceptance; the platform contract records the separate VoiceOver, Narrator,
NVDA, TalkBack and Orca task checks needed for those support claims.

## Security and network behavior

- Network-facing routes require Basic Auth. Set `BCE_AUTH_USER` and `BCE_AUTH_PASSWORD` together; the password must contain at least 16 bytes.
- Basic Auth does not encrypt traffic. Keep the app unexposed or access it through a TLS-terminating reverse proxy. The candidate TrueNAS app defaults to container-only exposure but offers explicit host-port publishing, defaulting to loopback 127.0.0.1. Remote access requires deliberate operator configuration and TLS termination; this source option is not proof of qualified ingress.
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

When updating the projection, replace the versioned file and update its digest together, then restart the app. A mismatch fails closed. **The candidate TrueNAS source wizard already supports optional `research_data.graph_file` and `research_data.graph_sha256` inputs** (merged public PR #13). Its iX template mounts an existing operator-provided host file read-only at `/opt/bce/research/graph-preview.json` and passes the paired digest. It separately supports an optional pinned research-evidence file. Each file/digest pair is required together, and the source renderer refuses unpaired settings. The file must exist and be readable by container UID 65532; the wizard does not create it. Synthetic source-render testing is complete, but Foundry's native target lifecycle, actual TrueNAS permissions, and runtime graph read have not yet been accepted. No graph is bundled as policy data. A version-pinned research graph remains `pinned-preview-not-normative`; a normative CPE projection needs separate CPE authority and acceptance.

## Sweep time and admission

Sweep execution has a shared 55-second engine deadline, one active sweep per process, and a 75-second HTTP write allowance. Both direct execution and feedback-preview sweeps share these bounds. The write allowance includes the HTTP server's 15-second request-body read window plus handler execution and response serialization. The browser and command-line client share an 80-second timeout. Each sweep is bounded to at most 12 Bitmagnet terms (fanout four, eight seconds per request) and three sequential SearXNG requests (eight seconds per request, existing two-second pacing, process-local quota). The configured deadline covers the conservative 52-second sum of those limits plus scheduling and serialization margin. A saturated server rejects a new sweep with HTTP 503 and `code: sweep_busy`, `state: not_started` before backend work starts; clients do not retry automatically. Other Bitmagnet API traffic shares the local backend admission slots, so the deadline may yield an explicit partial or failed result under contention rather than guaranteeing full completion.

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

The current published release is v0.0.2. `VERSION` identifies the untagged maintenance candidate under review; it is not a published release. See [CHANGELOG.md](CHANGELOG.md) and [maintenance qualification](docs/maintenance-release.md).

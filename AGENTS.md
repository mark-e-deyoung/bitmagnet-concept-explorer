# Agent and automation instructions — Bitmagnet Concept Explorer

This public repository owns the standalone **read-only Go application source**. The product/DLE authority is [mark-e-deyoung/bitmagnet-tools#18](https://github.com/mark-e-deyoung/bitmagnet-tools/issues/18) (private to authorized collaborators). Canonical portfolio working method: [Way of Working v0.0.4](https://github.com/SemperSupra/engineering-governance-private/blob/main/WAY_OF_WORKING.md). Preserve repository-local hard boundaries; do not copy private source, host data, secrets, metadata corpus or query history into this public tree.

## Cold start for any actor

1. Read this file, public `README.md`, and the active owning GitHub issue/PR. Reconcile current `main`, open work branches, release/source version, CI receipts, and held gates before writing.
2. Use `DISCOVER → PLAN → ACT → VERIFY` inside an authorized bounded workset. Create an isolated branch/PR, retain first failing tests and exact revision IDs, and report results to the owning DLE; do not mutate `main` or another actor's worktree.
3. Use public/free GitHub-hosted Actions **only for public-safe source or synthetic fixtures**. No private Actions minutes, private repository export, local NAS/corpus exposure, unsolicited real-provider traffic, production deployment, releases, merges, or official catalog publication.

## Supported machine interfaces

- `GET /healthz` is unauthenticated liveness, **not** proof of a live Bitmagnet/CPE dependency.
- Authenticated `GET /api/v1/capabilities` reports supported operations and whether a Bitmagnet endpoint is *configured*; the `execute` flag does not prove network reachability. Check it before sending a query.
- Authenticated `GET /api/v1/concepts[?q=...]` and `GET /api/v1/concepts/{id}` read an **optional pinned, provisional graph**. Missing graph is an explicit unavailable state; it is not an empty ontology.
- `POST /api/v1/query/plan` and `POST /api/v1/query/execute` are bounded read-only search controls; optional `include_web=true` forwards query terms to a configured SearXNG search provider and requires explicit user authorization.
- The `cmd/bce` CLI is a thin client over the same HTTP API. Example after starting a loopback development server: `go run ./cmd/bce -operation capabilities`, `go run ./cmd/bce -operation concepts -query animation`, `go run ./cmd/bce -operation plan -query animation`. Use `go run ./cmd/bce -help` for the source-native flag inventory, rather than treating this file as a second schema.
- For non-loopback service access, the client rejects plain HTTP; provide a qualified TLS origin and `BCE_AUTH_USER` plus `BCE_AUTH_PASSWORD`. Never embed credentials in arguments, URLs or logs.
- **No MCP server or privileged agent-control API is claimed.** Use the existing versioned HTTP/CLI capabilities; any future MCP adapter requires separate interface and security qualification.

## Evidence/authority invariants

- Bitmagnet is a source of **indexed metadata observations**, not content or legal/policy truth. Search results and title/filename lexical matches do not establish concept presence. Preserve origin, language, failures, scope, source tags, conflicting metadata and partial results. No match is non-diagnostic.
- `bce-graph-preview-v1` and `BCE_CPE_GRAPH_FILE` plus `BCE_CPE_GRAPH_SHA256` only provide integrity of operator-supplied provisional bytes. A correct checksum is **not** CPE source authenticity or approval. Only the owner of the accepted Concept Authority can promote semantics.
- Never download BitTorrent payloads, tag/mutate Bitmagnet, reconfigure the crawler/DHT or update CPE through this service. Discovery proposal previews are research; submission/adjudication/promotion are held upstream.
- The TrueNAS source package is a **Foundry candidate**, not a native Foundry deployment authority. Foundry [#274](https://github.com/SemperSupra/truenas-app-foundry-private/issues/274) and [#347](https://github.com/SemperSupra/truenas-app-foundry-private/issues/347) own admission/materialization/lifecycle. Docker Compose smoke does not prove native TrueNAS stop/start/NOOP/rollback.
- Three audiences (humans, automation, agents) share one query engine and truth model. Browser interactions, CLI/API equivalence and agent recovery all require separate evidence; a declared capability is not a qualified end-to-end journey.

## Fast source-safe verification

```sh
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly go test ./...
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly go vet ./...
node tests/ui/verify-coverage-manifest.mjs
```

Public UI qualification is in `.github/workflows/public-ui-ux-full.yml`; Foundry/graph source tests run in the respective public workflows. Check exact-head Actions outcomes and logs. Do not infer acceptance from a queued or failed run.

## Stop conditions and handoff

Stop at missing target permissions, invalid manifest/digest, semantic authority ambiguity, provider/privacy violation, unsafe ingress, undefined UI/API contract, or consequential production lifecycle mutation. Preserve a minimized failure and a bounded next action in the owning GitHub issue or PR. Do not ask the human to relay actor state that exists in GitHub.

# Route and audience contract

`GET /healthz` is the only unauthenticated route when Basic Auth is configured. The browser, CLI, and agent-facing API share the same authenticated HTTP handler; each registered GET/POST path is covered by `TestAuthenticationCoversRegisteredRouteMatrix`. With a syntactically configured local target, each malformed authenticated POST in the matrix returns HTTP 400 before backend work. Unauthenticated requests receive HTTP 401, a Basic Auth challenge, and `Cache-Control: no-store`.

The browser has direct controls for capabilities, concepts, query plan/execute, observed-result graph, inference proposals, metadata facets, sweep plan/execute, sweep limits, media lookup, and local query feedback preview. Its feedback form uses the current query, selected concepts, manual expansions, filters, sort and grouping, with external search disabled. The API and CLI additionally accept a sweep in the feedback envelope. The browser has no dedicated sweep-feedback control, CPE projection editor, standalone discovery-preview control, or metadata-comparison binding.

## Operation discovery and input parity

The table records the exact HTTP route and CLI operation. `-input FILE` reads the complete request from a file; `-input -` reads stdin and is the default for JSON-input operations. Query, filter and consent flags do not amend a JSON-input request. Put those fields in its JSON envelope. Run `bce -help` for the complete flag inventory.

| Purpose | HTTP route | CLI `-operation` | CLI input |
| --- | --- | --- | --- |
| Configuration and affordances | `GET /api/v1/capabilities` | `capabilities` | None |
| Process-local sweep budgets | `GET /api/v1/sweeps/limits` | `limits` | None |
| Browse optional provisional concepts | `GET /api/v1/concepts` | `concepts` | Optional `-query` |
| Inspect one provisional concept | `GET /api/v1/concepts/{id}` | `concept` | `-concept-id` |
| Preview retrieval terms | `POST /api/v1/query/plan` | `plan` | Query, concept, expansion, filter and view flags |
| Execute bounded metadata retrieval | `POST /api/v1/query/execute` | `execute` | Query, concept, expansion, filter and view flags; explicit `-web` opt-in |
| Native metadata aggregates | `POST /api/v1/query/facets` | `facets` | `-query`, `-facet-groups`, `-languages`, `-media` |
| Observed-result lexical graph | `POST /api/v1/query/observed-graph` | `observed-graph` | JSON input |
| Historical adapter preflight | `POST /api/v1/cpe/adapter-plan/preflight` | `cpe-preflight` | JSON input |
| Provisional inference | `POST /api/v1/inference/propose` | `infer` | JSON input |
| Plan bounded multilingual retrieval | `POST /api/v1/sweeps/plan` | `sweep-plan` | JSON input |
| Execute one bounded multilingual sweep | `POST /api/v1/sweeps/execute` | `sweep` | JSON input |
| Preview supplied sweep evidence | `POST /api/v1/sweeps/discovery/preview` | `discovery-preview` | JSON input |
| Gather and preview keyed feedback evidence | `POST /api/v1/discovery/feedback/preview` | `feedback-preview` | JSON input containing exactly one `query` or `sweep` |
| Look up source-attributed media candidates | `POST /api/v1/media/lookup` | `media-lookup` | JSON input |
| Compare caller-supplied metadata observations | `POST /api/v1/metadata/compare` | `metadata-compare` | JSON input |
| Historical FTS lowering preview | `POST /api/v1/cpe/adapter-plan/lower-preview` | API only | JSON body |
| Legacy discovery-proposal preview | `POST /api/v1/discovery-proposals/preview` | API only | JSON body |

JSON request budgets come from `internal/contracts` and are shared by API handlers and CLI input checks: query plan/execute 16 KiB, facets 8 KiB, observed graph 1 MiB, historical preflight/lower-preview 32 KiB, inference and legacy discovery 8 KiB, sweep plan/execute/discovery preview, feedback preview, media lookup and metadata comparison 16 KiB. CLI-supported JSON operations use their route's exact cap and reject cap+1 locally. Native facets reject local-only numeric and adult-metadata filters, because the upstream aggregate cannot honor them.

Capabilities list product-level affordances; route inventory and specialized exceptions are recorded here so the three audiences do not imply equivalent UI for every research endpoint.

## Automation and agent outcome handling

Use configuration discovery before an execution request. `GET /healthz` is liveness only; `capabilities.execute` means a syntactically configured endpoint, not a successful dependency probe. Missing provisional graph data is unavailable, not an empty ontology. Configuration flags do not authorize external requests.

The CLI sends one request, uses the shared 80-second client timeout, and does not follow redirects or retry. It prints a received response body to stdout followed by a newline. Local argument, input, transport and response-budget errors are plain text on stderr. Responses larger than 4 MiB are rejected as a whole; the CLI does not emit a truncated JSON prefix. HTTP failures preserve their response body on stdout, including a non-JSON authentication or proxy response. Check the exit code before parsing and use the API directly when HTTP headers or exact status codes are needed.

| CLI exit | Meaning | Caller action |
| --- | --- | --- |
| `0` | A 2xx HTTP response was received within the output budget, or help was displayed | Interpret the operation's `state`, failures, source outcomes and warnings. A `partial` or `failed` research result can arrive with HTTP 200. |
| `2` | Local arguments, origin, paired authentication or input could not be used | Correct the local request. No service request was sent. |
| `3` | Transport failed, response reading failed, or response exceeded the output budget | The server may have started work. Preserve an unknown outcome; do not automatically replay the request. |
| `4` | Non-2xx HTTP response, including an unfollowed redirect | Inspect the returned body. Correct authentication, route or input, or handle an explicit no-work-started capacity outcome. |

HTTP 400 indicates malformed, unknown-field, trailing or oversized request JSON. HTTP 422 indicates a rejected semantic request. Feedback previews return HTTP 503 with `code: "feedback_not_configured"` and `state: "not_started"` when the operator's local evidence HMAC key is unavailable. This configuration failure is distinct from temporary sweep contention.

## Shared sweep admission, deadline and recovery

`runSweep` owns admission and the 55-second execution deadline. Both `POST /api/v1/sweeps/execute` and a sweep envelope sent to `POST /api/v1/discovery/feedback/preview` share the same one-active-sweep process slot. The feedback route retains its 20-second limit only for ordinary query feedback. A caller's earlier deadline or cancellation still takes precedence. Planning remains available while execution is busy, and invalid input is rejected before provider work.

When the sweep slot is occupied, both execution routes return HTTP 503 with the same machine-readable response:

```json
{
  "error": "sweep capacity is busy; no work started and no automatic retry was made",
  "code": "sweep_busy",
  "state": "not_started"
}
```

Neither route queues or retries this request. A later attempt is a new explicit action. There is no guaranteed retry time. The bounds cover one process, not multiple replicas or an upstream provider's total traffic. The separate shared Bitmagnet admission slots and SearXNG pacing, quota, refusal and consent rules still apply.

Sweep results use `complete`, `partial` or `failed`; retain the result's failures and scope. Feedback from a partial or failed source returns `source_partial_or_failed_no_proposal` without a proposal. No-hit feedback returns `no_matching_observation_not_concept_absence`. `local_review_export_only` is a local research draft, not transmission, persistence, CPE acceptance or semantic promotion. Cancellation releases the sweep slot and cancels in-flight provider work; a disconnected caller may receive no final response.

Focused synthetic regression cases are in `sweep_shared_contract_test.go` and `cmd/bce/main_test.go`: contention across both routes and direct engine calls, cancellation and subsequent recovery, feedback continuing past the former 20-second wrapper, preservation of a shorter caller deadline, CLI partial-result handling, and non-followed/non-successful redirects. Actual pass/fail evidence belongs to the exact-revision CI receipt; the presence of a test or a capability declaration is not end-to-end acceptance.

## Local metadata comparison preview

`POST /api/v1/metadata/compare` and `bce -operation metadata-compare -input FILE` compare one caller-supplied reference record against 1–16 already observed source records. Shared 16 KiB budget; unknown fields, malformed input, oversized records and unsupported source/status/identity namespaces are rejected. No provider calls, result URL, content hash, content delivery, persistence or CPE writes. Input source labels and catalog IDs are claims, not authenticated provider receipts. Same ID is explicitly an unverified candidate; conflicts on comparable IDs, kind, year or edition block positive identity grading. Title similarity alone is never a resolved match. Missing fields and failed or partial observations are not evidence of absence. Response returns ordinal and field comparison status rather than echoing titles/IDs. All unresolved provenance shares one unqualified ancestry; no independent witnesses claimed. Browser binding to existing observations is a subsequent task.

# Sweep request deadline contract

**Authority:** `mark-e-deyoung/bitmagnet-tools#18`, independent-review baton `6533850296`.
**Source base:** public PR #1 exact head `f4c47771bb37380c4ff57b1bb94f32e6263e4efb`.
**Scope:** synthetic local HTTP fixtures only. No live Bitmagnet, CPE, Foundry, catalog, or deployment work.

## DISCOVER — reproduced failure

A TCP server with the production `http.Server` timeouts (`ReadHeaderTimeout=5s`, `ReadTimeout=15s`, `WriteTimeout=15s`, `IdleTimeout=30s`) received a valid sweep request. The Bitmagnet fixture returned immediately; three sequential synthetic SearXNG requests each delayed 5.4 seconds. The handler completed after the response write deadline. The client observed `EOF` after 16.204234924 seconds even though all three provider requests ran. The API therefore lost a completed result. The former CLI's 25-second timeout also sat below the 35-second handler deadline.

The local task and provider budgets are:

- Up to 12 Bitmagnet terms, four active requests, 8 seconds per request: 3 waves / 24 seconds when one sweep owns the shared slots.
- Up to 3 sequential SearXNG requests, 8 seconds each and 2 seconds minimum between request starts: at most 28 seconds.
- Conservative isolated-work total: 52 seconds. Direct Bitmagnet API traffic shares the provider slots and can delay a sweep; the server deadline returns partial/failed output in that case rather than promising full completion.

## PLAN — bounded policy

Use a 55-second sweep handler deadline (3 seconds beyond the 52-second isolated budget), 75-second HTTP write timeout, and shared 80-second browser/CLI caller timeout. The server's write deadline starts before request body reading, so its 75 seconds include up to 15 seconds of request read, the 55-second handler budget, and response margin. Admit one sweep at a time per process; reject saturation immediately with HTTP 503 before backend work starts. Preserve the existing four-slot Bitmagnet bound, SearXNG two-slot/pacing/quota/refusal controls, one-shot behavior, and no automatic retries. This keeps the admitted workload bounded while preserving the existing plan query contract.

Full results use the existing `complete` state. Source failures with usable data return `partial`; no usable source result returns `failed`. Caller disconnect/cancellation cancels in-flight requests; a disconnected caller cannot receive a final response. None of these results establishes corpus absence.

## ACT — changes

- Replaced the 35-second sweep handler deadline with 55 seconds and raised the production HTTP write timeout from 15 to 75 seconds.
- Added fail-fast process-local single-sweep admission (HTTP 503, no backend work, no automatic retry).
- Added the `deadline_seconds`, active sweep cap, and client timeout to the limits/result contract.
- Put the 80-second caller budget in `internal/deadline` and use it in the CLI. The browser sweep flow uses the same 80-second abort bound, keeps its buttons disabled while active, and reports cancellation/failure without retry or absence inference.
- Documented the deadline, contention qualification, and result states in the README.

## VERIFY — executable fixtures

The committed regression suite uses a real TCP listener configured with production timeout values plus local delayed HTTP backends. It covers a fast complete API result, a complete API result after 15 seconds, the legacy 15-second EOF failure, CLI complete output after 25 seconds, CLI partial output after exhausted local quota with no extra provider call, fail-fast admission with zero backend calls, and cancellation propagation with no retry. Independent review also found the old 60-second write timeout could expire after a 6-second request-body delay plus a 55-second handler; a real TCP regression reproduced loss of that completed result at 61.005 seconds. API/CLI execute paths use only synthetic metadata.

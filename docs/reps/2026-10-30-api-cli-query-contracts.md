# API/CLI/query contract follow-up

**Authority:** product `mark-e-deyoung/bitmagnet-tools#18`, current batch C–E in comment `6534095309`; earlier source-derived findings in `6533982804`.
**Scope:** local synthetic requests and source-only changes. No Bitmagnet, CPE service, external search, TrueNAS, Foundry, catalog, or deployment access.

## DISCOVER

- The CLI allowed insecure HTTP only for three exact loopback host:port strings, so valid alternate local ports were rejected. The CLI's single 1 MiB JSON input cap exceeded server caps for preflight, inference, and sweep routes.
- The HTTP auth wrapper was intended to protect every route except `GET /healthz`, but tests exercised only a small subset.
- Pinned projection lexicalizations and user-authored direct FTS text reached the same native query string. The historical CPE lower-preview already rejected parser-ambiguous lexical forms, while direct query syntax was intentionally passed through.

## PLAN / ACT

- Validate loopback by parsed hostname/IP, independent of port; retain HTTPS for non-loopback and reject URL credentials, paths, query, or fragment.
- Centralize per-route JSON budgets in `internal/contracts`; both handlers and CLI use the same values. CLI rejects cap+1 locally.
- Exercise the complete registered GET/POST route inventory through the authentication wrapper, including 401 challenge/cache headers and malformed authenticated POSTs on local synthetic configuration.
- Preserve user-authored direct Bitmagnet FTS unchanged. Treat pinned projection terms as lexical candidates: allow only letters, marks, numbers and single ASCII spaces. Reject ambiguous generated terms during planning rather than guessing a quote/escape transformation. Manual unverified expansions remain user-authored. This retrieval syntax rule does not change concept or policy semantics and remains subject to independent owner review against the pinned tokenizer.

## VERIFY

Focused tests: `TestValidateServiceOriginAllowsLoopbackPortsAndRejectsUnsafeHTTP`, `TestCLIJSONInputLimitsMatchRouteContracts`, `TestRegisteredPostRoutesEnforceSharedBodyBudgets`, `TestAuthenticationCoversRegisteredRouteMatrix`, `TestUserFTSStaysNativeWhilePinnedLexicalizationsRejectOperators`, and `TestSweepPlanKeepsUserFTSAndRejectsUnsafePinnedLexicalization`.

The initial focused compile exposed two integration errors (`serviceURL` unused after origin refactor; missing `fmt` import in sweep validation). Both were corrected, gofmt was applied, and the focused suite passed in the pinned offline Go 1.27.1 builder. Exact combined-head qualification and its first script-invocation correction are recorded in the owning DLE.

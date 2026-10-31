# Response integrity follow-up

**Authority:** product DLE `mark-e-deyoung/bitmagnet-tools#18`, follow-on findings in comment `6533982804` and execution baton `6534342145`.
**Scope:** synthetic local HTTP fixtures and UI-state helper checks. No provider, Bitmagnet, TrueNAS, CPE, Foundry, catalog, or deployment access.

## DISCOVER

- SearXNG used a 2 MiB `LimitReader` and one JSON decode. The source-equivalent reproducer accepted a valid JSON prefix followed by a second object, and could accept a valid prefix while the response exceeded the byte cap.
- Native facet totals and bucket counts/estimate flags decoded into zero-value scalars; omitted or null fields looked like exact zero counts.
- The CLI read at most 4 MiB and printed that prefix with exit success for HTTP 200, even when it was truncated and invalid JSON.
- The browser cleared current result cards only after a successful response. On a later timeout, 422 or 401, old rows and external observations could remain visible under the new query.

## PLAN / ACT

- Reuse `decodeBoundedJSON` for SearXNG so the full body is checked for both a byte budget and trailing JSON.
- Decode facet totals and each bucket's count/estimate flag as pointers; reject missing/null/negative values and retain valid explicit zero.
- Read CLI responses with a max+1 probe; return an error and write no stdout prefix when over budget.
- Clear query-specific output on request start and failure; increment query, graph, facet, sweep, limits, and shared sweep-display generations so older asynchronous responses cannot restore or overwrite current evidence. Keep failures explicitly non-diagnostic.

## VERIFY

- Go regressions: `TestSearXNGRejectsTrailingAndOversizedJSON`, `TestNativeFacetRequiresPresentNonnegativeTotalsButAcceptsExplicitZero`, `TestNativeFacetBucketsRequirePresentNonnegativeCountsAndEstimateFlags`, `TestCLIRejectsOversizedHTTP200ResponseWithoutReturningPrefix`.
- UI-state check: `node web/query_state.test.mjs` under Node v22.14.0; a dependency-free DOM stub executes production facet, sweep, and limits request functions with deferred responses, asserts each request starts, clears query evidence, then proves late successful responses cannot restore it. It also reverses sweep/limits completion order to verify the shared display follows the newest request, and checks new request, timeout, HTTP 422 and HTTP 401 clearing. This is not a full browser/visual test.
- Focused Go regressions passed in the pinned offline Go 1.27.1 container. The exact combined qualification result is recorded in issue #18.

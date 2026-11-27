# R16 — Multilingual Bitmagnet/SearXNG research feedback, privacy-keyed and nonpromotable

**Authority:** `mark-e-deyoung/bitmagnet-tools#18` product and `mark-e-deyoung/concept-policy-engine-private#202` discovery CPE. This is a local, review-only improvement to `/api/v1/discovery/feedback/preview`, not a CPE intake protocol, accepted ontology, source-rights transfer, publisher identity proof, or a second global crawler.

The same authenticated HTTP endpoint and `bce -operation feedback-preview -input request.json` now support exactly ONE of `query` (existing one-query local Bitmagnet mode) OR `sweep` (existing bounded multilingual sweep engine). The sweep is still limited by the existing up-to-12 local and up-to-3 SearXNG queries, source-wide 3/hour JSON quota, per-request external consent and local operator admission. Without `BCE_DISCOVERY_HMAC_KEY` at least 32 bytes, the service refuses to generate keyed feedback.

## Example — local multilingual, no external search

```json
{
  "sweep": {
    "query": "a researcher-authored local subject",
    "languages": ["en", "de", "th"],
    "concept_ids": [],
    "max_bitmagnet_queries": 3,
    "include_searxng": false
  },
  "proposal_kind": "lexicalization_candidate",
  "candidate_label": "a researcher-selected candidate term",
  "language": "de"
}
```

The proposal is only generated from **actual** returned data (not caller-generated evidence hashes), and only when the selected sources complete successfully. When there are zero observations, a provider refusal/rate limit, a missing graph or malformed scope, or partial execution, no CPE proposal is returned. A caller-supplied research projection digest must match the selected pinned preview graph; one cannot invent an accepted CPE projection by supplying a syntactically plausible digest.

## External multilingual search

To include the configured private/local SearXNG engine, a request must set `include_searxng=true` and `external_consent=true`, and the administrator must separately set `BCE_SEARXNG_SWEEP_ALLOWED=true` and a private/loopback `BCE_SEARXNG_URL`. Terms may be forwarded by SearXNG's underlying engines: review the query prior to granting consent. This endpoint does not fetch TMDb, IMDb, Jellyfin or other sources. Provider-native media lookup remains a separate operation with separate explicit TMDb consent.

Unlike legacy `sweep-research-v0` display evidence, this new review-only feedback **does not reuse unsalted SHA256 of URLs, infohashes, titles or search terms**. It derives domain-separated HMAC-SHA256 references from actual bounded observations with a local operator key, and returns at most 16 keyed evidence records. All unresolved Bitmagnet and SearXNG observations are treated as **one linked unqualified ancestry**, even if many hits, engines or language variants appear; no automated semantic corroboration. Labels are user-supplied hypotheses and may themselves be sensitive: keep the preview local.

## Qualification and outstanding CPE boundary

Source-only fictional `httptest` cases cover English/German/Thai local sweep, HMAC key separation and stable replay, private SearXNG explicit consent/absence, cross-source one ancestry, 429/partial fail-closed, false projection digest, and query+sweep mixed requests. No real external web search in public CI.

The output schema remains `cpe-discovery-proposal-research-v1` with `not_accepted_not_transmitted`. **It is NOT wire-compatible with independent research R37 `cpe-discovery-proposal-research-v0`** and must not be relabeled, submitted, auto-merged, used as a legal rights decision, or promoted without CPE #202's signed, versioned intake/grounding. CPE controls all deduplication, mapping, hard negatives, multilingual corroboration and acceptance, and protected local metadata must never be uploaded to free public runners.

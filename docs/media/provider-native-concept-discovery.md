# Provider-native concept discovery — source-safe adapter contract

**Purpose:** Search related media by a *provider-native* genre/keyword identifier, not by pretending that CPE concept IDs already map to TMDb/Jellyfin. Research product owner: `bitmagnet-tools#18`. Semantics/mappings authority: CPE `#201/#202/#348`. This is an extension of `docs/media/identity-related-rdte.md`.

## Explicit API input

`POST /api/v1/media/lookup` also accepts up to three `tmdb_genre_ids`, up to three `tmdb_keyword_ids`, and up to three `jellyfin_genres`. **Either** source-native discovery filters **or** title, IMDb/TMDb/Jellyfin item identity may be requested, but never mixed within one call; include-related is unavailable for such open-ended provider search. Each source must be explicitly in `providers`.

Example (illustrative, fictional intent, TMDb source-native IDs not asserted as CPE):

```json
{
  "kind": "movie",
  "providers": ["tmdb", "jellyfin"],
  "tmdb_genre_ids": [16, 878],
  "jellyfin_genres": ["Animation", "Science Fiction"],
  "language": "de-DE",
  "allow_external_lookup": false
}
```

TMDb will be `requires_explicit_external_consent` until the caller sets `allow_external_lookup=true` and the server has `BCE_TMDB_READ_TOKEN`. A permitted lookup uses the official bounded `GET /discover/movie|tv?with_genres=&with_keywords=&page=1` source-native filter path (maximum one page); local Jellyfin uses private GET `/Items?Genres=&IncludeItemTypes=Movie,Series&Limit=10`. No downstream corpus writes, no acquisition, no Jellyfin mutation, no CPE update or AI provider forwarding.

Both return **`provider_native_concept_results_unmapped`** with individually attributed candidates and `MatchBasis=provider_native_concept_candidate_unmapped` (TMDb) or `source_native_genre_candidate_unmapped` (Jellyfin). This is no more than *provider retrieval evidence*. It does not assert CPE relationships, independent corroboration, local library completeness, topical truth, or correct title identity.

## Next acceptance seam

An accepted CPE source-specific lexical/mapping projection can, after separate authority qualification, lower a CPE concept ID into supported **provider-specific** native genre/keyword IDs or Jellyfin library genre labels, with provenance, locale, source rights and version. The adapter must then bind that projection digest and source-specific mapping to every outgoing request, include hard-negative and ambiguity tests, and measure precision/recall and new concept proposal quality against an independent source oracle. Until then, humans and agents must use these provider IDs **explicitly**, not infer that a CPE concept ID maps to a metadata category by title similarity.

No real TMDb credentials/Jellyfin inventory were used for public CI; fixture source terms are fictional and do not become CPE authority. Follow media-discovery private #7 rights and #25 Jellyfin identity/behavior normalization. UI and CLI discovery are distinct acceptance gates; the HTTP API alone is not a complete human journey.

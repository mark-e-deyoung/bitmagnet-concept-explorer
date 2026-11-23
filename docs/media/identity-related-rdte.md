# Media identity and related-items lookup — source-only RDTE candidate

**Owner:** Bitmagnet Concept Explorer (read-only, explicit user query). **Product DLE:** private `mark-e-deyoung/bitmagnet-tools#18`. This is an additive candidate, not a replacement for `media-tools#40`'s existing GET-only Jellyfin inventory producer, `media-discovery-private#25` canonical behavioral snapshot interpretation, `media-discovery-private#7` TMDb identity authority, or CPE `#201/#202` semantics.

## Interfaces and current scope

`POST /api/v1/media/lookup` accepts bounded JSON:

```json
{
  "kind": "movie",
  "title": "A user-selected work title",
  "year": 2025,
  "language": "de-DE",
  "imdb_id": "tt1234567",
  "providers": ["tmdb", "jellyfin"],
  "include_related": false,
  "allow_external_lookup": false
}
```

At most three explicit providers. `kind` is `movie` or `series`. `title`, `tmdb_id`, `imdb_id`, or `jellyfin_item_id` must be supplied; use the most specific *confirmed* identity available. No caller-supplied URLs, paths, tokens, filters or downstream mutation. Text searches return *candidate identities*, never confirmed ownership or a claim that a title exists/does not exist anywhere.

- **Jellyfin**: local/private IP only, server-side `BCE_JELLYFIN_BASE_URL` and `BCE_JELLYFIN_TOKEN`, read-only `GET /Items` or selected `GET /Items/{id}`; `include_related` requires an explicitly selected Jellyfin ID and calls `GET /Items/{id}/Similar`. Use Jellyfin 10.11/12 compatible `Authorization: MediaBrowser Token="..."`, never legacy `X-Emby-Token`. Duplicate names and conflicting TMDb/IMDb provider IDs remain unconfirmed, not falsely owned. The returned first page is bounded, not an inventory proof.
- **TMDb**: server-only `BCE_TMDB_READ_TOKEN`; official HTTPS `https://api.themoviedb.org/3`, or deliberate loopback fixture base `BCE_TMDB_API_BASE` for RDTE. Requires `allow_external_lookup=true` **per invocation**, otherwise no external request. Query `/search/movie|tv`, `/find/{imdb_id}`, or a confirmed `/movie|tv/{tmdb_id}` endpoint. Related title recommendations require the explicit TMDb ID and never imply semantic equivalence. Limit 10 candidate and eight related results, one page, no automatic retries. Token never belongs in a URL, browser code, query history or logs.
- **IMDb**: `providers:["imdb"]` currently returns `licensed_source_not_configured`. We do *not* scrape IMDb, query undocumented endpoints or misuse non-commercial datasets. A known IMDb title ID may be used in an authorized TMDb `/find` crosswalk. Direct IMDb API or local official dataset access remains a separate license/rights and retention review.
- **Other sources**: TVmaze has a documented read-only series search/IMDb-ID lookup and source attribution/ShareAlike requirements; it is a candidate source for a later qualified adapter, not secretly invoked in this MVP.

### Server and security boundaries

The application routes sit behind the existing server Basic Auth and non-loopback TLS ingress requirement. Jellyfin is private/loopback only; remote TMDb is exact allowlist HTTPS. Requests only send a selected title/ID after a per-query affirmative egress flag, not bulk Bitmagnet results or local corpus. Max 1 MiB JSON responses, seven-second per request timeout, redirects and proxy inheritance disabled, bounded IDs and page sizes. Provider failures, 401/403, 404, 429, stale or ambiguous responses preserve explicit states, not absence. Do not add account sessions, write endpoints, automatic acquisitions, download/stream/file endpoints, or source-controlled secrets.

**TMDb attribution:** “This product uses the TMDB API but is not endorsed or certified by TMDB.” Live image/metadata display and the full attribution/rights review remain gated by provider terms. This candidate normalizes only identity fields and bounded provider genre IDs, not synopses, posters, biographies, ratings, behavioral events or ML training content. Existing private media-discovery policy forbids sending TMDb content to LLM/embedding/model paths without further permission.

## Follow-on acceptance and CPE bridge

A human picks a Bitmagnet observed record and confirms work identity using structured provider IDs, type, year and disambiguation. Jellyfin can answer which related titles are **in this user-authorized local library**; TMDb can suggest titles by external catalog relationship. The two source views remain separate and partial, with source/timestamp/identity evidence, never mixed with canonical work or personal reaction authority. **Related** means upstream recommendation, not accepted CPE concept relationship.

A future CPE projection crosswalk maps provider-native genre/keyword/tag evidence into *candidate* `ConceptDiscoveryProposal` entries, with language, source IDs/rights and hard negatives, replay and human adjudication. It must not simply treat TMDb genre or Jellyfin search membership as accepted concept. The owner of the CPE #202 intake controls all promotion.

### Independent RDTE and deployment gates

Run source-only public Go/HTTP fixtures for exact-ID vs title collision, wrong ID, absent, HTTP/JSON errors, SSRF, credential leakage, no-consent egress denial, related-source result ambiguity and English/German/Thai original text. Do not connect GitHub-hosted runners to the actual user's Jellyfin, TrueNAS, TMDb account or private Bitmagnet corpus. Confirm updated upstream provider terms, key permissions, attribution and actual source behavior before any personal live request.

A green source PR is not a native TrueNAS Foundry app qualification, production export, accepted CPE authority or evidence of Jellyfin ownership. Parent product #18 controls release and target decisions.

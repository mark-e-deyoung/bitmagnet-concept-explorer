# Interactive evidence surfaces — UI/UX/DX R2

**Research proposal, 2026-10-31.** Exact upstream design base: public draft PR #32, `64066175fdd124743bc2a3d3a2966e02b6c6be4c`. Owning product DLE: `mark-e-deyoung/bitmagnet-tools#18`. This is a separate stacked public PR; the private main already received R1 at `94f9c47`, and must not be overwritten.

## Design intent

Bridge and VanillaCord show accessible, task-first **static public surfaces** and a project-native visual identity. The Explorer also contains high-density interactive systems that they did not exercise: selectable SVG concept graphs, bounded observed-evidence graphs, source-derived facet summaries, query deadlines, raw evidence/provenance, model-hypothesis opt-in, batch/multilingual sweep limits and partial/unavailable source states. A branded shell without coherent interaction states is insufficient.

The product-native metaphor remains **multiple bounded evidence tracks converging on an inspectable junction**. SupraCraft organization identity, Bridge illustrations, VanillaCord art, upstream Bitmagnet marks, and unqualified CPE facts must not be copied or implied.

### Shared interaction grammar

| Component | Interactive state and information | Human | Automation/agent |
| --- | --- | --- | --- |
| Provisional concept graph | selected/unselected nodes, keyboard navigation, relationships, theme/forced-colors palette | SVG + equivalent native selectors; selected state announced, no automatic query | Existing `/api/v1/concepts` stable source |
| Observed evidence graph | returned title-lexical candidates and actual source edges | same graph tokens but distinct observed color; exact selected node and source detail, full labels via accessible names | Existing `/api/v1/query/observed-graph` supplied-result-only source |
| Metadata result | observed type, language, peers, flags, matched terms and full infohash | structured `dl` and accessible copy, never a content recommendation | Existing response JSON/CLI; no HTML scraping |
| Facet groups | group labels, bounded 60 buckets, estimates, counts, failure | native `details` / `summary` and readable bucket values; no false clickable policy controls | Existing `/api/v1/query/facets` provenance, estimate flags unmodified |
| Query and sweeps | idle, busy, completed HTTP, blocked, failure | text status plus deterministic border token, cannot equate an empty/error source with corpus absence | Existing typed status, source and error fields remain authoritative |
| Themes | Light/Dark/System, zoom, reduced motion, forced colors, 320px | CSS-custom-property-driven graph nodes/links/selection; no per-theme JS SVG regeneration; horizontal pan of diagrams only | Presentation-only; no API, CPE or concept promotion |

### Visual and accessibility invariants

- Four semantic states are explicitly **textual**, not color-only: busy, ready, blocked, error. An HTTP success is not an accepted CPE claim.
- **No hard-coded SVG fills or strokes** in dynamically generated graph shapes/edges/labels; CSS theme variables change all populated graphs with theme selection, including forced colors.
- **Two observable graph classes:** pinned relationships and query-specific title matches remain semantically different. Selection is reflected in both SVG controls and native button alternatives; diagram roles must not bury nested interactive elements under `role=img`.
- **No unintended scroll masking.** Graph is an intentionally two-dimensional, horizontally pannable region at 320px; ordinary forms, prose and result lists reflow without horizontal document overflow.
- **No extra request** from choosing a concept node, opening a facet disclosure, changing theme, or looking at graph metadata. Consent and budget admission remain on the existing explicit actions.
- Unicode and long labels remain text content, never HTML from provider responses. SVG labels may truncate visually; accessible names and list controls retain full names.
- Source `is_estimate`, `adult_metadata`, source failure, provisional semantic status and observation-vs-inference scopes must not be erased by visualization.
- No additional runtime dependencies, fonts, external images, network permissions, payload fetching, DHT/trackers or private data.
- **Security and provenance:** all staged public tests use fake 40-byte infohashes and synthetic fixture labels only. Real publisher `.torrent` binary custody stays owner-only Google Drive; code, tests, source decisions, SHA evidence and baton in Git/DLE.

### Qualification and remaining work

1. Public $0 GHA Playwright/axe on actual **populated** graphs, facets and errors across light/dark/forced colors, 320px overflow, theme switch without new data calls, keyboard/selection parity, explicit sweep consent. Attach real Chromium screenshots/traces.
2. Exact head Go unit/race/vet/build and unchanged live Bitmagnet/PostgreSQL/Explorer original 5-source metainfo regression. Preserve first failures.
3. Red/blue peer review: ensure standard API/CLI states remain authoritative and graphs are never policy decisions; exercise no-JS progressive text, 200/400% zoom and actual NVDA/VoiceOver/TalkBack at user acceptance.
4. Separately qualify error/partial/empty visual fixtures, Lighthouse/long lists, timer behavior and copy-to-clipboard fallback. No axe PASS equals screenreader certification.
5. Review merge sequence and official product integration only via owning private DLE and independent acceptance. **Do not merge/release/deploy** this research draft or preview-only Cloudflare PR #31.

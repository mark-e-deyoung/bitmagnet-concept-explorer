# C7 — one source read model, three read-only audiences

This source-only inspection adapter does **not** create a new service,
persist a database, scrape Gutenberg, accept CPE concepts, or alter characters.
It uses the exact same original public book SHA and candidate JSON as the
Gutenberg miner, validates source identity and each evidence line, and
exposes a read-only view for three audiences:

- Human: `-format=html` accessible, source-grounded, responsive static HTML
  with source SHA, candidate count, scientific UNKNOWN and the original
  lines; HTML template escapes source text (script-injection negative tested).
- Automation: `-format=json` deterministic typed JSON records, no writes,
  explicit statuses and exact source SHA.
- Agent: `-capabilities` typed JSON modes and holds. There are **no** agent
  write/apply operations or permissions to bypass rights/CPE/release gates.

Run after the original source miner:

```sh
go run ./research/gutenbergmotion/inspect -capabilities
go run ./research/gutenbergmotion/inspect -source 12926.txt -report motion-candidates.json -format=json > inspection.json
go run ./research/gutenbergmotion/inspect -source 12926.txt -report motion-candidates.json -format=html > human-inspection.html
```

The first milestone is a **read-only inspection UX**, not full UI/UX/DX
acceptance: browser-engine tests, VoiceOver, screen-reader workflow, editable
choreography, HTTP parity, cancellation/recovery, native phone playback and
complete actor interpretation are NOT_RUN/UNKNOWN. A green static HTML
structural test cannot certify those functions. No accepted CPE fact or
validated full human motion is claimed. No merge/release/deploy.

## Original source read-only search flow — three-audience increment

The new `-query` option is source-bound, case-insensitive, limited to
80 Unicode characters, and read-only. It is offered by the **same validated
source read model** to (1) human responsive HTML with an accessible labelled
live-search and Clear control, (2) automation CLI JSON with total and filtered
counts and selected source IDs, and (3) agent capability discovery and the
same typed read-only command contract.

```sh
go run ./research/gutenbergmotion/inspect -capabilities
go run ./research/gutenbergmotion/inspect -source 12926.txt -report candidates.json -format=json -query="right shoulder" > selected.json
go run ./research/gutenbergmotion/inspect -source 12926.txt -report candidates.json -format=html -query="right shoulder" > selected.html
```

The full source corpus retains all 64 original candidates. The browser
performs an initial source-bound prefilled search without any network or
mutation; the native GHA checks its actual headless Chromium DOM card count
and hidden states against independent Go CLI JSON counts. Existing
desktop/mobile Chromium screenshots and source/tamper/XSS tests remain.
A green filter demonstration does **not** qualify native phone VoiceOver,
full keyboard typing/form journey, human edit/undo, HTTP action parity,
long-running task cancellation or private MVP product acceptance. Those
remain independent NOT_RUN/UNKNOWN.

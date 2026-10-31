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

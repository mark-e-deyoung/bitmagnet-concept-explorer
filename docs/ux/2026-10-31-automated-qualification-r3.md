# UI/UX/DX R3 public qualification gates

Research-only QA workset. No product release or merge is implied.

Base: public Explorer R2 draft PR #33 at `0be1f760656c39df428eb220beeabcff74d60768`.

## Implemented free public-runner validation

- Existing Playwright Chromium + axe WCAG 2.2 checks of populated evidence, keyboard control, color themes, forced colors, reflow, consent and failure states.
- New independent Chromium, Firefox and WebKit browser matrix on the same source, testing keyboard concept selection, read-only search, selected evidence, facet estimates, multilingual text, 320 CSS-pixel reflow and error handling.
- New Lighthouse user flow on an isolated localhost synthetic test service. Desktop and mobile each include a navigation audit, an interaction timespan and a populated-state snapshot. Reports and screenshots are retained.
- Existing real upstream Bitmagnet/PostgreSQL/public publisher metainfo integration and Go unit/race/vet/amd64/arm64 checks run separately.

Lighthouse research score thresholds: desktop performance 80, mobile performance 70, navigation accessibility 90, best-practices 85, populated snapshot accessibility 90. Always retain raw audit scores and diagnostics. SEO is not gated because research previews are intentionally marked noindex. Laboratory performance is not a field-user metric.

The test fixture uses public synthetic records and no external adapters. Both GHA jobs require Node 22 or later, fixed direct npm dependencies, a generated temporary dependency lock and clean npm ci. The lock is uploaded for review; committing a transitive lock remains a separate reproducibility acceptance task.

## Remaining human acceptance

NVDA, VoiceOver, TalkBack, true optical zoom, independent visual judgment, user journey completion and performance under large real-world corpora are not certified by automation. Track them as UNKNOWN until directly observed.

Follow Discover / Observe / Plan / Apply / Verify. Preserve initial failures, changes, exact commit SHAs, artifacts and handoff in Bitmagnet product DLE #18. This draft does not authorize merging, deployment, semantic promotion or data publication.

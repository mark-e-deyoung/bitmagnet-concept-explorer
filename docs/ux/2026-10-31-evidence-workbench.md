# Concept Explorer — evidence workbench UI/UX/DX candidate

**Date:** 2026-10-31  
**Status:** draft / source-safe / unmerged  
**Scope:** public `mark-e-deyoung/bitmagnet-concept-explorer` UI, starting from exact Fedora original-source PR #30. This is not a new canonical brand identity, accepted CPE interpretation, TrueNAS release, or public disclosure of private RDTE corpus.

## Product-native design method and reference practices

The public SupraCraft Bridge and VanillaCord sites provide two independently recognizable products with shared quality standards rather than shared illustrations or copied trade dress:

- [Bridge brand guidelines](https://github.com/SupraCraft/Bridge/blob/master/docs/brand-guidelines.md): literal project mechanism, restrained technical palette, factual copy, semantic no-JS public descriptions, generated machine surfaces
- [VanillaCord brand guidelines](https://github.com/SupraCraft/VanillaCord/blob/master/docs/brand-guidelines.md): task-first public information architecture, one product-native metaphor, operator journeys on the page, machine-readable endpoints, clear safety boundary
- [Bridge Pages](https://supracraft.github.io/Bridge/) and [VanillaCord Pages](https://supracraft.github.io/VanillaCord/) are public reference URLs but were not available for **live browser rendering** from this session; designs were inspected at exact public Git source blob revisions, not claimed as live visual tests
- [W3C WCAG 2.2](https://www.w3.org/TR/wcag/): meaningful labels, visible keyboard focus, adequate target sizes and contrast, reflow and accessible interactive semantics

**Not adopted:** SupraCraft organization affiliation, marks, copper-only palette, its project imagery, Minecraft cues, Bridge's bridge glyph, VanillaCord's cord or copyrighted vendor motifs. The Explorer is a separate product, outside the SupraCraft branding authority.

### Identity hypothesis (candidate, not final logo)

**Name/mechanism/user job:** Bitmagnet Concept Explorer displays indexed Bitmagnet metadata across direct and experimental concept retrieval tracks, with a clear observed-evidence boundary.

**Literal visual sentence:** Three bounded evidence paths converge at one inspectable junction. The small nav pictogram and restrained hero geometry are an exploratory interface cue; no ownership claim over Bitmagnet's upstream branding or CPE acceptance.

**Negative space:** No mysterious AI brain, network globe, glowing hexagons, generic vector graph logo, fake data-corpus completeness indicator, torrent payload/download icon, proprietary trademarks or claimed scientific certainty. Do not use colors or charts to imply that an observed term is an accepted semantic fact.

**Visual tokens:** legible neutral shell; teal/ivory technical atmosphere, restrained copper for the secondary track, systematic contrast in Light/Dark/System. Product-native choices are candidates pending independent brand/discovery review, not SupraCraft-adopted colors.

## Current first-failure usability analysis (source-grounded)

1. Original public RDTE preview placed a full advanced filter form above any explanation of tasks and outputs. A new user encountered language IDs, seed/leech thresholds, adult flags, sort/group choices, web consent and graph options without a reason to act.
2. Prior CSS was fixed dark with long single-line rules, weak hierarchy, one large title, repeated dark panels and no explicit main/footer/navigation landmarks; active tracks appeared as undifferentiated cards.
3. Direct/concept/overlap result cards began empty; the UI did not explicitly explain the differences before the first query. Failure-vs-absence guard existed in result strings but was not visible as an opening mental model.
4. Primary human path, advanced multilingual investigations and machine interfaces were interleaved. `/api/v1/capabilities` and `/api/v1/concepts` existed but were hard for a visitor to discover.
5. A malformed `max="3 value="3"` attribute was observed in the public Explorer's sweep budget control; this branch fixes that concrete HTML defect independently of the private product's existing local fix.
6. Browsers, screen readers, contrast at 200% zoom, reduced-motion/forced-colors, and actual task completion had not been demonstrated on this particular UI head.

## Candidate information architecture

**Search → Results → Concepts → Sweep**, with a skip link, main and footer landmarks, anchored task navigation and native theme group.

**Primary path:** Literal phrase input, visible constraints/capabilities, optional example-fills (not automatic search), one submit action. Advanced filters and manual term expansion start collapsed in a clearly labeled `details` element. No network action from opening advanced controls.

**Compare:** Separate direct / experimental concept / shared match columns with explanatory empty states and run status. The evidence classifications, CPE policy boundaries, and source failures are preserved verbatim and in API data, not converted into recommendations.

**Investigate:** Existing graph/facet/sweep functionality and identifiers are preserved. Expensive/advanced source operations remain opt-in. Raw JSON technical evidence is available in a disclosure instead of occupying the main sightline.

**Automation/agents:** Existing read-only JSON endpoints and CLI contract do not change. The human UI links to machine capabilities and concept projections. Source IDs, response models, and trace semantics remain invariant.

## Executable quality gate

| Surface | Check | Qualification |
| --- | --- | --- |
| Source | Go unit, race, vet, amd64/arm64, existing browser-state regression | Native public GHA exact head |
| HTML/JS | Native landmarks, unique IDs, labels, progressive disclosure, three themes, contrast threshold, inline script syntax | `node web/ux_contract.test.mjs` |
| Real browser | Chromium desktop 1440, mobile 390, 320 px reflow, search task, evidence columns, theme persistence, keyboard focus | Playwright |
| Accessibility | Axe WCAG 2.2 A/AA automated scan + controls, visible labels & focus + forced-colors/reduced-motion CSS source | Playwright + independent assistive human follow-up |
| Visual QA | Real full-page desktop and narrow screenshots, inspect hierarchy, density, graph legibility, contrast and clipping | Playwright artifact + independent human review |
| Performance | No new runtime JavaScript libraries, CDN images, font network fetch or additional API calls on initial load | Source/read-only run + Lighthouse follow-up |
| Reality | Public GHA / real upstream Bitmagnet+Postgres Explorer later rep on exact rebased head | **NOT** equivalent to static screenshot tests |

**Caution:** Playwright and axe versions are pinned in `package.json` as test-only dependencies, not shipped with the Go service. A transitive lockfile and actual browser/assistive review are additional gates; do not conflate source checks with a user-tested release.

## Priority backlog after this first pass

- **P0:** Repair all first-failure CI, axe and browser screenshots; ensure keyboard form submit, focus and error recovery; enforce visual clarity under Light/Dark/System.
- **P1:** Independent real users complete "find direct", "add a term", "inspect why", and "distinguish no-data from source-failure" without help; measure time to first meaningful result and task success.
- **P1:** Full keyboard/zoom/forced-colors and screenreader sessions (NVDA on Windows, VoiceOver on macOS/iOS, TalkBack on Android), including SVG concept graph nonvisual equivalents. Axe alone does not establish accessibility.
- **P1:** Source-safe preview branch integration that renders *this* UI on a bounded GHA Cloudflare viewer; current preview PR #31 is intentionally NEVER MERGE and should not receive permanent design changes.
- **P1:** Review canonical private `bitmagnet-tools/concept-explorer` alignment to public implementation. Port only via owning DLE/PR and independent API/CLI/agent parity checks; no silent cross-repo overwrite.
- **P2:** Product-native brand identity/name collision and small-asset review; engineering typography, responsive graphics and graphical relationship provenance.
- **P2:** CI screenshots, Lighthouse budgets, interaction time/bytes, explicit accessibility feedback, durable public release/discovery documents and version-neutral machine interface catalog.

## Authority, storage and holds

Store source, CSS/HTML, tests, design decisions and exact-SHA handoff in public Git and owning Bitmagnet #18 DLE; store larger visual QA PNG/video as short-lived public GHA artifacts without private information. CPE original `.torrent` binaries and owner-only quarantine identifiers remain in private Drive and private product Git, never public UI screenshots. No CPE ontology/fact acceptance, rights promotion, scheduled harvesting, private source exposure, merged product branch, release or TrueNAS production deployment occurs here.

# Bounded Debian RDTE metainfo source rep — 2026-10-31

## Grounding and scope
Debian official BitTorrent distribution list: https://www.debian.org/CD/torrent-cd/ . The independent Debian CD mirror directory https://mirrors.edge.kernel.org/debian-cd/current/amd64/bt-cd/ lists `debian-13.7.0-amd64-netinst.iso.torrent` as an authentic distribution item (dated 2026-09-12), distinct from Ubuntu. This source is a *mirror of an official publisher distribution*, not an independent rights certificate.

The collector only fetches the original `.torrent` metainfo over HTTPS from this fixed hostname, rejects HTTP redirect downgrades, checks strict bounded bencode, excludes private/v2 metainfo, computes the SHA-1 of the original unmodified raw `info` bytes and whole metainfo SHA-256, and records source URL, versioned source item, rights **claim**, source events and audit manifest. No ISO payload, DHT, tracker, swarm, peer, or private corpus content is fetched.

## Native qualification
This draft is stacked on independent real-system Explorer PR #18, not a change to the original source-plugin authority in `mark-e-deyoung/bitmagnet-tools/scripts/sources`. The public Actions rep must prove (a) at least 3 unique original metainfo identities, (b) exactly one Debian family manifest entry and (c) that exact Debian infohash is actually imported to upstream Bitmagnet v0.10.0/PostgreSQL and retrieved as a direct Explorer hit. A source failure must fail closed, preserve its first failure/event, and must not be replaced by a synthetic fixture, an Internet Archive item, or a different mirror without an observed reason.

The qualified source URL is a *versioned source record*, not a promise of perpetual stability: the `current` path can move. Immutable replay uses captured SHA-addressed original bytes and a manifest. A source terms review and an approved private/controlled object custody decision are still separate gates; no public/policy/CPE acceptance occurs here.

## Holds
Keep this PR open/draft/unmerged. Do not activate default-branch cron, auto-accumulate, create a new harvester/control plane, publish binaries or images, deploy TrueNAS, promote CPE concepts/facts, or touch local/private data. GHA runner artifact is only temporary; approved custody is managed separately via CPE #348 and product DLE Bitmagnet #18.
## Observed remote-source failure and corrective action
First head `a217f95d5309bce406f392f621e82e2ef470603d`, native run `41200660555` failed closed because `ftp.be.debian.org` returned TLS certificate for `mirror.as35701.net`. Error is preserved in immutable native log and PR #25 first-failure comment; no certificate check disabled and no Debian record fabricated. Switched to the independently indexed `mirrors.edge.kernel.org` Debian CD mirror; acceptance requires new **exact-head** native source and real-system CI.

Transport red-team: an allowlisted-host redirect could otherwise downgrade from HTTPS to HTTP; this draft adds an explicit HTTPS check for every redirect. Qualification must be rerun at the new exact head and does not retroactively change prior results.

## Two-mirror exact-byte admission follow-up
After one successful live import of three real publisher metainfo files, repeated native runner requests to `mirrors.edge.kernel.org` returned `context deadline exceeded (Client.Timeout exceeded while awaiting headers)` at 30 seconds; these are preserved historical FAIL receipts in PR #25. The original source bytes have independently checked SHA-256 and raw infohash in private evidence quarantine; the previously successful run is valid, but does not imply reliable mirror availability.

Added independent Debian mirror `https://debian.nic.cz/debian-cd/current/amd64/bt-cd/debian-13.7.0-amd64-netinst.iso.torrent`, documented by the publisher mirror list and current mirror index. Tries `debian.nic.cz` then `mirrors.edge.kernel.org` at most once each under an existing 30-second per-source timeout and HTTPS/redirect allowlist. A candidate is admitted **only** if entire original metainfo SHA-256 matches `690700ed8de8d0987fcf8c136507471b1588b996696ca34f9cf972ee09b849e0` AND raw infohash matches `7acf8fb590b2060dd9c3146ef770169d593433b0`; each rejected origin/reason is preserved in the manifest source events. **Never disable TLS or silently substitute a new Debian ISO revision.** Three-item minimum and original Debian-through-real-Bitmagnet FTS verification stay mandatory. This change may extend a run's source fetch window by up to 30 seconds; no unbounded retry or payload/peer access is added.

The independent original 3-binary archive remains in owner-only Google Drive quarantine and is **not** a new public corpus release. The expected full metainfo SHA-256 is from the already verified raw artifact; source-distribution rights remain `source_claim_not_independently_qualified`. All activation, merge, CPE, release and TrueNAS holds unchanged.

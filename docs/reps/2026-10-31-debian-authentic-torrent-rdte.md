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

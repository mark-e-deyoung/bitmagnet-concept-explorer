# Bounded Debian RDTE metainfo source rep — 2026-10-31

## Grounding and scope
Debian official BitTorrent distribution list: https://www.debian.org/CD/torrent-cd/ . The Debian-operated mirror directory https://ftp.be.debian.org/debian-cd/current/amd64/bt-cd/ lists `debian-13.7.0-amd64-netinst.iso.torrent` as an authentic distribution item (dated 2026-09-12), distinct from Ubuntu. This source is a *mirror of an official publisher distribution*, not an independent rights certificate.

The collector only fetches the original `.torrent` metainfo over HTTPS from this fixed hostname, checks strict bounded bencode, excludes private/v2 metainfo, computes the SHA-1 of the original unmodified raw `info` bytes and whole metainfo SHA-256, and records source URL, versioned source item, rights **claim**, source events and audit manifest. No ISO payload, DHT, tracker, swarm, peer, or private corpus content is fetched.

## Native qualification
This draft is stacked on independent real-system Explorer PR #18, not a change to the original source-plugin authority in `mark-e-deyoung/bitmagnet-tools/scripts/sources`. The public Actions rep must prove (a) at least 3 unique original metainfo identities, (b) exactly one Debian family manifest entry and (c) that exact Debian infohash is actually imported to upstream Bitmagnet v0.10.0/PostgreSQL and retrieved as a direct Explorer hit. A source failure must fail closed, preserve its first failure/event, and must not be replaced by a synthetic fixture, an Internet Archive item, or a different mirror without an observed reason.

The qualified source URL is a *versioned source record*, not a promise of perpetual stability: the `current` path can move. Immutable replay uses captured SHA-addressed original bytes and a manifest. A source terms review and an approved private/controlled object custody decision are still separate gates; no public/policy/CPE acceptance occurs here.

## Holds
Keep this PR open/draft/unmerged. Do not activate default-branch cron, auto-accumulate, create a new harvester/control plane, publish binaries or images, deploy TrueNAS, promote CPE concepts/facts, or touch local/private data. GHA runner artifact is only temporary; approved custody is managed separately via CPE #348 and product DLE Bitmagnet #18.
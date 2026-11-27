# Offline v1 metainfo identity comparison

The RDTE collector still accepts only its existing explicitly admitted publisher sources and exact previously pinned bytes. The helper compares **already available** bounded v1 bytes against two prior digests: whole-file SHA-256 and original bencoded info SHA-1. The tests distinguish exact byte replay from same-infohash wrapper variants and different infohashes. Only exact byte replay passes the existing pinned-original gate. Neither matching digest authenticates the prior issuer or establishes distribution rights.

This is not a general metainfo validator, a peer metadata client, a recovery service, or v2/hybrid support. The existing decoder's scope is the bounded publisher RDTE; full structural/piece-layer/path validation needs a separate qualification. No new source, external lookup, payload, tracker or DHT traffic. Tests are entirely synthetic.

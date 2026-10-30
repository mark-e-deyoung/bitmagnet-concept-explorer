# Bitmagnet Concept Explorer

This app provides a read-only interface to Bitmagnet indexed metadata. It does not download payloads, tag torrents, or write to Bitmagnet.

<b>Network and credentials</b><br>
The UI and API require Basic Auth credentials. Basic Auth does not encrypt traffic. Keep the service unexposed or access it through a TLS-terminating reverse proxy. The package exposes its service only for inter-app access and does not offer direct host-port publishing. The Bitmagnet integration accepts a private or loopback IP or localhost URL ending in /graphql; other DNS names and public addresses are rejected.

<b>Release status</b><br>
Version 0.0.1 is a public source preview. Foundry lifecycle qualification and official TrueNAS catalog review remain outstanding.

# Bitmagnet Concept Explorer

This app provides a read-only interface to Bitmagnet indexed metadata. It does not download payloads, tag torrents, or write to Bitmagnet.

<b>Network and credentials</b><br>
The UI and API require Basic Auth credentials. Basic Auth does not encrypt traffic. Keep the service unexposed or access it through a TLS-terminating reverse proxy. The candidate source has a host-port mapping: by default the selected host IP is 127.0.0.1, not every NAS interface. If a remote client needs access, configure a trusted host IP and put a TLS-terminating reverse proxy in front of Basic Auth before exposing traffic. The Bitmagnet integration accepts a private or loopback IP or localhost URL ending in /graphql; other DNS names and public addresses are rejected by the application. Inside the container, localhost refers to the Explorer container itself and NOT the TrueNAS host or Bitmagnet app. Use an explicitly configured reachable private endpoint.

<b>Release status</b><br>
Version 0.0.1 is a public source preview. Foundry lifecycle qualification and official TrueNAS catalog review remain outstanding.

<b>Concept graph data</b><br>
The optional graph loader accepts only a digest-pinned provisional display/query-expansion snapshot, not policy rules or an accepted CPE decision model. The TrueNAS package does not currently expose a graph file mount or digest setting, so operator-provided graph data cannot yet be loaded through the catalog UI.

<b>Resource limits and acceptance</b><br>
The pinned TrueNAS `ix_lib` 2.3.18 resource renderer automatically consumes `resources.limits.cpus` and `resources.limits.memory`; the app template does not need to mention those keys directly. Render the final Compose with the exact library before claiming resource enforcement. This is a source candidate only: the GHCR package pull, official catalog metadata/icon, target exposure, TrueNAS F0–F5 lifecycle, and any subsequent publication require separate verification and owner approval.

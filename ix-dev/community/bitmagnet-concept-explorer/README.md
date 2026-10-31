# Bitmagnet Concept Explorer

This app provides a read-only interface to Bitmagnet indexed metadata. It does not download payloads, tag torrents, or write to Bitmagnet.

<b>Network and credentials</b><br>
The UI and API require Basic Auth credentials. Basic Auth does not encrypt traffic. Keep the service unexposed or access it through a TLS-terminating reverse proxy. The default **exposed** setting only exposes container TCP port 3086 to reachable container networks: the upstream TrueNAS 2.3.18 source renderer emits `expose: [3086/tcp]` and no host `ports` mapping. The separately selectable **published** mode binds a NAS host port only to explicitly selected host IPs; the default is 127.0.0.1, and remote exposure requires deliberate operator configuration and TLS termination. An exposed container port is not a promise that a reverse proxy in another network can reach the service; that path needs an independent topology and TLS test. The Bitmagnet integration accepts a private or loopback IP or localhost URL ending in /graphql; other DNS names and public addresses are rejected by the application. Inside the container, localhost refers to the Explorer container itself and NOT the TrueNAS host or Bitmagnet app. Use an explicitly configured reachable private endpoint.

<b>Release status</b><br>
Version 0.0.1 is a public source preview. Foundry lifecycle qualification and official TrueNAS catalog review remain outstanding.

<b>Concept graph data</b><br>
The optional graph loader accepts only a SHA-256-pinned provisional display/query-expansion snapshot, not policy rules or an accepted CPE decision model. The candidate app wizard now provides OPTIONAL read-only host-file paths and exact SHA-256 digests for both the concept graph and attributed public research evidence. Specify a host-side file and its matching byte digest together, or leave both fields blank. The renderer refuses unpaired fields and does not create host files. Mounted source files must already exist and be readable by container UID 65532; this is not yet verified against a live TrueNAS host. Catalog claims, artwork metadata and recorded rights observations remain research source claims—not independently accepted CPE facts, permission to use an artwork, or automatic concept promotion.

<b>Resource limits and acceptance</b><br>
The pinned TrueNAS `ix_lib` 2.3.18 resource renderer automatically consumes `resources.limits.cpus` and `resources.limits.memory`; the app template does not need to mention those keys directly. Render the final Compose with the exact library before claiming resource enforcement. This is a source candidate only: the GHCR package pull, official catalog metadata/icon, target exposure, TrueNAS F0–F5 lifecycle, and any subsequent publication require separate verification and owner approval.

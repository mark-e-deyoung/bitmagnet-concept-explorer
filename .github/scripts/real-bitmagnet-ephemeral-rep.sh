#!/usr/bin/env bash
# Real upstream Bitmagnet v0.10.0, PostgreSQL and real BCE binaries on one
# disposable GitHub-hosted runner. No mock services, DHT, public ingress or
# existing-user corpus. All imported metadata below is deliberately fictional.
set -Eeuo pipefail

: "$RUNNER_TEMP"
OUT="$RUNNER_TEMP/bce-real-bitmagnet-rep"
mkdir -p "$OUT"
SERVICE_PID=""
NETWORK="bce-real-rep"
DB="bce-real-postgres"
BITMAGNET="bce-real-bitmagnet"
POSTGRES_IMAGE="postgres@sha256:0cefaabd882e42bcb2a2f3dc56d3ce4770560ba62fd7f197f1a2e008fd5731ab"
BITMAGNET_IMAGE="ghcr.io/bitmagnet-io/bitmagnet@sha256:cf2c16fac5b51f1c18e630b4710175d35d7e44b83012cfd05a2b43712feec05e"
export NO_PROXY="127.0.0.1,localhost"
export no_proxy="$NO_PROXY"

cleanup() {
  local status="$1"
  trap - EXIT
  if [[ -n "$SERVICE_PID" ]]; then
    kill "$SERVICE_PID" 2>/dev/null || true
    wait "$SERVICE_PID" 2>/dev/null || true
  fi
  if [[ "$status" != "0" ]]; then
    echo "::error::Real Bitmagnet cloud rep FAILED; inspect retained GitHub Actions first-failure logs"
    echo "=== BCE service log ==="
    tail -n 80 "$OUT/bce.log" 2>/dev/null || true
    echo "=== Real Bitmagnet log ==="
    docker logs --tail 100 "$BITMAGNET" 2>&1 || true
    echo "=== PostgreSQL log ==="
    docker logs --tail 45 "$DB" 2>&1 || true
  fi
  docker rm -f "$BITMAGNET" "$DB" >/dev/null 2>&1 || true
  docker network rm "$NETWORK" >/dev/null 2>&1 || true
  exit "$status"
}
trap 'cleanup $?' EXIT

echo "REAL-BITMAGNET: source=$(git rev-parse HEAD)"
go version
docker version --format 'Docker server {{.Server.Version}}'
docker pull --quiet "$POSTGRES_IMAGE"
docker pull --quiet "$BITMAGNET_IMAGE"
POSTGRES_IDENTITY="$(docker image inspect "$POSTGRES_IMAGE" --format '{{index .RepoDigests 0}}')"
BITMAGNET_IDENTITY="$(docker image inspect "$BITMAGNET_IMAGE" --format '{{index .RepoDigests 0}}')"
if [[ "$POSTGRES_IDENTITY" != *@sha256:* || "$BITMAGNET_IDENTITY" != *@sha256:* ]]; then
  echo "::error::Missing OCI pull digest: cannot qualify image identity"
  exit 1
fi
echo "REAL-BITMAGNET: postgres_image=$POSTGRES_IDENTITY"
echo "REAL-BITMAGNET: bitmagnet_image=$BITMAGNET_IDENTITY"

docker network create "$NETWORK" >/dev/null
docker run -d --name "$DB" --network "$NETWORK" \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=public-fixture-only \
  -e POSTGRES_DB=bitmagnet --shm-size 1g "$POSTGRES_IMAGE" >/dev/null

healthy=0
for i in $(seq 1 60); do
  if docker exec "$DB" pg_isready -U postgres -d bitmagnet >/dev/null 2>&1; then healthy=1; break; fi
  sleep 2
done
if [[ "$healthy" != "1" ]]; then
  echo "::error::Real PostgreSQL did not reach readiness"
  exit 1
fi
echo "REAL-BITMAGNET: PostgreSQL ready"

# The upstream worker starts its actual HTTP/GraphQL, import, queue and
# classifier paths. DHT never starts; no BitTorrent ports are exposed.
# TMDB is disabled to keep the rep public-only and no external API calls.
docker run -d --name "$BITMAGNET" --network "$NETWORK" \
  -p 127.0.0.1:3333:3333 \
  -e POSTGRES_HOST="$DB" -e POSTGRES_PASSWORD=public-fixture-only \
  -e POSTGRES_DB=bitmagnet -e TMDB_ENABLED=false \
  "$BITMAGNET_IMAGE" worker run --keys=http_server --keys=queue_server >/dev/null

ready=0
for i in $(seq 1 75); do
  if curl --noproxy '*' -fsS --max-time 3 -H 'Content-Type: application/json' \
    -d '{"query":"query { version }"}' http://127.0.0.1:3333/graphql \
    -o "$OUT/version.json" 2>/dev/null \
    && jq -e '.data.version | type == "string" and length > 0' "$OUT/version.json" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done
if [[ "$ready" != "1" ]]; then
  echo "::error::Pinned upstream Bitmagnet GraphQL version never became available"
  exit 1
fi
echo "REAL-BITMAGNET: GraphQL version=$(jq -r '.data.version' "$OUT/version.json")"
if [[ "$(jq -r '.data.version' "$OUT/version.json")" != "v0.10.0" ]]; then
  echo "::error::Immutable Bitmagnet image does not report pinned source version v0.10.0"
  exit 1
fi

echo "REAL-BITMAGNET: importing 5 ORIGINAL PUBLIC SYNTHETIC metadata records using upstream /import"
curl --noproxy '*' -fsS --max-time 90 \
  -H 'Content-Type: application/json' -H 'Connection: close' \
  --data-binary @fixtures/public-bitmagnet-rep.ndjson \
  http://127.0.0.1:3333/import >"$OUT/import.txt"
cat "$OUT/import.txt"
if ! grep -q '5 items imported' "$OUT/import.txt" || ! grep -q 'import complete' "$OUT/import.txt"; then
  echo "::error::Real Bitmagnet importer did not acknowledge all public test metadata"
  exit 1
fi

# Load a *real public metadata source* instead of inventing hundreds of fake
# book names. Project Gutenberg expressly publishes bulk machine-readable
# catalogs for automation. This is ONE bulk download, NOT web page scraping.
# The real Bitmagnet importer receives catalog-derived titles/authors, but each
# infohash and size is deliberately fabricated as an isolated test identifier.
CATALOG_URL="https://www.gutenberg.org/cache/epub/feeds/pg_catalog.csv.gz"
CATALOG_MIRROR="https://dev.gutenberg.org/cache/epub/feeds/pg_catalog.csv.gz"
if curl -fsSL --retry 1 --connect-timeout 12 --max-time 180 --max-filesize 15000000 \
  "$CATALOG_URL" -o "$OUT/pg_catalog.csv.gz"; then
  echo "PUBLIC-GUTENBERG-CATALOG: official bulk metadata acquired from $CATALOG_URL"
else
  echo "::warning::Official Gutenberg bulk catalog origin unavailable; trying one official dev mirror"
  rm -f "$OUT/pg_catalog.csv.gz"
  curl -fsSL --retry 1 --connect-timeout 12 --max-time 180 --max-filesize 15000000 \
    "$CATALOG_MIRROR" -o "$OUT/pg_catalog.csv.gz"
  echo "PUBLIC-GUTENBERG-CATALOG: official bulk metadata acquired from $CATALOG_MIRROR"
fi

python3 .github/scripts/sample-public-gutenberg-catalog.py \
  --catalog "$OUT/pg_catalog.csv.gz" \
  --output "$OUT/public-catalog-only.ndjson" \
  --manifest "$OUT/public-catalog-manifest.json"
CATALOG_ROWS="$(jq -r '.selected' "$OUT/public-catalog-manifest.json")"
if [[ "$CATALOG_ROWS" -lt 180 ]]; then
  echo "::error::Real public catalog sample failed minimum corpus-size admission"
  exit 1
fi
echo "PUBLIC-GUTENBERG-CATALOG: import real metadata ($CATALOG_ROWS distinct source records), never actual torrents"
curl --noproxy '*' -fsS --max-time 150 \
  -H 'Content-Type: application/json' -H 'Connection: close' \
  --data-binary @"$OUT/public-catalog-only.ndjson" \
  http://127.0.0.1:3333/import >"$OUT/public-catalog-import.txt"
if ! grep -qF "$CATALOG_ROWS items imported" "$OUT/public-catalog-import.txt" || ! grep -q 'import complete' "$OUT/public-catalog-import.txt"; then
  echo "::error::Real Bitmagnet /import did not admit full genuine-catalog metadata sample"
  cat "$OUT/public-catalog-import.txt"
  exit 1
fi
echo "PUBLIC-GUTENBERG-CATALOG: real Bitmagnet importer acknowledged $CATALOG_ROWS actual catalog-derived titles"

# This public draft pack is a digest-pinned RESEARCH vocabulary, not CPE-accepted
# concepts or facts. The separate evidence JSON supplies attributed work data.
PACK_DIR="testdata/public-cpe-packs/gutenberg-lore-v0"
GRAPH="$PWD/$PACK_DIR/graph-preview.json"
EVIDENCE="$PWD/$PACK_DIR/catalog-evidence.json"
test -s "$GRAPH" && test -s "$EVIDENCE"
GRAPH_SHA="$(sha256sum "$GRAPH" | cut -d' ' -f1)"
echo "PUBLIC-CPE: candidate pack=$PACK_DIR sha256=$GRAPH_SHA"

# REAL TORRENT RDTE: fetch official publisher .torrent metainfo, hash the
# original bencoded info bytes to derive genuine torrent infohashes, retain
# source URLs and SHA-256 of intact metainfo for replay. No swarm, DHT,
# trackers or payload download. Internet Archive selection is independently
# source-filtered; Ubuntu publisher torrents are mandatory in this first rep.
RDTE_DIR="$OUT/rdte-real-metainfo"
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
go run ./cmd/rdte-torrent-collector -out "$RDTE_DIR" -minimum 3
REAL_TORRENT_ROWS="$(jq -r '.entries | length' "$RDTE_DIR/manifest.json")"
if [[ "$REAL_TORRENT_ROWS" -lt 3 ]]; then
  echo "::error::No qualifying genuine publisher-provided .torrent metainfo"
  exit 1
fi
# A minimum count alone does not prove independent source-family coverage.
DEBIAN_SOURCE_ROWS="$(jq '[.entries[] | select(.source == "debian:official-mirror")] | length' "$RDTE_DIR/manifest.json")"
if [[ "$DEBIAN_SOURCE_ROWS" != "1" ]]; then
  echo "::error::Debian official-mirror metainfo was not genuinely admitted"
  jq '{entries: [.entries[] | {source,infohash,metainfo_sha256}], source_events}' "$RDTE_DIR/manifest.json" || true
  exit 1
fi

curl --noproxy '*' -fsS --max-time 120 \
  -H 'Content-Type: application/json' -H 'Connection: close' \
  --data-binary @"$RDTE_DIR/bitmagnet-import.ndjson" \
  http://127.0.0.1:3333/import >"$OUT/real-torrent-import.txt"
if ! grep -qF "$REAL_TORRENT_ROWS items imported" "$OUT/real-torrent-import.txt" || ! grep -q 'import complete' "$OUT/real-torrent-import.txt"; then
  echo "::error::Real Bitmagnet upstream importer rejected verified real publisher metainfo"
  cat "$OUT/real-torrent-import.txt"
  exit 1
fi
echo "CPE-RDTE-REAL: genuine publisher torrent infohashes imported=$REAL_TORRENT_ROWS"

# Compile and start the ACTUAL Explorer service and standalone automation CLI.
# No service ports are reachable from outside this ephemeral runner.
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export CGO_ENABLED=0
go build -trimpath -o "$OUT/bce" .
go build -trimpath -o "$OUT/bce-cli" ./cmd/bce
BCE_AUTH_USER=public-rep BCE_AUTH_PASSWORD=synthetic-only-credential-2026 \
BCE_BITMAGNET_GRAPHQL=http://127.0.0.1:3333/graphql \
BCE_SEARXNG_URL= BCE_CPE_GRAPH_FILE="$GRAPH" BCE_CPE_GRAPH_SHA256="$GRAPH_SHA" \
  "$OUT/bce" -listen 127.0.0.1:3086 >"$OUT/bce.log" 2>&1 &
SERVICE_PID="$!"

ready=0
for i in $(seq 1 50); do
  if curl --noproxy '*' -fsS --max-time 3 http://127.0.0.1:3086/healthz -o "$OUT/health.json" 2>/dev/null; then
    ready=1; break
  fi
  sleep 1
done
if [[ "$ready" != "1" ]]; then
  echo "::error::Actual Explorer service did not become healthy"
  exit 1
fi

# No anonymous access to the product API. Health is intentionally unprotected.
status="$(curl --noproxy '*' -sS --max-time 4 -o /dev/null -w '%{http_code}' http://127.0.0.1:3086/api/v1/capabilities)"
if [[ "$status" != "401" ]]; then
  echo "::error::Unauthenticated actual Explorer capability endpoint returned HTTP $status"
  exit 1
fi
AUTH='public-rep:synthetic-only-credential-2026'
curl --noproxy '*' -fsS --max-time 4 -u "$AUTH" \
  http://127.0.0.1:3086/api/v1/capabilities >"$OUT/capabilities.json"
jq -e '.execute == true and .mutation == false and .cpe_write == false and (.audiences | index("human") != null and index("automation") != null and index("agent") != null)' "$OUT/capabilities.json" >/dev/null
curl --noproxy '*' -fsS --max-time 4 -u "$AUTH" http://127.0.0.1:3086/ >"$OUT/index.html"
if ! grep -qi 'concept explorer' "$OUT/index.html"; then
  echo "::error::Actual Explorer browser document not served"
  exit 1
fi

# Human/API/agent interface reads this public, digest-pinned concept type pack.
# It is *not* a normative CPE snapshot or a rights/creative-work verdict.
curl --noproxy '*' -fsS --max-time 5 -u "$AUTH" \
  http://127.0.0.1:3086/api/v1/concepts >"$OUT/public-cpe-concepts.json"
jq -e '
  .authority_status == "pinned-preview-not-normative" and
  .snapshot_id == "public-gutenberg-lore-research-20261031-v0" and
  (.concepts | length) == 10 and
  (any(.concepts[]; .id == "public:literary-work")) and
  (any(.concepts[]; .id == "public:fictional-character")) and
  (any(.concepts[]; .id == "public:source-claim")) and
  (any(.concepts[]; .id == "public:rights-claim")) and
  (any(.concepts[]; .id == "public:literary-work" and
      (any(.lexicalizations[]; .language == "de")) and
      (any(.lexicalizations[]; .language == "th"))))
' "$OUT/public-cpe-concepts.json" >/dev/null
echo "PUBLIC-CPE: original public Git-backed pack with 10 types, EN/DE/TH/JA/ES lexicalizations loaded from actual digest-pinned file"

# Import and classification are genuinely asynchronous. Poll the real
# PostgreSQL-backed Bitmagnet FTS through Explorer, not a mocked response.
cat >"$OUT/query.json" <<'QUERY'
{"query":"Barsoom","expansions":[{"term":"Mars","provenance":"manual-unverified"}]}
QUERY
found=0
for i in $(seq 1 60); do
  curl --noproxy '*' -sS --max-time 25 -u "$AUTH" \
    -H 'Content-Type: application/json' --data-binary @"$OUT/query.json" \
    http://127.0.0.1:3086/api/v1/query/execute -o "$OUT/execute.json" || true
  if jq -e '
    .state == "complete" and .read_only == true and
    (.track_counts.direct // 0) >= 1 and (.track_counts.concept // 0) >= 1 and
    (.results | length) >= 2 and
    (any(.results[]; (.tracks | index("direct") != null and index("concept") != null)))
  ' "$OUT/execute.json" >/dev/null 2>&1; then
    found=1
    break
  fi
  sleep 2
done
if [[ "$found" != "1" ]]; then
  echo "::error::Real Bitmagnet imported rows did not reach BOTH query tracks and an overlapping hit"
  echo "Last synthetic Explorer summary:"
  jq '{state,track_counts,failures,warnings,coverage,before_filters,after_filters}' "$OUT/execute.json" || true
  exit 1
fi
echo "REAL-BITMAGNET: direct=$(jq -r '.track_counts.direct' "$OUT/execute.json") concept=$(jq -r '.track_counts.concept' "$OUT/execute.json") merged=$(jq -r '.results|length' "$OUT/execute.json")"


# Qualify the ACTUAL pinned upstream native FTS parser. Unquoted
# "science fiction" is a conjunction, not phrase adjacency. A real imported
# record with an intervening word must match AND, but not the quoted phrase.
# This does not accept any CPE lexical or concept policy.
cat >"$OUT/and.json" <<'QUERY'
{"query":"science fiction"}
QUERY
cat >"$OUT/phrase.json" <<'QUERY'
{"query":"\"science fiction\""}
QUERY
fts=0
for i in $(seq 1 30); do
  curl --noproxy '*' -sS --max-time 25 -u "$AUTH" \
    -H 'Content-Type: application/json' --data-binary @"$OUT/and.json" \
    http://127.0.0.1:3086/api/v1/query/execute -o "$OUT/and-result.json" || true
  curl --noproxy '*' -sS --max-time 25 -u "$AUTH" \
    -H 'Content-Type: application/json' --data-binary @"$OUT/phrase.json" \
    http://127.0.0.1:3086/api/v1/query/execute -o "$OUT/phrase-result.json" || true
  if jq -e '.state == "complete" and .read_only == true and (.track_counts.direct // 0) >= 2 and (.results | length) >= 2' "$OUT/and-result.json" >/dev/null 2>&1 \
    && jq -e '.state == "complete" and .read_only == true and (.track_counts.direct // 0) == 1 and (.results | length) == 1 and .results[0].info_hash == "4444444444444444444444444444444444444444"' "$OUT/phrase-result.json" >/dev/null 2>&1; then
    fts=1
    break
  fi
  sleep 2
done
if [[ "$fts" != "1" ]]; then
  echo "::error::Actual Bitmagnet v0.10.0 multiword AND vs phrase adjacency distinction not verified"
  echo "Last AND search:"
  jq '{state,track_counts,results,failures}' "$OUT/and-result.json" || true
  echo "Last quoted phrase search:"
  jq '{state,track_counts,results,failures}' "$OUT/phrase-result.json" || true
  exit 1
fi
echo "REAL-BITMAGNET: pinned native FTS AND=$(jq -r '.results|length' "$OUT/and-result.json") quote-adjacent=$(jq -r '.results|length' "$OUT/phrase-result.json") PASS"

# Real public-source research lookup: Barsoom is a source-catalog setting,
# while "A Princess of Mars" is the actual title of Gutenberg eBook #62.
# The expansion is explicitly source-candidate, not an accepted CPE mapping.
# This deliberately uses Bitmagnet's actual indexed catalog-derived metadata.
cat >"$OUT/gutenberg-research-query.json" <<'QUERY'
{"query":"Barsoom","expansions":[{"term":"A Princess of Mars","provenance":"public-gutenberg-source-candidate:gutenberg:62"}]}
QUERY
ANCHOR_HASH="$(jq -r '.items[] | select(.gutenberg_ebook_id == 62) | .artificial_infohash_for_disposable_test_only' "$OUT/public-catalog-manifest.json")"
test -n "$ANCHOR_HASH"
found=0
for i in $(seq 1 120); do
  curl --noproxy '*' -sS --max-time 25 -u "$AUTH" \
    -H 'Content-Type: application/json' --data-binary @"$OUT/gutenberg-research-query.json" \
    http://127.0.0.1:3086/api/v1/query/execute -o "$OUT/public-cpe-real-retrieval.json" || true
  if jq -e --arg hash "$ANCHOR_HASH" '
    .state == "complete" and .read_only == true and
    (.track_counts.concept // 0) >= 1 and
    (any(.results[]; .info_hash == $hash and (.tracks | index("concept") != null)))
  ' "$OUT/public-cpe-real-retrieval.json" >/dev/null 2>&1; then
    found=1
    break
  fi
  sleep 2
done
if [[ "$found" != "1" ]]; then
  echo "::error::Official catalog A Princess of Mars metadata did not materialize in real native Bitmagnet FTS"
  jq '{state,track_counts,results,failures,coverage}' "$OUT/public-cpe-real-retrieval.json" || true
  exit 1
fi
echo "PUBLIC-CPE: real Bitmagnet indexed official bulk eBook #62; Barsoom source-candidate expansion retrieved matching catalog anchor, authority remains research-only"

# Native, real indexed torrent identity: prove that the actual Bitmagnet
# GraphQL→Explorer FTS returns a true publisher torrent infohash. Do not use
# the Gutenberg catalog's deliberately synthetic identifiers for this check.
UBUNTU_HASH="$(jq -r '[.entries[] | select(.source == "ubuntu:official")][0].infohash' "$RDTE_DIR/manifest.json")"
if [[ -z "$UBUNTU_HASH" || "$UBUNTU_HASH" == "null" ]]; then
  echo "::error::No real Ubuntu infohash in provenance manifest"
  exit 1
fi
authentic=0
for i in $(seq 1 90); do
  curl --noproxy '*' -sS --max-time 20 -u "$AUTH" \
    -H 'Content-Type: application/json' \
    --data '{"query":"ubuntu"}' \
    http://127.0.0.1:3086/api/v1/query/execute -o "$OUT/ubuntu-real-query.json" || true
  if jq -e --arg hash "$UBUNTU_HASH" '
    .state == "complete" and .read_only == true and
    (any(.results[]; .info_hash == $hash and (.tracks | index("direct") != null)))
  ' "$OUT/ubuntu-real-query.json" >/dev/null 2>&1; then
    authentic=1
    break
  fi
  sleep 2
done
if [[ "$authentic" != "1" ]]; then
  echo "::error::Real publisher torrent infohash not returned through actual Bitmagnet FTS"
  jq '{state,track_counts,results,failures}' "$OUT/ubuntu-real-query.json" || true
  exit 1
fi
echo "CPE-RDTE-REAL: authentic bencoded publisher infohash=$UBUNTU_HASH successfully returned from real Bitmagnet index and Explorer"

# Independently prove a second publisher family makes it through actual FTS and Explorer.
DEBIAN_HASH="$(jq -r '[.entries[] | select(.source == "debian:official-mirror")][0].infohash' "$RDTE_DIR/manifest.json")"
if [[ -z "$DEBIAN_HASH" || "$DEBIAN_HASH" == "null" ]]; then
  echo "::error::No authentic Debian hash in provenance manifest"
  exit 1
fi
debian_authentic=0
for i in $(seq 1 60); do
  curl --noproxy '*' -sS --max-time 20 -u "$AUTH" \
    -H 'Content-Type: application/json' \
    --data '{"query":"debian"}' \
    http://127.0.0.1:3086/api/v1/query/execute -o "$OUT/debian-real-query.json" || true
  if jq -e --arg hash "$DEBIAN_HASH" '
    .state == "complete" and .read_only == true and
    (any(.results[]; .info_hash == $hash and (.tracks | index("direct") != null)))
  ' "$OUT/debian-real-query.json" >/dev/null 2>&1; then
    debian_authentic=1
    break
  fi
  sleep 2
done
if [[ "$debian_authentic" != "1" ]]; then
  echo "::error::Debian publisher infohash absent from actual Bitmagnet FTS and Explorer"
  jq '{state,track_counts,results,failures}' "$OUT/debian-real-query.json" || true
  exit 1
fi
echo "CPE-RDTE-REAL: authentic Debian publisher infohash=$DEBIAN_HASH returned from real Bitmagnet index and Explorer"


# Separate CLI process uses actual authenticated HTTP service and real upstream.
BCE_AUTH_USER=public-rep BCE_AUTH_PASSWORD=synthetic-only-credential-2026 \
  "$OUT/bce-cli" -server http://127.0.0.1:3086 -operation execute \
  -query Barsoom -expand Mars >"$OUT/cli.json"
jq -e '.state == "complete" and .read_only == true and (.track_counts.direct // 0) >= 1 and (.track_counts.concept // 0) >= 1' "$OUT/cli.json" >/dev/null
echo "REAL-BITMAGNET: actual separate Explorer CLI real-query PASS"

if [[ -n "$GITHUB_STEP_SUMMARY" ]]; then
  {
    echo "### Real Bitmagnet + Explorer disposable cloud rep"
    echo "- **Result:** PASS (only after all checks above)"
    echo "- **Explorer commit:** $(git rev-parse HEAD)"
    echo "- **Bitmagnet real GraphQL version:** $(jq -r '.data.version' "$OUT/version.json")"
    echo "- **Bitmagnet OCI pulled identity:** \`$BITMAGNET_IDENTITY\`"
    echo "- **Postgres OCI pulled identity:** \`$POSTGRES_IDENTITY\`"
    echo "- **Seed:** 5 invented negative/overlap cases + $CATALOG_ROWS real public Gutenberg catalog-derived metadata records (only artificial torrent hashes)"
    echo "- **Public Gutenberg bulk SHA-256:** \`$(jq -r '.catalog_gzip_sha256' "$OUT/public-catalog-manifest.json")\`"
    echo "- **Public CPE research graph SHA-256:** \`$GRAPH_SHA\` (10 types, not CPE-authorized)"
    echo "- **Native source-candidate retrieval:** Gutenberg #62 anchor indexed and found using Barsoom research expansion"
    echo "- **Actual publisher .torrent files:** $REAL_TORRENT_ROWS retained by SHA-256 in Actions evidence; original exact v1 infohash verified through Bitmagnet FTS"
    echo "- **Torrent payload / P2P:** none; meta files only, no DHT, tracker, webseed or peer connections"
    echo "- **BCE tracks:** direct $(jq -r '.track_counts.direct' "$OUT/execute.json"), concept $(jq -r '.track_counts.concept' "$OUT/execute.json")"
    echo "- **Boundaries:** no DHT, public internet ingress, TMDB, CPE accepted facts, images published, private corpus or site deployment"
    echo "- **Cleanup:** both containers and runner-local service removed on job exit"
  } >>"$GITHUB_STEP_SUMMARY"
fi
echo "REAL-BITMAGNET: PASS full real ephemeral PostgreSQL/Bitmagnet/Explorer/CLI stack"

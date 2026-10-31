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

# Compile and start the ACTUAL Explorer service and standalone automation CLI.
# No service ports are reachable from outside this ephemeral runner.
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
export CGO_ENABLED=0
go build -trimpath -o "$OUT/bce" .
go build -trimpath -o "$OUT/bce-cli" ./cmd/bce
BCE_AUTH_USER=public-rep BCE_AUTH_PASSWORD=synthetic-only-credential-2026 \
BCE_BITMAGNET_GRAPHQL=http://127.0.0.1:3333/graphql \
BCE_SEARXNG_URL= BCE_CPE_GRAPH_FILE= BCE_CPE_GRAPH_SHA256= \
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
    echo "- **Seed:** three original synthetic metadata records via real upstream importer"
    echo "- **BCE tracks:** direct $(jq -r '.track_counts.direct' "$OUT/execute.json"), concept $(jq -r '.track_counts.concept' "$OUT/execute.json")"
    echo "- **Boundaries:** no DHT, public internet ingress, TMDB, CPE accepted facts, images published, private corpus or site deployment"
    echo "- **Cleanup:** both containers and runner-local service removed on job exit"
  } >>"$GITHUB_STEP_SUMMARY"
fi
echo "REAL-BITMAGNET: PASS full real ephemeral PostgreSQL/Bitmagnet/Explorer/CLI stack"

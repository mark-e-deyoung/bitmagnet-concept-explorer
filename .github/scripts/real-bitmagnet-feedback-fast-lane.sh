#!/usr/bin/env bash
# Public/free GitHub runner only: REAL PostgreSQL + pinned Bitmagnet v0.10.0
# + ACTUAL Go Explorer, using five fictional importer rows. No external corpus,
# personal Jellyfin/TMDb, DHT, swarm, CPE write, model or provider credentials.
set -Eeuo pipefail
test -n "$RUNNER_TEMP" && test -n "$BCE_EXPECTED_SOURCE_SHA"
test "$(git rev-parse HEAD)" = "$BCE_EXPECTED_SOURCE_SHA"
OUT="$(mktemp -d "$RUNNER_TEMP/bce-feedback-live-XXXXXX")"
TOKEN="$(basename "$OUT" | tr '[:upper:]' '[:lower:]' | tr -cd '[:alnum:]')"
DB="bce-feedback-db-$TOKEN"
BM="bce-feedback-bm-$TOKEN"
NET="bce-feedback-net-$TOKEN"
PID=""
URL="http://127.0.0.1:3333"
BCE="http://127.0.0.1:3086"
AUTH="fixture-auditor:fixture-password-only-public"
POSTGRES_IMAGE="postgres@sha256:0cefaabd882e42bcb2a2f3dc56d3ce4770560ba62fd7f197f1a2e008fd5731ab"
BITMAGNET_IMAGE="ghcr.io/bitmagnet-io/bitmagnet@sha256:cf2c16fac5b51f1c18e630b4710175d35d7e44b83012cfd05a2b43712feec05e"
export NO_PROXY="127.0.0.1,localhost" no_proxy="127.0.0.1,localhost"
cleanup() {
  local rc=$?
  trap - EXIT
  if [[ -n "$PID" ]]; then kill "$PID" >/dev/null 2>&1 || true; wait "$PID" >/dev/null 2>&1 || true; fi
  if (( rc != 0 )); then
    echo "::error::REAL_FEEDBACK_FIRST_FAILURE: rc=$rc (only synthetic local data)"
    tail -n 45 "$OUT/bce.log" 2>/dev/null || true
    docker logs --tail 65 "$BM" 2>&1 || true
    docker logs --tail 35 "$DB" 2>&1 || true
  fi
  docker rm -f "$BM" "$DB" >/dev/null 2>&1 || true
  docker network rm "$NET" >/dev/null 2>&1 || true
  rm -rf "$OUT"
  exit "$rc"
}
trap cleanup EXIT
echo "REAL_FEEDBACK: source=$(git rev-parse HEAD)"
echo "REAL_FEEDBACK: go=$(go version)"
docker pull --quiet "$POSTGRES_IMAGE" >/dev/null
docker pull --quiet "$BITMAGNET_IMAGE" >/dev/null
docker network create "$NET" >/dev/null
docker run -d --name "$DB" --network "$NET" \
  -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=public-fixture-only \
  -e POSTGRES_DB=bitmagnet --shm-size 512m "$POSTGRES_IMAGE" >/dev/null
ready=0
for i in $(seq 1 45); do
  if docker exec "$DB" pg_isready -U postgres -d bitmagnet >/dev/null 2>&1; then ready=1;break;fi
  sleep 2
done
test "$ready" = 1 || { echo "Postgres readiness failure" >&2;exit 1; }
echo "REAL_FEEDBACK: pinned PostgreSQL ready"
docker run -d --name "$BM" --network "$NET" -p 127.0.0.1:3333:3333 \
  -e POSTGRES_HOST="$DB" -e POSTGRES_PASSWORD=public-fixture-only \
  -e POSTGRES_DB=bitmagnet -e TMDB_ENABLED=false \
  "$BITMAGNET_IMAGE" worker run --keys=http_server --keys=queue_server >/dev/null
ready=0
for i in $(seq 1 55); do
  if curl --noproxy '*' -fsS --max-time 3 -H 'Content-Type: application/json' \
    --data '{"query":"query { version }"}' "$URL/graphql" -o "$OUT/version.json" 2>/dev/null \
    && jq -e '.data.version=="v0.10.0"' "$OUT/version.json" >/dev/null; then ready=1;break;fi
  sleep 2
done
test "$ready" = 1 || { echo "Pinned Bitmagnet v0.10.0 not ready" >&2;exit 1; }
echo "REAL_FEEDBACK: Bitmagnet v0.10.0 GraphQL ready"
curl --noproxy '*' -fsS --max-time 45 -H 'Content-Type: application/json' \
  --data-binary @fixtures/public-bitmagnet-rep.ndjson "$URL/import" -o "$OUT/import.txt"
grep -q "5 items imported" "$OUT/import.txt"
grep -q "import complete" "$OUT/import.txt"
echo "REAL_FEEDBACK: five fictional records imported by real upstream /import"
if [[ "${BCE_RDTE_AUTHENTIC_PUBLISHER:-0}" == "1" ]]; then
  # The publisher-only collector is independently qualified at pinned #81
  # first-source+frozen replay (GHA 45429886057 and 45430385981).
  # No ISO/torrent payload or P2P access. Keep original public evidence
  # outside OUT so the container/service cleanup cannot erase it.
  RDTE_DIR="$RUNNER_TEMP/bce-real-publisher-feedback-evidence"
  mkdir -p "$RDTE_DIR"
  echo "REAL_PUBLISHER_CHAIN: collecting original pinned publisher metainfo"
  GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly \
    timeout --signal=TERM --kill-after=10s 360s \
      go run ./cmd/rdte-torrent-collector -out "$RDTE_DIR" -minimum 5
  jq -e '
    .schema_version=="cpe-rdte-real-metainfo-v1" and
    .authority_status=="research_source_observations_not_accepted_CPE" and
    .no_payload_download==true and .no_swarm_or_dht==true and
    (.entries | length)>=5 and
    ([.entries[] | select(.source=="ubuntu:official")]|length)>=2 and
    ([.entries[] | select(.source=="arch:official")]|length)==1 and
    ([.entries[] | select(.source=="fedora:official")]|length)==1 and
    ([.entries[] | select(.source=="debian:official-mirror")]|length)==1 and
    all(.entries[]; .p2p_used==false and .payload_downloaded==false and
      .rights_status=="source_claim_not_independently_qualified")
  ' "$RDTE_DIR/manifest.json" >/dev/null
  PUBLISHER_COUNT="$(jq -r '.entries|length' "$RDTE_DIR/manifest.json")"
  DEBIAN_PUBLISHER_HASH="$(jq -r '[.entries[] | select(.source=="debian:official-mirror")][0].infohash' "$RDTE_DIR/manifest.json")"
  test "$PUBLISHER_COUNT" -ge 5
  test "$DEBIAN_PUBLISHER_HASH" != "null"
  curl --noproxy '*' -fsS --max-time 45 -H 'Content-Type: application/json' \
    --data-binary @"$RDTE_DIR/bitmagnet-import.ndjson" "$URL/import" -o "$OUT/publisher-import.txt"
  grep -Fq "$PUBLISHER_COUNT items imported" "$OUT/publisher-import.txt"
  grep -Fq "import complete" "$OUT/publisher-import.txt"
  echo "REAL_PUBLISHER_CHAIN: original_publisher_records_imported=$PUBLISHER_COUNT"
fi
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly CGO_ENABLED=0
go build -trimpath -o "$OUT/bce" .
BCE_AUTH_USER=fixture-auditor BCE_AUTH_PASSWORD=fixture-password-only-public \
BCE_BITMAGNET_GRAPHQL="$URL/graphql" BCE_SEARXNG_URL= \
BCE_DISCOVERY_HMAC_KEY="fixture-only-evidence-hmac-key-1234567890" \
  "$OUT/bce" -listen 127.0.0.1:3086 >"$OUT/bce.log" 2>&1 &
PID="$!"
ready=0
for i in $(seq 1 30); do
  if curl --noproxy '*' -fsS --max-time 2 "$BCE/healthz" >/dev/null 2>&1;then ready=1;break;fi
  sleep 1
done
test "$ready" = 1 || { echo "BCE service health failure" >&2;exit 1; }
status="$(curl --noproxy '*' -sS --max-time 5 -o /dev/null -w '%{http_code}' \
  "$BCE/api/v1/discovery/feedback/preview" -X POST)"
test "$status" = 401 || { echo "Feedback unauthorized status=$status; expected 401" >&2;exit 1; }
cat >"$OUT/hypothesis.json" <<'JSON'
{"query":{"query":"Barsoom","include_web":false},"proposal_kind":"lexicalization_candidate","candidate_label":"fictional-linguistic-hypothesis","language":"en"}
JSON
# Real PostgreSQL FTS is asynchronous. Bound this independently of the full
# 25-minute Gutenberg/Internet Archive publisher-metainfo RDTE program.
accepted=0
for i in $(seq 1 70); do
  curl --noproxy '*' -fsS --max-time 8 -u "$AUTH" \
    -H 'Content-Type: application/json' \
    --data-binary @"$OUT/hypothesis.json" \
    "$BCE/api/v1/discovery/feedback/preview" -o "$OUT/proposal.json" || true
  if jq -e '
    .schema_version=="bce-feedback-preview-v1" and
    .state=="local_review_export_only" and
    .observation_count>=1 and
    .proposal.schema_version=="cpe-discovery-proposal-research-v1" and
    .proposal.distinct_ancestry_count==1 and
    .proposal.authority=="not_accepted_not_transmitted" and
    .proposal.state=="local-review-export-only" and
    ((.proposal.evidence|length)>=1) and
    all(.proposal.evidence[]; .source=="bitmagnet" and
      (.evidence_ref|test("^sha256:[a-f0-9]{64}$")) and
      (.ancestry_ref|test("^sha256:[a-f0-9]{64}$")))
  ' "$OUT/proposal.json" >/dev/null 2>&1;then accepted=1;break;fi
  sleep 2
done
if [[ "$accepted" != 1 ]];then
  echo "::error::Actual Bitmagnet observation never produced a valid read-only proposal"
  jq '{state,observation_count,proposal:{schema_version:.proposal.schema_version,authority:.proposal.authority,distinct_ancestry_count:.proposal.distinct_ancestry_count}}' "$OUT/proposal.json" || true
  exit 1
fi
if grep -Eiq 'Barsoom|1111111111111111|2222222222222222|3333333333333333' "$OUT/proposal.json";then
  echo "::error::Raw local query or source infohash leaked in research feedback response"
  exit 1
fi
curl --noproxy '*' -fsS --max-time 9 -u "$AUTH" -H 'Content-Type: application/json' \
  --data-binary @"$OUT/hypothesis.json" \
  "$BCE/api/v1/discovery/feedback/preview" -o "$OUT/proposal-replay.json"
jq -es '.[0].proposal.proposal_id == .[1].proposal.proposal_id' "$OUT/proposal.json" "$OUT/proposal-replay.json" >/dev/null
echo "REAL_FEEDBACK_BITMAGNET_TO_CPE_PREVIEW_PASS: source real; metadata fictional; 1 unqualified ancestry; NO CPE write"
cat >"$OUT/web-refusal.json" <<'JSON'
{"query":{"query":"Barsoom","include_web":true},"proposal_kind":"lexicalization_candidate","candidate_label":"fictional-linguistic-hypothesis","language":"en"}
JSON
status="$(curl --noproxy '*' -sS --max-time 9 -u "$AUTH" \
  -H 'Content-Type: application/json' --data-binary @"$OUT/web-refusal.json" \
  -o "$OUT/refusal.json" -w '%{http_code}' \
  "$BCE/api/v1/discovery/feedback/preview")"
test "$status" = 422
echo "REAL_FEEDBACK_WEB_EGRESS_FAIL_CLOSED_PASS"
cat >"$OUT/tmdb-no-consent.json" <<'JSON'
{"kind":"movie","title":"Fictional Example","providers":["tmdb"],"allow_external_lookup":false}
JSON
curl --noproxy '*' -fsS --max-time 8 -u "$AUTH" \
  -H 'Content-Type: application/json' \
  --data-binary @"$OUT/tmdb-no-consent.json" "$BCE/api/v1/media/lookup" -o "$OUT/media.json"
jq -e '.outcomes[0].state=="requires_explicit_external_consent"' "$OUT/media.json" >/dev/null
echo "REAL_FEEDBACK_NO_EXTERNAL_PROVIDER_CALL_PASS"
if [[ "${BCE_RDTE_AUTHENTIC_PUBLISHER:-0}" == "1" ]]; then
  # Real Bitmagnet FTS is asynchronous. Query a verified publisher infohash,
  # not an invented fixture string, through the ACTUAL compiled Explorer.
  cat >"$OUT/debian-original-query.json" <<'JSON'
{"query":"debian","include_web":false}
JSON
  found=0
  for i in $(seq 1 75); do
    curl --noproxy '*' -sS --max-time 10 -u "$AUTH" \
      -H 'Content-Type: application/json' \
      --data-binary @"$OUT/debian-original-query.json" \
      "$BCE/api/v1/query/execute" -o "$OUT/publisher-execute.json" || true
    if jq -e --arg hash "$DEBIAN_PUBLISHER_HASH" '
      .state=="complete" and .read_only==true and
      any(.results[]; .info_hash==$hash and (.tracks|index("direct")!=null))
    ' "$OUT/publisher-execute.json" >/dev/null 2>&1; then found=1;break;fi
    sleep 2
  done
  if [[ "$found" != 1 ]]; then
    echo "::error::Authentic Debian original publisher infohash absent from pinned Bitmagnet FTS via compiled BCE"
    jq '{state,track_counts,failures,coverage}' "$OUT/publisher-execute.json" || true
    exit 1
  fi
  echo "REAL_PUBLISHER_CHAIN: original Debian infohash returned via Bitmagnet GraphQL->BCE FTS"

  # Agent/automation CLI calls same authenticated Go API with exact source ID.
  go build -trimpath -o "$OUT/bce-cli" ./cmd/bce
  BCE_AUTH_USER=fixture-auditor BCE_AUTH_PASSWORD=fixture-password-only-public \
    "$OUT/bce-cli" -server "$BCE" -operation execute -query debian >"$OUT/publisher-cli.json"
  jq -e --arg hash "$DEBIAN_PUBLISHER_HASH" '
    .state=="complete" and .read_only==true and
    any(.results[]; .info_hash==$hash and (.tracks|index("direct")!=null))
  ' "$OUT/publisher-cli.json" >/dev/null
  echo "REAL_PUBLISHER_CHAIN: automation CLI returned same original publisher identity"

  # Feedback HMACs local source IDs. Bitmagnet evidence shares one source
  # ancestry and CPE remains unmodified, nonnormative and untransmitted.
  cat >"$OUT/publisher-feedback-input.json" <<'JSON'
{"query":{"query":"debian","include_web":false},"proposal_kind":"lexicalization_candidate","candidate_label":"public-publisher-observation-review","language":"en"}
JSON
  curl --noproxy '*' -fsS --max-time 12 -u "$AUTH" \
    -H 'Content-Type: application/json' \
    --data-binary @"$OUT/publisher-feedback-input.json" \
    "$BCE/api/v1/discovery/feedback/preview" -o "$OUT/publisher-feedback.json"
  jq -e '
    .schema_version=="bce-feedback-preview-v1" and
    .state=="local_review_export_only" and .observation_count>=1 and
    .proposal.schema_version=="cpe-discovery-proposal-research-v1" and
    .proposal.distinct_ancestry_count==1 and
    .proposal.authority=="not_accepted_not_transmitted" and
    .proposal.state=="local-review-export-only" and
    all(.proposal.evidence[]; .source=="bitmagnet" and
       (.evidence_ref|test("^sha256:[0-9a-f]{64}$")) and
       (.ancestry_ref|test("^sha256:[0-9a-f]{64}$")))
  ' "$OUT/publisher-feedback.json" >/dev/null
  if grep -Fq "$DEBIAN_PUBLISHER_HASH" "$OUT/publisher-feedback.json";then
    echo "::error::Original publisher infohash escaped into privacy-keyed CPE feedback draft"
    exit 1
  fi
  echo "REAL_PUBLISHER_CHAIN_PASS: 5 pinned original publishers imported, FTS/agent CLI verified, local HMAC research only"
else
  echo "REAL_FEEDBACK_FAST_LANE_PASS: fictional data only; no authentic publisher torrent, private local corpus, CPE adjudication, or native TrueNAS F4 claimed"
fi

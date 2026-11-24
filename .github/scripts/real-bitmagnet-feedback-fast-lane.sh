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
echo "REAL_FEEDBACK_FAST_LANE_PASS: no authentic publisher torrent, private local corpus, CPE adjudication, or native TrueNAS F4 claimed"

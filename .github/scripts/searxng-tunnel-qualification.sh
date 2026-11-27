#!/usr/bin/env bash
# One-shot public integration qualification. No arbitrary queries, no private data.
set -Eeuo pipefail
test "$(git rev-parse HEAD)" = "$EXPECTED_SOURCE"
test "$GITHUB_EVENT_NAME" = "pull_request"
test "$GITHUB_HEAD_REF" = "rdte/searxng-tunnel-qualification-20261127"
OUT="$(mktemp -d "$RUNNER_TEMP/searxng-qual.XXXXXX")"
NAME="searxng-qualification-$GITHUB_RUN_ID"
GW=""
CF=""
cleanup() {
  rc=$?
  trap - EXIT
  if [ -n "$CF" ]; then kill "$CF" 2>/dev/null || true; fi
  if [ -n "$GW" ]; then kill "$GW" 2>/dev/null || true; fi
  docker rm -f "$NAME" >/dev/null 2>&1 || true
  sudo rm -rf -- "$OUT"
  exit "$rc"
}
trap cleanup EXIT
mkdir -p "$OUT/config"
KEY="$(openssl rand -hex 32)"
export RDTE_TOKEN="$(openssl rand -hex 32)"
echo "::add-mask::$KEY"
echo "::add-mask::$RDTE_TOKEN"
cat >"$OUT/config/settings.yml" <<EOF
use_default_settings:
  engines:
    keep_only:
      - wikipedia
      - duckduckgo
      - bing
general:
  debug: false
search:
  default_lang: en
  formats: [json]
  autocomplete: ''
server:
  secret_key: "$KEY"
  limiter: false
  public_instance: false
  image_proxy: false
engines:
  - name: wikipedia
    disabled: false
  - name: duckduckgo
    disabled: false
  - name: bing
    disabled: false
EOF
IMAGE="ghcr.io/searxng/searxng:2026.10.30-1cd13d98d@sha256:f1bbc31c7bef7a7d8dd2b105c8397eb7dea1eeb7eceebdd45c0c29aef38c0c91"
docker pull "$IMAGE"
docker run -d --name "$NAME" --cpus 2 --memory 2g \
  -p 127.0.0.1:18888:8080 -v "$OUT/config:/etc/searxng" "$IMAGE" >/dev/null
ready=0
for i in $(seq 1 40); do
  if curl --noproxy '*' -fsS --max-time 3 http://127.0.0.1:18888/config -o "$OUT/config.json" && jq -e '.engines | type == "array"' "$OUT/config.json" >/dev/null; then ready=1; break; fi
  sleep 2
done
test "$ready" = 1
jq -e '[.engines[] | select(.enabled == true) | .name] | sort == ["bing","duckduckgo","wikipedia"]' "$OUT/config.json" >/dev/null
echo "SearXNG config: three intended engines enabled"
docker inspect --format '{{.Image}}' "$NAME"
python3 .github/scripts/searxng-fixture-gateway.py >"$OUT/gateway.log" 2>&1 &
GW=$!
for i in $(seq 1 10); do
  if curl --noproxy '*' -fsS --max-time 2 -H "X-RDTE-Token: $RDTE_TOKEN" http://127.0.0.1:18889/healthz >/dev/null; then break; fi
  sleep 1
done
curl --noproxy '*' -fsS --max-time 3 -H "X-RDTE-Token: $RDTE_TOKEN" http://127.0.0.1:18889/healthz | jq -e '.fixture_only == true'
CF_VERSION="2026.10.7"
CF_SHA="1f99fb518b83dec2ba8d870a7cc991d82a77e6196295a3d2754ba17ab9f91872"
curl -fsSL --connect-timeout 10 --max-time 75 "https://github.com/cloudflare/cloudflared/releases/download/$CF_VERSION/cloudflared-linux-amd64" -o "$OUT/cloudflared"
echo "$CF_SHA  $OUT/cloudflared" | sha256sum -c -
chmod 700 "$OUT/cloudflared"
"$OUT/cloudflared" tunnel --no-autoupdate --url http://127.0.0.1:18889 >"$OUT/tunnel.log" 2>&1 &
CF=$!
URL=""
for i in $(seq 1 40); do
  kill -0 "$CF"
  URL="$(grep -Eo 'https://[a-z0-9-]+\.trycloudflare\.com' "$OUT/tunnel.log" | head -n1 || true)"
  if [ -n "$URL" ]; then break; fi
  sleep 2
done
test -n "$URL"
echo "::add-mask::$URL"
ready=0
for i in $(seq 1 12); do
  if curl -fsS --max-time 5 -H "X-RDTE-Token: $RDTE_TOKEN" "$URL/healthz" -o "$OUT/edge-health.json" && jq -e '.fixture_only == true' "$OUT/edge-health.json" >/dev/null; then ready=1; break; fi
  sleep 2
done
test "$ready" = 1
code="$(curl -sS --max-time 8 -w '%{http_code}' -o "$OUT/denied.json" "$URL/healthz")"
test "$code" = 401
jq -e '.error == "fixture_auth_required"' "$OUT/denied.json" >/dev/null
code="$(curl -sS --max-time 8 -H "X-RDTE-Token: $RDTE_TOKEN" -w '%{http_code}' -o "$OUT/denied-path.json" "$URL/search?q=anything")"
test "$code" = 404
jq -e '.error == "fixed_fixture_route_only"' "$OUT/denied-path.json" >/dev/null
echo "Transport PASS: HTTPS, runner-only authentication, arbitrary search route rejected"
pass=1
for lang in en de th; do
  if curl -fsS --max-time 45 -H "X-RDTE-Token: $RDTE_TOKEN" "$URL/probe/$lang" -o "$OUT/$lang.json"; then
    jq -c '.' "$OUT/$lang.json"
    jq -e --arg lang "$lang" '.fixture_only == true and .language_requested == $lang and .result_count > 0' "$OUT/$lang.json" >/dev/null || pass=0
  else
    echo "Fixture $lang failed; no retry"
    pass=0
  fi
done
test "$pass" = 1
echo "Three requested locales returned results. No relevance or native-language accuracy claim."
echo "SearXNG fixture tunnel qualification PASS; teardown now."

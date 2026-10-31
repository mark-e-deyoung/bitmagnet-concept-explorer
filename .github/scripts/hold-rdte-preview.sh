#!/usr/bin/env bash
# Bounded public-fixture-only preview hold AFTER separate URL artifact upload.
# This script only runs on GitHub's disposable public PR runner.
set -Eeuo pipefail
: "$RUNNER_TEMP"
OUT="$RUNNER_TEMP/bce-real-bitmagnet-rep"
test -s "$OUT/preview-pids.txt" && test -s "$OUT/preview-url.txt"
mapfile -t PIDS <"$OUT/preview-pids.txt"
if [[ "${#PIDS[@]}" != 3 ]]; then
  echo "::error::Missing verified disposable preview ownership baton"
  exit 1
fi
TUNNEL_PID="${PIDS[0]}"
PROXY_PID="${PIDS[1]}"
SERVICE_PID="${PIDS[2]}"
URL="$(cat "$OUT/preview-url.txt")"
if [[ ! "$URL" =~ ^https://[a-z0-9-]+\.trycloudflare\.com$ ]]; then
  echo "::error::Malformed published Cloudflare Quick Tunnel URL"
  exit 1
fi
cleanup() {
  local code="$?"
  trap - EXIT
  echo "RDTE-PREVIEW: tearing down bounded tunnel, readonly proxy, Explorer, Bitmagnet and PostgreSQL"
  for pid in "$TUNNEL_PID" "$PROXY_PID" "$SERVICE_PID"; do
    kill "$pid" 2>/dev/null || true
  done
  docker rm -f bce-real-bitmagnet bce-real-postgres >/dev/null 2>&1 || true
  docker network rm bce-real-rep >/dev/null 2>&1 || true
  exit "$code"
}
trap cleanup EXIT
for pid in "$TUNNEL_PID" "$PROXY_PID" "$SERVICE_PID"; do
  if ! kill -0 "$pid" 2>/dev/null; then
    echo "::error::Service died between URL publication and bounded hold"
    exit 1
  fi
done
echo "RDTE-PREVIEW: verified URL artifact is ready and the public-fixture-only UI remains alive"
for i in $(seq 1 35); do
  sleep 30
  for pid in "$TUNNEL_PID" "$PROXY_PID" "$SERVICE_PID"; do
    if ! kill -0 "$pid" 2>/dev/null; then
      echo "::error::Disposable tunnel/preview service died during bounded hold"
      exit 1
    fi
  done
  if [[ $((i % 10)) == 0 ]]; then
    echo "RDTE-PREVIEW: still alive elapsed=$((i * 30))s (URL only in existing single-file artifact)"
  fi
done
echo "RDTE-PREVIEW: normal expiration after bounded viewer window"

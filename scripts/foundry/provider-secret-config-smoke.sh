#!/usr/bin/env bash
# Source render only. Artificial credentials never leave disposable public runner.
set -Eeuo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APPS="$ROOT/truenas-apps"
PIN="77058ecb5b28199f35150912cacb13283a6fb244"
APP="$APPS/ix-dev/community/bitmagnet-concept-explorer"
WORK="$(mktemp -d "$RUNNER_TEMP/bce-provider-config-XXXXXX")"
cleanup(){ local code=$?; trap - EXIT; rm -rf "$WORK"; exit "$code"; }
trap cleanup EXIT
test "$(git -C "$APPS" rev-parse HEAD)" = "$PIN"
test "$(git -C "$ROOT" rev-parse HEAD)" = "$BCE_EXPECTED_SOURCE_SHA"
cp -a "$ROOT/ix-dev/community/bitmagnet-concept-explorer" "$APPS/ix-dev/community/"
python3 -m venv "$WORK/venv"
"$WORK/venv/bin/python" -m pip install -q --disable-pip-version-check 'PyYAML==6.0.2'
export APP WORK
"$WORK/venv/bin/python" - <<'PY'
import os
from pathlib import Path
app=Path(os.environ["APP"])
src=(app/"templates/test_values/loopback-published-values.yaml").read_text()
assert "media_lookup:" not in src
(app/"templates/test_values/provider-positive.yaml").write_text(src+"""
media_lookup:
  jellyfin_base_url: "http://127.0.0.1:8096"
  jellyfin_token: "fixture-jellyfin-token-012345678901"
  tmdb_read_token: "fixture-tmdb-token-012345678901"
  feedback_hmac_key: "fixture-feedback-hmac-key-012345678901234"
""")
(app/"templates/test_values/provider-invalid-pair.yaml").write_text(src+"""
media_lookup:
  jellyfin_base_url: "http://127.0.0.1:8096"
  jellyfin_token: ""
""")
(app/"templates/test_values/provider-invalid-hmac.yaml").write_text(src+"""
media_lookup:
  feedback_hmac_key: "short"
""")
PY
render() {
  (cd "$APPS"; "$WORK/venv/bin/python" .github/scripts/ci.py --app bitmagnet-concept-explorer \
    --train community --test-file "$1" --render-only=true --skip-metadata)
}
render provider-positive.yaml > "$WORK/positive.log" 2>&1
"$WORK/venv/bin/python" - <<'PY'
import os
from pathlib import Path
import yaml
app=Path(os.environ["APP"])
svc=yaml.safe_load((app/"templates/rendered/docker-compose.yaml").read_text())["services"]["bitmagnet-concept-explorer"]
env=svc["environment"]
if isinstance(env,list): env=dict(i.split("=",1) for i in env)
assert env["BCE_JELLYFIN_BASE_URL"]=="http://127.0.0.1:8096"
assert env["BCE_JELLYFIN_TOKEN"]=="fixture-jellyfin-token-012345678901"
assert env["BCE_TMDB_READ_TOKEN"]=="fixture-tmdb-token-012345678901"
assert env["BCE_DISCOVERY_HMAC_KEY"]=="fixture-feedback-hmac-key-012345678901234"
assert svc.get("read_only") is True
assert str(svc.get("user")) in ("65532","65532:65532")
assert all(p.get("host_ip")=="127.0.0.1" for p in svc.get("ports",[]))
print("FOUNDRY_MEDIA_ENV_RENDER_PASS")
PY
if render provider-invalid-pair.yaml > "$WORK/pair.log" 2>&1; then
  echo "FAIL: incomplete Jellyfin credential pair rendered" >&2; exit 1
fi
grep -q "must be configured together" "$WORK/pair.log"
if render provider-invalid-hmac.yaml > "$WORK/hmac.log" 2>&1; then
  echo "FAIL: short feedback HMAC key rendered" >&2; exit 1
fi
grep -q "at least 32" "$WORK/hmac.log"
echo "FOUNDRY_MEDIA_CONFIG_FAIL_CLOSED_PASS"
# Existing no-media fixture remains explicit: no opt-in credentials injected by default.
render loopback-published-values.yaml > "$WORK/default.log" 2>&1
"$WORK/venv/bin/python" - <<'PY'
import os
from pathlib import Path
import yaml
app=Path(os.environ["APP"])
svc=yaml.safe_load((app/"templates/rendered/docker-compose.yaml").read_text())["services"]["bitmagnet-concept-explorer"]
env=svc["environment"]
if isinstance(env,list): env=dict(i.split("=",1) for i in env)
for name in ("BCE_JELLYFIN_BASE_URL","BCE_JELLYFIN_TOKEN","BCE_TMDB_READ_TOKEN","BCE_DISCOVERY_HMAC_KEY"):
    assert name not in env, "unexpected opt-in provider environment"
print("FOUNDRY_MEDIA_DEFAULT_OFF_PASS")
PY

#!/usr/bin/env bash
# Exact-head, zero-cloud, no-APPLY private GARM Go qualification entrypoint.
# This script is not itself runner/job admission or authorization.
set -euo pipefail
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"
action="${BCE_QUALIFY_ACTION:-plan}"
case "$action" in plan|verify) ;; *) echo "INVALID_QUALIFICATION_ACTION" >&2;exit 64;; esac
expected="${BCE_EXPECTED_COMMIT:-}"
if [[ ! "$expected" =~ ^[0-9a-f]{40}$ ]];then echo "MISSING_EXACT_QUALIFICATION_HEAD" >&2;exit 64;fi
actual="$(git rev-parse --verify HEAD)"
if [[ "$actual" != "$expected" ]];then echo "EXACT_HEAD_MISMATCH" >&2;exit 65;fi
if [[ -n "$(git status --porcelain --untracked-files=normal)" ]];then echo "DIRTY_SOURCE_HOLD" >&2;exit 65;fi
if [[ "$action" == plan ]];then
  echo "PLAN_ONLY: source exact, clean, no tests/network or local HIL performed"
  exit 0
fi
for tool in go gofmt timeout;do
 if ! command -v "$tool" >/dev/null 2>&1;then echo "MISSING_QUALIFICATION_TOOL:$tool" >&2;exit 69;fi
done
echo "SOURCE_PIN_PASS"
echo "GO_TOOLCHAIN:$(go version)"
echo "RUN_OFFLINE_GO_BUILD"
timeout --signal=TERM 240s go build ./...
echo "RUN_OFFLINE_GO_BUILD_LINUX_AMD64"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 timeout --signal=TERM 300s go build ./...
echo "RUN_OFFLINE_GO_BUILD_LINUX_ARM64"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 timeout --signal=TERM 300s go build ./...
# Keep all normal tests genuinely offline even if the caller intends one HIL.
# A globally exported HIL flag would otherwise trigger extra HIL requests
# during both ordinary and race test phases.
hil_requested="${BCE_LOCAL_BITMAGNET_HIL:-0}"
export BCE_LOCAL_BITMAGNET_HIL=0
echo "RUN_OFFLINE_GO_TESTS"
timeout --signal=TERM 240s go test -count=1 -timeout=150s ./...
echo "RUN_OFFLINE_RACE_TESTS"
timeout --signal=TERM 360s go test -race -count=1 -timeout=240s ./...
echo "RUN_OFFLINE_GO_VET"
timeout --signal=TERM 180s go vet ./...
unformatted="$(find . -name '*.go' -type f -print0 | xargs -0 -r gofmt -l)"
if [[ -n "$unformatted" ]];then
 echo "GOFMT_HOLD: source requires formatting; no mutation performed" >&2
 exit 65
fi
echo "OFFLINE_GO_QUALIFICATION_PASS"
if [[ "$hil_requested" == 1 ]];then
 if [[ -z "${BCE_BITMAGNET_GRAPHQL:-}" ]];then echo "HIL_ENDPOINT_NOT_CONFIGURED" >&2;exit 64;fi
 echo "RUN_ONE_BITMAGNET_VERSION_ONLY_HIL"
 BCE_LOCAL_BITMAGNET_HIL=1 timeout --signal=TERM 15s go test -count=1 -run '^TestLocalBitmagnetVersionHIL$' -v .
 echo "BITMAGNET_VERSION_ONLY_HIL_PASS"
else
 echo "BITMAGNET_HIL_NOT_REQUESTED"
fi
echo "SOURCE_QUALIFICATION_COMPLETE_NOT_A_MERGE_OR_DEPLOY_AUTHORITY"

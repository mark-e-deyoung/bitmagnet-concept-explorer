#!/usr/bin/env bash
set -euo pipefail

mkdir -p build/ios-simulator

echo "==> Inspecting available iOS simulators"
xcrun simctl list devices available | grep -E "iPhone|iPad" || true

# Find the first available iPhone simulator UUID
DEVICE_UUID=$(xcrun simctl list devices available | grep -E "iPhone" | head -n1 | sed -E 's/.*\(([A-F0-9-]+)\).*/\1/' || true)

if [ -z "$DEVICE_UUID" ]; then
  echo "==> No pre-created iPhone simulator found; querying available runtimes..."
  RUNTIME=$(xcrun simctl list runtimes | grep -E "iOS" | tail -n1 | awk '{print $NF}' || true)
  DEVICE_TYPE="com.apple.CoreSimulator.SimDeviceType.iPhone-15"
  echo "==> Creating iOS Simulator: iPhone 15 with runtime $RUNTIME"
  DEVICE_UUID=$(xcrun simctl create "BCE-Qualification-iPhone" "$DEVICE_TYPE" "$RUNTIME")
fi

echo "==> Selected iOS Simulator Device UUID: $DEVICE_UUID"

# Boot the simulator in headless mode
echo "==> Booting iOS simulator in headless mode..."
xcrun simctl boot "$DEVICE_UUID"

# Ensure cleanup on exit
cleanup() {
  echo "==> Cleaning up background services and simulator..."
  if [ -n "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" 2>/dev/null || true
  fi
  xcrun simctl shutdown "$DEVICE_UUID" 2>/dev/null || true
}
trap cleanup EXIT

# Wait for simulator to finish booting
xcrun simctl bootstatus "$DEVICE_UUID" -b

echo "==> Simulator booted successfully"

# Start the UX fixture server
echo "==> Starting local UX fixture server on port 4175..."
UX_FIXTURE_PORT=4175 node scripts/ux/fixture-server.mjs > build/ios-simulator/fixture-server.log 2>&1 &
SERVER_PID=$!

# Wait for fixture server health check
echo "==> Awaiting fixture server health check..."
HEALTHY=0
for i in $(seq 1 40); do
  if curl --noproxy '*' -fsS http://127.0.0.1:4175/healthz >/dev/null 2>&1; then
    HEALTHY=1
    echo "==> Fixture server is healthy on 127.0.0.1:4175 (attempt $i)"
    break
  fi
  sleep 1
done

if [ "$HEALTHY" -ne 1 ]; then
  echo "==> ERROR: Fixture server failed to start within timeout"
  cat build/ios-simulator/fixture-server.log || true
  exit 1
fi

# Launch Mobile Safari in the iOS simulator pointing to our fixture server
echo "==> Launching Mobile Safari in iOS Simulator..."
xcrun simctl openurl "$DEVICE_UUID" "http://127.0.0.1:4175/"

# Wait for Mobile Safari to load page, execute JS, and complete initial DOM mount
sleep 8

# Capture framebuffer screenshot
echo "==> Capturing screenshot from iOS Simulator framebuffer..."
SCREENSHOT_PATH="build/ios-simulator/ios-safari-concept-explorer.png"
xcrun simctl io "$DEVICE_UUID" screenshot "$SCREENSHOT_PATH"

if [ -s "$SCREENSHOT_PATH" ]; then
  SIZE=$(wc -c < "$SCREENSHOT_PATH" | tr -d ' ')
  echo "==> SUCCESS: iOS Simulator Mobile Safari screenshot captured ($SIZE bytes at $SCREENSHOT_PATH)"
else
  echo "==> ERROR: Screenshot was not generated or is empty"
  exit 1
fi

# Write verification receipt
cat <<EOF > build/ios-simulator/receipt.json
{
  "device_uuid": "$DEVICE_UUID",
  "platform": "iOS Simulator",
  "browser": "Mobile Safari / WebKit",
  "url": "http://127.0.0.1:4175/",
  "captured_at": "$(date -u +'%Y-%m-%dT%H:%M:%SZ')",
  "screenshot_file": "ios-safari-concept-explorer.png",
  "screenshot_bytes": $SIZE,
  "status": "QUALIFIED"
}
EOF

echo "==> iOS Simulator qualification completed successfully."

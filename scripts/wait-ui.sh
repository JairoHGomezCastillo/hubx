#!/usr/bin/env bash
# Waits until the MFE serves its page and proxies /api/health to the gateway. Usage: wait-ui.sh [ui_url] [seconds]
set -euo pipefail

ui="${1:-http://localhost:3000}"
timeout="${2:-90}"

for ((i = 0; i < timeout; i++)); do
  if curl -sf "$ui/" >/dev/null && curl -sf "$ui/api/health" >/dev/null; then
    echo "ui ready: $ui (${i}s)"
    exit 0
  fi
  sleep 1
done
echo "ui not ready after ${timeout}s: $ui" >&2
exit 1

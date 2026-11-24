#!/usr/bin/env bash
# Free-tier services sleep: poll until the UI answers and the API answers through the gateway (and the UI's proxy).
# Usage: warm-up.sh <gateway-url> <ui-url> [timeout-seconds]
set -euo pipefail
gw="${1:?gateway url}" ui="${2:?ui url}" timeout="${3:-240}"
for ((t = 0; t < timeout; t += 10)); do
  if curl -sf -m 30 "$gw/api/health" >/dev/null && curl -sf -m 30 "$ui/" >/dev/null \
     && curl -sf -m 30 "$ui/api/health" >/dev/null; then
    echo "warm: $gw and $ui (${t}s)"; exit 0
  fi
  sleep 10
done
echo "::error::services not answering after ${timeout}s: $gw / $ui"
exit 1

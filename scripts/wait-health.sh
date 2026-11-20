#!/usr/bin/env bash
# Waits until the gateway answers /health and proxies /api/health. Usage: wait-health.sh [base_url] [seconds]
set -euo pipefail

base="${1:-http://localhost:8080}"
timeout="${2:-90}"

for ((i = 0; i < timeout; i++)); do
  if curl -sf "$base/health" >/dev/null && curl -sf "$base/api/health" >/dev/null; then
    echo "healthy: $base (${i}s)"
    exit 0
  fi
  sleep 1
done

echo "timed out after ${timeout}s waiting for $base/health and $base/api/health" >&2
exit 1

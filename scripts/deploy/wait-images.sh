#!/usr/bin/env bash
# Waits until the images of a commit exist in GHCR (built by .github/workflows/images.yml).
# Usage: wait-images.sh <image-prefix> <sha> [timeout-seconds], e.g. ghcr.io/owner/hubx abc123 -> hubx-api:abc123...
set -euo pipefail
prefix="${1:?image prefix}" sha="${2:?sha}" timeout="${3:-1500}"
for ((t = 0; t < timeout; t += 20)); do
  missing=()
  for s in api gw mfe; do
    docker manifest inspect "$prefix-$s:$sha" >/dev/null 2>&1 || missing+=("$s")
  done
  if [ "${#missing[@]}" -eq 0 ]; then echo "images ready for $sha (${t}s)"; exit 0; fi
  echo "waiting for images: ${missing[*]} (${t}s)"
  sleep 20
done
echo "::error::images for $sha not found after ${timeout}s; did the images workflow run for that commit?"
exit 1

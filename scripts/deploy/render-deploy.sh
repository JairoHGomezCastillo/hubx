#!/usr/bin/env bash
# Deploys one image to one image-backed Render service and waits until it is live.
# Usage: render-deploy.sh <service-id> <image-url> [timeout-seconds]. Needs RENDER_API_KEY.
# API: POST /v1/services/{id}/deploys {"imageUrl": ...} -> 201 deploy (id) or 202 queued (no body);
#      GET /v1/services/{id}/deploys/{deployId} -> status (live, *_failed, canceled, deactivated, or in progress).
set -euo pipefail
service="${1:?service id}" image="${2:?image url}" timeout="${3:-900}"
api="https://api.render.com/v1/services/$service/deploys"
auth=(-H "Authorization: Bearer ${RENDER_API_KEY:?RENDER_API_KEY is required}" -H "Accept: application/json")

body=$(mktemp); trap 'rm -f "$body"' EXIT
code=$(curl -sS -o "$body" -w '%{http_code}' -X POST "${auth[@]}" -H "Content-Type: application/json" \
  -d "{\"imageUrl\":\"$image\"}" "$api")
case "$code" in
  201) deploy=$(jq -r '.id' "$body") ;;
  202) sleep 5; deploy=$(curl -sS "${auth[@]}" "$api?limit=1" | jq -r '.[0].deploy.id') ;;
  *) echo "::error::Render deploy of $service failed: HTTP $code $(head -c 300 "$body")"; exit 1 ;;
esac
echo "deploy $deploy of $service with $image"

for ((t = 0; t < timeout; t += 10)); do
  status=$(curl -sS "${auth[@]}" "$api/$deploy" | jq -r '.status')
  case "$status" in
    live) echo "$service is live (${t}s)"; exit 0 ;;
    build_failed|update_failed|pre_deploy_failed|canceled|deactivated)
      echo "::error::$service deploy $deploy ended as $status"; exit 1 ;;
  esac
  sleep 10
done
echo "::error::$service deploy $deploy not live after ${timeout}s (last status: $status)"
exit 1

#!/usr/bin/env bash
# Integration run on a clean, isolated stack: PostgreSQL + migrations + API + gateway + MFE, then hubx-bender
# (Cucumber: API scenarios through the gateway, @e2e scenarios in a browser through the MFE), then everything is
# removed (containers and the database volume). Exit code = bender's.
set -euo pipefail
cd "$(dirname "$0")/.."

project=hubx-it
compose=(docker compose -p "$project")

ui_port="${MFE_PORT:-3000}"
if curl -s -o /dev/null http://localhost:8080/health || curl -s -o /dev/null "http://localhost:$ui_port/"; then
  echo "port 8080 or $ui_port is busy (is 'make up' running?). Run 'make down' first." >&2
  exit 2
fi

bash scripts/env.sh

cleanup() {
  status=$?
  if [ "$status" -ne 0 ]; then
    "${compose[@]}" logs --no-color --tail=80 migrate api gw mfe >&2 || true
  fi
  "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  exit "$status"
}
trap cleanup EXIT

"${compose[@]}" up -d --build --quiet-pull
bash scripts/wait-health.sh http://localhost:8080 90

bash scripts/wait-ui.sh "http://localhost:$ui_port" 90

[ -d hubx-bender/node_modules ] || npm --prefix hubx-bender ci --no-audit --no-fund
npm --prefix hubx-bender run browsers >/dev/null

BENDER_BASE_URL=http://localhost:8080 BENDER_UI_URL="http://localhost:$ui_port" npm --prefix hubx-bender run test

#!/usr/bin/env bash
# Integration run on a clean, isolated stack: PostgreSQL + migrations + API + gateway, then hubx-bender
# through the gateway, then everything is removed (containers and the database volume). Exit code = bender's.
set -euo pipefail
cd "$(dirname "$0")/.."

project=hubx-it
compose=(docker compose -p "$project")

if curl -s -o /dev/null http://localhost:8080/health; then
  echo "port 8080 is busy (is 'make up' running?). Run 'make down' first." >&2
  exit 2
fi

bash scripts/env.sh

cleanup() {
  status=$?
  if [ "$status" -ne 0 ]; then
    "${compose[@]}" logs --no-color --tail=80 migrate api gw >&2 || true
  fi
  "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  exit "$status"
}
trap cleanup EXIT

"${compose[@]}" up -d --build --quiet-pull
bash scripts/wait-health.sh http://localhost:8080 90

[ -d hubx-bender/node_modules ] || npm --prefix hubx-bender ci --no-audit --no-fund

# Load .env to pass JWT_SECRET to bender tests
set -a
source .env
set +a

BENDER_BASE_URL=http://localhost:8080 JWT_SECRET="$JWT_SECRET" npm --prefix hubx-bender run test

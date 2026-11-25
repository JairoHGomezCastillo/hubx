#!/usr/bin/env bash
# Integration run on a clean, isolated stack: PostgreSQL + migrations + API + gateway + MFE, then hubx-bender
# (Cucumber: API scenarios through the gateway, @e2e scenarios in a browser through the MFE), then everything is
# removed (containers and the database volume). Exit code = non-zero if either Cucumber pass fails.
# Flow: (1) run Cucumber excluding @empty-db; (2) truncate shipments and updates tables; (3) run Cucumber with @empty-db.
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

exit_code=0
cleanup() {
  local status=$?
  # Exit non-zero if startup failed (status != 0) OR any Cucumber pass failed (exit_code != 0)
  if [ "$status" -ne 0 ] || [ "$exit_code" -ne 0 ]; then
    # Print compose logs whenever the run failed
    "${compose[@]}" logs --no-color --tail=80 migrate api gw mfe >&2 || true
    exit_code=$((status != 0 ? status : exit_code))
  fi
  "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
  exit "$exit_code"
}
trap cleanup EXIT

"${compose[@]}" up -d --build --quiet-pull || exit $?
bash scripts/wait-health.sh http://localhost:8080 90 || exit $?

bash scripts/wait-ui.sh "http://localhost:$ui_port" 90 || exit $?

[ -d hubx-bender/node_modules ] || npm --prefix hubx-bender ci --no-audit --no-fund || exit $?
npm --prefix hubx-bender run browsers >/dev/null || exit $?

# Load .env to pass JWT_SECRET to bender (HU-01 forges an expired token for the session-expiration scenario).
set -a
source .env
set +a

# Pass 1: Run Cucumber excluding @empty-db scenarios.
echo "Running Cucumber pass 1 (excluding @empty-db)..."
if ! BENDER_BASE_URL=http://localhost:8080 BENDER_UI_URL="http://localhost:$ui_port" JWT_SECRET="$JWT_SECRET" \
  npm --prefix hubx-bender run test -- --tags "not @empty-db"; then
  exit_code=1
fi

# Pass 2: Truncate shipments and updates, then run @empty-db scenarios.
echo ""
echo "Truncating shipments and updates tables..."
"${compose[@]}" exec -T postgres psql -U hubx -d hubx -v ON_ERROR_STOP=1 -c "TRUNCATE updates, shipments;" || {
  echo "Failed to truncate tables" >&2
  exit_code=1
}

echo "Running Cucumber pass 2 (@empty-db)..."
if ! BENDER_BASE_URL=http://localhost:8080 BENDER_UI_URL="http://localhost:$ui_port" JWT_SECRET="$JWT_SECRET" \
  npm --prefix hubx-bender run test -- --tags "@empty-db"; then
  exit_code=1
fi

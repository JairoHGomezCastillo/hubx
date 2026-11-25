# Quickstart: View the shipment count by stage (HU-09)

Runnable validation guide. It proves the feature end-to-end through the gateway and the browser. It references the
contract (`contracts/hu09-stage-counts.md`) and the data model instead of repeating them. No implementation code here.

## Prerequisites

- Docker available; run everything from the repository root.
- Demo users from migration `0001_baseline_schema.sql` (operator, ground partner, customer) and the documented demo
  password. HU-01 sign-in, HU-02 create and HU-03 record-update endpoints are integrated (upstream dependency).

## Start the stack

```bash
make up            # PostgreSQL + API + gateway + MFE; migrations run on start
curl -s http://localhost:8080/api/health   # gateway -> API health
```

MFE at http://localhost:3000; every `/api` call goes through the gateway on :8080.

## API validation through the gateway

All calls hit the gateway; it forwards to the API, which authorizes the role.

```bash
# 1. Operator token (HU-01)
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}' | jq -r '.token')

# 2. Count by stage — 200, six stages in journey order, each with its count
curl -s http://localhost:8080/api/shipments/stage-counts -H "Authorization: Bearer $TOKEN" | jq
# Expect: stages[] has 6 entries in order Created, In flight, At destination airport,
# Assigned to partner, Out for delivery, Delivered; each has a numeric count; total = sum.

# 3. A stage with no shipments still appears with count 0 (do not special-case it)
curl -s http://localhost:8080/api/shipments/stage-counts -H "Authorization: Bearer $TOKEN" \
  | jq '.stages[] | select(.stage=="Out for delivery")'
# Expect: { "stage": "Out for delivery", "count": <0 or more> } — present even when 0.

# 4. Role without permission — ground partner gets 403, no totals
PARTNER=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"partner@hubx.co","password":"Hubx-Demo-2026"}' | jq -r '.token')
curl -s -o /dev/null -w 'partner=%{http_code}\n' \
  http://localhost:8080/api/shipments/stage-counts -H "Authorization: Bearer $PARTNER"
# Expect: partner=403

CUSTOMER=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"customer@hubx.co","password":"Hubx-Demo-2026"}' | jq -r '.token')
curl -s -o /dev/null -w 'customer=%{http_code}\n' \
  http://localhost:8080/api/shipments/stage-counts -H "Authorization: Bearer $CUSTOMER"
# Expect: customer=403

# 5. No session — 401
curl -s -o /dev/null -w 'anon=%{http_code}\n' http://localhost:8080/api/shipments/stage-counts
# Expect: anon=401
```

### Checking the counts reflect reality

To reproduce the backlog "Count by stage" scenario (3 in flight, 2 out for delivery, 5 delivered), create shipments
with HU-02 and advance them with HU-03 through the gateway (never by direct DB writes), then re-read the count and
confirm each stage total matches what you recorded, and the remaining stages read 0. On a fresh database with no
shipments, step 2 returns all six stages with count 0 and total 0.

## Browser happy path (@e2e)

1. Open http://localhost:3000 and sign in as the operator.
2. On the dashboard, open "Shipment count by stage".
3. See six rows in journey order (Created → Delivered), each with its stage name and a count; stages with no shipments
   show 0.
4. Sign out, sign in as a ground partner or customer: the count view is not offered, and a direct request is refused.

The `@e2e` scenario in `hubx-bender/features/HU09_stage-counts.feature` automates step 1–3 (the operator happy path).

## Automated acceptance and checks

```bash
make test-integration   # full stack: HU-09 Cucumber API scenarios + the @e2e browser happy path
go -C hubx-api test ./...        # backend unit tests (StageCounter, handler) with gomock
npm --prefix hubx-mfe run test   # MFE unit tests for the view and the service
npm --prefix hubx-mfe run build  # strict TypeScript build
make security                    # gosec + govulncheck, no findings
```

## Expected outcomes (map to Success Criteria)

- SC-001: every operator request returns exactly six stage totals in journey order.
- SC-002: each total matches the shipments in that stage; empty stages read 0.
- SC-003: an empty system returns six zero counts, never an empty result.
- SC-004: ground partner and customer are refused with `403` and receive no totals.
- SC-005: all HU-09 Cucumber scenarios pass, including the `@e2e` operator happy path.

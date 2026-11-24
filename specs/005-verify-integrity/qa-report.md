# QA Report: HU-05 Verify the integrity of the chain

**Story**: HU-05 Verify the integrity of the chain  
**Phase**: red  
**Agent**: qa-seguridad  
**Date**: 2026-11-25  
**Status**: OK

## Summary

**Verdict**: All 8 HU-05 scenarios written and RED for the right reason (missing route `/api/shipments/{code}/integrity`). The tamper helper successfully bypassed the immutability trigger. Regression green: all 43 scenarios from HU-01 through HU-04 passed. No undefined steps. Ready for backend-go and frontend-react implementation.

## Blockers

None.

## Artifacts Created

- ✅ `hubx-bender/features/HU05_verify-integrity.feature` — 7 scenarios (Scenario Outline with 3 examples expanded to 3 scenarios, plus 4 individual scenarios) + 1 `@e2e` happy path
- ✅ `hubx-bender/features/steps/HU05_verify-integrity.steps.ts` — Step definitions reusing `common.steps.ts` helpers
- ✅ `hubx-bender/features/support/tamper.ts` — Tamper helper using Node `spawn` (no shell), docker compose exec psql with shipment code validated and passed as psql variable
- ✅ `specs/005-verify-integrity/qa-report.md` — This report

## Gherkin Coverage

| Scenario | Feature Line | Test Type | Result (red) |
|---|---|---|---|
| Intact chain (operator) | 6-18 | API | ❌ 404 (missing route) |
| Intact chain (customer who owns) | 6-18 | API | ❌ 404 (missing route) |
| Intact chain (assigned partner) | 6-18 | API | ❌ 404 (missing route) |
| Shipment without updates | 20-24 | API | ❌ 404 (missing route) |
| Update improperly altered | 26-31 (tagged @needs-secret @needs-db) | API | ❌ 404 (missing route) |
| Code that matches no shipment | 33-36 | API | ❌ JSON parse error (HTML 404 from gateway) |
| Shipment the participant cannot see | 38-42 | API | ❌ JSON parse error (HTML 404 from gateway) |
| Verify an intact chain in the browser | 44-49 (tagged @e2e) | E2E | ❌ Timeout (missing MFE view) |

**Total**: 8 scenarios (7 from backlog + 1 @e2e), all RED because `GET /api/shipments/{code}/integrity` does not exist yet.

**Regression**: 43 scenarios from HU-01 through HU-04 passed.

## Test Design Notes

### Fixtures
- Fresh shipments created through `POST /api/shipments` (HU-02) via the gateway
- Advanced through `POST /api/shipments/{code}/updates` (HU-03) to reach the needed stage
- "Three updates" = Assigned to partner (In flight → At destination airport → Assigned to partner)
- "No updates" = Created (HU-02 creates shipments in Created with no update)
- Demo users from `memory.md`: operator@hubx.co, rapido@hubx.co (assigned ground partner), ana@hubx.co and andina@hubx.co (customers)
- Password: `Hubx-Demo-2026` (from `DEMO_PASSWORD` in `common.steps.ts`)

### Tamper Scenario
- Tagged `@needs-secret` and `@needs-db` so deployments never run it
- Uses `tamperUpdate(code, sequence, newStage)` from `features/support/tamper.ts`
- Runs `docker compose -p $BENDER_COMPOSE_PROJECT exec -T postgres psql -U hubx -d hubx -v code=<code> -v newStage=<stage> -v ON_ERROR_STOP=1 -f -`
- SQL piped to stdin: `BEGIN; SET LOCAL session_replication_role = replica; UPDATE updates SET stage = :'newStage' WHERE sequence = N AND shipment_id = (SELECT id FROM shipments WHERE code = :'code'); COMMIT;`
- Shipment code validated with `/^HBX-[0-9A-F]{8}$/` before use
- Stage value restricted to known stages (In flight, At destination airport, Assigned to partner, Out for delivery, Delivered)
- Compose project from `BENDER_COMPOSE_PROJECT` env var (defaults to `hubx-it`)
- No new npm dependency (uses Node built-in `node:child_process` `spawn`)
- **Result**: Successfully tampered update 2 to "Out for delivery" (confirmed in test log)

### @e2e Contract
The MFE view expected by the @e2e scenario:
- Route: `/verify-integrity` (linked from Dashboard with text "Verify integrity")
- Input labelled: **"Shipment code"** (case-insensitive)
- Button: **"Verify integrity"** (case-insensitive)
- Result texts when intact:
  - **"The chain is intact"** (case-insensitive, visible text)
  - **"3 updates checked"** (case-insensitive, visible text)

The @e2e step uses `getByRole('link', { name: /verify integrity/i })`, `getByLabel(/shipment code/i)`, `getByRole('button', { name: /verify integrity/i })`, `getByText(/the chain is intact/i)` and `getByText(/3 updates checked/i)`.

### Step Reuse
- `"<participant> has signed in"` defined in `common.steps.ts` (shared with HU-01, HU-02, HU-03)
- Phrasing now matches both "the shipment's integrity" and "that shipment's integrity" with regex `(?:the|that)`
- `prepareShipment(world, stage)` creates fixtures through the gateway (HU-02 + HU-03 endpoints)
- `authHeader(world)` builds `Authorization: Bearer <JWT>` header
- No ambiguous step definitions (Cucumber strict mode would fail)

## Evidence

### Integration test (prove RED for the right reason)

**Command**: `make test-integration`  
**Working directory**: `/Users/jaigomez/source/hubx-hu05`  
**Result**: 51 scenarios (43 passed, 8 failed), 329 steps (314 passed, 7 skipped, 8 failed), 1m 9.837s

**HU-05 failures (all expected)**:

```
Failed scenarios:
  1) Intact chain # features/HU05_verify-integrity.feature:15
       Then they are told that the chain is intact # features/steps/HU05_verify-integrity.steps.ts:99
           Error: verification response status
           expect(received).toBe(expected) // Object.is equality
           Expected: 200
           Received: 404

  2) Intact chain # features/HU05_verify-integrity.feature:16
       [same as #1, second example]
           Expected: 200
           Received: 404

  3) Intact chain # features/HU05_verify-integrity.feature:17
       [same as #1, third example]
           Expected: 200
           Received: 404

  4) Shipment without updates # features/HU05_verify-integrity.feature:19
       Then they are told that the chain is intact # features/steps/HU05_verify-integrity.steps.ts:99
           Expected: 200
           Received: 404

  5) Update improperly altered # features/HU05_verify-integrity.feature:27
       Then they are told that the chain is not intact # features/steps/HU05_verify-integrity.steps.ts:113
           Expected: 200
           Received: 404

  6) Code that matches no shipment # features/HU05_verify-integrity.feature:34
       Then they are told that no shipment exists with that code # features/steps/HU05_verify-integrity.steps.ts:122
           SyntaxError: Unexpected non-whitespace character after JSON at position 4 (line 1 column 5)

  7) Shipment the participant cannot see # features/HU05_verify-integrity.feature:39
       Then they are told that no shipment exists with that code # features/steps/HU05_verify-integrity.steps.ts:122
           SyntaxError: Unexpected non-whitespace character after JSON at position 4 (line 1 column 5)

  8) Verify an intact chain in the browser # features/HU05_verify-integrity.feature:46
       When the operator verifies the shipment's integrity in the browser # features/steps/HU05_verify-integrity.steps.ts:82
           Error: function timed out, ensure the promise resolves within 30000 milliseconds
```

**Tamper evidence**:
```
Tampered update 2 of HBX-7E527358 to stage "Out for delivery"
```

**Analysis**:
- Scenarios 1-5 fail with **404** because the route `/api/shipments/{code}/integrity` does not exist yet ✅
- Scenarios 6-7 fail with **JSON parse error** because the gateway returns HTML 404 (not JSON) for unknown routes ✅
- Scenario 8 (@e2e) **times out** because the MFE view `/verify-integrity` does not exist yet ✅
- The tamper step in scenario 5 succeeded: it altered update 2 to "Out for delivery" before the verification request
- **All 43 regression scenarios passed**: HU-01 through HU-04 remain green

**No undefined steps**: The test run shows no "Undefined scenarios" section and no step snippets to implement (the first run had 1 undefined, fixed by updating the step regex to match "that shipment's integrity").

### Security scan (baseline for verify phase)

Not run in phase `red` (read-only route, no new vulnerability surface). Will run in phase `verify`:
```bash
gosec ./...
govulncheck -C hubx-api ./...
govulncheck -C hubx-gw ./...
```

## OWASP API Security Top 10 Checklist (Planned for Verify)

This checklist will be completed in phase `verify` after implementation. Recorded here for the verify phase:

| OWASP Category | HU-05 Route | Test / Evidence | Result (verify) |
|---|---|---|---|
| **API1: BOLA** | GET /shipments/{code}/integrity | Scenario "Shipment the participant cannot see" (customer who does not own it, ground partner not assigned) → 404 identical to unknown code | To verify |
| API1 (cont.) | | Scenario "Code that matches no shipment" → 404 `SHIPMENT_NOT_FOUND` | To verify |
| API1 (cont.) | | Scenario Outline "Intact chain" with operator (any shipment), customer (own shipments), partner (assigned shipments) | To verify |
| **API2: Broken Authentication** | | JWT required (HU-01's `RequireAuth`); 401 when missing/invalid/expired (contract rule) | Reuses HU-01 |
| **API3: Broken Object Property Level Authorization** | | Response fields: shipmentCode, intact, updatesChecked, firstAlteredUpdate (contract); no internal IDs | To verify |
| **API4: Unrestricted Resource Consumption** | | Read-only, one indexed shipment lookup + one query of its updates (plan R6); no unbounded loop | To verify |
| **API5: Broken Function Level Authorization** | | Same visibility rule as HU-03 (operator, owner customer, assigned partner; plan R2); tested in Scenario Outline | To verify |
| **API6: Unrestricted Access to Sensitive Business Flows** | | Verification is read-only (FR-008); does not record/modify/delete; no rate-limiting yet (pre-existing finding, HU-02 qa-report) | Pre-existing |
| **API7: Server Side Request Forgery (SSRF)** | | No external URL input; shipment code validated format `^HBX-[0-9A-F]{8}$` | Not applicable |
| **API8: Security Misconfiguration** | | Gateway CORS/headers/rate-limit: pre-existing finding from HU-02 qa-report (not implemented yet, constitution VI) | Pre-existing |
| API8 (cont.) | | Secrets from env vars only (constitution III); JWT secret, DB password | Reuses baseline |
| **API9: Improper Inventory Management** | | Route documented in `contracts/hu05-verify-integrity.md`; added to hubx-api handler registry | To verify |
| **API10: Unsafe Consumption of APIs** | | No third-party API consumed | Not applicable |

**Focus for HU-05**: API1 (BOLA) — identical 404 for unknown and invisible shipments, correct visibility enforcement per role.

**Pre-existing findings** (not introduced by HU-05, recorded in HU-02 qa-report):
- API6/API8: Gateway security headers, CORS and rate limiting not implemented yet (constitution VI). Needs its own story.

## Recommendations for Implementation

1. **Backend-go**: 
   - Implement `ledger.Verify([]Update) VerificationResult` reusing the existing `ledger.Fingerprint` from HU-03
   - Add `LedgerRepository.List(ctx, shipmentID) ([]Update, error)` with `ORDER BY sequence ASC`, parameterized SQL
   - Fix HU-03's `Append`: truncate `RecordedAt` to microseconds before computing the fingerprint (plan R6)
   - Implement `shipment.Verifier` reusing HU-03's visibility rules
   - Add `GET /shipments/{code}/integrity` handler with flat 401 body (HU-01 `RequireAuth`)
   - Return identical 404 for unknown and invisible shipments
   - Regenerate mocks: `go -C hubx-api generate ./...`

2. **Frontend-react**:
   - Create `/verify-integrity` route and view
   - Input labelled "Shipment code", button "Verify integrity"
   - Result texts: "The chain is intact" / "The chain is not intact", "N updates checked", "first alteration is in the Nth update"
   - Error: "No shipment exists with that code"
   - Link from Dashboard with "Verify integrity"

3. **Devops** (verify phase):
   - Confirm `scripts/test-integration.sh` exports `BENDER_COMPOSE_PROJECT=hubx-it` to bender
   - Confirm deployments exclude `@needs-secret` scenarios (already in place per memory.md)

## Files Changed

- `hubx-bender/features/HU05_verify-integrity.feature` (new)
- `hubx-bender/features/steps/HU05_verify-integrity.steps.ts` (new, fixed step regex to match "that shipment's")
- `hubx-bender/features/support/tamper.ts` (new, using spawn + stdin for SQL)
- `specs/005-verify-integrity/qa-report.md` (this file)

## UI Contract for Frontend-React

The @e2e scenario asserts these exact UI elements (case-insensitive matching):

**Route**: `/verify-integrity`

**Form elements**:
- Input with label: `"Shipment code"`
- Button: `"Verify integrity"`

**Result texts** (intact chain with N updates):
- `"The chain is intact"`
- `"N updates checked"` (e.g. "3 updates checked")

**Result texts** (not intact, first alteration at position N):
- `"The chain is not intact"`
- `"first alteration is in the Nth update"` (replace N with the actual number, e.g. "2nd" or "second")

**Error text** (unknown/invisible shipment):
- `"No shipment exists with that code"`

**Navigation**:
- Dashboard must have a link with text `"Verify integrity"` that navigates to `/verify-integrity`

Frontend-react should implement the view against this contract, using accessible HTML (labels for inputs, role="status" or visible text for results).

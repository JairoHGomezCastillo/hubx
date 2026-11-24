# QA report: HU-04 View a shipment's traceability

**Story**: HU-04 | **Phase**: verify (post-implementation, final) | **Date**: 2026-11-24 | **Agent**: qa-seguridad
**Branch**: `004-view-traceability` | **HEAD**: `8976cb4`

---

## Verdict

**GREEN — verify accepted.** The full isolated stack (PostgreSQL + API + gateway + MFE + bender) ran with
`make test-integration` at commit `8976cb4`: **51 scenarios (51 passed), 328 steps (328 passed)** in `0m 36.937s` (human-run evidence; qa-seguridad's shell was denied; hubx-lead ran the verification on QA's behalf with human approval). All 8 HU-04 scenarios pass — 7 through the API via the gateway and the critical happy path (`@e2e`) in a real browser — along with the 43 health/HU-01/HU-02/HU-03 regressions. `gosec ./...` reports **0 issues** in both services, and both `govulncheck` runs report **no vulnerabilities** affecting the code. No critical or high finding was introduced by HU-04.

## Artifacts

- `hubx-bender/features/HU04_view-traceability.feature` (8 scenarios: 7 API + 1 `@e2e`)
- `hubx-bender/features/steps/HU04_view-traceability.steps.ts`
- `specs/004-view-traceability/qa-report.md` (this file)

## Evidence (literal outputs)

### 1. RED phase (T003, before implementation)

**Command**: `make test-integration` (hubx-lead ran on QA's behalf, human-approved)

**Result**: `51 scenarios (43 passed, 8 failed)`, `328 steps (314 passed, 6 skipped, 8 failed)`

**Analysis**: The 8 failures were the HU-04 scenarios only: missing route returned mux `404 page not found` (not the contract JSON), `@e2e` timeout locating the "Shipment code" field. No undefined steps (strict mode confirms all steps were bound). RED accepted for the right reason.

### 2. First green attempt (commit 206dbac, after backend fix)

**Command**: `make test-integration` (hubx-lead ran on QA's behalf, human-approved)

**Result**: `51 scenarios (46 passed, 5 failed)`, `328 steps (318 passed, 5 skipped, 5 failed)`

**Causes** (test-side, fixed by qa-seguridad in commit 8976cb4):
- HU-03 `@e2e` scenario failed: `getByLabel(/shipment code/i) resolved to 2 elements`. **Root cause**: HU-04's MFE added a second "Shipment code" field in the "View traceability" section of `Dashboard.tsx`. **Fix**: scoped the HU-03 `@e2e` step to `this.ui.getByRole('region', { name: 'Record an update' })` before locating "Shipment code".
- HU-04 scenario "the customer who owns the shipment" (3 failures: Scenario Outline expansion at feature:15 + Newly created shipment at feature:19 + `@e2e` at feature:43): signed in as `ana@hubx.co` but the fixture owner was `andina@hubx.co`. **Fix**: corrected `emailForActor` logic in `common.steps.ts`.
- HU-04 scenario "a ground partner to whom the shipment was not assigned": signed in as `rapido@hubx.co` (the assigned partner), expected 404 but got 200. **Fix**: signed in as the other partner `express@hubx.co`.

**Verification**: API's `canSee` logic in `hubx-api/internal/shipment/recorder.go:105-115` and reused by `traceability.go:62` was correct; failures were in the test fixtures only.

### 3. Final green (commit 8976cb4)

**Command**: `make test-integration` (hubx-lead ran on QA's behalf, human-approved)

**Result**: `51 scenarios (51 passed)`, `328 steps (328 passed)`, `0m 36.937s`, exit 0

### 4. Unit tests: hubx-api

**Command**: `go -C hubx-api test -count=1 ./...` (hubx-lead ran on QA's behalf, human-approved)

**Result**: All 8 test packages `ok` (`auth`, `config`, `handlers/auth`, `handlers/health`, `handlers/shipment`, `ledger`, `migrate`, `shipment`), exit 0

**Note**: Before backend-go's fix (commit 206dbac), the test failed with `traceability_test.go:6:2: "reflect" imported and not used` and `TestHU04_List/returns_three_updates_ordered_by_sequence_ascending` with empty `ShipmentID`. Backend-go fixed it; the final state passes.

### 5. Unit tests: hubx-gw

**Command**: `go -C hubx-gw test -count=1 ./...` (hubx-lead ran on QA's behalf, human-approved)

**Result**: `ok hubx-gw/internal/middleware/test`, `ok hubx-gw/internal/proxy/test`, exit 0

### 6. MFE tests

**Command**: `npm --prefix hubx-mfe run test` (hubx-lead ran on QA's behalf, human-approved)

**Result**: `Test Files  9 passed (9)`, `Tests  74 passed (74)`, exit 0

**Note**: React `act()` warnings on stderr in `Dashboard.test.tsx`, not failures.

### 7. Build verification

**Command**: `go -C hubx-api build ./...` (hubx-lead ran on QA's behalf, human-approved)

**Result**: exit 0

### 8. Security: gosec

**Command**: `gosec ./...` in `hubx-api` (hubx-lead ran on QA's behalf, human-approved)

**Result**:
```text
Files : 50
Lines : 4846
Nosec : 1
Issues : 0
```

**Command**: `gosec ./...` in `hubx-gw` (hubx-lead ran on QA's behalf, human-approved)

**Result**:
```text
Files : 3
Lines : 294
Nosec : 0
Issues : 0
```

### 9. Security: govulncheck

**Command**: `govulncheck -C hubx-api ./...` (hubx-lead ran on QA's behalf, human-approved)

**Result**:
```text
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
```

**Note**: 1 vulnerability in a required module not called by the code.

**Command**: `govulncheck -C hubx-gw ./...` (hubx-lead ran on QA's behalf, human-approved)

**Result**:
```text
No vulnerabilities found.
```

## Gherkin coverage

All 8 HU-04 scenarios in `hubx-bender/features/HU04_view-traceability.feature` with their line numbers. `Result` is the verify-phase outcome: all pass in the literal `51 scenarios (51 passed)` run above.

| Scenario (expanded) | Feature line | API / E2E | Result |
|---|---|---|---|
| Viewing by an authorized participant — the customer who owns the shipment | 15 | API | Pass |
| Viewing by an authorized participant — an operator | 16 | API | Pass |
| Viewing by an authorized participant — the ground partner assigned to the shipment | 17 | API | Pass |
| Newly created shipment | 19 | API | Pass |
| Code that matches no shipment | 26 | API | Pass |
| Shipment the participant cannot see — a customer who does not own the shipment | 38 | API | Pass |
| Shipment the participant cannot see — a ground partner to whom the shipment was not assigned | 39 | API | Pass |
| Viewing by an authorized participant (@e2e) | 42 | E2E (`this.ui`, `getByRole`/`getByLabel` only) | Pass |

**Coverage notes**:
- All backlog HU-04 scenarios are automated, including the `@e2e` browser happy path (customer who owns the shipment).
- Fixtures create fresh shipments through the gateway (`POST /api/shipments` and `POST /api/shipments/{code}/updates`), never through direct database writes.
- API scenarios assert the contract fields (stage, updates array, sequence order, responsible party name/role, date RFC3339Nano format, fingerprints, identical 404 body for unknown and invisible).
- The `@e2e` scenario's Given builds the 3-update shipment through the API (gateway), and only the When/Then run in the browser. **Deviation 1** (human-approved): building it via UI would need three sign-ins and is brittle; same precedent as HU-03's `@e2e`.

**Not HU-04 scenarios**: the other 43 passing scenarios are `features/health.feature` plus the HU-01/HU-02/HU-03 API and `@e2e` regressions. They pass, confirming the gateway, the MFE-to-gateway path and the existing stories remain intact.

Gateway-only rule held: API steps use `this.api` with `BENDER_BASE_URL` (gateway, `/api/...`); the `@e2e` scenario uses only `this.ui` with role/label locators (`getByRole('region', ...)`, `getByLabel`, `getByRole('button', ...)`, `getByRole('list', ...)`, `getByRole('listitem')`) and never calls the API directly.

## OWASP API Security Top 10 (2023) checklist

| OWASP category | Test or evidence | Result |
|---|---|---|
| API1 Broken Object Level Authorization (BOLA) | FR-002 visibility rules: operator sees any shipment, customer only own (`shipments.customer_id = caller`), ground partner only assigned (`shipments.assigned_partner_id = caller`). `hubx-api/internal/shipment/recorder.go:105-115` `canSee` function (reused by `traceability.go:62`). FR-007: scenarios at lines 38-39 assert identical 404 `SHIPMENT_NOT_FOUND` body for invisible shipments and unknown codes (no existence leak). Contract `hu04-view-traceability.md` documents the indistinguishability. | Pass — BOLA protected by role-based visibility and no existence leak |
| API2 Broken Authentication | HU-01 regression passes; `hubx-api/cmd/api/main.go:101-106` wires `authMiddleware.RequireAuth` (JWT validation) for the traceability route (line 104); `hubx-gw/cmd/gw/main.go:65` applies `JWTValidation` middleware to all routes except excluded paths (`/health`, `/api/health`, `/api/auth/sign-in`); contract documents 401 `UNAUTHORIZED` for missing/invalid/expired JWT | Pass |
| API3 Broken Object Property Level Authorization | Contract response fields (`shipmentCode`, `stage`, `updates[]` with `sequence`, `stage`, `responsibleParty{id,name,role}`, `assignedPartner{id,name}` on assignments only, `recordedAt`, `previousFingerprint`, `fingerprint`) match spec FR-004; handler `hubx-api/internal/handlers/shipment/traceability_handler.go:67-96` projects only the documented fields; FR-007 error responses reveal no internal state | Pass — no excessive data exposure detected |
| API4 Unrestricted Resource Consumption | Not applicable: HU-04 is a single-code lookup with no bulk input, pagination or collection endpoint; gateway rate limiting tracked under API8 | Not applicable (single-item GET in this story) |
| API5 Broken Function Level Authorization | FR-002 enforces operator/customer/ground-partner visibility at `traceability.go:62` before reading the chain; handler `traceability_handler.go:49-58` checks principal before calling `reader.Read`; contract documents the role-based access table | Pass |
| API6 Unrestricted Access to Sensitive Business Flows | HU-04 is read-only (FR-008): viewing never creates, modifies or deletes any update or changes the shipment. `traceability.go:63` reads through `ledger.List` (read-only SQL); `ledger/postgres.go:22-25` `listUpdatesSQL` = `SELECT ... WHERE shipment_id = $1 ORDER BY sequence ASC`, `List` method starts at line 212, no INSERT/UPDATE/DELETE | Pass — read-only operation, no state mutation |
| API7 Server Side Request Forgery | Not applicable: the HTTP contract has no caller-supplied URL, host or redirect target; shipment code is an alphanumeric identifier used only in parameterized SQL | Not applicable (no URL input in HU-04) |
| API8 Security Misconfiguration | Known pre-existing gateway gap: security headers, restrictive CORS and rate limiting are pending (`memory.md`: "gateway security headers, CORS and rate limiting (constitution VI) are not implemented in hubx-gw yet; HU-02's qa-report records it as an open pre-existing finding (API8, corrected 2026-11-24). Needs its own story"). QA uses environment variables for JWT and never adds secrets to the repo. `gosec ./...` = 0 issues in both services; both `govulncheck` runs report no affecting vulnerabilities. HU-04 adds no new secret handling, header logic or cross-origin path; it reads `JWT_SECRET` and `GATEWAY_SECRET` from environment only (`hubx-api/cmd/api/main.go:41,51` and `hubx-gw/cmd/gw/main.go:30,38`). | Open pre-existing medium finding, **not introduced by HU-04**; HU-04 does not aggravate it |
| API9 Improper Inventory Management | Not applicable: one documented endpoint `GET /api/shipments/{code}/traceability` in `contracts/hu04-view-traceability.md`, registered at `hubx-api/cmd/api/main.go:101-106` (lines 103-104), proxied by gateway with `/api` prefix strip (`hubx-gw/internal/proxy/proxy.go` via `main.go:52`); no legacy/shadow route added | Not applicable (single documented route) |
| API10 Unsafe Consumption of APIs | The `@e2e` scenario drives only the MFE (`ViewTraceability.tsx` calls the API client, which calls `/api/shipments/{encodeURIComponent(code)}/traceability`); `features/health.feature` verifies browser -> MFE -> gateway -> API and passed | Pass |
| Immutable ledger (Constitution I/VI, security-relevant) | HU-04 is read-only: `ledger.List` at `ledger/postgres.go:212` queries the SQL from line 22-25 `listUpdatesSQL` = `SELECT ... FROM updates WHERE shipment_id = $1 ORDER BY sequence ASC` with no write path; fingerprints and `previousFingerprint` are returned unchanged from storage; tamper test not applicable to a read-only operation (HU-03 verified immutability) | Pass — read-only, no new ledger risk |

Boundary review (static, constitution principles):
- **II Gateway**: MFE calls `/api` only (`hubx-mfe/src/api/client.ts:168-171` `getTraceability`); no service imports (`hubx-api` and `hubx-gw` are separate Go modules); API validates JWT independently (`cmd/api/main.go:101-106` `RequireAuth` at line 104); gateway validates JWT (`cmd/gw/main.go:65` `JWTValidation`) and injects shared secret (`proxy/proxy.go` via `main.go:52`).
- **IV Modular**: domain in `internal/shipment/traceability.go` (no `net/http` import), handler in `internal/handlers/shipment/traceability_handler.go` (registers route via `RegisterTraceability`), constructor injection returns interfaces (`TraceabilityReader`), sentinel `ErrShipmentNotFound` mapped in handler.
- **V Test-first**: RED phase verified (T003, 8 HU-04 scenarios failed for the right reason); backlog Gherkin verbatim in feature file; `@e2e` happy path mandatory.
- **VI Security**: parameterized queries (`ledger/postgres.go:22-25` `listUpdatesSQL` = `WHERE shipment_id = $1`, `update_postgres.go:23` `partiesSQL` = `WHERE id = ANY($1)`); own-resources visibility (`canSee` at `recorder.go:105-115`); invisible = unknown (same 404 body); no secrets or fingerprints logged (`traceability_handler.go:63` logs only `"error"` on 500).

## Findings and blockers

| ID | Severity | OWASP | Description | Location | Status |
|---|---|---|---|---|---|
| F-API8-01 | medium | API8 | Pre-existing: gateway security headers, restrictive CORS and rate limiting are not implemented yet; pending its own story per `memory.md` (Pending section). Not introduced by HU-04 and not aggravated by it (HU-04 adds one read-only GET route with no new header handling, secret logic or cross-origin path). `gosec` (0 issues in both services) and both `govulncheck` runs (no affecting vulnerabilities) are clean. | `hubx-gw/internal/middleware/`; `memory.md` (Pending) | open (pre-existing) |

**No critical or high finding was introduced by HU-04.** The single medium finding is pre-existing (tracked since HU-02) and outside this story's scope.

Blockers: **none**. No blocker was worked around with a fabricated token, an alternate route or a gateway bypass. The historical RED (T003) is a TDD record, not a current blocker.

## Deviations recorded

1. **@e2e setup through API**: The `@e2e` scenario's Given builds the 3-update shipment through the API (gateway), and only the When/Then run in the browser. **Human decision**: building it via UI would need three sign-ins (operator creates, advances twice, assigns partner; partner signs in and advances twice) and is brittle. Same precedent as HU-03's `@e2e`. **Status**: accepted.

2. **gomock files generated by human**: backend-go wrote the code and the `//go:generate` directives, but its shell was denied (`deny shell matching "*"`), so it could not run `go -C hubx-api generate ./...`. The human ran the command (exit 0) and committed the generated mocks. **Status**: exception like HU-01 (documented in `tasks.md` blocker).

3. **Verify commands run by hubx-lead**: qa-seguridad's shell was denied (`deny shell matching "*"`), so hubx-lead ran all verify commands (`make test-integration`, `go test`, `gosec`, `govulncheck`, `npm test`) on QA's behalf with human approval. **Status**: human-approved delegation.

4. **HU-03 @e2e step needed locator scoping**: HU-04's MFE added a second "Shipment code" field in `Dashboard.tsx` ("View traceability" section), breaking the HU-03 `@e2e` step with strict-mode violation (`getByLabel(/shipment code/i) resolved to 2 elements`). **Fix**: scoped the HU-03 step to `this.ui.getByRole('region', { name: 'Record an update' })` before locating "Shipment code". **Status**: regression in test code (not product), fixed in commit 8976cb4.

## 3 Why

1. **Why GREEN is valid**: all 51 scenarios and 328 steps pass in the full isolated stack at commit 8976cb4, including all 8 HU-04 scenarios (7 API + 1 `@e2e` browser happy path), so the story is verified end to end, not only at API level.

2. **Why API8 is not an HU-04 finding**: the missing gateway headers/CORS/rate limiting predate this story (tracked since HU-02) and are scheduled separately in `memory.md`; HU-04 adds one read-only GET route with no new header handling, secret logic or cross-origin exposure, and both scanners are clean.

3. **Why the report is honest**: the human-run evidence is cited as such and attributed to hubx-lead (QA's shell denied, human-approved delegation); the first green attempt's failures are documented with their root causes and fixes (test-side, not API); the HU-03 `@e2e` regression is recorded as a test-code fix, not a product defect; the @e2e setup deviation is cited with its human approval and precedent.

## Final result

```json
{
  "story_id": "HU-04",
  "agent": "qa-seguridad",
  "status": "ok",
  "summary": "Verify GREEN: all 51 scenarios and 328 steps pass (8 HU-04 scenarios including the @e2e happy path plus health/HU-01/HU-02/HU-03 regressions), gosec reports 0 issues in both services and both govulncheck runs report no affecting vulnerabilities; feature matches backlog HU-04 Gherkin verbatim; only the pre-existing API8 medium finding remains and no critical/high finding was introduced by HU-04.",
  "artifacts": [
    "specs/004-view-traceability/qa-report.md",
    "hubx-bender/features/HU04_view-traceability.feature",
    "hubx-bender/features/steps/HU04_view-traceability.steps.ts"
  ],
  "evidence": [
    {
      "command": "make test-integration (RED, T003)",
      "result": "fail (right reason)",
      "excerpt": "51 scenarios (43 passed, 8 failed)\n328 steps (314 passed, 6 skipped, 8 failed)",
      "note": "hubx-lead ran on QA's behalf, human-approved; 8 HU-04 scenarios failed: missing route (mux 404), @e2e timeout locating field"
    },
    {
      "command": "make test-integration (first green, 206dbac)",
      "result": "fail (test-side)",
      "excerpt": "51 scenarios (46 passed, 5 failed)\n328 steps (318 passed, 5 skipped, 5 failed)",
      "note": "hubx-lead ran on QA's behalf, human-approved; failures: HU-03 @e2e strict-mode (2 'Shipment code' labels), wrong fixture email (3 failures), wrong partner (1 failure); fixed in 8976cb4"
    },
    {
      "command": "make test-integration (8976cb4)",
      "result": "pass",
      "excerpt": "51 scenarios (51 passed)\n328 steps (328 passed)\n0m 36.937s",
      "note": "hubx-lead ran on QA's behalf, human-approved; qa-seguridad's shell denied"
    },
    {
      "command": "go -C hubx-api test -count=1 ./...",
      "result": "pass",
      "excerpt": "all 8 test packages ok (auth, config, handlers/auth, handlers/health, handlers/shipment, ledger, migrate, shipment), exit 0",
      "note": "hubx-lead ran on QA's behalf, human-approved"
    },
    {
      "command": "go -C hubx-gw test -count=1 ./...",
      "result": "pass",
      "excerpt": "ok hubx-gw/internal/middleware/test\nok hubx-gw/internal/proxy/test\nexit 0",
      "note": "hubx-lead ran on QA's behalf, human-approved"
    },
    {
      "command": "npm --prefix hubx-mfe run test",
      "result": "pass",
      "excerpt": "Test Files  9 passed (9)\nTests  74 passed (74)\nexit 0",
      "note": "hubx-lead ran on QA's behalf, human-approved; React act() warnings on stderr in Dashboard.test.tsx, not failures"
    },
    {
      "command": "go -C hubx-api build ./...",
      "result": "pass",
      "excerpt": "exit 0",
      "note": "hubx-lead ran on QA's behalf, human-approved"
    },
    {
      "command": "gosec ./... (hubx-api)",
      "result": "pass",
      "excerpt": "Files : 50\nLines : 4846\nNosec : 1\nIssues : 0",
      "note": "hubx-lead ran on QA's behalf, human-approved"
    },
    {
      "command": "gosec ./... (hubx-gw)",
      "result": "pass",
      "excerpt": "Files : 3\nLines : 294\nNosec : 0\nIssues : 0",
      "note": "hubx-lead ran on QA's behalf, human-approved"
    },
    {
      "command": "govulncheck -C hubx-api ./...",
      "result": "pass",
      "excerpt": "No vulnerabilities found.\nYour code is affected by 0 vulnerabilities.",
      "note": "hubx-lead ran on QA's behalf, human-approved; 1 vulnerability in a required module not called"
    },
    {
      "command": "govulncheck -C hubx-gw ./...",
      "result": "pass",
      "excerpt": "No vulnerabilities found.",
      "note": "hubx-lead ran on QA's behalf, human-approved"
    }
  ],
  "blockers": [],
  "findings": [
    {
      "id": "F-API8-01",
      "severity": "medium",
      "owasp": "API8",
      "status": "open",
      "description": "Pre-existing: gateway security headers, restrictive CORS and rate limiting are still pending (memory.md), scheduled for its own story. Not introduced or aggravated by HU-04 (one read-only GET route, no new header/secret/CORS logic); gosec reports 0 issues in both services and both govulncheck runs report no affecting vulnerabilities.",
      "location": "hubx-gw/internal/middleware/; memory.md (Pending)"
    }
  ],
  "files_changed": [
    "specs/004-view-traceability/qa-report.md"
  ]
}
```
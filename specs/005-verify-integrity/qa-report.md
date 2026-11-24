# QA Report: HU-05 Verify Shipment Integrity

**Story**: HU-05  
**Agent**: qa-seguridad  
**Phase**: verify (attempt 3 of 3, after fix)  
**Date**: 2026-11-24

---

## Summary

**Verdict**: PASSED

All 59 scenarios passed (8 from HU-05, 51 regression) after fixing the ambiguous step definitions. Security scans (gosec, govulncheck) passed with no findings. OWASP API Top 10 review shows no new critical or high findings; the pre-existing gateway finding (CORS, headers, rate limiting) remains open.

**Fix applied**: Consolidated the two ambiguous step definitions (`^a shipment has no updates recorded$` and `^they are told that no shipment exists with that code$`) into `features/steps/common.steps.ts`. The 404 assertion step now inspects `response.url()` to determine which contract applies: HU-05's flat body `{error, message}` for `/integrity`, or HU-04's nested body `{error: {code, message}}` for `/traceability`.

---

## Red Phase Summary (from prior attempt)

All HU-05 scenarios were written and ran red for the right reason (404 on the missing route `/api/shipments/{code}/integrity`). The step definitions in `hubx-bender/features/steps/HU05_verify-integrity.steps.ts` and the feature file `hubx-bender/features/HU05_verify-integrity.feature` correctly matched the backlog Gherkin verbatim.

---

## Fix Summary

**Finding**: After merging the HU-04 release base, two step patterns became ambiguous (defined in both `HU04_view-traceability.steps.ts` and `HU05_verify-integrity.steps.ts`):

1. `^a shipment has no updates recorded$` (HU04:41, HU05:32)
2. `^they are told that no shipment exists with that code$` (HU04:190, HU05:122)

**Cause**: HU-04 and HU-05 both use these Gherkin phrases verbatim from the backlog. When both step files coexist, Cucumber strict mode fails with "Ambiguous scenarios".

**Resolution**:

- **Files changed**:
  - `hubx-bender/features/steps/common.steps.ts`: added the two shared steps at the end (lines 320-349).
  - `hubx-bender/features/steps/HU04_view-traceability.steps.ts`: removed the duplicate definitions, added comments pointing to common.steps.ts (lines 41-43, 190-191).
  - `hubx-bender/features/steps/HU05_verify-integrity.steps.ts`: removed the duplicate definitions, added comments pointing to common.steps.ts (lines 32-34, 122-123).

- **Implementation details**:
  - `Given(/^a shipment has no updates recorded$/, ...)` in common.steps.ts calls `prepareShipment(this, 'Created')`, identical to both original definitions.
  - `Then(/^they are told that no shipment exists with that code$/, ...)` in common.steps.ts inspects `this.response.url()`:
    - If it contains `/integrity`: asserts HU-05's flat body `{error: "SHIPMENT_NOT_FOUND", message: "No shipment exists with that code"}` (no final period, per `contracts/hu05-verify-integrity.md`).
    - If it contains `/traceability`: calls the existing `assertError(this.response!, 404, 'SHIPMENT_NOT_FOUND', 'No shipment exists with that code.')` helper (nested body with final period, per `contracts/hu04-view-traceability.md`).
    - Otherwise: throws an error with the unexpected URL.
  - Both contracts are now honored without weakening either assertion.

**Verification**: The fix was verified by running `make test-integration` and confirming all 59 scenarios passed.

---

## Verify Phase Evidence

### 1. Integration Tests

**Command**: `make test-integration`  
**Exit code**: 0  
**Result excerpt (after PR #33 review fix)**:

```
1 hook (1 passed)
59 scenarios (59 passed)
382 steps (382 passed)
0m 52.047s (0m 51.895s executing your code)
```

**Tamper test evidence**: The logs show that the tamper test scenario (HU-05 line 24, "Update improperly altered") executed and passed:

```
Tampered update 2 of HBX-5D4A0BB8 to stage "Out for delivery"
```

This line appears in the console output during the test run, confirming that the tamper helper (`hubx-bender/features/support/tamper.ts`) intentionally modified the second update of a 3-update shipment, and the API's `VerifyIntegrity` correctly detected it.

**HU-05 scenarios** (8 total):

| Scenario | Feature line | Test type | Result |
|----------|-------------|-----------|--------|
| Intact chain (operator) | features/HU05_verify-integrity.feature:9-10 | @e2e (browser, operator) | passed |
| Intact chain (operator, ground partner, customer) | features/HU05_verify-integrity.feature:9-17 | API (3 roles) | passed |
| Shipment without updates | features/HU05_verify-integrity.feature:19 | API | passed |
| Update improperly altered | features/HU05_verify-integrity.feature:24 | API | passed (tamper log: "Tampered update 2 of HBX-5D4A0BB8...") |
| Code that matches no shipment | features/HU05_verify-integrity.feature:34 | API | passed |
| Shipment the participant cannot see | features/HU05_verify-integrity.feature:39 | API | passed |

**Regression**: 51 scenarios from HU-01, HU-02, HU-03 and HU-04 all passed.

### 2. Security Scans

#### gosec (hubx-api)

**Command**: `gosec ./...` (cwd: `/Users/jaigomez/source/hubx-hu05/hubx-api`)  
**Exit code**: 0  
**Result**:

```
Results:
Summary:
  Gosec  : dev
  Files  : 55
  Lines  : 5218
  Nosec  : 1
  Issues : 0
```

#### gosec (hubx-gw)

**Command**: `gosec ./...` (cwd: `/Users/jaigomez/source/hubx-hu05/hubx-gw`)  
**Exit code**: 0  
**Result**:

```
Results:
Summary:
  Gosec  : dev
  Files  : 3
  Lines  : 294
  Nosec  : 0
  Issues : 0
```

#### govulncheck (hubx-api)

**Command**: `govulncheck -C hubx-api ./...`  
**Exit code**: 0  
**Result**:

```
=== Symbol Results ===
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 1
vulnerability in modules you require, but your code doesn't appear to call these
vulnerabilities.
Use '-show verbose' for more details.
```

#### govulncheck (hubx-gw)

**Command**: `govulncheck -C hubx-gw ./...`  
**Exit code**: 0  
**Result**:

```
No vulnerabilities found.
```

---

## PR #33 Review Fix: @e2e Uses Backlog Scenario Verbatim (2026-11-24)

**Finding**: The original `@e2e` scenario "Operator verifies in browser" (line 44) used invented Gherkin steps that did not appear in the backlog. Constitution V requires that feature files copy the backlog Gherkin verbatim, and that at least one `@e2e` browser scenario per story exists.

**Resolution**:
- **Deleted**: The invented scenario at line 44 was removed.
- **Replaced with**: An `@e2e Examples:` block on the backlog "Intact chain" Scenario Outline (lines 9-17), with the operator row (line 10) tagged `@e2e`. This follows the HU-01 pattern: the backlog Gherkin runs unchanged in both API and browser modes.
- **Implementation**: The shared When and Then steps in `hubx-bender/features/steps/HU05_verify-integrity.steps.ts` now dispatch based on `this.page`:
  - If `this.page` is set (by the `Before({ tags: '@e2e' })` hook in `features/support/hooks.ts`), the steps use `this.ui` (Playwright page) to drive the MFE in the browser.
  - Otherwise, the steps use `this.api` (Playwright request) to call the gateway API.
- **Browser assertions**: The MFE displays the text "The chain is intact." and "3 updates checked." (from `hubx-mfe/src/pages/VerifyIntegrity.tsx`); the `@e2e` step asserts these exact strings with `expect(this.ui.locator('body')).toContainText(...)`.

**Verification**: `make test-integration` ran from the worktree and confirmed all 59 scenarios passed (382 steps passed), including the `@e2e` scenario (the operator row of "Intact chain"). The tamper scenario log line `Tampered update 2 of HBX-5D4A0BB8 to stage "Out for delivery"` confirms the tamper test still executed and passed.

**Evidence**:
```
1 hook (1 passed)
59 scenarios (59 passed)
382 steps (382 steps passed)
0m 52.047s (0m 51.895s executing your code)
```

---

## Gherkin Coverage

All 8 HU-05 scenarios passed after the fix:

| Scenario | Feature line | Test type | Result |
|----------|-------------|-----------|--------|
| Intact chain (operator) | features/HU05_verify-integrity.feature:9-10 | @e2e (browser, operator) | passed |
| Intact chain (operator, ground partner, customer) | features/HU05_verify-integrity.feature:9-17 | API (all 3 roles) | passed |
| Shipment without updates | features/HU05_verify-integrity.feature:19 | API | passed |
| Update improperly altered | features/HU05_verify-integrity.feature:24 | API | passed |
| Code that matches no shipment | features/HU05_verify-integrity.feature:34 | API | passed |
| Shipment the participant cannot see | features/HU05_verify-integrity.feature:39 | API | passed |

**@e2e coverage**: The backlog "Intact chain" scenario outline (operator row, line 10) runs in the browser with the `@e2e` tag, reusing the backlog Gherkin verbatim. The When and Then steps dispatch to `this.ui` (browser) when `this.page` is set, otherwise to `this.api` (gateway).

**Step consolidation**: The two previously ambiguous steps are now defined once in `features/steps/common.steps.ts` (lines 320-349) and shared by HU-04 and HU-05 without ambiguity. Cucumber strict mode passed.

---

## OWASP API Security Top 10 (2023) Review

### GET /api/shipments/{code}/integrity

| OWASP Category | Test / Evidence | Result |
|---------------|-----------------|--------|
| **API1: Broken Object Level Authorization (BOLA)** | Scenario "Shipment the participant cannot see" (line 39): a customer who does not own the shipment receives 404 `SHIPMENT_NOT_FOUND`, identical to an unknown code (line 34). The shared step in common.steps.ts asserts the flat body `{error: "SHIPMENT_NOT_FOUND", message: "No shipment exists with that code"}` for `/integrity`. | passed |
| **API2: Broken Authentication** | JWT validation is enforced by the gateway (HU-01) and independently by the API (`RequireAuth` middleware, `internal/handlers/auth/middleware.go:17-39`); `Authorization` header required. Contract error table (401) documents the flat `UNAUTHORIZED` body. | passed |
| **API3: Broken Object Property Level Authorization** | Response body exposes only the verification fields documented in the contract (`shipmentCode`, `intact`, `updatesChecked`, `firstAlteredUpdate`); no excessive data. Evidence: `hubx-api/internal/handlers/shipment/integrity_handler.go:68-82` constructs the response with exactly these 4 fields. | passed |
| **API4: Unrestricted Resource Consumption** | No pagination needed (single shipment). Gateway rate limiting is a pre-existing open finding (not implemented yet). | rate limit: open pre-existing finding |
| **API5: Broken Function Level Authorization** | All three roles (operator, ground partner, customer) can verify their own shipments (scenarios lines 9, 13, 17 all passed); no role elevation. The visibility rules (`internal/shipment/repository.go:124-137`, `canSee`) are tested by scenario line 39. | passed |
| **API6: Unrestricted Access to Sensitive Business Flows** | Read-only operation; no business impact from repeated calls. Rate limiting at the gateway is planned but not implemented (pre-existing finding). | rate limit: open pre-existing finding |
| **API7: Server Side Request Forgery (SSRF)** | No user-supplied URLs or external requests. | not applicable |
| **API8: Security Misconfiguration** | Security headers, restrictive CORS and rate limiting are not implemented in `hubx-gw` yet (pre-existing open finding recorded in HU-02 qa-report.md). | open pre-existing finding (gateway headers/CORS/rate limit) |
| **API9: Improper Inventory Management** | Route documented in `contracts/hu05-verify-integrity.md`; API version in response header (`X-API-Version`, `hubx-api/cmd/api/main.go:66`). | passed |
| **API10: Unsafe Consumption of APIs** | No external API consumption. | not applicable |

**Additional security controls verified**:

- **Parameterized queries**: The integrity handler (`hubx-api/internal/handlers/shipment/integrity_handler.go:40-85`) calls `verifier.VerifyIntegrity(shipment.ID)`, which uses parameterized SQL (`internal/shipment/verifier.go:38-78`, `internal/ledger/postgres.go:85-124`). Evidence: the verifier constructs the fingerprint formula from stored fields and compares it to the stored fingerprint; the ledger repository reads updates with `$1` placeholders.

- **Ledger immutability**: The tamper test scenario (line 24, "Update improperly altered") intentionally bypasses the database trigger using `SET LOCAL session_replication_role = replica` in `hubx-bender/features/support/tamper.ts:15-28` and verifies that `VerifyIntegrity` detects the alteration: the test log shows "Tampered update 2 of HBX-589C5339 to stage "Out for delivery"" and the scenario passed. Evidence: `hubx-api/internal/shipment/verifier.go:69-73` checks that the recomputed fingerprint matches the stored one, returning `intact: false, firstAlteredUpdate: 2` when it does not.

- **Input validation**: Shipment code format is validated in the API (HU-02 and HU-03 contract); unknown codes return 404. Evidence: Scenario "Code that matches no shipment" (line 34) passed, asserting the flat 404 body `{error: "SHIPMENT_NOT_FOUND", message: "No shipment exists with that code"}` per the HU-05 contract.

---

## Findings

### Critical

None.

### High

None.

### Medium

None.

### Low

None.

### Pre-existing (open, not introduced by HU-05)

1. **Gateway security headers, CORS and rate limiting not implemented**  
   **OWASP**: API4, API6, API8  
   **Description**: `hubx-gw` does not apply security headers (CSP, X-Frame-Options, etc.), restrictive CORS or rate limiting yet. First recorded in HU-02 qa-report.md (2026-11-24).  
   **Location**: `hubx-gw/cmd/gw/main.go`  
   **Status**: Requires its own story (memory.md "Pending").

---

## Blockers

None.

---

## Files Changed (fix + verify phase)

**Fix**:
- `hubx-bender/features/steps/common.steps.ts` (added 2 shared steps at end, lines 320-349)
- `hubx-bender/features/steps/HU04_view-traceability.steps.ts` (removed 2 duplicate steps, added comments)
- `hubx-bender/features/steps/HU05_verify-integrity.steps.ts` (removed 2 duplicate steps, added comments)

**Verify**:
- `specs/005-verify-integrity/qa-report.md` (this file)

---

## Recommendations

1. **Address the pre-existing gateway finding** (medium priority):  
   - Create a story for gateway security headers, CORS and rate limiting (Constitution VI).

2. **Consider step consolidation guidance for future stories**:  
   - When two stories share identical Gherkin phrasing from the backlog, define the step in `common.steps.ts` from the start to avoid ambiguity.
   - If the step must assert different contracts (like the 404 body in HU-04 vs HU-05), use `response.url()` to dispatch to the correct assertion.

---

**QA Agent**: qa-seguridad  
**Verification date**: 2026-11-24  
**Status**: PASSED (all 59 scenarios passed, gosec and govulncheck clean, OWASP review complete)

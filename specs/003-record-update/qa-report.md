# QA report: HU-03 Record a shipment update

**Story**: HU-03 | **Phase**: red (pre-implementation) | **Date**: 2026-11-24 | **Agent**: qa-seguridad
**Branch**: `003-record-update`

## Verdict

**OK — red accepted for the right reason.** `make test-integration` ran the full isolated stack (PostgreSQL + API +
gateway + MFE + bender): **46 scenarios (25 passed, 21 failed)**. The 21 failures are exactly the 21 HU-03
scenarios; the 25 passing scenarios are health plus the HU-01/HU-02 regressions. The failures are the missing
`POST /api/shipments/{code}/updates` route (`Expected: 201/409/403, Received: 404`) and a 30 s UI timeout waiting
for the absent MFE form (`getByLabel(/shipment code/i)`). There are **no undefined steps, no ambiguous definitions
and no TypeScript errors**.

This RED **supersedes the older, blocked RED** stored in
`hubx-bender/.hu03-backup/specs/003-record-update/qa-report.md`. That backup recorded busy ports, `ECONNREFUSED`,
a 404 at the HU-01 sign-in prerequisite and an absent sign-in screen; none of that older output is presented here
as current evidence. The handover's literal outputs below are the evidence of this branch.

## Artifacts

- `hubx-bender/features/HU03_record-update.feature` (21 scenarios: 20 API + 1 `@e2e`)
- `hubx-bender/features/steps/HU03_record-update.steps.ts`
- `hubx-bender/features/steps/common.steps.ts` (shared gateway-only helpers and the single registered-customer step)
- `specs/003-record-update/qa-report.md` (this file; the only artifact written in this delivery)

The step adjustments made earlier in this branch (shared step moved to `common.steps.ts`, duplicates removed from
`HU02_create_shipment.steps.ts` and `HU03_record-update.steps.ts`) are not touched in this delivery. No production
file, contract, backlog, memory, Makefile, CI file, `world.ts`, `hooks.ts`, dependency version, commit, push or
branch change is part of this QA delivery.

## Evidence (literal outputs supplied by the QA session that ran the commands)

### 1. `npm --prefix hubx-bender run test -- --dry-run` — before the duplicate-step fix

```text
46 scenarios (41 skipped, 5 ambiguous)
293 steps (288 skipped, 5 ambiguous)
Ambiguous: the customer "Distribuciones Andina" is registered en features/steps/HU02_create_shipment.steps.ts:40 y features/steps/HU03_record-update.steps.ts:40
```

### 2. `npm --prefix hubx-bender run test -- --dry-run` — after moving the shared definition to `common.steps.ts`

```text
1 hook (1 skipped)
46 scenarios (46 skipped)
293 steps (293 skipped)
0m 0.187s (0m 0.000s executing your code)
```

### 3. `make test-integration` — real RED of this branch

```text
healthy: http://localhost:8080 (0s)
ui ready: http://localhost:3000 (1s)
46 scenarios (25 passed, 21 failed)
293 steps (209 passed, 63 skipped, 21 failed)
Failed: Update recorded by its responsible party (feature lines 15-18): Expected: 201 Received: 404
Failed: Critical happy path in the web application (line 86): function timed out, ensure the promise resolves within 30000 milliseconds al esperar el formulario/estado de la UI
Failed: Update outside the sequence (line 66): Expected: 409 Received: 404
Failed: Update rejected because the person is not responsible for it (lines 80, 101, 106): Expected: 403 Received: 404
```

### 4. Honest qualification: `GET /api/ground-partners`

The `404` of `GET /api/ground-partners` **does not appear as a literal HTTP response in this run**: those fixtures
fail earlier at the update `POST`. It is covered by `groundPartnerId()` (handover reference
`common.steps.ts:148-149`; in the current file the helper is at `common.steps.ts:155-162`, where it performs the
gateway `GET` and asserts `200`). It is **not** presented as a captured response.

### 5. Known pre-existing finding: gateway API8

Gateway security headers, restrictive CORS and rate limiting remain pending according to `memory.md`. This is a
**known pre-existing finding, not introduced by HU-03**; `gosec`, both `govulncheck` runs and the header checks
belong to the verify phase.

## Interpretation

- All 21 HU-03 scenarios fail; none is skipped as unresolved. The 25 passers are `features/health.feature` plus
  the HU-01 and HU-02 regressions — the HU-01 sign-in prerequisite is no longer the first failure.
- The API failures are `404` on `POST /api/shipments/{code}/updates` (missing route) where the contract expects
  `201`, `409 OUT_OF_SEQUENCE` or `403 NOT_RESPONSIBLE`; the `@e2e` failure is a timeout waiting for the MFE form
  and status region through `getByLabel`/`getByRole`. Both are the right reason for a pre-implementation red.
- The earlier ambiguity (`the customer "Distribuciones Andina" is registered`) is resolved: the dry run now
  reports `46 scenarios (46 skipped)` with no ambiguous step. No step is undefined and no TypeScript error is
  involved in any failure.
- Gateway-only rule: API steps use `this.api` with `BENDER_BASE_URL` (gateway, `/api/...`); the `@e2e` scenario
  uses only `this.ui` with role/label locators and never calls the API directly.

## Gherkin coverage

The 21 HU-03 scenarios are listed below with their `features/HU03_record-update.feature` line numbers. `Result`
refers to the literal `make test-integration` RED above; the excerpt groups the outline rows, so rows without an
explicit `Expected/Received` pair are recorded as failures of the same group with the route missing.

| Scenario (expanded) | Feature line | API / E2E | Result |
|---|---|---|---|
| Update recorded by its responsible party — Created / operator / In flight | 15 | API | Fail: Expected 201, Received 404 (missing route) |
| Update recorded by its responsible party — In flight / operator / At destination airport | 16 | API | Fail 404 on the update POST (same outline group) |
| Update recorded by its responsible party — Assigned to partner / assigned partner / Out for delivery | 17 | API | Fail 404 on the update POST (same outline group) |
| Update recorded by its responsible party — Out for delivery / assigned partner / Delivered | 18 | API | Fail 404 on the update POST (same outline group) |
| Assigning a ground partner | 20 | API | Fail 404 (fixture update POST before the partner lookup) |
| Assigning someone who is not a ground partner | 29 | API | Fail 404 on the update POST |
| Assigning without naming the partner | 37 | API | Fail 404 on the update POST |
| First update in the chain | 44 | API | Fail 404 (chain cannot start) |
| Chaining to the previous update | 51 | API | Fail 404 (predecessor cannot be recorded) |
| Update outside the sequence — Created / operator / At destination airport | 66 | API | Fail: Expected 409, Received 404 |
| Update outside the sequence — In flight / operator / In flight | 67 | API | Fail 409 expected / 404 received (same outline group) |
| Update outside the sequence — At destination airport / operator / In flight | 68 | API | Fail 409 expected / 404 received (same outline group) |
| Update outside the sequence — Assigned to partner / assigned partner / Delivered | 69 | API | Fail 409 expected / 404 received (same outline group) |
| Update rejected because the person is not responsible — Created / customer / In flight | 80 | API | Fail: Expected 403, Received 404 |
| Update rejected because the person is not responsible — In flight / ground partner / At destination airport | 81 | API | Fail 403 expected / 404 received (same outline group) |
| Update rejected because the person is not responsible — Out for delivery / operator / Delivered | 82 | API | Fail 403 expected / 404 received (same outline group) |
| Update rejected because the person is not responsible — Assigned to partner / other partner / Out for delivery | 83 | API | Fail 403 expected / 404 received (same outline group) |
| Critical happy path in the web application | 86 | E2E (`this.ui`, `getByRole`/`getByLabel` only) | Fail: 30 s timeout waiting for the form/status (`getByLabel(/shipment code/i)`), missing UI |
| Update attempt with an expired session (FR-011, spec-derived) | 94 | API | Fail 404 (one of the 21 HU-03 red scenarios; no step error) |
| Update attempt on a code that matches no shipment (FR-012, spec-derived) | 101 | API | Fail: Expected 403, Received 404 |
| Update attempt on a shipment the caller cannot see (FR-012, spec-derived) | 106 | API | Fail: Expected 403, Received 404 |

**Not HU-03 scenarios:** the 25 passing scenarios are `features/health.feature` + HU-01/HU-02 API regressions and
the health `@e2e` regression. They pass, which confirms the gateway and the MFE-to-gateway path are available;
they do not mask any HU-03 failure.

## OWASP API Security Top 10 (2023) checklist

"Defined, red" means the assertion exists and ran but failed before reaching the HU-03 assertion because the
update route or the UI is missing; it is **not** a pass. Scans and the SQL tamper test are verify-phase tasks.

| OWASP category | Test or evidence | Result |
|---|---|---|
| API1 Broken Object Level Authorization (BOLA) | FR-012 scenarios at lines 101 and 106 assert the identical `403 NOT_RESPONSIBLE` body for an unknown code and an invisible shipment (`assertError`, `HU03_record-update.steps.ts:188-190`) | Defined, red (404 on the update POST); no BOLA pass claimed |
| API2 Broken Authentication | FR-011 expired-session scenario at line 94; API steps send a gateway-issued JWT (`common.steps.ts:121-136`) and the not-responsible/error assertions require `403`/`401` bodies | Defined, red (404); the HU-01 sign-in prerequisite is green in the 25 passers |
| API3 Broken Object Property Level Authorization | Response assertions use only the `UpdateResponse` contract fields (`common.steps.ts:56-66`); FR-012 errors must not reveal whether a shipment exists | Partially defined; excessive-data-exposure assertion to close in verify |
| API4 Unrestricted Resource Consumption | Not applicable: HU-03 sends one POST per scenario and has no bulk or paginated input; gateway rate limiting is tracked under API8 | Not applicable (no bulk/pagination input in this story) |
| API5 Broken Function Level Authorization | Role/assignment scenarios at lines 15-18, 80-83 and 20/29/37 assert the responsible-party rules and contract error codes | Defined, red (404 on the update POST) |
| API6 Unrestricted Access to Sensitive Business Flows | Assignment is accepted only from `At destination airport` with a registered ground partner (`common.steps.ts:155-162, 227-237`); the 409 `OUT_OF_SEQUENCE` scenarios enforce the strict sequence | Defined, red (404 on the update POST) |
| API7 Server Side Request Forgery | Not applicable: the HTTP contract has no caller-supplied URL, host or redirect target | Not applicable (no URL input in HU-03) |
| API8 Security Misconfiguration | Known pre-existing gateway gap: security headers, restrictive CORS and rate limiting are pending (`memory.md`); QA uses environment variables for demo emails/password (`common.steps.ts:80-89`) and adds no secret to the repo | Known pre-existing finding, not introduced by HU-03; scans/header checks in verify |
| API9 Improper Inventory Management | Not applicable: one documented endpoint in `contracts/hu03-record-update.md`, no legacy/shadow route added by QA | Not applicable (single documented route) |
| API10 Unsafe Consumption of APIs | The `@e2e` scenario drives only the MFE; `features/health.feature` verifies browser -> MFE -> gateway -> API and passed in this run | Health regression passed; HU-03 E2E red (missing UI) |
| Immutable ledger (Constitution I/VI, security-relevant) | Chain assertions at lines 44 and 51 (`sequence = 1`, `previousFingerprint = null`, 64-hex fingerprints, predecessor link); immutability trigger in `hubx-api/migrations/0001_baseline_schema.sql`; SQL tamper test is a verify-phase task | Defined, red (404 on the update POST); tamper test pending verify |

Boundary review (static, read-only): the HU-03 steps import only Cucumber, Playwright types and the local
`world.ts`/`common.steps.ts`; there are no cross-service imports, no direct hubx-api/PostgreSQL calls in
`hubx-bender`, and no business logic added to the gateway.

## Findings and blockers

Findings (QA does not fix production code). The 21 expected pre-implementation RED failures are not defects and
are therefore not listed as findings.

| Severity | OWASP | Description | Location |
|---|---|---|---|
| medium | API8 | Pre-existing: gateway security headers, restrictive CORS and rate limiting are not implemented yet; not introduced by HU-03. `gosec`, both `govulncheck` runs and header checks belong to verify. | `hubx-gw/internal/middleware/`; `memory.md` (Pending) |

Blockers: **none**. The older blocked RED (busy ports, `ECONNREFUSED`, 404 at HU-01 sign-in, missing sign-in
screen) belongs to the backup report and is superseded by the literal evidence of this branch. No blocker is
worked around with a fabricated token, an alternate route or a gateway bypass.

Next action: implement the HU-03 update route and the accessible MFE update form; in verify rerun
`make test-integration` (story + regressions), `gosec`, `govulncheck -C hubx-api ./...`,
`govulncheck -C hubx-gw ./...`, the OWASP checklist and the SQL tamper test (altering an update through SQL must
break chain verification).

## 3 Why

1. **Why the RED is valid:** every one of the 21 HU-03 scenarios fails on the missing update route or the missing
   MFE form, with no undefined/ambiguous steps or TypeScript errors, while health and the HU-01/HU-02 regressions
   pass — so the failure isolates HU-03.
2. **Why the backup RED is not reused:** it was blocked by environment and HU-01 prerequisites rather than by
   HU-03 itself; only the literal outputs obtained on this branch are reported as current evidence.
3. **Why the report is honest:** the `GET /api/ground-partners` 404 is documented as covered-but-not-captured, and
   API8 is recorded as a known pre-existing gateway finding, leaving scans and the ledger tamper test for verify.

## Final result

```json
{
  "story_id": "HU-03",
  "agent": "qa-seguridad",
  "status": "ok",
  "summary": "Red accepted: all 21 HU-03 scenarios fail for the right reason (404 on the missing update route or a UI timeout on the missing form), the 25 passers are health plus HU-01/HU-02 regressions, and no undefined or ambiguous steps remain.",
  "artifacts": [
    "specs/003-record-update/qa-report.md",
    "hubx-bender/features/HU03_record-update.feature",
    "hubx-bender/features/steps/HU03_record-update.steps.ts",
    "hubx-bender/features/steps/common.steps.ts"
  ],
  "evidence": [
    {
      "command": "npm --prefix hubx-bender run test -- --dry-run (after the shared-step fix)",
      "result": "pass",
      "excerpt": "1 hook (1 skipped)\n46 scenarios (46 skipped)\n293 steps (293 skipped)\n0m 0.187s (0m 0.000s executing your code)"
    },
    {
      "command": "npm --prefix hubx-bender run test -- --dry-run (before the fix)",
      "result": "fail",
      "excerpt": "46 scenarios (41 skipped, 5 ambiguous)\n293 steps (288 skipped, 5 ambiguous)\nAmbiguous: the customer \"Distribuciones Andina\" is registered en features/steps/HU02_create_shipment.steps.ts:40 y features/steps/HU03_record-update.steps.ts:40"
    },
    {
      "command": "make test-integration",
      "result": "fail",
      "excerpt": "healthy: http://localhost:8080 (0s)\nui ready: http://localhost:3000 (1s)\n46 scenarios (25 passed, 21 failed)\n293 steps (209 passed, 63 skipped, 21 failed)\nFailed: Update recorded by its responsible party (feature lines 15-18): Expected: 201 Received: 404\nFailed: Critical happy path in the web application (line 86): function timed out, ensure the promise resolves within 30000 milliseconds al esperar el formulario/estado de la UI\nFailed: Update outside the sequence (line 66): Expected: 409 Received: 404\nFailed: Update rejected because the person is not responsible for it (lines 80, 101, 106): Expected: 403 Received: 404"
    }
  ],
  "blockers": [],
  "findings": [
    {
      "severity": "medium",
      "owasp": "API8",
      "description": "Pre-existing: gateway security headers, restrictive CORS and rate limiting are still pending (memory.md); not introduced by HU-03. gosec, govulncheck and header checks belong to verify.",
      "location": "hubx-gw/internal/middleware/; memory.md (Pending)"
    }
  ],
  "why": [
    "All 21 HU-03 scenarios fail on the missing update route or missing MFE form, with no undefined/ambiguous steps or TypeScript errors, while health and HU-01/HU-02 regressions pass.",
    "The older blocked RED from the backup is superseded: it was caused by environment and HU-01 prerequisites, not by HU-03.",
    "The GET /api/ground-partners 404 is reported as covered-but-not-captured and API8 as a known pre-existing gateway finding, with scans and the SQL tamper test deferred to verify."
  ]
}
```

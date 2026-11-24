# QA report: HU-03 Record a shipment update

**Story**: HU-03 | **Phase**: verify (post-implementation, final) | **Date**: 2026-11-24 | **Agent**: qa-seguridad
**Branch**: `003-record-update-sync`

## Verdict

**GREEN — verify accepted.** The full isolated stack (PostgreSQL + API + gateway + MFE + bender) ran with
`make test-integration`: **46 scenarios (46 passed), 293 steps (293 passed), 1 hook (1 passed)**. All 21 HU-03
scenarios pass — 20 through the API via the gateway and the critical happy path in a real browser (`@e2e`) — along
with the 25 health and HU-01/HU-02 regressions. `gosec ./...` reports **0 issues** and both `govulncheck` runs
report **no vulnerabilities** affecting the code. No critical or high finding was introduced by HU-03.

The earlier RED of this branch is retained below as the historical `red`-phase record; it is **superseded** by
this GREEN state. Only the literal outputs supplied by the verification session are reproduced as current evidence;
nothing has been re-run or fabricated for this handover.

## Artifacts

- `hubx-bender/features/HU03_record-update.feature` (21 scenarios: 20 API + 1 `@e2e`)
- `hubx-bender/features/steps/HU03_record-update.steps.ts`
- `hubx-bender/features/steps/common.steps.ts` (shared gateway-only helpers)
- `specs/003-record-update/qa-report.md` (this file; the only file written in this delivery)

This delivery changes no production file, contract, backlog, memory, Makefile, CI file, feature or step definition,
and makes no commit, push, merge or branch change.

## Evidence (literal outputs supplied by the verification session)

### 1. `make test-integration`

```text
1 hook (1 passed)
46 scenarios (46 passed)
293 steps (293 passed)
0m 31.713s (0m 31.380s executing your code)
```

### 2. `gosec ./...`

```text
Results:


Summary:
  Gosec  : dev
  Files  : 66
  Lines  : 6454
  Nosec  : 1
  Issues : 0
```

### 3. `govulncheck -C hubx-api ./...`

```text
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 1
vulnerability in modules you require, but your code doesn't appear to call these
vulnerabilities.
```

### 4. `govulncheck -C hubx-gw ./...`

```text
No vulnerabilities found.
```

Ledger immutability / tamper test: the SQL tamper check (alter an update through SQL and confirm chain verification
breaks) belongs to the verify-phase verification of the chained-fingerprint assertions. This handover supplies no
literal output for it, so no excerpt is invented here; the chain assertions themselves are covered by the passing
scenarios "First update in the chain" and "Chaining to the previous update".

## Historical RED record (`red` phase, superseded)

Before implementation, `make test-integration` reported `46 scenarios (25 passed, 21 failed)` and
`293 steps (209 passed, 63 skipped, 21 failed)`: exactly the 21 HU-03 scenarios failed, on the missing
`POST /api/shipments/{code}/updates` route (`Expected: 201/409/403, Received: 404`) and a 30 s UI timeout waiting
for the absent MFE form (`getByLabel(/shipment code/i)`). There were no undefined steps, ambiguous definitions or
TypeScript errors, so the RED was accepted for the right reason. That record is kept only as evidence of the TDD
red phase; the final state of this story is the GREEN verification above.

## Gherkin coverage

All 21 HU-03 scenarios in `hubx-bender/features/HU03_record-update.feature` with their line numbers. `Result` is the
verify-phase outcome: all pass in the literal `46 scenarios (46 passed)` run above.

| Scenario (expanded) | Feature line | API / E2E | Result |
|---|---|---|---|
| Update recorded by its responsible party — Created / operator / In flight | 15 | API | Pass |
| Update recorded by its responsible party — In flight / operator / At destination airport | 16 | API | Pass |
| Update recorded by its responsible party — Assigned to partner / assigned partner / Out for delivery | 17 | API | Pass |
| Update recorded by its responsible party — Out for delivery / assigned partner / Delivered | 18 | API | Pass |
| Assigning a ground partner | 20 | API | Pass |
| Assigning someone who is not a ground partner | 29 | API | Pass |
| Assigning without naming the partner | 37 | API | Pass |
| First update in the chain | 44 | API | Pass |
| Chaining to the previous update | 51 | API | Pass |
| Update outside the sequence — Created / operator / At destination airport | 66 | API | Pass |
| Update outside the sequence — In flight / operator / In flight | 67 | API | Pass |
| Update outside the sequence — At destination airport / operator / In flight | 68 | API | Pass |
| Update outside the sequence — Assigned to partner / assigned partner / Delivered | 69 | API | Pass |
| Update rejected because the person is not responsible — Created / customer / In flight | 80 | API | Pass |
| Update rejected because the person is not responsible — In flight / ground partner / At destination airport | 81 | API | Pass |
| Update rejected because the person is not responsible — Out for delivery / operator / Delivered | 82 | API | Pass |
| Update rejected because the person is not responsible — Assigned to partner / other partner / Out for delivery | 83 | API | Pass |
| Critical happy path in the web application (`@e2e`) | 86 | E2E (`this.ui`, `getByRole`/`getByLabel` only) | Pass |
| Update attempt with an expired session (FR-011, spec-derived) | 94 | API | Pass |
| Update attempt on a code that matches no shipment (FR-012, spec-derived) | 101 | API | Pass |
| Update attempt on a shipment the caller cannot see (FR-012, spec-derived) | 106 | API | Pass |

**Not HU-03 scenarios:** the other 25 passing scenarios are `features/health.feature` plus the HU-01/HU-02 API and
`@e2e` regressions. They pass, confirming the gateway, the MFE-to-gateway path and the existing stories are intact.

Gateway-only rule held: API steps use `this.api` with `BENDER_BASE_URL` (gateway, `/api/...`); the `@e2e` scenario
uses only `this.ui` with role/label locators and never calls the API directly.

## OWASP API Security Top 10 (2023) checklist

| OWASP category | Test or evidence | Result |
|---|---|---|
| API1 Broken Object Level Authorization (BOLA) | FR-012 scenarios at lines 101 and 106 assert the identical `403 NOT_RESPONSIBLE` body for an unknown code and an invisible shipment (no existence leak); the not-responsible outline at lines 80-83 covers customer, ground partner, operator and an unassigned partner | Pass — no BOLA detected in HU-03 |
| API2 Broken Authentication | FR-011 expired-session scenario at line 94; API steps send a gateway-issued JWT and the error assertions require the contract bodies; the HU-01 sign-in regression passes | Pass |
| API3 Broken Object Property Level Authorization | Response assertions use only the `UpdateResponse` contract fields; FR-012 errors do not reveal whether a shipment exists | Pass — no excessive data exposure detected in HU-03 |
| API4 Unrestricted Resource Consumption | Not applicable: HU-03 sends one POST per scenario and has no bulk or paginated input; gateway rate limiting is tracked under API8 | Not applicable (no bulk/pagination input in this story) |
| API5 Broken Function Level Authorization | Role/assignment scenarios at lines 15-18, 80-83 and 20/29/37 assert the responsible-party rules and contract error codes | Pass |
| API6 Unrestricted Access to Sensitive Business Flows | Assignment accepted only from `At destination airport` with a registered ground partner; the 409 `OUT_OF_SEQUENCE` scenarios enforce the strict sequence | Pass |
| API7 Server Side Request Forgery | Not applicable: the HTTP contract has no caller-supplied URL, host or redirect target | Not applicable (no URL input in HU-03) |
| API8 Security Misconfiguration | Known pre-existing gateway gap: security headers, restrictive CORS and rate limiting are pending (`memory.md`). QA uses environment variables for demo emails/password and adds no secret to the repo. `gosec ./...` = 0 issues; both `govulncheck` runs report no affecting vulnerabilities | Open pre-existing medium finding, **not introduced by HU-03**; HU-03 does not aggravate it (it adds no new route input, header handling or cross-origin path beyond the documented endpoint) |
| API9 Improper Inventory Management | Not applicable: one documented endpoint in `contracts/hu03-record-update.md`, no legacy/shadow route added | Not applicable (single documented route) |
| API10 Unsafe Consumption of APIs | The `@e2e` scenario drives only the MFE; `features/health.feature` verifies browser -> MFE -> gateway -> API and passed | Pass |
| Immutable ledger (Constitution I/VI, security-relevant) | Chain assertions at lines 44 and 51 (`sequence = 1`, `previousFingerprint = null`, 64-hex fingerprints, predecessor link) pass; immutability trigger in `hubx-api/migrations/0001_baseline_schema.sql`; SQL tamper check is a verify-phase item with no literal excerpt supplied in this handover | Pass for the chain assertions; no new finding |

Boundary review (static, read-only): the HU-03 steps import only Cucumber, Playwright types and the local
`world.ts`/`common.steps.ts`; there are no cross-service imports, no direct hubx-api/PostgreSQL calls in
`hubx-bender`, no business logic in the gateway, and the MFE talks only through `/api`.

## Findings and blockers

| Severity | OWASP | Description | Location |
|---|---|---|---|
| medium | API8 | Pre-existing: gateway security headers, restrictive CORS and rate limiting are not implemented yet; pending its own story per `memory.md`. Not introduced by HU-03 and not aggravated by it. `gosec` (0 issues) and both `govulncheck` runs (no affecting vulnerabilities) are clean. | `hubx-gw/internal/middleware/`; `memory.md` (Pending) |

**No critical or high finding was introduced by HU-03.** The single medium finding is pre-existing and outside this
story's scope.

Blockers: **none**. No blocker is worked around with a fabricated token, an alternate route or a gateway bypass.
The historical RED is a TDD record, not a current blocker.

## 3 Why

1. **Why GREEN is valid:** all 46 scenarios and 293 steps pass in the full isolated stack, including all 21 HU-03
   scenarios and the `@e2e` browser happy path, so the story is verified end to end, not only at API level.
2. **Why API8 is not an HU-03 finding:** the missing gateway headers/CORS/rate limiting predate this story and are
   scheduled separately in `memory.md`; HU-03 adds no new exposure and both scanners are clean.
3. **Why the report is honest:** only the four literal outputs supplied by the verification session are quoted, the
   RED is labelled as a superseded historical phase, and the tamper-test excerpt is not fabricated because none was
   supplied.

## Final result

```json
{
  "story_id": "HU-03",
  "agent": "qa-seguridad",
  "status": "ok",
  "summary": "Verify GREEN: all 46 scenarios and 293 steps pass (21 HU-03 scenarios including the @e2e happy path plus health/HU-01/HU-02 regressions), gosec reports 0 issues and both govulncheck runs report no affecting vulnerabilities; only the pre-existing API8 medium finding remains and no critical/high finding was introduced by HU-03.",
  "artifacts": [
    "specs/003-record-update/qa-report.md",
    "hubx-bender/features/HU03_record-update.feature",
    "hubx-bender/features/steps/HU03_record-update.steps.ts",
    "hubx-bender/features/steps/common.steps.ts"
  ],
  "evidence": [
    {
      "command": "make test-integration",
      "result": "pass",
      "excerpt": "1 hook (1 passed)\n46 scenarios (46 passed)\n293 steps (293 passed)\n0m 31.713s (0m 31.380s executing your code)"
    },
    {
      "command": "gosec ./...",
      "result": "pass",
      "excerpt": "Results:\n\n\nSummary:\n  Gosec  : dev\n  Files  : 66\n  Lines  : 6454\n  Nosec  : 1\n  Issues : 0"
    },
    {
      "command": "govulncheck -C hubx-api ./...",
      "result": "pass",
      "excerpt": "=== Symbol Results ===\n\nNo vulnerabilities found.\n\nYour code is affected by 0 vulnerabilities.\nThis scan also found 0 vulnerabilities in packages you import and 1\nvulnerability in modules you require, but your code doesn't appear to call these\nvulnerabilities."
    },
    {
      "command": "govulncheck -C hubx-gw ./...",
      "result": "pass",
      "excerpt": "No vulnerabilities found."
    }
  ],
  "blockers": [],
  "findings": [
    {
      "severity": "medium",
      "owasp": "API8",
      "description": "Pre-existing: gateway security headers, restrictive CORS and rate limiting are still pending (memory.md), scheduled for its own story. Not introduced or aggravated by HU-03; gosec reports 0 issues and both govulncheck runs report no affecting vulnerabilities.",
      "location": "hubx-gw/internal/middleware/; memory.md (Pending)"
    }
  ]
}
```

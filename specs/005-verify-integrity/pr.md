## Story
HU-05 Verify the integrity of a shipment's chain. Spec: `specs/005-verify-integrity/`

Release branch: `release/v0.4.0-ledger`

A signed-in participant enters a shipment code and verifies that its traceability chain has not been altered.
Verification recomputes every update's fingerprint from the recorded data using the existing HU-03 definition and
compares it with the stored fingerprint and the link to the previous update. An intact chain reports how many updates
were checked (zero for a shipment without updates); an altered chain reports "not intact" and the position of the
first altered update. Operators verify any shipment, customers their own, ground partners the ones assigned to them;
an unknown code and a shipment the caller cannot see give the identical "No shipment exists with that code" answer
(cross-cutting rule 4). New route `GET /api/shipments/{code}/integrity` (`contracts/hu05-verify-integrity.md`),
read-only, no migration, no gateway change, no new dependency.

## Evidence
- make test-integration (qa-seguridad verify, 2026-11-24): `59 scenarios (59 passed)`, `382 steps (382 passed)`,
  `0m 48.906s`, exit 0 (8 HU-05 scenarios + 51 regression). The tamper scenario logged
  `Tampered update 2 of HBX-589C5339 to stage "Out for delivery"` and reported `intact: false, firstAlteredUpdate: 2`.
  Red before implementation: HU-05 scenarios failed with 404 on the missing route `/api/shipments/{code}/integrity`.
  Fix after the HU-04 base merge: two Gherkin phrases (`^a shipment has no updates recorded$`,
  `^they are told that no shipment exists with that code$`) were defined verbatim in both HU-04 and HU-05 step files
  and became ambiguous under strict mode; qa-seguridad consolidated them into `features/steps/common.steps.ts`, with
  the 404 step dispatching on `response.url()` (flat `{error, message}` body for `/integrity`, nested body for
  `/traceability`), honoring both contracts.
- go test / npm test (rerun by hubx-lead at 5ffb09d):
  - `go -C hubx-api test -count=1 ./...`: all test packages `ok` (auth, config, handlers/auth, handlers/health,
    handlers/shipment, ledger, migrate, shipment).
  - `go -C hubx-api build ./...` and `go -C hubx-api vet ./...`: exit 0.
  - `go -C hubx-gw test -count=1 ./...`: `ok hubx-gw/internal/middleware/test`, `ok hubx-gw/internal/proxy/test`;
    `go -C hubx-gw build ./...` and `vet ./...` exit 0.
  - `npm --prefix hubx-mfe run test`: `Test Files  11 passed (11)`, `Tests  85 passed (85)`.
- gosec / govulncheck:
  - `gosec ./...` hubx-api: `Files : 55`, `Lines : 5218`, `Nosec : 1`, `Issues : 0`; hubx-gw: `Files : 3`,
    `Lines : 294`, `Nosec : 0`, `Issues : 0`.
  - `govulncheck -C hubx-api ./...`: `No vulnerabilities found.` `Your code is affected by 0 vulnerabilities.`
    (1 vulnerability in a required module, not called). `govulncheck -C hubx-gw ./...`: `No vulnerabilities found.`
  - `make agents-zones`: `agent zones ok (origin/main..HEAD)`.
- deploy (devops, staging, 2026-11-24): `specs/005-verify-integrity/deploy.md`. Run
  https://github.com/JairoHGomezCastillo/hubx/actions/runs/45493671765, commit 5ffb09d, environment staging. hubx-api,
  hubx-gw and hubx-mfe all `live`; migrations `up to date` (baseline 0001, no new migration); smoke
  `57 scenarios (57 passed)`. Lead curl checks: `https://hubx-gw-stg.onrender.com/health` →
  `{"service":"hubx-gw","status":"ok"}`, `/api/health` → `{"status":"ok","service":"hubx-api"}`,
  `https://hubx-mfe-stg.onrender.com/` → 200.
- speckit.analyze (spec, plan, tasks, contract against the code): FR-001..FR-010 each map to code and a passing
  scenario; the plan matches the implementation file by file (`ledger.Verify`, `shipment.Verifier`,
  `handlers/shipment/integrity_handler.go`, the MFE `features/integrity` view). Deviations:
  1. HU-03 microsecond clarification: `contracts/hu03-record-update.md` gains 5 lines (and `ledger` truncates the
     update instant to microseconds before the fingerprint) because PostgreSQL `timestamptz` keeps only microseconds;
     without truncation a round-tripped intact chain would read as altered. Recorded in `tasks.md`/`spec.md`
     clarifications and the `[agent:hubx-lead]` commit; a contract is a lead-owned path.
  2. The two ambiguous shared steps were moved to `common.steps.ts` (see test-integration above); test change only,
     product unchanged.
  3. `@e2e` setup: the Given builds the three-update shipment through the API (gateway) and only When/Then run in the
     browser (HU-03/HU-04 precedent, accepted by the human).
  4. Generated mocks (`verifier_mock.go`) are committed in backend-go's delivery; the human ran
     `go -C hubx-api generate ./...` when the sub-agent's shell was denied (exception like HU-01/HU-04).
  5. `tasks.md` checkboxes were left unchecked by the sub-agents; the closing `[agent:hubx-lead]` commit reconciles
     them against the cited evidence.
- qa-report.md: `specs/005-verify-integrity/qa-report.md` (verdict PASSED; only open finding is the pre-existing
  gateway one: security headers, CORS and rate limiting, medium, needs its own story).

## Checklist
- [x] Every backlog scenario has a test (`hubx-bender/features/HU05_verify-integrity.feature`, Gherkin verbatim +
      one `@e2e`)
- [x] No secrets, no cross-service imports, MFE only via /api
- [x] Only each agent's own paths changed (`make agents-zones` ok; diffstat attached)
- [x] Backlog status updated for this story only
- [x] deploy.md recorded (staging deploy validated before this PR)

Commits (one per accepted delivery): f07ca56 `[agent:hubx-lead]` spec, plan, tasks, integrity contract, HU-03
microsecond clarification; c2205ec `[agent:qa-seguridad]` red (feature, steps, DB tamper helper); 2dea05b
`[agent:backend-go]` integrity route, ledger List/Verify, shipment Verifier, microsecond truncation (with generated
mocks); 4b8b350 `[agent:frontend-react]` verify view, service and dashboard link; 5ffb09d
`[agent:qa-seguridad]` verify + shared steps + qa-report; closing `[agent:hubx-lead]` commit with tasks, backlog,
memory, deploy.md and this file. devops validated the staging deploy (no commit; deploy.md written for the lead to
commit).

<details><summary>git diff --stat</summary>

```
$ git diff --stat dbb383c   # merge base with release/v0.4.0-ledger
 contracts/hu03-record-update.md                    |   5 +-
 contracts/hu05-verify-integrity.md                 |  84 +++++++
 hubx-api/cmd/api/main.go                           |   4 +
 .../handlers/shipment/integrity_handler.go         | 110 +++++++++
 .../handlers/shipment/mocks/verifier_mock.go       |  57 +++++
 .../shipment/test/integrity_handler_test.go        | 238 +++++++++++++++++++
 hubx-api/internal/ledger/postgres.go               |   5 +-
 hubx-api/internal/ledger/test/helpers_test.go      |  39 ++++
 hubx-api/internal/ledger/test/list_test.go         | 126 ++++++++++
 hubx-api/internal/ledger/test/postgres_test.go     |  20 +-
 hubx-api/internal/ledger/test/truncate_test.go     |  95 ++++++++
 hubx-api/internal/ledger/test/verify_test.go       | 255 ++++++++++++++++++++
 hubx-api/internal/ledger/verify.go                 |  78 +++++++
 hubx-api/internal/shipment/errors.go               |   3 +-
 hubx-api/internal/shipment/mocks/verifier_mock.go  |  57 +++++
 hubx-api/internal/shipment/test/verifier_test.go   | 196 ++++++++++++++++
 hubx-api/internal/shipment/verifier.go             |  62 +++++
 hubx-bender/features/HU05_verify-integrity.feature |  51 ++++
 .../features/steps/HU04_view-traceability.steps.ts |  14 +-
 .../features/steps/HU05_verify-integrity.steps.ts  | 131 +++++++++++
 hubx-bender/features/steps/common.steps.ts         |  27 +++
 hubx-bender/features/support/tamper.ts             | 102 ++++++++
 hubx-mfe/src/App.tsx                               |  10 +-
 .../features/integrity/VerifyIntegrity.test.tsx    | 163 +++++++++++++
 .../src/features/integrity/VerifyIntegrity.tsx     | 125 ++++++++++
 .../integrity/VerifyIntegrityContainer.tsx         |  56 +++++
 hubx-mfe/src/pages/Dashboard.tsx                   |  23 ++
 hubx-mfe/src/pages/VerifyIntegrity.tsx             |  35 +++
 hubx-mfe/src/services/integrityService.test.ts     |  74 ++++++
 hubx-mfe/src/services/integrityService.ts          |  32 +++
 specs/005-verify-integrity/plan.md                 | 114 +++++++++
 specs/005-verify-integrity/qa-report.md            | 259 +++++++++++++++++++++
 specs/005-verify-integrity/quickstart.md           |  20 ++
 specs/005-verify-integrity/spec.md                 | 175 ++++++++++++++
 specs/005-verify-integrity/tasks.md                |  84 +++++++
 35 files changed, 2910 insertions(+), 19 deletions(-)
```
`deploy.md` and this `pr.md` are added by the closing lead commit on top of the stat above.
</details>

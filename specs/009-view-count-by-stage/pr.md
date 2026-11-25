## Story
HU-09 View the shipment count by stage. Spec: `specs/009-view-count-by-stage/`

Release branch: `release/v0.5.0-view-count-by-stage`

An operator opens "Shipment count by stage" and sees the six stages in journey order, each with its total (zero when
empty). Ground partners and customers are refused with "You do not have permission to see that information" and get no
totals. Endpoint `GET /api/shipments/stage-counts` (`contracts/hu09-stage-counts.md`); no migration, no new dependency,
no gateway change, no ledger access.

## Evidence
- make test-integration (qa-seguridad, 9977730): pass 1 `63 scenarios (63 passed)`, `407 steps (407 passed)`;
  pass 2 (`@empty-db`) `2 scenarios (2 passed)`, `12 steps (12 passed)`. All six HU-09 scenarios green, including the
  `@e2e` "Count by stage" in Chromium.
- go test (lead rerun): `go -C hubx-api test ./...` all `ok` (incl. `internal/shipment/test`,
  `internal/handlers/shipment/test`); `go -C hubx-api vet ./...` clean; `go -C hubx-gw test ./...` `ok`.
- npm test (lead rerun): `Test Files 15 passed (15)`, `Tests 103 passed (103)`; `npm --prefix hubx-mfe run build`
  `✓ built`.
- gosec / govulncheck (qa-seguridad bd9956c, lead rerun): `gosec ./...` `Files: 64, Lines: 5859, Nosec: 1, Issues: 0`;
  `govulncheck -C hubx-api ./...` `Your code is affected by 0 vulnerabilities` (1 module-level, GO-2026-5932 in
  golang.org/x/crypto openpgp, not called); `govulncheck -C hubx-gw ./...` `No vulnerabilities found.`
- speckit.analyze (lead, spec/plan/tasks vs code):
  1. Empty-state scenarios: "No shipments in HUBX" and "Stages without shipments" cannot hold on the shared seeded
     database. Human decision (option A): they are tagged `@empty-db` and `scripts/test-integration.sh` runs them in a
     second pass after `TRUNCATE updates, shipments` in the isolated `hubx-it` database (devops, 2cca4d5). Steps never
     write to the database. The plan expected no OPS change.
  2. Order: devops worked before the QA red because the red needed the empty pass.
  3. `stage_counts_errors.go` was not created: the 403/500 mapping is inline in `stage_counts_handler.go`.
  4. backend-go's `go fmt ./...` reformatted 15 pre-existing hubx-api files (import order and whitespace only:
     `git diff -w` leaves 1 import reorder and 1 trailing blank line). Accepted in 1c99e21.
  5. The MFE added `src/pages/StageCounts.tsx` and a route in `App.tsx`, not listed in the plan. The first view needed a
     button click to load; QA verify caught it in the `@e2e` and 1cbd209 loads on open.
  6. T022 (deployment validation) skipped by human decision; no staging deploy or smoke for this story.
- qa-report.md: `specs/009-view-count-by-stage/qa-report.md`

## Checklist
- [x] Every backlog scenario has a test
- [x] No secrets, no cross-service imports, MFE only via /api
- [x] Only my own paths changed (git diff --stat attached)
- [x] Backlog status updated for this story only

## Risks
- `release/v0.5.0` and `release/v0.5.0-view-count-by-stage` both map to `v0.5.0`; whichever merges into `main` second
  fails `release.yml` (`tag v0.5.0 already exists`). Decide which one ships as v0.5.0 before merging.

<details><summary>git diff --stat</summary>

```
 contracts/hu09-stage-counts.md                     | 101 ++++++
 hubx-api/cmd/api/main.go                           |   5 +
 hubx-api/internal/auth/jwt.go                      |  10 +-
 hubx-api/internal/auth/repository.go               |   8 +-
 hubx-api/internal/auth/service.go                  |  14 +-
 hubx-api/internal/auth/test/jwt_test.go            |  14 +-
 hubx-api/internal/auth/test/service_test.go        |   4 +-
 hubx-api/internal/config/config.go                 |  16 +-
 hubx-api/internal/handlers/auth/handler.go         |  10 +-
 hubx-api/internal/handlers/auth/middleware.go      |   8 +-
 hubx-api/internal/handlers/auth/register.go        |   8 +-
 .../internal/handlers/auth/test/handler_test.go    |  16 +-
 .../internal/handlers/auth/test/middleware_test.go |  16 +-
 .../handlers/shipment/mocks/stage_counter_mock.go  |  57 ++++
 hubx-api/internal/handlers/shipment/register.go    |  11 +
 .../handlers/shipment/stage_counts_handler.go      |  92 ++++++
 .../shipment/test/stage_counts_handler_test.go     | 265 +++++++++++++++
 .../shipment/test/traceability_handler_test.go     |  10 +-
 .../handlers/shipment/traceability_handler.go      |   4 +-
 hubx-api/internal/ledger/test/postgres_test.go     |   2 +-
 hubx-api/internal/ledger/test/verify_test.go       |   1 -
 .../shipment/mocks/stage_count_repository_mock.go  |  57 ++++
 hubx-api/internal/shipment/stage_count.go          |  69 ++++
 hubx-api/internal/shipment/stage_count_postgres.go |  44 +++
 .../internal/shipment/stage_count_repository.go    |  12 +
 .../internal/shipment/test/stage_count_test.go     | 178 ++++++++++
 hubx-bender/features/HU09_stage-counts.feature     |  43 +++
 .../features/steps/HU09_stage-counts.steps.ts      | 230 +++++++++++++
 hubx-mfe/src/App.tsx                               |  10 +-
 .../src/features/stage-counts/StageCounts.test.tsx | 125 +++++++
 hubx-mfe/src/features/stage-counts/StageCounts.tsx |  92 ++++++
 .../features/stage-counts/StageCountsContainer.tsx |  16 +
 hubx-mfe/src/pages/Dashboard.tsx                   |  18 +-
 hubx-mfe/src/pages/StageCounts.tsx                 |  23 ++
 hubx-mfe/src/services/stageCountService.test.ts    | 134 ++++++++
 hubx-mfe/src/services/stageCountService.ts         |  55 ++++
 scripts/test-integration.sh                        |  45 ++-
 .../checklists/requirements.md                     |  39 +++
 specs/009-view-count-by-stage/data-model.md        | 106 ++++++
 specs/009-view-count-by-stage/plan.md              | 183 +++++++++++
 specs/009-view-count-by-stage/qa-report.md         | 365 +++++++++++++++++++++
 specs/009-view-count-by-stage/quickstart.md        |  94 ++++++
 specs/009-view-count-by-stage/research.md          |  87 +++++
 specs/009-view-count-by-stage/spec.md              | 135 ++++++++
 specs/009-view-count-by-stage/tasks.md             | 163 +++++++++
 45 files changed, 2909 insertions(+), 86 deletions(-)
```

Plus the closing commit (pr.md, tasks.md, backlog.md, memory.md, README.md).
</details>

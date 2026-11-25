## Story
HU-04 View a shipment's traceability. Spec: `specs/004-view-traceability/`

Release branch: `release/v0.4.0-ledger`

A signed-in participant enters a shipment code and sees the current stage and every update, oldest first, with stage,
responsible party and date. Operators see any shipment, customers their own, ground partners the ones assigned to
them; any other code gets the same "No shipment exists with that code." answer. New route
`GET /api/shipments/{code}/traceability` (`contracts/hu04-view-traceability.md`), read-only, no migration, no gateway
change, no new dependency.

## Evidence
- make test-integration (run by hubx-lead with human approval at 8976cb4): `51 scenarios (51 passed)`,
  `328 steps (328 passed)`, `0m 36.937s`, exit 0. Red before implementation: `51 scenarios (43 passed, 8 failed)`,
  HU-04 scenarios only. First run after the backend fix: `51 scenarios (46 passed, 5 failed)`, all five test-side,
  fixed by qa-seguridad in 8976cb4.
- go test / npm test:
  - `go -C hubx-api test -count=1 ./...`: 8 test packages `ok` (auth, config, handlers/auth, handlers/health,
    handlers/shipment, ledger, migrate, shipment). Red before the fix: `traceability_test.go:6:2: "reflect" imported
    and not used` and `TestHU04_List/returns_three_updates_ordered_by_sequence_ascending` (empty `ShipmentID`).
  - `go -C hubx-api build ./...` and `go -C hubx-api vet ./...`: exit 0.
  - `go -C hubx-gw test -count=1 ./...`: `ok hubx-gw/internal/middleware/test`, `ok hubx-gw/internal/proxy/test`.
  - `npm --prefix hubx-mfe run test`: `Test Files  9 passed (9)`, `Tests  74 passed (74)`.
- gosec / govulncheck:
  - `gosec ./...` hubx-api: `Files : 50`, `Lines : 4846`, `Nosec : 1`, `Issues : 0`; hubx-gw: `Files : 3`,
    `Lines : 294`, `Nosec : 0`, `Issues : 0`.
  - `govulncheck -C hubx-api ./...`: `No vulnerabilities found.` `Your code is affected by 0 vulnerabilities.`
    (1 vulnerability in a required module, not called). `govulncheck -C hubx-gw ./...`: `No vulnerabilities found.`
  - `make agents-zones`: `agent zones ok (origin/main..HEAD)`.
- speckit.analyze (spec, plan, tasks, contract against the code): every FR-001..FR-009 maps to code and a passing
  scenario; plan matches the implementation file by file. Deviations:
  1. `@e2e` setup: the Given builds the three-update shipment through the API (gateway) and only When/Then run in the
     browser. Accepted by the human (three UI sign-ins would be brittle; HU-03 precedent).
  2. Generated mocks: the human ran `go -C hubx-api generate ./...` (exit 0) because backend-go's shell was denied;
     the files are committed in backend-go's delivery (206dbac), exception like HU-01.
  3. Verify commands (`make test-integration`, unit tests, gosec, govulncheck) were run by hubx-lead for qa-seguridad,
     with human approval; `qa-report.md` was written by qa-seguridad.
  4. HU-03 `@e2e` step changed: HU-04's second "Shipment code" field made `getByLabel(/shipment code/i)` ambiguous;
     the step is now scoped to the "Record an update" region (8976cb4). Test change only; product unchanged.
  5. `TraceabilityReader.Read` falls back to the party id as name (and an empty role) when a party id is missing from
     the lookup (`hubx-api/internal/shipment/traceability.go`). Neither plan nor contract describes this case. No
     scenario exercises it; flagged for review.
  6. `hubx-api/internal/handlers/shipment/mocks/traceability_handler_mock.go` comes from a `//go:generate` directive
     on a file with no interfaces, so the mock is empty (10 lines). Harmless; not in the plan.
  7. Dates in `spec.md`, `plan.md` and the contract header said 2026-11-25; corrected to 2026-11-24, the day of the
     runs.
- qa-report.md: `specs/004-view-traceability/qa-report.md` (verdict green; only open finding is pre-existing
  F-API8-01, medium: gateway security headers, CORS and rate limiting).

## Checklist
- [x] Every backlog scenario has a test (`hubx-bender/features/HU04_view-traceability.feature`, Gherkin verbatim +
      one `@e2e`)
- [x] No secrets, no cross-service imports, MFE only via /api
- [x] Only my own paths changed (git diff --stat attached; `make agents-zones` ok)
- [x] Backlog status updated for this story only

Commits (one per accepted delivery): e994c12 `[agent:hubx-lead]` spec, plan, tasks, contract; 349be06 and ebb64b9
`[agent:qa-seguridad]` red; cd4c1b1 `[agent:frontend-react]`; 206dbac `[agent:backend-go]` (with generated mocks);
8976cb4 `[agent:qa-seguridad]` fixture fix; a2f31d4 `[agent:qa-seguridad]` qa-report; closing `[agent:hubx-lead]`
commit with tasks, backlog, memory and this file. devops confirmed no CI/compose/deploy change (no commit).

<details><summary>git diff --stat</summary>

```
$ git diff --stat 34f4982   # merge base with release/v0.4.0-ledger
 backlog/backlog.md                         |   2 +-
 contracts/hu04-view-traceability.md        |  93 ++++
 hubx-api/cmd/api/main.go                   |   6 +
 .../shipment/mocks/repository_mock.go      |  39 ++
 .../mocks/traceability_handler_mock.go     |  10 +
 .../mocks/traceability_reader_mock.go      |  57 +++
 .../test/traceability_handler_test.go      | 310 ++++++++++++
 .../shipment/traceability_handler.go       | 140 ++++++
 .../internal/ledger/mocks/postgres_mock.go | 104 ++++
 .../ledger/mocks/repository_mock.go        |  15 +
 hubx-api/internal/ledger/postgres.go       |  66 +++
 hubx-api/internal/ledger/repository.go     |   7 +-
 .../internal/ledger/test/postgres_test.go  | 179 +++++++
 hubx-api/internal/shipment/errors.go       |   3 +
 .../mocks/ledger_repository_mock.go        |  87 ++++
 .../shipment/mocks/repository_mock.go      |  39 ++
 .../shipment/mocks/traceability_mock.go    |  57 +++
 hubx-api/internal/shipment/repository.go   |  12 +
 .../shipment/test/traceability_test.go     | 392 ++++++++++++++++
 hubx-api/internal/shipment/traceability.go | 137 ++++++
 .../internal/shipment/update_postgres.go   |  30 ++
 .../HU04_view-traceability.feature         |  48 ++
 .../steps/HU03_record-update.steps.ts      |  12 +-
 .../steps/HU04_view-traceability.steps.ts  | 197 ++++++++
 hubx-bender/features/steps/common.steps.ts |  12 +-
 hubx-mfe/src/api/client.test.ts            | 105 ++++-
 hubx-mfe/src/api/client.ts                 |  75 +++
 .../traceability/ViewTraceability.test.tsx | 181 +++++++
 .../traceability/ViewTraceability.tsx      | 135 ++++++
 .../ViewTraceabilityContainer.tsx          |  16 +
 hubx-mfe/src/pages/Dashboard.test.tsx      |  29 +-
 hubx-mfe/src/pages/Dashboard.tsx           |   3 +
 memory.md                                  |  12 +
 specs/004-view-traceability/plan.md        | 105 +++++
 specs/004-view-traceability/pr.md          | 109 +++++
 specs/004-view-traceability/qa-report.md   | 288 ++++++++++++
 specs/004-view-traceability/spec.md        | 148 ++++++
 specs/004-view-traceability/tasks.md       |  90 ++++
 38 files changed, 3332 insertions(+), 18 deletions(-)
```
The release branch advanced after this branch was cut (PR #28, `tooling/review-story`, eec68e9); its files
(`.claude/skills/review-story/SKILL.md`, `scripts/review-story.sh`, one line of `AGENTS.md`) are not part of this
story and are untouched here.
</details>

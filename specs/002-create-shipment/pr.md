## Story
HU-02 Create a shipment. Spec: `specs/002-create-shipment/`. Release: v0.3.0 (`release/v0.3.0-create-shipment`).

Operators create a shipment (origin, destination, flight, weight, customer by email) through
`POST /api/shipments` (`contracts/shipments-create.md`). The API assigns a unique `HBX-XXXXXXXX` code, the stage is
"Created", and no update is recorded on creation. Ground partners and customers get 403 before their data is examined.
The MFE adds a "Create shipment" page for operators. No migration (reuses `shipments` from 0001).

Clarification Q2 (human decision, 2026-11-24, Option A): a weight with more than one decimal place (e.g. 0.15 kg) is
rejected with the weight-range message ("The weight must be between 0.1 and 5000 kg"). It is not deduced from the
database column precision. The code already implemented it; no code or test change.

## Evidence
- make test-integration (full stack: PostgreSQL, API, gateway, MFE; API scenarios and `@e2e` in Chromium):
  ```
  1 hook (1 passed)
  25 scenarios (25 passed)
  146 steps (146 passed)
  0m 6.921s (0m 6.871s executing your code)
  ```
- go test: `go -C hubx-api test ./...` all `ok` (auth, config, handlers/auth, handlers/health, handlers/shipment,
  migrate, shipment); `go -C hubx-gw test ./...` all `ok` (middleware, proxy).
- npm test: `npm --prefix hubx-mfe run test` -> `Test Files 5 passed (5)`, `Tests 43 passed (43)`.
- gosec: `gosec -quiet ./...` in hubx-api and hubx-gw: no output (0 issues).
- govulncheck: hubx-api "Your code is affected by 0 vulnerabilities" (1 in a required module, not called);
  hubx-gw "No vulnerabilities found."
- make agents-zones: `agent zones ok (origin/release/v0.3.0-create-shipment..HEAD)`.
- speckit.analyze: deviations recorded in `specs/002-create-shipment/tasks.md` ("Deviations" 1-10). Main ones:
  partial TDD red evidence for the API; "not created" asserted by status and missing `code` (no read endpoint until
  HU-04); gateway-secret and body-cap checks verified by code review only.
- qa-report.md: `specs/002-create-shipment/qa-report.md`. Open medium finding API8 (pre-existing, out of scope for
  HU-02): hubx-gw does not yet apply security headers, restrictive CORS or rate limiting (Constitution VI). Tracked
  in `memory.md` Pending; needs its own story. One low finding (validation order of a string `weightKg`), accepted.

## Checklist
- [x] Every backlog scenario has a test (`hubx-bender/features/HU02_create_shipment.feature`, 12 examples + 1 `@e2e`)
- [x] No secrets, no cross-service imports, MFE only via /api
- [x] Only my own paths changed (git diff --stat attached; `make agents-zones` ok)
- [x] Backlog status updated for this story only

## Notes for the reviewer
- Q1 (customer indicated by email) and Q3 (English UI) were resolved by hubx-lead from existing artifacts; please
  confirm.
- The branch was not merged with `origin/release/v0.3.0-create-shipment` locally (the lead's `git merge` is denied by
  its permission profile). The release head 52837ed has the same tree as this branch's base 3f43b02
  (`git diff --stat 3f43b02 origin/release/v0.3.0-create-shipment` is empty), so there is nothing to integrate; use
  "Update branch" in GitHub if a merge commit is wanted.

<details><summary>git diff --stat</summary>

```
 backlog/backlog.md                            |   2 +-
 contracts/shipments-create.md                 | 113 ++++
 hubx-api/cmd/api/main.go                      |   5 +-
 .../internal/handlers/shipment/handler.go     | 176 ++++++
 .../internal/handlers/shipment/register.go    |  24 +
 .../handlers/shipment/test/handler_test.go    | 429 ++++++++++++++
 hubx-api/internal/shipment/code.go            |  37 ++
 hubx-api/internal/shipment/errors.go          |  26 +
 hubx-api/internal/shipment/mocks/code_mock.go |  54 ++
 .../shipment/mocks/repository_mock.go         |  71 +++
 .../internal/shipment/mocks/service_mock.go   |  57 ++
 hubx-api/internal/shipment/repository.go      |  85 +++
 hubx-api/internal/shipment/service.go         | 121 ++++
 hubx-api/internal/shipment/shipment.go        |  31 ++
 hubx-api/internal/shipment/test/code_test.go  |  36 ++
 .../internal/shipment/test/service_test.go    | 526 ++++++++++++++++++
 .../features/HU02_create_shipment.feature     |  62 +++
 .../steps/HU02_create_shipment.steps.ts       | 372 +++++++++++++
 hubx-mfe/src/App.tsx                          |  10 +-
 hubx-mfe/src/pages/CreateShipment.test.tsx    | 278 +++++++++
 hubx-mfe/src/pages/CreateShipment.tsx         | 287 ++++++++++
 hubx-mfe/src/pages/Dashboard.tsx              |  19 +
 hubx-mfe/src/services/shipmentService.test.ts | 237 ++++++++
 hubx-mfe/src/services/shipmentService.ts      |  52 ++
 memory.md                                     |  11 +
 specs/002-create-shipment/data-model.md       |  45 ++
 specs/002-create-shipment/plan.md             | 104 ++++
 specs/002-create-shipment/qa-report.md        | 314 +++++++++++
 specs/002-create-shipment/quickstart.md       |  26 +
 specs/002-create-shipment/spec.md             | 151 +++++
 specs/002-create-shipment/tasks.md            | 123 ++++
 31 files changed, 3880 insertions(+), 4 deletions(-)
```
</details>

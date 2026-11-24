## Story
HU-03 Record a shipment update. Spec: `specs/003-record-update/` (branch `003-record-update-sync` -> `main`).

## Evidence
- make test-integration: `1 hook (1 passed)` / `46 scenarios (46 passed)` / `293 steps (293 passed)` / `0m 31.713s (0m 31.380s executing your code)` — the 21 HU-03 scenarios (including the `@e2e` happy path) plus health/HU-01/HU-02 regressions. Two earlier runs also ended `46 scenarios (46 passed)`.
- go test / npm test: `go -C hubx-api test -count=1 ./...` all packages `ok` (including `internal/ledger/test 0.007s`, `internal/shipment/test 0.010s`, `internal/handlers/shipment/test 0.009s`, `internal/handlers/auth/test`); `go -C hubx-gw test -count=1 ./...` → `ok hubx-gw/internal/middleware/test 0.007s`, `ok hubx-gw/internal/proxy/test 0.009s`; `go -C hubx-api build ./...` and `go -C hubx-gw build ./...` no output; MFE production build in Docker → `#40 [mfe build 6/6] RUN npm run build (tsc -b && vite build) ... ✓ built in 321ms`; MFE unit tests (`client.test.ts`, `RecordUpdate.test.tsx`, `Dashboard.test.tsx`) included in the delivery.
- gosec / govulncheck: `gosec ./...` → `Files: 66`, `Lines: 6454`, `Nosec: 1`, `Issues: 0`; `govulncheck -C hubx-api ./...` → `No vulnerabilities found.` (0 affecting; 1 in a required module never called); `govulncheck -C hubx-gw ./...` → `No vulnerabilities found.`
- speckit.analyze: 6 deviations recorded (D1 branch name in spec/plan; D2 gateway nested `SESSION_EXPIRED` body required by `contracts/hu03-record-update.md:61`; D3 MFE tests and redirect; D4 handler file names and pending tasks; D5 expired-session scenario is spec-derived from FR-011, not backlog verbatim; D6 setup tasks), 0 critical issues, 100 % requirement coverage (17 requirements / 34 tasks). D1–D4 corrected in `8c5617d`; D5 and D6 documented as accepted/closed. API8 (gateway security headers, CORS, rate limiting) is pre-existing, medium, and not introduced or aggravated by HU-03.
- qa-report.md: `specs/003-record-update/qa-report.md`

## Checklist
- [x] Every backlog scenario has a test
- [x] No secrets, no cross-service imports, MFE only via /api
- [x] Only my own paths changed (git diff --stat attached)
- [x] Backlog status updated for this story only

<details><summary>git diff --stat</summary>

```
 backlog/backlog.md                                 |   2 +-
 contracts/hu03-record-update.md                    | 132 ++++++
 hubx-api/cmd/api/main.go                           |  12 +
 .../internal/handlers/auth/session_middleware.go   |  60 +++
 .../handlers/auth/test/session_middleware_test.go  | 122 ++++++
 hubx-api/internal/handlers/shipment/handler.go     |   2 +-
 .../handlers/shipment/mocks/handler_mock.go        |  97 +++++
 .../handlers/shipment/mocks/recorder_mock.go       |  58 +++
 .../handlers/shipment/mocks/repository_mock.go     | 179 ++++++++
 .../handlers/shipment/test/update_handler_test.go  | 349 ++++++++++++++++
 .../handlers/shipment/test/update_session_test.go  | 150 +++++++
 .../internal/handlers/shipment/update_errors.go    |  76 ++++
 .../internal/handlers/shipment/update_handler.go   | 213 ++++++++++
 hubx-api/internal/ledger/errors.go                 |  75 ++++
 hubx-api/internal/ledger/mocks/postgres_mock.go    | 264 ++++++++++++
 hubx-api/internal/ledger/mocks/repository_mock.go  |  72 +++++
 hubx-api/internal/ledger/postgres.go               | 246 +++++++++++
 hubx-api/internal/ledger/repository.go             |  13 +
 hubx-api/internal/ledger/service.go                |  62 +++
 hubx-api/internal/ledger/test/postgres_test.go     | 427 ++++++++++++++++++++
 hubx-api/internal/ledger/test/service_test.go      | 159 ++++++++++
 hubx-api/internal/ledger/test/update_test.go       | 234 +++++++++++
 hubx-api/internal/ledger/update.go                 | 101 +++++
 hubx-api/internal/shipment/mocks/postgres_mock.go  | 207 ++++++++++
 .../internal/shipment/mocks/repository_mock.go     | 108 +++++
 hubx-api/internal/shipment/recorder.go             | 116 ++++++
 hubx-api/internal/shipment/repository.go           |  20 +
 hubx-api/internal/shipment/shipment.go             |  24 +-
 hubx-api/internal/shipment/stage.go                |  74 ++++
 hubx-api/internal/shipment/test/postgres_test.go   | 448 +++++++++++++++++++++
 hubx-api/internal/shipment/test/recorder_test.go   | 439 ++++++++++++++++++++
 hubx-api/internal/shipment/update_errors.go        |  22 +
 hubx-api/internal/shipment/update_postgres.go      | 159 ++++++++
 hubx-bender/features/HU03_record-update.feature    | 110 +++++
 hubx-bender/features/steps/HU01_sign_in.steps.ts   |   8 +
 .../features/steps/HU02_create_shipment.steps.ts   |  35 +-
 .../features/steps/HU03_record-update.steps.ts     | 215 ++++++++++
 hubx-bender/features/steps/common.steps.ts         | 269 ++++++++++++-
 hubx-gw/internal/middleware/auth.go                |  69 +++-
 .../internal/middleware/test/session_error_test.go | 162 ++++++++
 hubx-mfe/src/App.tsx                               |  38 +-
 hubx-mfe/src/api/client.test.ts                    | 133 ++++++
 hubx-mfe/src/api/client.ts                         | 143 +++++++++
 .../shipment-updates/RecordUpdate.test.tsx         | 165 ++++++++++
 .../src/features/shipment-updates/RecordUpdate.tsx | 231 +++++++++++
 .../shipment-updates/RecordUpdateContainer.tsx     |  81 ++++
 hubx-mfe/src/pages/Dashboard.test.tsx              |  49 +++
 hubx-mfe/src/pages/Dashboard.tsx                   |   3 +
 hubx-mfe/src/services/httpClient.ts                |  17 +-
 memory.md                                          |  31 +-
 specs/003-record-update/data-model.md              |  94 +++++
 specs/003-record-update/plan.md                    | 211 ++++++++++
 specs/003-record-update/pr.md                      |  80 +
 specs/003-record-update/qa-report.md               | 209 ++++++++++
 specs/003-record-update/quickstart.md              |  73 ++++
 specs/003-record-update/research.md                | 112 +++++
 specs/003-record-update/spec.md                    | 213 ++++++++++
 specs/003-record-update/tasks.md                   | 200 +++++++++
 58 files changed, 7582 insertions(+), 91 deletions(-)
```
</details>

## Story
HU-10 Download the traceability evidence. Spec: `specs/010-download-evidence/`

**Source**: backlog HU-10. **Story branch**: `010-download-evidence`. **Release branch**: `release/v0.10.0-demo`
(validated with `bash scripts/release-version.sh release/v0.10.0-demo` → `version=v0.10.0`).

**Deploy validation skipped by human decision (demo mode); staging is deployed and smoke-tested automatically when this PR is merged.**

## Evidence
- make test-integration: `1 hook (1 passed) / 66 scenarios (66 passed) / 424 steps (424 passed) / 1m 5.472s (1m 5.232s executing your code) / INTEGRATION_EXIT=0`. All 7 HU-10 scenarios passed, including the `@e2e` browser download of `evidence-HBX-6118F6AD.pdf` (3814 bytes, `%PDF-` magic, 3 chained fingerprints). No older story regressed.
- go test / npm test: `go -C hubx-api test ./...` → all packages `ok`, `EXIT=0`; `go -C hubx-gw test ./...` → `ok`, `EXIT=0`; `go -C hubx-api build ./...` → `EXIT=0`; `go -C hubx-api vet ./...` and `go -C hubx-gw vet ./...` → `EXIT=0`; `go -C hubx-api generate ./...` → `EXIT=0` (gomock mocks generated). `npm --prefix hubx-mfe run typecheck` → `EXIT=0`; `npm --prefix hubx-mfe run test` → `15 files passed, 106 tests passed`; `npm --prefix hubx-mfe run lint` → `EXIT=0` (4 warnings, all in pre-existing files); `npm --prefix hubx-mfe run build` → `EXIT=0`.
- gosec / govulncheck: `gosec ./...` in `hubx-api` → `Issues : 0`, `GOSEC_API_EXIT=0`; `gosec ./...` in `hubx-gw` → `Issues : 0`, `GOSEC_GW_EXIT=0`; `govulncheck -C hubx-api ./...` → `No vulnerabilities found`, `GOVULN_API_EXIT=0`; `govulncheck -C hubx-gw ./...` → `No vulnerabilities found`, `GOVULN_GW_EXIT=0`. The two findings originally reported in HU-10 code (HIGH G115 at `pdf.go:150`, MEDIUM G705 at `handler.go:81`) were closed structurally in commit `f81a551` without `#nosec` suppression and without changing the PDF bytes.
- speckit.analyze: 13/13 FR and 4/4 buildable SC have tasks; 30 tasks; 0 duplications, 0 ambiguities, 0 critical issues introduced by HU-10. Two LOW deviations were corrected in the closing commit `6d4fc17` (`spec.md` status, contract path in the `plan.md` tree). Remaining recorded deviations: the pre-existing gateway gap (constitution VI: security headers, restrictive CORS and rate limiting, `hubx-gw/cmd/gw/main.go:44-70`, already in `memory.md → Pending`) and the human-ordered demo-mode skip of deploy validation.
- qa-report.md: `specs/010-download-evidence/qa-report.md` (final verify on `f81a551`: green suite, zero security findings, one pre-existing medium finding).
- Agent zones: `node scripts/check-agent-zones.mjs origin/main` → `agent zones ok (origin/main..HEAD)`; no CI, Dockerfile, compose, Makefile or infrastructure file changed.

## Checklist
- [x] Every backlog scenario has a test (6 scenarios + the two-role Scenario Outline, Gherkin verbatim, plus one `@e2e` happy path).
- [x] No secrets, no cross-service imports, MFE only via `/api` through the gateway.
- [x] Only my own paths changed (git diff --stat attached); each `[agent:<role>]` commit is confined to that role's zone.
- [x] Backlog status updated for this story only (`In review (PR pending)`).

## Risks and human decisions
- Demo mode: the human explicitly asked to skip the deploy validation of lead step 6d. No `deploy.md` and no devops delegation; staging is deployed and smoke-tested automatically when this PR is merged.
- `origin/release/v0.10.0-demo` stayed at `853944c` while the story branch was cut from the local release at `8c47690` (v0.4.0 base, 44 commits ahead). GitHub rejected the release update with `GH013: Changes must be made through a pull request`, and no agent pushes to `release/*`; this PR therefore also shows those pre-existing commits. The HU-10 changes themselves are the 48 files in the diff below.
- Permission change: the human authorized enabling the shell layer in the five `.opencode/agents/*.md` files (commit `a10ff6e`, untagged, no agent zone violation); `make agents-sync` and `make agents-check` passed.

<details><summary>git diff --stat</summary>

```
$ git diff --stat 8c47690..HEAD
 .opencode/agents/backend-go.md                     |   5 +
 .opencode/agents/devops.md                         |   5 +
 .opencode/agents/frontend-react.md                 |   5 +
 .opencode/agents/hubx-lead.md                      |   5 +
 .opencode/agents/qa-seguridad.md                   |   5 +
 README.md                                          |   9 +-
 backlog/backlog.md                                 |   6 +-
 contracts/hu10-download-evidence.md                |  77 +++++
 docs/GUIA-EQUIPO-HUBX.md                           |   2 +
 docs/sdd-agent-blockers.md                         |   1 +
 hubx-api/cmd/api/main.go                           |   7 +
 hubx-api/internal/evidence/errors.go               |  13 +
 hubx-api/internal/evidence/evidence.go             |  53 ++++
 hubx-api/internal/evidence/mocks/ledger_repository_mock.go |  87 +++++
 hubx-api/internal/evidence/mocks/reader_mock.go    |  58 ++++
 hubx-api/internal/evidence/mocks/shipment_repository_mock.go | 218 +++++++++
 hubx-api/internal/evidence/pdf.go                  | 170 +++++++++
 hubx-api/internal/evidence/reader.go               | 144 +++++
 hubx-api/internal/evidence/test/pdf_test.go        | 241 +++++++++++
 hubx-api/internal/evidence/test/reader_test.go     | 258 ++++++++++++++++
 hubx-api/internal/handlers/evidence/handler.go     | 130 ++++++++++
 hubx-api/internal/handlers/evidence/mocks/handler_mock.go |  98 +++++
 hubx-api/internal/handlers/evidence/test/handler_test.go | 239 ++++++++++++++
 hubx-api/internal/shipment/recorder.go             |   9 +-
 hubx-api/internal/shipment/test/postgres_test.go   |  55 +++-
 hubx-api/internal/shipment/traceability.go         |   2 +-
 hubx-api/internal/shipment/update_postgres.go      |  32 +-
 hubx-api/internal/shipment/verifier.go             |   2 +-
 hubx-bender/features/HU10_download-evidence.feature |  46 +++
 hubx-bender/features/steps/HU10_download-evidence.steps.ts | 344 +++++++++++++++++++++
 hubx-bender/features/steps/common.steps.ts         |  20 +-
 hubx-bender/features/support/hooks.ts              |   3 +-
 hubx-mfe/src/App.tsx                               |  10 +-
 hubx-mfe/src/features/evidence/DownloadEvidence.test.tsx | 134 ++++++++
 hubx-mfe/src/features/evidence/DownloadEvidence.tsx | 129 ++++++++
 hubx-mfe/src/features/evidence/DownloadEvidenceContainer.tsx |  44 +++
 hubx-mfe/src/pages/Dashboard.tsx                   |  18 +-
 hubx-mfe/src/pages/DownloadEvidence.tsx            |  24 ++
 hubx-mfe/src/services/evidenceService.test.ts     | 170 ++++++
 hubx-mfe/src/services/evidenceService.ts           | 102 ++++++
 memory.md                                          |  23 ++
 specs/010-download-evidence/data-model.md          |  56 ++++
 specs/010-download-evidence/plan.md                | 125 ++++++++
 specs/010-download-evidence/qa-report.md           | 282 +++++++++++++++++
 specs/010-download-evidence/quickstart.md          |  37 +++
 specs/010-download-evidence/research.md            |  71 +++++
 specs/010-download-evidence/spec.md                | 152 +++++++++
 specs/010-download-evidence/tasks.md               | 223 +++++++++++++
 48 files changed, 3923 insertions(+), 26 deletions(-)
```

</details>

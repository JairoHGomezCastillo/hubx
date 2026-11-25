## Story
HU-10 Download the traceability evidence. Spec: `specs/010-download-evidence/`

**Source**: backlog HU-10. **Story branch**: `010-download-evidence`. **Release branch**: `release/v0.10.0-demo`
(validated with `bash scripts/release-version.sh release/v0.10.0-demo` → `version=v0.10.0`).

**Deploy validation skipped by human decision (demo mode); staging is deployed and smoke-tested automatically when this PR is merged.**

## Evidence
- make test-integration (re-verify on `bfdc9de` after the human-requested presentation fix): `1 hook (1 passed) / 66 scenarios (66 passed) / 424 steps (424 passed) / 1m 7.402s (1m 7.177s executing your code) / INTEGRATION_EXIT=0`. All 7 HU-10 scenarios passed, including the `@e2e` browser download of `evidence-HBX-80EDEFE3.pdf` (10631 bytes, `%PDF-` magic). A read-only parser of that downloaded PDF found all asserted phrases, `CANONICAL_BLOCKS=3`, three contiguous seven-line canonical blocks and `PRINTED_CHAIN_MATCHES_RECOMPUTED=true`. No older story regressed.
- go test / npm test: `go -C hubx-api test ./...` → all packages `ok`, `TEST_EXIT=0`; `go -C hubx-gw test ./...` → `ok`, `EXIT=0`; `go -C hubx-api build ./...` → `BUILD_EXIT=0`; `go -C hubx-api vet ./...` → `VET_EXIT=0` and `go -C hubx-gw vet ./...` → `EXIT=0`; `go -C hubx-api generate ./...` → `GENERATE_EXIT=0` (gomock mocks generated). The restyled writer added four black-box tests (`TestHU10_RenderUsesHubxVisualStyle`, `TestHU10_RenderWrapsProseButNotCanonicalLines`, `TestHU10_RenderKeepsCanonicalBlocksAsConsecutiveStrings`, `TestHU10_RenderRepeatsChromeAndNeverSplitsBlocksAcrossPages`, `TARGETED_EXIT=0`). `npm --prefix hubx-mfe run typecheck` → `EXIT=0`; `npm --prefix hubx-mfe run test` → `15 files passed, 106 tests passed`; `npm --prefix hubx-mfe run lint` → `EXIT=0` (4 warnings, all in pre-existing files); `npm --prefix hubx-mfe run build` → `EXIT=0`.
- gosec / govulncheck (re-run on `bfdc9de`): `gosec ./...` in `hubx-api` → `Files : 64`, `Lines : 6437`, `Issues : 0`, `GOSEC_API_EXIT=0`; `gosec ./...` in `hubx-gw` → `Files : 3`, `Lines : 294`, `Issues : 0`, `GOSEC_GW_EXIT=0`; `govulncheck -C hubx-api ./...` → `No vulnerabilities found`, `GOVULN_API_EXIT=0`; `govulncheck -C hubx-gw ./...` → `No vulnerabilities found`, `GOVULN_GW_EXIT=0`. The two findings originally reported in HU-10 code (HIGH G115 at `pdf.go:150`, MEDIUM G705 at `handler.go:81`) were closed structurally in `f81a551` without `#nosec` suppression; the presentation fix in `bfdc9de` kept the scans at zero.
- speckit.analyze: 13/13 FR and 4/4 buildable SC have tasks; 30 tasks at analysis time (32 after the two review-fix tasks); 0 duplications, 0 ambiguities, 0 critical issues introduced by HU-10. Two LOW deviations were corrected in `6d4fc17` (`spec.md` status, contract path in the `plan.md` tree). The presentation fix changed no contract, stage, role or permission, so the analysis has no new deviation. Remaining recorded deviations: the pre-existing gateway gap (constitution VI: security headers, restrictive CORS and rate limiting, `hubx-gw/cmd/gw/main.go:44-70`, already in `memory.md → Pending`) and the human-ordered demo-mode skip of deploy validation.
- qa-report.md: `specs/010-download-evidence/qa-report.md` (re-verified on `bfdc9de`: green suite, restyled-PDF parser evidence, zero security findings, one pre-existing medium finding).
- Agent zones: `node scripts/check-agent-zones.mjs origin/main` → `agent zones ok (origin/main..HEAD)`; no CI, Dockerfile, compose, Makefile or infrastructure file changed.

## Checklist
- [x] Every backlog scenario has a test (6 scenarios + the two-role Scenario Outline, Gherkin verbatim, plus one `@e2e` happy path).
- [x] No secrets, no cross-service imports, MFE only via `/api` through the gateway.
- [x] Only my own paths changed (git diff --stat attached); each `[agent:<role>]` commit is confined to that role's zone.
- [x] Backlog status updated for this story only (`In review (PR #40)`).

## Risks and human decisions
- Human review of PR #40 found the first PDF plain and with clipped long prose. `backend-go` restyled it in `bfdc9de` with the MFE palette and word wrapping; `qa-seguridad` re-verified it in `f764a7b`. This PR is updated with both commits; no merge was performed.
- Demo mode: the human explicitly asked to skip the deploy validation of lead step 6d. No `deploy.md` and no devops delegation; staging is deployed and smoke-tested automatically when this PR is merged.
- `origin/release/v0.10.0-demo` stayed at `853944c` while the story branch was cut from the local release at `8c47690` (v0.4.0 base, 44 commits ahead). GitHub rejected the release update with `GH013: Changes must be made through a pull request`, and no agent pushes to `release/*`; this PR therefore also shows those pre-existing commits. The HU-10 changes themselves are the 49 files in the diff below.
- Permission change: the human authorized enabling the shell layer in the five `.opencode/agents/*.md` files (commit `a10ff6e`, untagged, no agent zone violation); `make agents-sync` and `make agents-check` passed.

<details><summary>git diff --stat</summary>

```
$ git diff --stat 8c47690
 .opencode/agents/backend-go.md                     |   5 +
 .opencode/agents/devops.md                         |   5 +
 .opencode/agents/frontend-react.md                 |   5 +
 .opencode/agents/hubx-lead.md                      |   5 +
 .opencode/agents/qa-seguridad.md                   |   5 +
 README.md                                          |   9 +-
 backlog/backlog.md                                 |   6 +-
 contracts/hu10-download-evidence.md                |  77 ++++
 docs/GUIA-EQUIPO-HUBX.md                           |   2 +
 docs/sdd-agent-blockers.md                         |   1 +
 hubx-api/cmd/api/main.go                           |   7 +
 hubx-api/internal/evidence/errors.go               |  13 +
 hubx-api/internal/evidence/evidence.go             |  53 +++
 .../evidence/mocks/ledger_repository_mock.go       |  87 +++++
 hubx-api/internal/evidence/mocks/reader_mock.go    |  58 +++
 .../evidence/mocks/shipment_repository_mock.go     | 218 ++++++++++
 hubx-api/internal/evidence/pdf.go                  | 394 +++++++++++++++++++++
 hubx-api/internal/evidence/reader.go               | 144 ++++++++
 hubx-api/internal/evidence/test/pdf_test.go        | 385 ++++++++++++++++++++
 hubx-api/internal/evidence/test/reader_test.go     | 258 ++++++++++++++
 hubx-api/internal/handlers/evidence/handler.go     | 130 +++++++++++
 .../handlers/evidence/mocks/handler_mock.go        |  98 +++++
 .../handlers/evidence/test/handler_test.go         | 239 +++++++++++
 hubx-api/internal/shipment/recorder.go             |   9 +-
 hubx-api/internal/shipment/test/postgres_test.go   |  55 ++-
 hubx-api/internal/shipment/traceability.go         |   2 +-
 hubx-api/internal/shipment/update_postgres.go      |  32 +-
 hubx-api/internal/shipment/verifier.go             |   2 +-
 .../features/HU10_download-evidence.feature        |  46 +++
 .../features/steps/HU10_download-evidence.steps.ts | 344 ++++++++++++++++++
 hubx-bender/features/steps/common.steps.ts         |  20 +-
 hubx-bender/features/support/hooks.ts              |   3 +-
 hubx-mfe/src/App.tsx                               |  10 +-
 .../features/evidence/DownloadEvidence.test.tsx    | 134 ++++++++
 .../src/features/evidence/DownloadEvidence.tsx     | 129 ++++++++
 .../evidence/DownloadEvidenceContainer.tsx         |  44 +++
 hubx-mfe/src/pages/Dashboard.tsx                   |  18 +-
 hubx-mfe/src/pages/DownloadEvidence.tsx            |  24 ++
 hubx-mfe/src/services/evidenceService.test.ts      | 170 +++++++
 hubx-mfe/src/services/evidenceService.ts           | 102 +++++++
 memory.md                                          |  28 ++
 specs/010-download-evidence/data-model.md          |  56 ++++
 specs/010-download-evidence/plan.md                | 125 +++++++
 specs/010-download-evidence/pr.md                  |  85 +++++
 specs/010-download-evidence/qa-report.md           | 316 +++++++++++++++++
 specs/010-download-evidence/quickstart.md          |  37 +++
 specs/010-download-evidence/research.md            |  71 +++++
 specs/010-download-evidence/spec.md                | 152 +++++++++
 specs/010-download-evidence/tasks.md               | 254 +++++++++++++
 49 files changed, 4446 insertions(+), 26 deletions(-)
```

</details>

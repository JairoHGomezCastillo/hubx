# Tasks: Gateway perimeter hardening

Only `hubx-lead` marks a task, citing the evidence of the agent that ran it.

## Phase 1: test environment
- [x] T001 [OPS] `compose.yaml`: `gw` limits from env, `gw-limited` (profile `test`); `scripts/test-integration.sh` and `scripts/review-story.sh` start it, set 10000 on `gw` and export `BENDER_LIMITED_BASE_URL`. Evidence: 7a6d97b, `docker compose --profile test config --quiet` no error; stack exercised by the QA runs. `review-story.sh` itself was not run
- [x] T002 [OPS] `deploy/render.staging.yaml` limits; `.github/workflows/deploy.yml` excludes `@needs-limited-gateway`; note in `deploy/README.md`. Evidence: 7a6d97b, diff reviewed; workflow and Blueprint not executed

## Phase 2: red
- [x] T003 [QA] `hubx-bender/features/S011_gateway-hardening.feature` with the spec's scenarios verbatim, steps in `features/steps/S011_gateway-hardening.steps.ts`. Evidence: 94c4723
- [x] T004 [QA] `make test-integration` red: `87 scenarios (78 passed, 9 failed)`, no undefined or ambiguous steps; 2 scenarios are guards that pass in red

## Phase 3: implementation
- [x] T005 [GW] `internal/config` with tests. Evidence: 6d34d4d
- [x] T006 [GW] `internal/ratelimit` with tests (window, recovery, isolation, bound, eviction, concurrency). Evidence: 6d34d4d, `go -C hubx-gw test -race -count=1 ./...` all ok
- [x] T007 [GW] security headers, CORS and limit middlewares with tests; validated `sub` in context. Evidence: 6d34d4d
- [x] T008 [GW] proxy header cleanup, wiring in `internal/server` and `cmd/gw`, `application.properties`, `hubx-gw/AGENTS.md`. Evidence: 6d34d4d

## Phase 4: HU-09 correction (review of PR #41)
- [x] T009 [QA] dated correction in `specs/009-view-count-by-stage/qa-report.md`, additions only. Evidence: 0354ea9
- [x] T010 [QA] HU-09 `@e2e` asserts the browser totals; Gherkin untouched. Evidence: 0354ea9, green in the red and verify runs

## Phase 5: verify and close
- [x] T011 [QA] `87 scenarios (87 passed)`, `2 scenarios (2 passed)`, go tests ok, gosec `Issues : 0`, govulncheck clean. Evidence: 78bb8c1
- [x] T012 [OPS] `make agents-zones` up to 78bb8c1: `agent zones ok (origin/main..HEAD)`
- [x] T013 [LEAD] analyze, `memory.md`, `README.md`, `pr.md`
- [ ] T014 [OPS] deploy validation: PENDING (needs the push, and the two staging variables set by a human)

## Phase 6: human decisions of 2026-11-25 and extra scope
- [ ] T015 [OPS] compose sign-in default 10, `CORS_ALLOWED_ORIGINS` on `gw-limited`, `deploy/README.md` corrected
- [ ] T016 [QA] red: approved origin on `gw-limited`, none configured on the main gateway; HU-10 names statement
- [ ] T017 [GW] defaults 10 and empty, tests, `hubx-gw/AGENTS.md`
- [ ] T018 [API] HU-10 names statement in `internal/evidence` with unit test
- [ ] T019 [QA] verify: full suite, go tests, gosec, govulncheck, both qa-reports
- [ ] T020 [LEAD] `docs/diagramas` updated to the current state
- [ ] T021 [LEAD] closing: `memory.md`, `pr.md`; [OPS] `make agents-zones`, `make agents-check`

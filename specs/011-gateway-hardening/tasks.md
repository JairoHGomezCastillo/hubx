# Tasks: Gateway perimeter hardening

Only `hubx-lead` marks a task, citing the evidence of the agent that ran it.

## Phase 1: test environment
- [ ] T001 [OPS] `compose.yaml`: `gw` limits from env, `gw-limited` (profile `test`); `scripts/test-integration.sh` (and `scripts/review-story.sh` if it runs the suite) starts it, sets 10000 on `gw` and exports `BENDER_LIMITED_BASE_URL`
- [ ] T002 [OPS] `deploy/render.staging.yaml` limits; `.github/workflows/deploy.yml` excludes `@needs-limited-gateway`; note in `deploy/README.md`

## Phase 2: red
- [ ] T003 [QA] `hubx-bender/features/S011_gateway-hardening.feature` with the spec's scenarios verbatim, steps in `features/steps/S011_gateway-hardening.steps.ts`
- [ ] T004 [QA] `make test-integration` red for the right reason; literal output in `qa-report.md`

## Phase 3: implementation
- [ ] T005 [GW] `internal/config` with tests (precedence, missing file, malformed line, invalid limits and origins)
- [ ] T006 [GW] `internal/ratelimit` with tests (window, recovery, isolation, bound, eviction, concurrency)
- [ ] T007 [GW] security headers, CORS and limit middlewares with tests; validated `sub` in context
- [ ] T008 [GW] proxy header cleanup, wiring in `cmd/gw`, `application.properties`, `hubx-gw/AGENTS.md`

## Phase 4: HU-09 correction (review of PR #41)
- [ ] T009 [QA] dated correction in `specs/009-view-count-by-stage/qa-report.md` (API4, rate limiting claim), history kept
- [ ] T010 [QA] HU-09 `@e2e` asserts the browser totals (+3/+2/+5, others unchanged); Gherkin untouched

## Phase 5: verify and close
- [ ] T011 [QA] full suite, go tests, gosec, govulncheck, OWASP review in `specs/011-gateway-hardening/qa-report.md`
- [ ] T012 [OPS] `make agents-zones`
- [ ] T013 [LEAD] analyze, `memory.md`, `README.md`, `pr.md`
- [ ] T014 [OPS] deploy validation: PENDING until the human confirms the release

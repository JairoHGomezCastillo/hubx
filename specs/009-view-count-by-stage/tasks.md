# Tasks: View the shipment count by stage (HU-09)

**Input**: Design documents from `/specs/009-view-count-by-stage/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, `contracts/hu09-stage-counts.md`, quickstart.md

**Tests**: mandatory (constitution Principle V). The QA `red` batch MUST be accepted before any implementation task starts.

**Organization**: grouped by user story (US1→US2) and tagged by owning agent: `[QA]`, `[API]`, `[GW]`, `[MFE]`, `[OPS]`, `[LEAD]`.

## Format: `[ID] [P?] [Story?] [Agent] Description with file path`

- **[P]**: different files, no dependency on an incomplete task.
- **[Story]**: user story from spec.md; omitted in Setup, Foundational and Polish phases.
- Only hubx-lead marks a task `- [x]`, and only after an accepted subagent delivery with cited evidence.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: confirm the existing foundation; no project initialization and no new migration are needed.

- [x] T001 [API] Confirm migration `hubx-api/migrations/0001_baseline_schema.sql` already provides `shipments.stage` (CHECK on the six stages) and the index `shipments_stage_created_at_idx ON shipments (stage, created_at DESC)`; confirm NO HU-09 migration is added (read-only inspection, no file added to `hubx-api/migrations/`) — evidence: lead read-only inspection, `hubx-api/migrations/` holds only `0001_baseline_schema.sql` (stage CHECK line 24, index line 33)
- [x] T002 [QA] (7a33fbb; bender loads under Node >= 22 and `make test-integration` ran both passes) Verify the QA toolchain is installed (`npm --prefix hubx-bender ci`, `npm --prefix hubx-bender run browsers`) and that `BENDER_BASE_URL` points to the gateway in `hubx-bender/features/support/world.ts`

**Checkpoint**: foundation confirmed; no file from this phase blocks QA.

---

## Phase 2: Foundational — QA red (BLOCKING)

**Purpose**: acceptance tests exist and fail for the right reason before any production code is written.

- [x] T003 [QA] (7a33fbb; scenarios "No shipments in HUBX" and "Stages without shipments" tagged `@empty-db`, human decision option A) Write `hubx-bender/features/HU09_stage-counts.feature` with the HU-09 Gherkin copied verbatim from `backlog/backlog.md` (the four scenario groups: "Count by stage", "Stages without shipments", "No shipments in HUBX", and the "Role without permission to view the count" Scenario Outline with its `ground partner` and `customer` examples) plus at least one `@e2e` critical happy path (operator views the six counts in journey order)
- [x] T004 [QA] (7a33fbb; lead dry-run: 65 scenarios, 419 steps, none undefined/ambiguous) Write `hubx-bender/features/steps/HU09_stage-counts.steps.ts` and extend `hubx-bender/features/steps/common.steps.ts` as needed: sign-in through the gateway, build the stage fixtures by creating shipments (HU-02) and advancing them (HU-03) through the gateway only, call `GET /api/shipments/stage-counts`, and assert the response shape from `contracts/hu09-stage-counts.md`; `@e2e` steps use `this.ui` with `getByRole`/`getByLabel` only and never call the API
- [x] T005 [QA] (7a33fbb; `make test-integration` pass 1 `63 scenarios (59 passed, 4 failed)`, pass 2 `2 scenarios (2 failed)`, every HU-09 failure is 404 on the missing endpoint) Run `npm --prefix hubx-bender run test:api` and `npm --prefix hubx-bender run test:e2e`; record the literal red output in `specs/009-view-count-by-stage/qa-report.md`, proving failures are the missing endpoint/UI (e.g. `Expected 200/403, Received 404`), not broken steps
- [x] T006 [QA] (7a33fbb; qa-report.md OWASP table API1-API10, API2/API5/API8 pending verify) Add the OWASP API Security Top 10 checklist to `specs/009-view-count-by-stage/qa-report.md`, covering function-level authorization (API5: operator-only, non-operator `403` with no totals, from FR-006), broken authentication (API2: session rule from FR-007) and the absence of injection surface (API8: the parameter-free aggregate)

**Checkpoint (⚠️ CRITICAL)**: no API, MFE or OPS implementation task may start until T003–T006 are accepted in `red`.

---

## Phase 3: User Story 1 - See the shipment count for every stage (Priority: P1) 🎯 MVP

**Goal**: an operator gets one total per stage, the six stages always present and in journey order, absent stages shown as zero, an empty system showing six zeros.

**Independent Test**: with shipments spread across stages and an operator signed in, `GET /api/shipments/stage-counts` returns exactly six `{stage,count}` entries in journey order; each count matches the shipments in that stage; a stage with none reads 0; on an empty database all six read 0 and `total` is 0.

### Tests for User Story 1 (write first, expect red)

- [x] T007 [P] [US1] [API] (1c99e21; lead rerun `go -C hubx-api test ./...`: `ok hubx-api/internal/shipment/test`) Write black-box unit tests in `hubx-api/internal/shipment/test/stage_count_test.go` for `StageCounter.Count`, using a gomock `StageCountRepository` from `hubx-api/internal/shipment/mocks/`: asserts exactly six entries in journey order (Created→Delivered), zero-fills stages absent from the repository map, computes `Total` as the sum, handles the empty map (six zeros, total 0), and propagates a repository error unchanged
- [x] T008 [P] [US1] [API] (1c99e21; `ok hubx-api/internal/handlers/shipment/test`) Write black-box handler tests in `hubx-api/internal/handlers/shipment/test/stage_counts_handler_test.go` for `GET /shipments/stage-counts` with an operator principal, using a mocked `StageCounter` from `hubx-api/internal/handlers/shipment/mocks/`: asserts `200` with the JSON body of `contracts/hu09-stage-counts.md` (six stages in order, `count` and `total` fields) and `500 INTERNAL_SERVER_ERROR` when the counter returns an error

### Implementation for User Story 1

- [x] T009 [US1] [API] (1c99e21) Define `StageCount`, `StageCountReport` and the `StageCounter` service in `hubx-api/internal/shipment/stage_count.go` (no `net/http` import), reusing the journey order from `hubx-api/internal/shipment/stage.go`; build the fixed six-entry, journey-ordered, zero-filled report and the `Total` sum (data-model rules 2–4)
- [x] T010 [US1] [API] (1c99e21; mocks generated by `go -C hubx-api generate ./...`) Define the `StageCountRepository` interface (`CountByStage(ctx) (map[Stage]int, error)`) in `hubx-api/internal/shipment/stage_count_repository.go` with the `//go:generate go tool mockgen` directive, and run `go -C hubx-api generate ./...` to produce the mock in `hubx-api/internal/shipment/mocks/`
- [x] T011 [US1] [API] (1c99e21) Implement `hubx-api/internal/shipment/stage_count_postgres.go`: a single parameter-free aggregate `SELECT stage, COUNT(*) FROM shipments GROUP BY stage` returning a `map[Stage]int`; parameterized `database/sql` usage only, errors wrapped for the handler to map to `500`
- [x] T012 [US1] [API] (1c99e21; deviation: error mapping is inline in `stage_counts_handler.go`, no `stage_counts_errors.go`) Implement `RegisterStageCounts(mux, requireAuth, counter)` and the `GET /shipments/stage-counts` handler in `hubx-api/internal/handlers/shipment/stage_counts_handler.go`: read the operator role from the validated JWT claims (same `auth.GetClaims` pattern as the HU-02 create handler), call `StageCounter.Count`, and serialize the contract body; map errors in `hubx-api/internal/handlers/shipment/stage_counts_errors.go` (`500 INTERNAL_SERVER_ERROR`)
- [x] T013 [US1] [API] (1c99e21; `go -C hubx-api vet ./...` clean) Wire the `StageCountRepository`, `StageCounter` and `RegisterStageCounts(mux, authMiddleware.RequireAuth, ...)` in `hubx-api/cmd/api/main.go`, reusing the existing pgx pool and the HU-01 JWT/shared-secret middleware; run `go -C hubx-api generate ./...` and `go -C hubx-api test ./...`
- [x] T014 [US1] [GW] (backend-go read-only confirmation, no hubx-gw change; `go -C hubx-gw test ./...` ok; route green through the gateway in 9977730) Confirm `hubx-gw/internal/proxy/proxy.go` forwards `GET /api/shipments/stage-counts` with `/api` stripped and `Authorization` forwarded, and that no gateway file needs a change for HU-09 (read-only confirmation; record in `specs/009-view-count-by-stage/qa-report.md` or the PR if a change ever proves necessary)
- [x] T015 [P] [US1] [MFE] (4e4f0f4; lead rerun `Test Files 15 passed (15)`, `Tests 103 passed (103)`) Add `hubx-mfe/src/services/stageCountService.ts` (typed `GET /shipments/stage-counts` via the single `/api` client) and its unit test `hubx-mfe/src/services/stageCountService.test.ts` (success returns six ordered `{stage,count}` entries; `403` and `401` surfaced to the caller)
- [x] T016 [US1] [MFE] (4e4f0f4 + 1cbd209 load on open; build `✓ built`; deviation: also `src/pages/StageCounts.tsx` and a route in `App.tsx`) Add the accessible view `hubx-mfe/src/features/stage-counts/StageCounts.tsx` and `hubx-mfe/src/features/stage-counts/StageCountsContainer.tsx` (six rows in journey order, stage name + count, English text, role/label locators for `@e2e`, session-expiry handling), with unit test `hubx-mfe/src/features/stage-counts/StageCounts.test.tsx`; render it from `hubx-mfe/src/pages/Dashboard.tsx` only for `user?.role === 'operator'`; run `npm --prefix hubx-mfe run test` and `npm --prefix hubx-mfe run build`

**Checkpoint**: US1 is independently testable; the `@e2e` operator happy path must be green before US2 starts.

---

## Phase 4: User Story 2 - Keep the count operator-only (Priority: P2)

**Goal**: a ground partner or customer requesting the count is refused with the permission message and receives no totals.

**Independent Test**: signed in as a ground partner or a customer, `GET /api/shipments/stage-counts` returns `403 FORBIDDEN` with the message `You do not have permission to see that information` and no `stages`/`total` body; the MFE does not offer the view to those roles.

### Tests for User Story 2 (write first, expect red)

- [x] T017 [P] [US2] [API] (1c99e21) Extend `hubx-api/internal/handlers/shipment/test/stage_counts_handler_test.go` with the refusal cases: a `ground partner` principal and a `customer` principal both get `403 FORBIDDEN` with the contract message and body, and the `StageCounter` is never invoked (assert the mock gets no `Count` call); add an invalid/missing-session case asserting `401 UNAUTHORIZED` passthrough from the middleware

### Implementation for User Story 2

- [x] T018 [US2] [API] (1c99e21; 403 before the counter, inline mapping) Add the operator-only authorization to the `GET /shipments/stage-counts` handler in `hubx-api/internal/handlers/shipment/stage_counts_handler.go`: when `claims.Role != "operator"`, return `403 FORBIDDEN` with the message `You do not have permission to see that information` before calling the counter (FR-006); map it in `hubx-api/internal/handlers/shipment/stage_counts_errors.go` and keep the body free of any totals; run `go -C hubx-api test ./...`
- [x] T019 [US2] [MFE] (4e4f0f4, 1cbd209) Ensure `hubx-mfe/src/pages/Dashboard.tsx` hides the stage-count entry for non-operator roles and that `hubx-mfe/src/features/stage-counts/StageCountsContainer.tsx` surfaces a clear permission message if a `403` is received; cover it in `hubx-mfe/src/features/stage-counts/StageCounts.test.tsx`; run `npm --prefix hubx-mfe run test` and `npm --prefix hubx-mfe run build`

**Checkpoint**: refusal scenarios green; US1 still green.

---

## Phase 5: Polish & Cross-Cutting Concerns

**Purpose**: full verification, deployment check, analysis and pull request.

- [x] T020 [QA] (9977730; pass 1 `63 scenarios (63 passed)`, pass 2 `2 scenarios (2 passed)`) Run `make test-integration` (full stack, all HU-09 Cucumber API scenarios and the `@e2e` happy path) and record the literal result in `specs/009-view-count-by-stage/qa-report.md`
- [x] T021 [QA] (bd9956c; `gosec ./...` `Files: 64, Lines: 5859, Nosec: 1, Issues: 0`; `govulncheck -C hubx-api ./...` 0 reachable, GO-2026-5932 module-level not called; `govulncheck -C hubx-gw ./...` `No vulnerabilities found.`; lead reran all three) Run `gosec ./...` in both Go services and `govulncheck -C hubx-api ./...`, `govulncheck -C hubx-gw ./...`; attach outputs to `specs/009-view-count-by-stage/qa-report.md` and close the OWASP checklist
- [ ] T022 [OPS] (out of scope by human decision 2026-11-24: no deployment validation; devops instead delivered the `@empty-db` pass in 2cca4d5) Verify `make test-integration` and `make up` still work with the new route; confirm no CI, Dockerfile or compose change is required for HU-09; record evidence in `specs/009-view-count-by-stage/deploy.md` only if a change ever proves necessary
- [x] T023 [LEAD] (analyze deviations in `pr.md`; backlog, memory.md and README updated in the closing commit) Run `speckit.analyze`, record deviations between spec, plan, tasks and code, update `backlog/backlog.md` (HU-09 status/branch/PR) and `memory.md`, and write `specs/009-view-count-by-stage/pr.md` with the literal evidence
- [ ] T024 [LEAD] Commit each accepted delivery tagged `[agent:role] HU-09 ...`, push `009-view-count-by-stage` and open the PR to the story's release branch; never merge, force-push or push to `main`/`release/*`/`hotfix/*`

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 (Setup)**: no dependency; lead/QA/API verification only.
- **Phase 2 (QA red)**: BLOCKING. T003–T006 must be accepted before T007 or any later implementation task.
- **Phase 3 (US1)**: depends on Phase 2; the MVP.
- **Phase 4 (US2)**: depends on US1's handler and endpoint (it adds the role gate to the same handler).
- **Phase 5 (Polish)**: depends on all accepted story phases.

### User story dependencies

- **US1 (P1)**: no dependency on other stories; MVP.
- **US2 (P2)**: extends US1's handler with the operator-only gate and the MFE view's role visibility; it is testable on its own by driving the endpoint as a non-operator.

### Within each user story

1. Tests first, failing for the right reason (constitution Principle V).
2. Domain type and interface before the service; service before the postgres implementation; handler after the service; MFE service before the view.
3. `go generate` after interface changes; never edit generated mocks by hand.

### Delegation batches (lead's implement order)

1. `qa-seguridad` **red** → T003–T006. Snapshot, JSON with evidence, accept, commit `[agent:qa-seguridad]`.
2. `backend-go` (API) and `frontend-react` (MFE) in parallel on non-overlapping paths → T007–T016 as two deliveries, one commit each. `[GW]` T014 is a read-only confirmation handled by `backend-go`/`devops`.
3. `backend-go` + `frontend-react` → T017–T019 (US2 gate and MFE visibility).
4. `qa-seguridad` **verify** → T020–T021, with all API and `@e2e` scenarios green and no critical/high finding.
5. `devops` → T022 if any infrastructure work is needed.
6. `hubx-lead` → T023–T024 after `speckit.analyze`.

### Parallel opportunities

- T007 and T008 (independent `test/` files) can be written together.
- API (hubx-api) and MFE (hubx-mfe) work has no overlapping path, so T009–T013 and T015–T016 run in parallel within batch 2.
- T017 (API refusal tests) and the MFE visibility work are in different files and can proceed in parallel.
- [P] tasks use different files and have no dependency on an incomplete task.

---

## Implementation Strategy

### MVP first (US1 only)

1. Confirm Phase 1 and accept QA red (Phase 2).
2. Deliver US1 (T007–T016) and validate the `@e2e` operator happy path.
3. Stop at that checkpoint before adding the role-refusal work.

### Incremental delivery

1. QA red → foundation of evidence.
2. US1 → operator sees the six stage counts (MVP).
3. US2 → non-operator refusal and MFE role visibility.
4. QA verify → security → analyze → PR; a human reviews and merges.

---

## Notes

- Every Go package with tests must have `test/` (package `test`, black-box) and `mocks/` (generated with `go tool mockgen`), per constitution Principle IV. HU-09 extends the existing `internal/shipment` and `internal/handlers/shipment` packages; it creates no new domain.
- No new migration and no new dependency: any task proposing them is out of HU-09's scope. The count reads the existing `shipments.stage` projection only; it never touches the `updates` ledger.
- The aggregate query is parameter-free, so there is no injection surface; authorization is function-level from the validated JWT (OWASP API5).
- The MFE validation is `npm run test` plus `npm run build` and the mandatory `@e2e` browser scenario; a red `@e2e` blocks the story even when all API scenarios are green.
- The lead commits once per accepted subagent delivery and never merges; merging is a human decision.
- The story request must name its release branch (`release/vX.Y.Z[-name]`); if it does not exist, hubx-lead creates it from `main` with the human's confirmation before T024.

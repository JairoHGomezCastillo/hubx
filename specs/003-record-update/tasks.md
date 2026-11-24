# Tasks: Record a shipment update (HU-03)

**Input**: Design documents from `/specs/003-record-update/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, `contracts/hu03-record-update.md`, quickstart.md

**Tests**: mandatory (constitution Principle V). The QA `red` batch MUST be accepted before any implementation task starts.

**Organization**: grouped by user story (US1→US3) and tagged by owning agent: `[QA]`, `[API]`, `[GW]`, `[MFE]`, `[OPS]`, `[LEAD]`.

## Format: `[ID] [P?] [Story?] [Agent] Description with file path`

- **[P]**: different files, no dependency on an incomplete task.
- **[Story]**: user story from spec.md; omitted in Setup, Foundational and Polish phases.
- Only hubx-lead marks a task `- [x]`, and only after an accepted subagent delivery with cited evidence.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: confirm the existing foundation; no project initialization and no new migration are needed.

- [ ] T001 [API] Confirm migration `hubx-api/migrations/0001_baseline_schema.sql` already provides users/shipments/updates and that no HU-03 migration is added; keep `0002_seed_ledger` reserved
- [ ] T002 [QA] Verify the QA toolchain is installed (`npm --prefix hubx-bender ci`, `npm --prefix hubx-bender run browsers`) and confirm `BENDER_BASE_URL` points to the gateway in `hubx-bender/features/support/world.ts`

**Checkpoint**: foundation confirmed; no file from this phase blocks QA.

---

## Phase 2: Foundational — QA red (BLOCKING)

**Purpose**: acceptance tests exist and fail for the right reason before any production code is written.

- [x] T003 [QA] Write `hubx-bender/features/HU03_record-update.feature` with the HU-03 Gherkin scenarios copied verbatim from `backlog/backlog.md` (the 8 scenario groups, including both Scenario Outlines and all examples) and at least one `@e2e` critical happy path — accepted in `3bc2d51`; Gherkin verbatim plus the `@e2e` browser happy path, with the literal RED recorded in `qa-report.md`
- [x] T004 [QA] Write `hubx-bender/features/steps/HU03_record-update.steps.ts` and extend `hubx-bender/features/steps/common.steps.ts` with sign-in through the gateway, fixture advancement through the update endpoint, and POST/response helpers; `@e2e` steps use `this.ui` with `getByRole`/`getByLabel` only and never call the API — accepted in `3bc2d51`; no gateway bypass, no undefined steps, role/label locators only
- [x] T005 [QA] Run `npm --prefix hubx-bender run test:api` and `npm --prefix hubx-bender run test:e2e`; record literal red output in `specs/003-record-update/qa-report.md`, proving failures are due to the missing endpoint/UI, not broken steps — accepted from `qa-seguridad` sessions `ses_df2396ff1fferi4rOkxEngtCyZ` (executed the real RED) and `ses_df232f9edffe4wAKQKyp1CfPlD` (wrote the report), `status: ok`, `blockers: []`; literal `make test-integration`: `46 scenarios (25 passed, 21 failed)`, all 21 HU-03 failures from `Expected: 201/409/403, Received: 404` on the missing update route or the `@e2e` 30 s UI timeout; dry-run after the shared-step fix `46 scenarios (46 skipped)`, no undefined/ambiguous steps; report at `specs/003-record-update/qa-report.md`
- [x] T006 [QA] Add the OWASP API Security Top 10 checklist to `specs/003-record-update/qa-report.md`, including BOLA/visibility, authentication and immutable-ledger checks derived from FR-011 and FR-012 — accepted in `3bc2d51`; written by `qa-seguridad`, never by hubx-lead

**Checkpoint (⚠️ CRITICAL)**: no API, MFE or OPS implementation task may start until T003–T006 are accepted in `red`.

---

## Phase 3: User Story 1 - Record the next stage update (Priority: P1) 🎯 MVP

**Goal**: the responsible party advances the shipment and appends a chained update with party and date.

**Independent Test**: a Created shipment with an operator moves to In flight; the first update has no predecessor and a later update links to it through its fingerprint.

### Tests for User Story 1 (write first, expect red)

- [x] T007 [P] [US1] [API] Write black-box unit tests in `hubx-api/internal/ledger/test/` for the canonical fingerprint function and `Append` chain behavior, using gomock mocks from `hubx-api/internal/ledger/mocks/` — accepted in `85ec054`; RED: `internal/ledger/test/service_test.go:94:22: undefined: ledger.NewService` / `FAIL hubx-api/internal/ledger/test [build failed]`; GREEN with the temporary Go 1.27.4 Docker runner: `ok hubx-api/internal/ledger/test 0.010s`
- [x] T008 [P] [US1] [API] Write black-box unit tests in `hubx-api/internal/shipment/test/` for `Recorder.Record` on Created→In flight and In flight→At destination airport, using gomock mocks from `hubx-api/internal/shipment/mocks/` — accepted in `5c57dd2`; RED: `undefined: shipment.NewRecorder`, `FAIL .../test [build failed]`; GREEN with pinned GOROOT: `ok hubx-api/internal/shipment/test 0.008s`, all module packages `ok`, no temporary directive left
- [x] T009 [P] [US1] [API] Write black-box handler tests in `hubx-api/internal/handlers/shipment/test/` for `POST /shipments/{code}/updates` success and 201 body shape, with mocks in `hubx-api/internal/handlers/shipment/mocks/` — accepted in `8c18ab6`; RED→GREEN with pinned GOROOT, final `ok hubx-api/internal/handlers/shipment/test`; includes resolved-name and ID-fallback cases from the QA contract fix

### Implementation for User Story 1

- [x] T010 [US1] [API] Define the ledger domain in `hubx-api/internal/ledger/update.go` and the `LedgerRepository` interface (`Last`, `Append`) in `hubx-api/internal/ledger/repository.go`; all ledger access goes through this interface (Principle I) — accepted in `85ec054`; only `//go:generate go tool mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks` remains, no temporary directive
- [x] T011 [US1] [API] Implement the canonical SHA-256 fingerprint input from `contracts/hu03-record-update.md` in `hubx-api/internal/ledger/update.go` (`-` for nulls, RFC3339Nano UTC, no trailing newline) — accepted in `85ec054`; covered by `internal/ledger/test/update_test.go`, GREEN `ok hubx-api/internal/ledger/test 0.010s` via the temporary runner
- [x] T012 [US1] [API] Implement `hubx-api/internal/ledger/postgres.go`: insert-only `Append` inside one transaction with `SELECT ... FOR UPDATE`, expected-sequence check, shipment projection update, parameterized SQL only — accepted in `cd98089`; RED: `undefined: ledger.NewPostgresRepository`, `mocks.NewMockRow`, `FAIL hubx-api/internal/ledger/test [build failed]`; GREEN with the temporary `go:generate` directive: `ok hubx-api/internal/ledger/test 0.009s`, all module packages `ok`; no temporary directive left (`grep go:generate` → only the two mockgen lines)
- [x] T013 [US1] [API] Define `Shipment`, `Stage` and the transition table in `hubx-api/internal/shipment/shipment.go`, repository interfaces in `hubx-api/internal/shipment/repository.go` and sentinel errors in `hubx-api/internal/shipment/errors.go` — accepted in `a42a09a`; build with GOROOT pinned to 1.27.4 produced an empty output file (success), `grep go:generate` shows only the legitimate mockgen directive; literal `go build` failure is the documented host GOROOT 1.27.3 mismatch
- [x] T014 [US1] [API] Implement `Recorder.Record` in `hubx-api/internal/shipment/recorder.go` for the two operator transitions, reading the predecessor fingerprint from `LedgerRepository.Last` — accepted in `5c57dd2`; strict transition/actor/partner rules and FR-012 covered by `internal/shipment/test/recorder_test.go`; GREEN `ok hubx-api/internal/shipment/test 0.008s`
- [x] T015 [US1] [API] Implement `Register(mux, ...)` and `POST /shipments/{code}/updates` in `hubx-api/internal/handlers/shipment/handler.go` with a `PrincipalSource` interface and sentinel→HTTP mapping in `hubx-api/internal/handlers/shipment/errors.go` — accepted in `8c18ab6`; contract error codes/messages mapped, `PartyDirectory` injectable so 201 names come from users (fallback to ID after commit); GREEN `ok hubx-api/internal/handlers/shipment/test`
- [ ] T016 [US1] [API] Wire config, pgx pool, repositories, `Recorder` and the handler in `hubx-api/cmd/api/main.go`, keeping the HU-01 JWT/shared-secret middleware; run `go -C hubx-api generate ./...` and `go -C hubx-api test ./...`
- [ ] T017 [P] [US1] [MFE] Add `hubx-mfe/src/api/client.ts` as the single `/api` client with the JWT in memory (never localStorage)
- [ ] T018 [US1] [MFE] Add the accessible update form in `hubx-mfe/src/features/shipment-updates/RecordUpdate.tsx` (shipment code, stage select, submit, English text, role/label locators) and render it from `hubx-mfe/src/App.tsx`; run `npm --prefix hubx-mfe run build`

**Checkpoint**: US1 is independently testable; the `@e2e` happy path must be green before US2 starts.

---

## Phase 4: User Story 2 - Assign a ground partner (Priority: P2)

**Goal**: an operator assigns a registered ground partner from At destination airport, and that partner becomes responsible for delivery.

**Independent Test**: a shipment at At destination airport moves to Assigned to partner, the update names the operator as responsible party and the chosen partner as delivery owner.

### Tests for User Story 2 (write first, expect red)

- [ ] T019 [P] [US2] [API] Extend `hubx-api/internal/shipment/test/` and `hubx-api/internal/handlers/shipment/test/` with assignment cases: valid partner, customer instead of partner, and missing partner id

### Implementation for User Story 2

- [x] T020 [US2] [API] Add the registered-ground-partner check to `hubx-api/internal/shipment/repository.go` and `hubx-api/internal/shipment/postgres.go`; implement `GET /ground-partners` in `hubx-api/internal/handlers/shipment/handler.go` per `contracts/hu03-record-update.md` — COMPLETE: endpoint accepted in `9caf123` (GREEN `ok hubx-api/internal/handlers/shipment/test 0.007s`) and SQL layer accepted in `a9c5818` (GREEN `ok hubx-api/internal/shipment/test 0.009s`, all hubx-api packages ok); parameterized SQL, `sql.ErrNoRows`→`(nil,nil)`, non-nil empty list and `Name` compatible with `PartyDirectory`; no temporary directive left
- [ ] T021 [US2] [API] Implement the Assigned to partner transition in `hubx-api/internal/shipment/recorder.go`: partner id required, responsible party is the operator, projection sets `assigned_partner_id`, and assignment cannot be repeated or replaced (FR-002, FR-009)
- [ ] T022 [US2] [MFE] Add the labelled ground-partner `<select>` populated from `GET /api/ground-partners` in `hubx-mfe/src/features/shipment-updates/RecordUpdate.tsx`, using partner names as labels and ids as values
- [ ] T023 [US2] [API] Regenerate gomock mocks with `go -C hubx-api generate ./...` and rerun `go -C hubx-api test ./...`

**Checkpoint**: assignment scenarios green; US1 still green.

---

## Phase 5: User Story 3 - Reject updates that break the rules (Priority: P3)

**Goal**: out-of-sequence and unauthorized attempts are rejected with the stated reason and change nothing.

**Independent Test**: a shipment in a given stage with a non-entitled caller gets the rejection and keeps its stage and chain.

### Tests for User Story 3 (write first, expect red)

- [ ] T024 [P] [US3] [API] Extend `hubx-api/internal/shipment/test/` and `hubx-api/internal/ledger/test/` with all out-of-sequence and not-responsible examples from the backlog, plus the FR-012 invisible/unknown-code cases

### Implementation for User Story 3

- [ ] T025 [US3] [API] Enforce the strict transition table, no repeat/no going back, and Delivered as terminal in `hubx-api/internal/shipment/recorder.go`, returning `ErrOutOfSequence` (FR-006)
- [ ] T026 [US3] [API] Enforce role/assignment responsibility and FR-012 in `hubx-api/internal/shipment/recorder.go`, so an invisible or unknown shipment returns the same `NOT_RESPONSIBLE` as a wrong caller (FR-007, FR-011, FR-012)
- [ ] T027 [US3] [API] Return `CONCURRENT_UPDATE` on a stale expected sequence or unique violation in `hubx-api/internal/ledger/postgres.go` and map it to 409 in `hubx-api/internal/handlers/shipment/errors.go`
- [ ] T028 [US3] [API] Verify every rejected attempt creates no update and changes no projection field; extend tests in the `test/` subpackages and regenerate mocks with `go -C hubx-api generate ./...`

**Checkpoint**: all backlog scenarios automated and green; no open high/critical finding.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: full verification, deployment check, analysis and pull request.

- [ ] T029 [QA] Run `make test-integration` (full stack, all Cucumber API scenarios and the `@e2e` happy path) and record the literal result in `specs/003-record-update/qa-report.md`
- [ ] T030 [QA] Run `gosec ./...` in both Go services and `govulncheck -C hubx-api ./...`, `govulncheck -C hubx-gw ./...`; attach outputs to `specs/003-record-update/qa-report.md` and close the OWASP checklist
- [ ] T031 [GW] Confirm `hubx-gw/internal/proxy/proxy.go` needs no change (any method is proxied with `/api` stripped and Authorization forwarded) and that no gateway file was modified by HU-03
- [ ] T032 [OPS] Verify `make test-integration` and `make up` still work with the new routes; no CI, Dockerfile or compose change is required for HU-03; record evidence in `specs/003-record-update/deploy.md` if a change is ever needed
- [ ] T033 [LEAD] Run `speckit.analyze`, record deviations between spec, plan, tasks and code, update `backlog/backlog.md` and `memory.md`, and write `specs/003-record-update/pr.md` with the literal evidence
- [ ] T034 [LEAD] Commit each accepted delivery tagged `[agent:role] HU-03 ...`, push `003-record-update` and open the PR to `main` (human decision 2026-11-24 after v0.1.0 was merged), once the current release branch is merged into `main` by a human; never merge

---

## Dependencies & Execution Order

### Phase dependencies

- **Phase 1 (Setup)**: no dependency; lead/QA/API verification only.
- **Phase 2 (QA red)**: BLOCKING. T003–T006 must be accepted before T007 or any later implementation task.
- **Phases 3–5 (US1→US3)**: depend on Phase 2; US2 and US3 integrate with US1's `Recorder` and endpoint and run in that order.
- **Phase 6 (Polish)**: depends on all accepted story phases.

### User story dependencies

- **US1 (P1)**: no dependency on other stories; MVP.
- **US2 (P2)**: needs US1's transition engine and endpoint.
- **US3 (P3)**: needs US1 and US2 paths to exercise rejection cases.

### Within each user story

1. Tests first, failing for the right reason (constitution Principle V).
2. Domain interfaces and entities before services; services before handlers; handler before MFE wiring.
3. `go generate` after interface changes; never edit generated mocks by hand.

### Delegation batches (lead's implement order)

1. `qa-seguridad` **red** → T003–T006. Snapshot, JSON with evidence, accept, commit `[agent:qa-seguridad]`.
2. `backend-go` (API) and `frontend-react` (MFE) in parallel on non-overlapping paths → T007–T028 as two deliveries, one commit each.
3. `qa-seguridad` **verify** → T029–T030, with all API and `@e2e` scenarios green and no critical/high finding.
4. `devops` → T031–T032 if any infrastructure work is needed.
5. `hubx-lead` → T033–T034 after `speckit.analyze`.

### Parallel opportunities

- T007, T008 and T009 (independent `test/` packages) can be written together.
- API (hubx-api) and MFE (hubx-mfe) work has no overlapping path, so it runs in parallel within batch 2.
- [P] tasks use different files and have no dependency on an incomplete task.

---

## Implementation Strategy

### MVP first (US1 only)

1. Confirm Phase 1 and accept QA red (Phase 2).
2. Deliver US1 (T007–T018) and validate the `@e2e` happy path.
3. Stop at that checkpoint before adding assignment or rejection work.

### Incremental delivery

1. QA red → foundation of evidence.
2. US1 → record the next stage update (MVP).
3. US2 → assign a ground partner.
4. US3 → reject invalid attempts.
5. QA verify → security → analyze → PR; a human reviews and merges.

---

## Notes

- Every Go package with tests must have `test/` (package `test`, black-box) and `mocks/` (generated with `go tool mockgen`), per constitution Principle IV.
- No new migration and no new dependency: any task proposing them is out of HU-03's scope.
- The MFE has no component-test runner; its validation is `npm run build` plus the mandatory `@e2e` browser scenario.
- A red `@e2e` scenario blocks the story even when all API scenarios are green.
- The lead commits once per accepted subagent delivery and never merges.
- Human decision 2026-11-24: implement the isolated domains and `RecordUpdate.tsx` before HU-01 is merged (PR #13),
  because the clean-stack red run is blocked only by the missing sign-in route/UI. T003/T004/T006 have evidence but
  T005 stays unaccepted until the final green run. No wiring file shared with PR #13 may be edited in this phase;
  wiring, red re-run, verify and PR happen only after the human merges PR #13. This deviation will be reviewed in Analyze.
- Verification deviation 2026-11-24 (human-approved): the OpenCode process starts with `GOROOT=/usr/local/go`
  (compiler 1.27.3) while the Go tool is 1.27.4, so literal `go -C hubx-api test ./...` fails in this session even
  though it passes in the user's terminal. `devops` created a temporary Docker runner at
  `/tmp/opencode/hubx-go-runner/` (Go 1.27.4, `GOWORK=off`) fully outside the repository; the accepted verification
  command for this session is `make -f /tmp/opencode/hubx-go-runner/Makefile go-test` (`ok` for all packages,
  `0` repo files changed). It is not a deliverable and must be deleted at closing.
- `go.uber.org/mock v0.6.0` is added as a module tool for `go tool mockgen`, already permitted by constitution 1.4.0
  and `memory.md`; the earlier "no new dependency" note refers to production dependencies. `go.work.sum` only gained
  checksums from resolving that tool.

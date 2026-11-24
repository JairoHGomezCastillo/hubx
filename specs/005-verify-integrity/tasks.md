# Tasks: Verify the integrity of a shipment's chain (HU-05)

**Input**: Design documents from `/specs/005-verify-integrity/`

**Prerequisites**: plan.md, spec.md, quickstart.md, `contracts/hu05-verify-integrity.md`, `contracts/hu03-record-update.md` (fingerprint formula, microsecond clarification)

**Tests**: mandatory (constitution Principle V). The QA `red` batch MUST be accepted before any implementation task starts.

**Organization**: grouped by user story (US1→US3) and tagged by owning agent: `[QA]`, `[API]`, `[GW]`, `[MFE]`, `[OPS]`, `[LEAD]`.

## Format: `[ID] [P?] [Story?] [Agent] Description with file path`

- **[P]**: different files, no dependency on an incomplete task.
- **[Story]**: user story from spec.md; omitted in Setup, Foundational and Polish phases.
- Only hubx-lead marks a task `- [x]`, and only after an accepted subagent delivery with cited evidence.

---

## Phase 1: Setup

- [ ] T001 [LEAD] Confirm no migration is needed (HU-05 only reads `updates` from `0001_baseline_schema.sql`); `0002_seed_ledger` stays reserved
- [ ] T002 [LEAD] Write `contracts/hu05-verify-integrity.md` and the microsecond clarification in `contracts/hu03-record-update.md` rule 1

---

## Phase 2: Foundational — QA red (BLOCKING)

- [ ] T003 [QA] Write `hubx-bender/features/HU05_verify-integrity.feature` with the HU-05 Gherkin copied verbatim from `backlog/backlog.md` (5 scenario groups, 7 scenarios after expanding the outline) plus one `@e2e` happy path (operator verifies an intact chain in the browser)
- [ ] T004 [QA] Write `hubx-bender/features/steps/HU05_verify-integrity.steps.ts`: fixtures created through the gateway (create shipment, record updates with the HU-03 endpoint); `@e2e` steps use `getByRole`/`getByLabel` only
- [ ] T005 [QA] Write `hubx-bender/features/support/tamper.ts`: `node:child_process` `execFile` (no shell) running `docker compose -p "$BENDER_COMPOSE_PROJECT" exec -T postgres psql` with `SET LOCAL session_replication_role = replica` to alter the second update of a fresh shipment; the tamper scenario is tagged `@needs-secret` so deployments never run it; no new dependency
- [ ] T006 [QA] Run the suite, record the literal RED (missing route/UI, not broken steps) and the OWASP API Top 10 checklist (API1 BOLA focus) in `specs/005-verify-integrity/qa-report.md`

**Checkpoint**: no API, MFE or OPS task starts until T003–T006 are accepted.

---

## Phase 3: User Story 1 - Verify an intact chain (P1) 🎯 MVP

### Tests (write first, expect red)

- [ ] T007 [P] [US1] [API] Unit tests in `hubx-api/internal/ledger/test/` for `ledger.Verify` (empty, intact 3) and `LedgerRepository.List` (ordered, parameterized), gomock mocks in `hubx-api/internal/ledger/mocks/`
- [ ] T008 [P] [US1] [API] Unit tests in `hubx-api/internal/ledger/test/` proving `Append` truncates `RecordedAt` to microseconds before computing the fingerprint and stores/returns the same instant (contract HU-03 rule 1)
- [ ] T009 [P] [US1] [API] Unit tests in `hubx-api/internal/shipment/test/` for `shipment.Verifier` (operator any, owner customer, assigned partner)
- [ ] T010 [P] [US1] [API] Handler tests in `hubx-api/internal/handlers/shipment/test/` for `GET /shipments/{code}/integrity` 200 body shape and 401 via `RequireAuth`

### Implementation

- [ ] T011 [US1] [API] `hubx-api/internal/ledger/repository.go` + `postgres.go`: `List(ctx, shipmentID)` with `ORDER BY sequence ASC`; `Append` truncates to microseconds (`Truncate(time.Microsecond)`)
- [ ] T012 [US1] [API] `hubx-api/internal/ledger/verify.go`: `Verify([]Update) VerificationResult` using the existing `Fingerprint`
- [ ] T013 [US1] [API] `hubx-api/internal/shipment/verifier.go`: visibility reusing the HU-03 rule; `ErrShipmentNotFound` for invisible shipments
- [ ] T014 [US1] [API] `hubx-api/internal/handlers/shipment/integrity_handler.go` + wiring in `hubx-api/cmd/api/main.go`; regenerate mocks with `go -C hubx-api generate ./...`
- [ ] T015 [P] [US1] [MFE] Tests first, then `hubx-mfe/src/features/integrity/` view (code input labelled "Shipment code", button "Verify integrity", result text) through the `/api` client; route/link in `hubx-mfe/src/App.tsx` and `hubx-mfe/src/pages/Dashboard.tsx`

---

## Phase 4: User Story 2 - Detect the first altered update (P2)

- [ ] T016 [US2] [API] Unit tests then implementation in `ledger.Verify` for: altered field, altered fingerprint, broken previous link, missing/reordered sequence; only the first position reported
- [ ] T017 [US2] [MFE] Render "not intact" with the first altered update position

---

## Phase 5: User Story 3 - Privacy (P3)

- [ ] T018 [US3] [API] Tests and implementation: unknown code, customer not owner, ground partner not assigned all return the identical 404 `SHIPMENT_NOT_FOUND` body
- [ ] T019 [US3] [MFE] Render "No shipment exists with that code"

---

## Phase 6: Verify and operations

- [ ] T020 [QA] `verify`: `make test-integration` green (API + `@e2e`), `make security` (gosec, govulncheck), OWASP review, update `qa-report.md`
- [ ] T021 [OPS] `scripts/test-integration.sh`: export `BENDER_COMPOSE_PROJECT=hubx-it` to bender; confirm deployments exclude `@needs-secret`

---

## Phase 7: Polish

- [ ] T022 [LEAD] `speckit.analyze`; record deviations in `pr.md`
- [ ] T023 [LEAD] Backlog Story Status, `memory.md`, README/docs; `pr.md`; push and open the PR into `release/v0.4.0-ledger`

## Dependencies

T003–T006 → T007–T019 (API and MFE in parallel, no shared files) → T020 → T021 → T022–T023.

# Tasks: Download the traceability evidence

**Input**: Design documents from `/specs/010-download-evidence/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, `contracts/hu10-download-evidence.md`

**Tests**: mandatory (Constitution V). QA writes the feature and steps first, in red; Go unit tests use gomock in
`test/` and generated `mocks/`.

**Organization**: by agent `[QA]`, `[API]`, `[GW]`, `[MFE]`, `[OPS]` and by user story; every task names exact paths.

## Format: `[ID] [P?] [Story] [Agent] Description with file path`

- **[P]**: different files, no dependency on an unfinished task
- **[Story]**: US1 (P1), US2 (P2), US3 (P3); setup/foundational/polish have none
- Only `hubx-lead` marks a task `- [x]`, citing the accepted subagent's evidence

---

## Phase 1: Setup (Shared Infrastructure)

- [x] T001 Confirm branch `010-download-evidence`, `Source: backlog HU-10` in `specs/010-download-evidence/spec.md`, and the agreed `contracts/hu10-download-evidence.md`

  Evidence: `create-new-feature.sh --number 010 --short-name download-evidence` returned
  `BRANCH_NAME":"010-download-evidence"`; `git branch --show-current` = `010-download-evidence`; spec line 3 is
  `Source: backlog HU-10`; contract committed in `2137e43` and corrected in `a893dad`.

---

## Phase 2: Foundational (Blocking Prerequisites) — QA red

**Purpose**: acceptance tests in red for all three stories; no implementation may start before this phase is accepted.

**⚠️ CRITICAL**: the feature Gherkin MUST be copied verbatim from `backlog/backlog.md` HU-10, with at least one `@e2e` happy path.

- [x] T002 [P] [QA] Write `hubx-bender/features/HU10_download-evidence.feature` with the HU-10 Gherkin verbatim (six scenarios plus the two-role Scenario Outline) and `@e2e` on the "Evidence for a shipment with updates" happy path
- [x] T003 [QA] Write `hubx-bender/features/steps/HU10_download-evidence.steps.ts`: API steps through the gateway, PDF byte/text assertions, download event handling for `@e2e`, and actor mappings in `features/steps/common.steps.ts` for any new Gherkin phrasing
- [x] T004 [QA] Extend `hubx-bender/features/support/hooks.ts` only if needed for download capture; keep all API traffic in `world.api` and the `@e2e` in `world.ui`, with role-and-label locators scoped to the evidence region
- [x] T005 [QA] Run `npm --prefix hubx-bender test` and record the literal red output (undefined steps or 404), proving the scenarios fail for the right reason

  Evidence (qa-seguridad, phase red, 2026-11-25): `make test-integration` → `66 scenarios (59 passed, 7 failed)`,
  `424 steps (410 passed, 7 skipped, 7 failed)`, no undefined/ambiguous steps in the final run. Each of the 7 HU-10
  failures is the intended red reason: missing Dashboard download link (`@e2e`), 404 on the missing `/evidence`
  route (third-party, no-updates, 404-code), 404 instead of 403 (roles), and gateway no-JWT message (unauthenticated).
  First run exposed two step defects (undefined `tries…`, ambiguous `nobody has signed in`), both fixed in this delivery.

**Checkpoint**: red evidence accepted by hubx-lead; implementation may now start.

---

## Phase 3: User Story 1 - Download evidence for a shipment with updates (Priority: P1) 🎯 MVP

**Goal**: an operator downloads a PDF with the shipment data and the three updates in chain order, with stage,
responsible party, date and fingerprint, generation date and verification instructions.

**Independent Test**: sign in as operator, prepare a shipment with three updates through the gateway, request
`GET /api/shipments/{code}/evidence` and verify the PDF contains all required fields; drive the same happy path in
Chromium via the MFE download button.

### Tests for US1 (written first in Phase 2)

- [x] T006 [QA] Verify US1 scenarios in `hubx-bender/features/HU10_download-evidence.feature` are red before implementation and green afterwards, including the third-party fingerprint-check scenario

  Evidence: red run `66 scenarios (59 passed, 7 failed)` (commit `e2e7b06`); final green run `66 scenarios (66 passed)`,
  with "Evidence for a shipment with updates" (`@e2e`) and "Verification by a third party" both listed as `passed` in
  `specs/010-download-evidence/qa-report.md` (commit `6a647de`).

### Implementation for US1 — API

- [x] T007 [P] [API] Add `hubx-api/internal/evidence/evidence.go` with `Evidence`, `ShipmentData`, `UpdateEvidence` and the self-contained `Instructions` string from `contracts/hu03-record-update.md`
- [x] T008 [P] [API] Add `hubx-api/internal/evidence/pdf.go`: stdlib-only PDF writer, uncompressed text streams, escaped PDF strings, valid xref; no new dependency in `hubx-api/go.mod`
- [x] T009 [API] Add `hubx-api/internal/evidence/reader.go`: `Reader.Read(ctx, code, principal)` using `ShipmentRepository.FindByCode`, exported `shipment.CanSee`, `LedgerRepository.List` and `PartyRepository.Parties`; no ledger write
- [x] T010 [P] [API] Extend `hubx-api/internal/shipment/update_postgres.go` `FindByCode` projection with origin, destination, flight, weight, customer name and creation time, keeping `WHERE code = $1` parameterized
- [x] T011 [P] [API] Export `CanSee` in `hubx-api/internal/shipment/recorder.go` and update HU-03/HU-04/HU-05 callers to use it
- [x] T012 [API] Add `hubx-api/internal/handlers/evidence/handler.go`: `Register(mux, requireAuth, reader, principals)`, `GET /shipments/{code}/evidence`, `application/pdf` and `Content-Disposition: attachment`
- [x] T013 [API] Wire the route in `hubx-api/cmd/api/main.go` with the existing flat `RequireAuth`; no gateway change, no migration
- [x] T014 [P] [API] Add `//go:generate go tool mockgen` directives and generated mocks in `hubx-api/internal/evidence/mocks/` and `hubx-api/internal/handlers/evidence/mocks/`
- [x] T015 [P] [API] Write black-box unit tests in `hubx-api/internal/evidence/test/` and `hubx-api/internal/handlers/evidence/test/` covering the PDF content, empty chain, visibility and error mapping; run `go -C hubx-api generate ./...` then `go -C hubx-api test ./...`
- [x] T016 [GW] Confirm `hubx-gw` needs no change for the flat 401 family and run `go -C hubx-gw test ./...`; do not touch gateway route exclusions or `Authorization`

  Evidence (backend-go, phase implement, 2026-11-25): commit `2f9c9ae` (18 files, `hubx-api/**` only).
  `go -C hubx-api generate ./...` exit 0; `go -C hubx-api test ./...` → all packages `ok`, EXIT=0;
  `go -C hubx-api build ./...` EXIT=0; `go -C hubx-gw test ./...` → `ok`, EXIT=0 (gateway untouched);
  `go -C hubx-api vet ./...` and `go -C hubx-gw vet ./...` EXIT=0. `gofmt -l` lists only pre-existing files,
  none from HU-10. No new dependency, no migration, no gateway change.

### Implementation for US1 — MFE

- [x] T017 [P] [MFE] Add `hubx-mfe/src/services/evidenceService.ts`: `downloadEvidence(code)` via `fetchWithAuth`, blob URL and `<a download="evidence-<code>.pdf">`, URL revoked after click
- [x] T018 [P] [MFE] Add `hubx-mfe/src/features/evidence/DownloadEvidence.tsx` and `DownloadEvidenceContainer.tsx` with an accessible region, a "Shipment code" label and a "Download evidence" button, mapping service errors to explicit outcomes
- [x] T019 [MFE] Add `hubx-mfe/src/pages/DownloadEvidence.tsx`, register `/download-evidence` in `hubx-mfe/src/App.tsx` behind `ProtectedRoute` and add an operator-only link from `hubx-mfe/src/pages/Dashboard.tsx`
- [x] T020 [P] [MFE] Add unit tests in `hubx-mfe/src/features/evidence/DownloadEvidence.test.tsx` and `hubx-mfe/src/services/evidenceService.test.ts`; run `npm --prefix hubx-mfe run test` and `npm --prefix hubx-mfe run build`

  Evidence (frontend-react, phase implement, 2026-11-25): `npm --prefix hubx-mfe run typecheck` exit 0,
  `npm --prefix hubx-mfe run test` → `15 files, 106 tests passed`, `npm --prefix hubx-mfe run lint` exit 0
  (4 warnings, all in pre-existing files), `npm --prefix hubx-mfe run build` exit 0. Commit `bdf93ec` touches only
  `hubx-mfe/**`; no new dependency, no `window.location`, no direct API call.

**Checkpoint**: US1 independently testable (API + browser download).

---

## Phase 4: User Story 2 - Evidence for a shipment without updates (Priority: P2)

**Goal**: the document states that a newly created shipment has no updates, without omitting the shipment data.

**Independent Test**: request evidence for a shipment in Created and check the PDF contains the shipment data and
`The shipment has no updates yet.`

- [x] T021 [API] Cover the empty-chain sentence in `hubx-api/internal/evidence/pdf.go` and its unit test in `hubx-api/internal/evidence/test/`

  Evidence: commit `2f9c9ae`; `pdf_test.go` asserts `The shipment has no updates yet.` and `go -C hubx-api test ./...` is green.

- [x] T022 [QA] Verify the "Evidence for a shipment without updates" scenario in `hubx-bender/features/HU10_download-evidence.feature` is green through the gateway

  Evidence: `SCENARIO line=20 | Evidence for a shipment without updates | steps=7 | passed` in
  `specs/010-download-evidence/qa-report.md` (commit `6a647de`).

**Checkpoint**: US2 independently testable.

---

## Phase 5: User Story 3 - Clear refusal when evidence cannot be given (Priority: P3)

**Goal**: no document for unknown code, wrong role or missing session, with privacy preserved.

**Independent Test**: request evidence with an unknown code (404), with a ground partner/customer (403) and with no
session (401); assert no PDF body in each case.

- [x] T023 [API] Implement and unit-test the flat error mapping in `hubx-api/internal/handlers/evidence/handler.go` and `hubx-api/internal/evidence/errors.go`: 401 `UNAUTHORIZED`, 403 `FORBIDDEN`, 404 `SHIPMENT_NOT_FOUND` for unknown **and** invisible shipments, 500 `INTERNAL_SERVER_ERROR`

  Evidence: commit `2f9c9ae`; `handler_test.go` covers 401/403/404/500 and `go -C hubx-api test ./...` is green.
  The 401 message is the gateway's (`contracts/auth-sign-in.md`); this route fixes no message of its own.
- [x] T024 [MFE] Map 401/403/404 to visible outcomes in `hubx-mfe/src/features/evidence/DownloadEvidenceContainer.tsx` so no file is saved on error

  Evidence: included in the `frontend-react` delivery above (commit `bdf93ec`); 401 is detected via `fetchWithAuth`'s
  token clearing, 403/404 messages are the exact contract strings, and unit tests verify no download is created on error.
- [x] T025 [QA] Verify the three refusal scenarios in `hubx-bender/features/HU10_download-evidence.feature` are green through the gateway

  Evidence: `SCENARIO line=27 | Code that matches no shipment | passed`,
  `line=39 | Role without permission: ground partner | passed`, `line=40 | ... customer | passed` and
  `line=42 | Person who has not signed in | passed` in `specs/010-download-evidence/qa-report.md` (commit `6a647de`).

**Checkpoint**: all three stories independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [x] T026 [QA] Run the full suite (`make test-integration`), `gosec ./...` and `govulncheck -C hubx-api ./...` / `-C hubx-gw ./...`; write literal evidence in `specs/010-download-evidence/qa-report.md`

  Evidence (qa-seguridad, phase verify, commit `6a647de`): `make test-integration` → `66 scenarios (66 passed)`,
  `424 steps (424 passed)`, `INTEGRATION_EXIT=0`; `gosec ./...` → `Issues : 0` / exit 0 in hubx-api and hubx-gw;
  both `govulncheck` runs → `No vulnerabilities found`. Only the pre-existing gateway gap (API4/API8) remains.

- [x] T027 [OPS] Run `make agents-zones` and confirm no CI, Dockerfile, compose or infrastructure file changed (demo mode: no devops deploy validation)

  Evidence (hubx-lead): `node scripts/check-agent-zones.mjs origin/main` → `agent zones ok (origin/main..HEAD)` after
  commits `2f9c9ae`, `bdf93ec`, `573808c`, `f81a551` and `6a647de`; `git diff` against `origin/main` contains no CI,
  Dockerfile, compose or infrastructure change. Demo mode: no devops delegation, no `deploy.md`.
- [ ] T028 Run `speckit.analyze` and record deviations in `specs/010-download-evidence/pr.md`
- [ ] T029 Update `backlog/backlog.md` (HU-10 status in review), `memory.md` (decisions and lessons) and `README.md` (delivered story) for this story only
- [ ] T030 Write `specs/010-download-evidence/pr.md` from `.github/pull_request_template.md`, including the demo-mode note, and push the story branch and open the PR to `release/v0.10.0-demo`

---

## Dependencies & Execution Order

- **Setup (Phase 1)**: no dependencies.
- **Foundational (Phase 2, QA red)**: blocks all implementation; the feature file and steps must fail for the right reason first.
- **US1 → US2 → US3**: each depends only on Phase 2; US2 and US3 reuse US1's reader/handler and can follow it closely.
- **QA verify (T026)**: depends on all three stories; a red `@e2e` blocks the story.
- **Polish (T028–T030)**: depends on a green QA report; demo mode skips devops deploy validation.

### Within each user story

Tests first (Phase 2) → domain entities → reader/service → handler → MFE service/view/route → integration.

### Parallel opportunities

- T002 and T003–T004 touch different QA files; T007/T008/T014/T015 and T017/T018/T020 are independent of each other.
- `[API]` and `[MFE]` deliveries run in parallel after Phase 2 because they share no files.
- T010 and T011 touch the same `internal/shipment` package and must be in the same backend-go delivery.

### Parallel Example: User Story 1

```bash
# Backend-go delivery: go generate + go test over evidence, handlers/shipment wiring and gateway check
# Frontend-react delivery: evidenceService + DownloadEvidence view/container + page/route/link + unit tests
# They share no files, so both can be delegated at the same time.
```

## Implementation Strategy

### MVP first (US1 only)

1. Setup + QA red (Phases 1–2).
2. Implement US1 in API and MFE; verify API and `@e2e` green.
3. Only then add US2 and US3.

### Incremental delivery

Each story adds content or error handling to the same endpoint without changing the others' behavior; the full suite
stays green after each increment.

### Notes

- One commit per accepted subagent delivery, tagged `[agent:<role>] HU-10 ...`, made by `hubx-lead` immediately after
  the snapshot comparison; only `hubx-lead` ticks tasks.
- Demo mode: no `deploy.md`, no devops delegation; `pr.md` records the human decision.

# Tasks: View my assigned shipments (HU-07)

Source: backlog HU-07 | Spec: [spec.md](spec.md) | Plan: [plan.md](plan.md) |
Contract: [contracts/hu07-view-assigned.md](../../contracts/hu07-view-assigned.md)

Order: `[QA]` red -> `[API]` and `[MFE]` in parallel -> `[QA]` verify -> analyze -> pr. `[OPS]` deploy validation is
skipped at the human's request. Only hubx-lead marks `- [x]`, citing evidence.

## Phase 1: Acceptance tests in red (qa-seguridad, `red`)

- [x] T001 [QA] `hubx-bender/features/HU07_view-assigned.feature`: the four backlog HU-07 scenarios verbatim (feature
  header as in the backlog) plus one `@e2e` happy path for a ground partner in the web application.
- [x] T002 [QA] `hubx-bender/features/steps/HU07_view-assigned.steps.ts` (+ shared helpers in `common.steps.ts` only if
  reused): fixtures through the gateway (create, advance, assign, drain the partner's pending list), actor mapping
  for "the first partner", "the partner", "an operator", "a customer", assertions against the contract (order by
  `assignedAt`, only own codes, 403 `FORBIDDEN` body, empty list + `message`).
- [x] T003 [QA] Run against the stack and show every HU-07 scenario red for the right reason (route missing: 404/405),
  with HU-01..HU-03 still green. Evidence (lead ran `make test-integration`, QA shell denied; HU-04 precedent):
  `65 scenarios (59 passed, 6 failed)`; five API scenarios `Expected: 200|403 Received: 404`, `@e2e`
  `getByRole('region', { name: /my assigned shipments/i }) ... element(s) not found`. A first run had 3 scenarios red
  for the wrong reason (step arity); qa-seguridad fixed them in `fix`.

## Phase 2: API (backend-go, `implement`) — TDD

- [x] T004 [API] Tests first in `internal/ledger/test/`: `PendingAssignments` query, parameters, scan and ordering.
- [x] T005 [API] `internal/ledger`: `Assignment` type, `LedgerRepository.PendingAssignments`, Postgres implementation.
- [x] T006 [API] Tests first in `internal/shipment/test/assigned_test.go`: role gate (`ErrNoPermission` for operator and
  customer), offset bounds (`ErrInvalidOffset`), limit 21 request, `NextOffset`, ledger order preserved, empty page.
- [x] T007 [API] `internal/shipment`: `ShipmentRepository.FindByIDs`, `AssignedLister`, `NewAssignedLister`, sentinels.
- [x] T008 [API] Tests first in `internal/handlers/shipment/test/assigned_handler_test.go`: 200 body, `message` only when
  empty, 403/400/500 flat bodies, offset parsing, partner from claims only.
- [x] T009 [API] `internal/handlers/shipment/assigned_handler.go` with `RegisterAssigned`; wire it in `cmd/api/main.go`
  with `RequireAuth`.
- [x] T010 [API] `go -C hubx-api generate ./...`, `go -C hubx-api test ./...`, `go -C hubx-gw test ./...` green.

T004-T010 evidence (commit 1afa4c9): backend-go ran `go -C hubx-api generate ./...` in its `fix` round (mocks carry
the MockGen header); the lead reran `go -C hubx-api test -count=1 ./...` (every test package `ok`) and
`go -C hubx-gw test -count=1 ./...` (`ok middleware/test`, `ok proxy/test`). The first delivery did not compile
(missing `context` import, unused variable, stale mocks) and went back as `fix`.

## Phase 3: MFE (frontend-react, `implement`) — TDD, parallel with Phase 2

- [x] T011 [MFE] Tests first: `shipmentService.test.ts` (`listAssignedShipments`), feature component tests (ordered rows,
  empty message, "Show next 20", 401 -> `onSessionExpired`, error alert), `Dashboard.test.tsx` (section only for
  `ground partner`).
- [x] T012 [MFE] `listAssignedShipments` in `src/services/shipmentService.ts`; `src/features/assigned-shipments/`
  component + container; render in `Dashboard.tsx` for ground partners.
- [x] T013 [MFE] `npm --prefix hubx-mfe run test`, `run lint`, `run build` green.

T011-T013 evidence (commit 5f8816b), rerun by the lead: `Test Files  14 passed (14)`, `Tests  105 passed (105)`;
lint exit 0 (4 pre-existing warnings, none in HU-07 files); `tsc -b && vite build` ok. The first delivery deleted
7 HU-02 `createShipment` tests and went back as `fix`; they are restored.

## Phase 4: Verify (qa-seguridad, `verify`)

- [x] T014 [QA] `make test-integration`: every scenario green, HU-07 API and `@e2e` included. Evidence (run by the
  lead at 5f8816b, QA shell denied): `65 scenarios (65 passed)`, `421 steps (421 passed)`, exit 0.
- [ ] T015 [QA] `gosec ./...`, `govulncheck -C hubx-api ./...`, `govulncheck -C hubx-gw ./...`; OWASP API Top 10 review.
  OWASP review done (qa-report.md). **gosec and govulncheck NOT RUN**: qa-seguridad's shell was denied and the lead's
  allowed commands do not include them. A human must run them before merge (constitution VI).
- [x] T016 [QA] `specs/007-view-assigned/qa-report.md` with the cited evidence.

## Phase 5: Deploy validation — SKIPPED

- [ ] T017 [OPS] ~~Deploy `007-view-assigned` to staging~~. **Skipped at the human's explicit request** (2026-11-24);
  recorded in `deploy.md` and `pr.md`.

## Phase 6: Close (hubx-lead)

- [x] T018 analyze: contrast spec, plan, tasks and code; record deviations in `pr.md`.
- [x] T019 backlog Story Status, `memory.md` (decisions, lessons, Pending), README if affected.
- [x] T020 `pr.md`, push `007-view-assigned`, open the PR into `release/v0.6.0-demo-07`.

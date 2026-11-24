# Tasks: View a shipment's traceability (HU-04)

Spec: [spec.md](./spec.md) · Plan: [plan.md](./plan.md) · Contract: `contracts/hu04-view-traceability.md`

Only hubx-lead marks a task `- [x]`, citing the subagent evidence.

## Phase 1 — QA red

- [ ] T001 [QA] `hubx-bender/features/HU04_view-traceability.feature`: backlog HU-04 scenarios verbatim + one `@e2e`
      happy path (customer who owns the shipment, through the MFE).
- [ ] T002 [QA] `hubx-bender/features/steps/HU04_view-traceability.steps.ts`: fixtures through the gateway only (HU-02
      create, HU-03 advance to "Assigned to partner"), assertions on order, stage, responsible party, date, and the
      identical 404 body for unknown and invisible shipments.
- [ ] T003 [QA] Run the HU-04 scenarios against the stack: red for the right reason (missing route / missing view).

## Phase 2 — Implementation (parallel)

### [API] backend-go — hubx-api

- [ ] T010 [API] Tests first: `internal/ledger/test` for `List` (order, empty, scan error).
- [ ] T011 [API] `LedgerRepository.List` + PostgreSQL implementation; regenerate `internal/ledger/mocks`.
- [ ] T012 [API] Tests first: `internal/shipment/test/traceability_test.go` (visibility matrix, unknown, empty chain,
      order, one party lookup, repository errors).
- [ ] T013 [API] `internal/shipment/traceability.go`, `ErrShipmentNotFound`, `PartyRepository.Parties`; regenerate
      `internal/shipment/mocks`.
- [ ] T014 [API] Tests first: `internal/handlers/shipment/test/traceability_handler_test.go` (200 shape, identical 404
      for unknown and invisible, 401, 500).
- [ ] T015 [API] `internal/handlers/shipment/traceability_handler.go` + `RegisterTraceability`; regenerate mocks;
      wire in `cmd/api/main.go`.
- [ ] T016 [API] `go -C hubx-api test ./...`, `go -C hubx-api build ./...`, `gosec ./...`, `govulncheck -C hubx-api ./...`.

### [GW] backend-go — hubx-gw

- [ ] T020 [GW] No code change; confirm `go -C hubx-gw test ./...` and `govulncheck -C hubx-gw ./...` stay green.

### [MFE] frontend-react — hubx-mfe

- [ ] T030 [MFE] Tests first: `src/api/client.test.ts` for `getTraceability` (path, JWT, 200/404/401/5xx outcomes).
- [ ] T031 [MFE] `getTraceability` in `src/api/client.ts`.
- [ ] T032 [MFE] Tests first: `src/features/traceability/ViewTraceability.test.tsx` (list order, fields, empty chain,
      not-found message).
- [ ] T033 [MFE] `ViewTraceability.tsx`, `ViewTraceabilityContainer.tsx`, rendered in `src/pages/Dashboard.tsx`.
- [ ] T034 [MFE] `npm --prefix hubx-mfe run test` and `npm --prefix hubx-mfe run build`.

## Phase 3 — QA verify

- [ ] T040 [QA] `make test-integration` (all features, API and `@e2e`) green.
- [ ] T041 [QA] gosec + govulncheck (both services), OWASP API Security Top 10 review (API1 BOLA focus),
      `specs/004-view-traceability/qa-report.md`.

## Phase 4 — DevOps

- [ ] T050 [OPS] Confirm no CI/compose/deploy change is needed (no new service, env var or migration); CI workflows
      cover the new tests unchanged.

## Phase 5 — Lead

- [ ] T060 analyze (spec ↔ plan ↔ tasks ↔ code), record deviations.
- [ ] T061 backlog status, memory.md, README, `pr.md`.

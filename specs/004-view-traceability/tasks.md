# Tasks: View a shipment's traceability (HU-04)

Spec: [spec.md](./spec.md) · Plan: [plan.md](./plan.md) · Contract: `contracts/hu04-view-traceability.md`

Only hubx-lead marks a task `- [x]`, citing the subagent evidence.

## Phase 1 — QA red

- [x] T001 [QA] `hubx-bender/features/HU04_view-traceability.feature`: backlog HU-04 scenarios verbatim + one `@e2e`
      happy path (customer who owns the shipment, through the MFE). Evidence: file delivered by qa-seguridad (red).
- [x] T002 [QA] `hubx-bender/features/steps/HU04_view-traceability.steps.ts`: fixtures through the gateway only (HU-02
      create, HU-03 advance to "Assigned to partner"), assertions on order, stage, responsible party, date, and the
      identical 404 body for unknown and invisible shipments.
- [x] T003 [QA] Run the HU-04 scenarios against the stack: red for the right reason (missing route / missing view).
      Evidence: qa-seguridad's shell was denied (`deny shell matching "*"`), so hubx-lead ran the verification
      `make test-integration`: `51 scenarios (43 passed, 8 failed)`, `328 steps (314 passed, 6 skipped, 8 failed)`;
      the 8 failures are the HU-04 scenarios only (no `200 traceability response`, mux `404 page not found` not JSON,
      `@e2e` timeout locating the "Shipment code" field); no undefined step (strict mode).

## Phase 2 — Implementation (parallel)

### [API] backend-go — hubx-api

- [x] T010 [API] Tests first: `internal/ledger/test` for `List` (order, empty, scan error).
- [x] T011 [API] `LedgerRepository.List` + PostgreSQL implementation; regenerate `internal/ledger/mocks`.
- [x] T012 [API] Tests first: `internal/shipment/test/traceability_test.go` (visibility matrix, unknown, empty chain,
      order, one party lookup, repository errors).
- [x] T013 [API] `internal/shipment/traceability.go`, `ErrShipmentNotFound`, `PartyRepository.Parties`; regenerate
      `internal/shipment/mocks`.
- [x] T014 [API] Tests first: `internal/handlers/shipment/test/traceability_handler_test.go` (200 shape, identical 404
      for unknown and invisible, 401, 500).
- [x] T015 [API] `internal/handlers/shipment/traceability_handler.go` + `RegisterTraceability`; regenerate mocks;
      wire in `cmd/api/main.go`.
- [x] T016 [API] `go -C hubx-api test ./...`, `go -C hubx-api build ./...`, `gosec ./...`, `govulncheck -C hubx-api ./...`.
      Evidence (commit 206dbac): red after mock generation, `traceability_test.go:6:2: "reflect" imported and not used`
      and `TestHU04_List/returns_three_updates_ordered_by_sequence_ascending` (empty `ShipmentID`); backend-go fixed
      both in one `fix` round. Rerun by hubx-lead: `go -C hubx-api test -count=1 ./...` all 8 test packages `ok`;
      `go -C hubx-api build ./...` and `go -C hubx-api vet ./...` exit 0; gosec `Files : 50 ... Issues : 0`;
      govulncheck `Your code is affected by 0 vulnerabilities.`

### [GW] backend-go — hubx-gw

- [x] T020 [GW] No code change; confirm `go -C hubx-gw test ./...` and `govulncheck -C hubx-gw ./...` stay green.
      Evidence (run by hubx-lead): `ok hubx-gw/internal/middleware/test`, `ok hubx-gw/internal/proxy/test`;
      govulncheck `No vulnerabilities found.`; gosec `Files : 3 ... Issues : 0`.

### [MFE] frontend-react — hubx-mfe

- [x] T030 [MFE] Tests first: `src/api/client.test.ts` for `getTraceability` (path, JWT, 200/404/401/5xx outcomes).
- [x] T031 [MFE] `getTraceability` in `src/api/client.ts`.
- [x] T032 [MFE] Tests first: `src/features/traceability/ViewTraceability.test.tsx` (list order, fields, empty chain,
      not-found message).
- [x] T033 [MFE] `ViewTraceability.tsx`, `ViewTraceabilityContainer.tsx`, rendered in `src/pages/Dashboard.tsx`.
- [x] T034 [MFE] `npm --prefix hubx-mfe run test` and `npm --prefix hubx-mfe run build`. Evidence (commit cd4c1b1),
      rerun by hubx-lead after one `fix` round (two failing tests: duplicate "Shipment code" label in the Dashboard
      test, wrong 401 mock body): `Test Files  9 passed (9)`, `Tests  74 passed (74)`; build `tsc -b && vite build`
      `✓ built in 130ms`.

> Blocker (T010-T016, T020): backend-go wrote the code and tests and added the `//go:generate` directives, but its
> shell is denied in this session (`deny shell matching "*"`), so it could not run `go -C hubx-api generate ./...`.
> Without the generated mocks, `go -C hubx-api test ./...` fails to build `internal/ledger/test`,
> `internal/shipment/test` and `internal/handlers/shipment/test` (missing `MockRows`, `MockDB.QueryContext`,
> `MockLedgerRepository.List`, `MockPartyRepository`, `MockTraceabilityReader`). Production code builds
> (`go -C hubx-api build ./...` and `go vet` on the changed packages exit 0). Waiting for a human decision.
>
> Resolved (human decision): the human ran `go -C hubx-api generate ./...` (exit 0, mechanical from the existing
> `//go:generate` directives); the generated mocks are committed with backend-go's delivery (206dbac) as an exception
> like HU-01.

## Phase 3 — QA verify

- [x] T040 [QA] `make test-integration` (all features, API and `@e2e`) green. Evidence (run by hubx-lead with human
      approval, qa-seguridad's shell denied): first run after 206dbac `51 scenarios (46 passed, 5 failed)`, all
      test-side (HU-03 `@e2e` locator ambiguity, wrong fixture actors); qa-seguridad fixed them (8976cb4); rerun
      `51 scenarios (51 passed)`, `328 steps (328 passed)`, exit 0.
- [x] T041 [QA] gosec + govulncheck (both services), OWASP API Security Top 10 review (API1 BOLA focus),
      `specs/004-view-traceability/qa-report.md`. Evidence: qa-seguridad delivered the report (a2f31d4, citations
      corrected in one `fix` round); only open finding is the pre-existing F-API8-01 (medium).

## Phase 4 — DevOps

- [x] T050 [OPS] Confirm no CI/compose/deploy change is needed (no new service, env var or migration); CI workflows
      cover the new tests unchanged. Evidence: devops `status: ok`, read-only (`.github/workflows/ci.yml` runs each
      service's tests and `make test-integration`; `hubx-mfe/Dockerfile` proxies `/api/` generically; no new env var
      in `compose.yaml` or `deploy/render.yaml`; `package.json` files unchanged). No devops commit.

## Phase 5 — Lead

- [x] T060 analyze (spec ↔ plan ↔ tasks ↔ code), record deviations. Evidence: deviations listed in `pr.md`.
- [x] T061 backlog status, memory.md, `pr.md`. README unchanged: HU-04 is not merged or released yet.

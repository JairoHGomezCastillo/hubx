# Implementation Plan: View a shipment's traceability

**Branch**: `004-view-traceability` | **Date**: 2026-11-24 | **Spec**: [spec.md](./spec.md) |
**Release branch**: `release/v0.4.0-ledger`

## Summary

A signed-in participant enters a shipment code and sees the shipment's current stage and its ledger of updates, oldest
first, each with stage, responsible party and date. hubx-api adds one read-only route that looks up the shipment,
applies the HU-03 visibility rule (operator: any; customer: own; ground partner: assigned) and reads the chain through
`LedgerRepository`. Unknown and invisible shipments get the same 404. The MFE adds a traceability view on the
dashboard that calls `/api` through the gateway. No migration, no gateway change, no new dependency.

## Technical Context

**Language/Version**: Go 1.27.4; TypeScript strict, React, Vite (versions unchanged from HU-03).

**Primary Dependencies**: none new. Go stdlib `net/http` + `ServeMux`, `log/slog`, `database/sql` + pgx v5.12.0,
`go.uber.org/mock` v0.6.0 (tests). MFE vitest + Testing Library (already declared). QA Cucumber 13.3.0 + Playwright
1.65.0 + tsx 4.23.15.

**Storage**: PostgreSQL, schema `0001_baseline_schema.sql` (already applied). HU-04 needs **no migration**: it reads
`shipments` (code, stage, customer_id, assigned_partner_id), `updates` (ordered by the `UNIQUE (shipment_id, sequence)`
index) and `users` (name, role). Migration number `0002` stays reserved for `0002_seed_ledger`.

**Testing**: Go black-box unit tests in `test/` with gomock mocks in `mocks/`; MFE vitest; Cucumber feature
`hubx-bender/features/HU04_view-traceability.feature` (backlog Gherkin verbatim + one `@e2e`); `make test-integration`;
`gosec ./...`, `govulncheck -C hubx-api ./...`, `govulncheck -C hubx-gw ./...`.

**Performance**: one shipment lookup by unique code, one ordered ledger read, one party lookup for the distinct party ids
(`WHERE id = ANY($1)`): three queries per request regardless of chain length, no N+1.

**Constraints**: read-only; privacy indistinguishability; parameterized SQL; English UI.

**Scope**: one API route, one MFE view, one feature file. No verification (HU-05), no lists (HU-06/07), no evidence
(HU-10).

## HTTP contract

`contracts/hu04-view-traceability.md`: `GET /api/shipments/{code}/traceability` → 200 `{shipmentCode, stage, updates[]}`;
404 `SHIPMENT_NOT_FOUND` "No shipment exists with that code." for both unknown and invisible shipments; 401 is the
HU-01 flat body (the gateway's nested `SESSION_EXPIRED` body is limited to the HU-03 routes, so HU-04 uses the API's
`RequireAuth`, keeping both layers consistent with no gateway change).

## Design

### hubx-api

| Path | Change |
|---|---|
| `internal/ledger/repository.go` | [modified] add `List(ctx, shipmentID string) ([]Update, error)` to `LedgerRepository`: every update of the shipment ordered by `sequence` ascending; empty slice when none (Principle I: all ledger access through this interface) |
| `internal/ledger/postgres.go` | [modified] parameterized `SELECT ... FROM updates WHERE shipment_id = $1 ORDER BY sequence ASC`; needs a multi-row `Query` on the injectable `DB` (add `QueryContext` + `Rows` to the interface) |
| `internal/ledger/mocks/*` | [regenerated] `go -C hubx-api generate ./...` |
| `internal/shipment/traceability.go` | [new] `TraceabilityReader` interface + `NewTraceabilityReader(shipments ShipmentRepository, ledger ledger.LedgerRepository, parties PartyRepository) TraceabilityReader`. `Read(ctx, code, principal) (*Traceability, error)`: `FindByCode`; `nil` or `!canSee` → `ErrShipmentNotFound`; `ledger.List`; resolve the distinct responsible and assigned party ids once. Reuses `canSee` from `recorder.go` (same rule, no copy) |
| `internal/shipment/errors.go` | [modified] sentinel `ErrShipmentNotFound` |
| `internal/shipment/repository.go` / `update_postgres.go` | [modified] `PartyRepository.Parties(ctx, ids []string) (map[string]Party, error)` (`Party{ID, Name, Role}`), one parameterized query `SELECT id, name, role FROM users WHERE id = ANY($1)` |
| `internal/shipment/mocks/*`, `internal/shipment/test/traceability_test.go` | [regenerated] / [new] black-box tests: operator any, customer own, customer other → not found, partner assigned, partner unassigned → not found, unknown → not found, repository error → wrapped error (the handler answers 500; that answer does not depend on whether the shipment exists), empty chain, order preserved, single party lookup |
| `internal/handlers/shipment/traceability_handler.go` | [new] `RegisterTraceability(mux, requireAuth, reader, principals)` mounts `GET /shipments/{code}/traceability`; maps `ErrShipmentNotFound` → 404, other errors → 500 (logged with slog, no fingerprint or token logged); writes the contract JSON (`updates: []` never `null`, `recordedAt` RFC3339Nano UTC) |
| `internal/handlers/shipment/test/traceability_handler_test.go`, `mocks/` | [new] / [regenerated] handler tests with gomock: 200 body shape, 404 body identical for unknown and invisible, 401 when no principal, 500 body |
| `cmd/api/main.go` | [modified] wiring only: `shipment.RegisterTraceability(mux, authMiddleware.RequireAuth, reader, shipment.NewClaimsPrincipalSource())` |

The handler package already imports `internal/handlers/auth` for claims; no new import across services.

### hubx-gw

No change. `GET /api/shipments/{code}/traceability` is already JWT-validated and proxied with `Authorization` intact;
`usesSessionErrorContract` returns false for it, so the flat 401 body matches the API's `RequireAuth`.

### hubx-mfe

| Path | Change |
|---|---|
| `src/api/client.ts` (+ `client.test.ts`) | [modified] `getTraceability(code)` → `GET /api/shipments/{encodeURIComponent(code)}/traceability`, explicit outcomes: `ok`, `not-found`, `session-expired`, `error` |
| `src/features/traceability/ViewTraceability.tsx` (+ test) | [new] accessible form: label "Shipment code", button "View traceability"; result: heading "Traceability of <code>", text "Current stage: <stage>", an ordered list (role `list`, name "Updates") with one item per update showing stage, responsible party name (and role) and the date with time zone; "This shipment has no updates yet." when empty; "No shipment exists with that code." on 404 (role `alert`). Text kept in single text nodes (lesson from HU-02: no split `<strong>` labels) |
| `src/features/traceability/ViewTraceabilityContainer.tsx` | [new] wires the client, handles session expiry like `RecordUpdateContainer` |
| `src/pages/Dashboard.tsx` (+ test) | [modified] renders the traceability view for every role |

### hubx-bender

| Path | Change |
|---|---|
| `features/HU04_view-traceability.feature` | [new] backlog HU-04 Gherkin verbatim + one `@e2e` "Viewing by an authorized participant" (customer who owns the shipment) through the MFE |
| `features/steps/HU04_view-traceability.steps.ts` | [new] fixtures: create a fresh shipment via HU-02 for a known customer, advance it through HU-03 to "Assigned to partner" (three updates) assigning `rapido@hubx.co`; reuse the common `has signed in` step; never write to the DB |

## Test strategy (TDD)

1. QA `red`: feature + steps run against the stack before implementation; every HU-04 scenario fails because the route
   returns 404 from the mux (no route) or the UI lacks the view, not because of a step error.
2. backend-go and frontend-react in parallel, unit tests first (red → green).
3. QA `verify`: full `make test-integration` (all features, API and `@e2e`), gosec, govulncheck, OWASP API Top 10
   review (API1 BOLA is the key risk here), `qa-report.md`.

## Constitution Check

| Principle | Decision | Result |
|---|---|---|
| I Ledger | Read through new `LedgerRepository.List`; no write path; immutability untouched | PASS |
| II Gateway | MFE calls `/api` only; no service imports; API validates JWT (`RequireAuth`) and the shared secret (`RequireGatewaySecret`) | PASS |
| III Stack | No new dependency, no new property, no migration | PASS |
| IV Modular | Domain `internal/shipment` (no `net/http`), handler `internal/handlers/shipment` with `RegisterTraceability`, `test/` + generated `mocks/`, constructors return interfaces, sentinel mapped in handler | PASS |
| V Test-first | QA red first; backlog Gherkin verbatim; `@e2e` happy path; unit tests first | PASS |
| VI Security | Parameterized queries; own-resources only; invisible = unknown (same status and body); no secrets in logs | PASS |
| VII Language | English everywhere, glossary terms | PASS |

No violation; Complexity Tracking not applicable.

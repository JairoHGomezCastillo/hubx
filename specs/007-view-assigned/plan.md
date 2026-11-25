# Implementation Plan: View my assigned shipments

**Branch**: `007-view-assigned` | **Release branch**: `release/v0.6.0-demo-07` | **Date**: 2026-11-24 |
**Spec**: [spec.md](spec.md) | **Contract**: [contracts/hu07-view-assigned.md](../../contracts/hu07-view-assigned.md)

Source: backlog HU-07

## Summary

A signed-in ground partner gets the shipments assigned to them that are not yet delivered, oldest assignment first,
20 at a time. Operators and customers get a 403. Read-only: one new API route, one new MFE section on the Dashboard,
one new feature file. No migration and no gateway change.

## Technical Context

- **Base**: `main` at `8c47690` (v0.4.0, PR #36), after the human re-based the release branch (it was first cut at
  v0.3.4). It contains HU-01 to HU-05 and the UI refresh (`AppShell`, Dashboard `workspace` layout).
- **Existing code reused**: `LedgerRepository` already has `Last`, `List`, `Append` (HU-03/HU-04);
  `ShipmentRepository` has `FindByCode`, `FindByID`; `PartyRepository.Parties` exists (HU-04) but HU-07 does not need
  party names. Adding `PendingAssignments` to `LedgerRepository` regenerates both `ledger/mocks` and
  `shipment/mocks/ledger_repository_mock.go`; adding `FindByIDs` to `ShipmentRepository` regenerates its mocks and
  every existing test that builds a fake `ShipmentRepository` must keep compiling.
- **API**: Go 1.27.4, stdlib `net/http` `ServeMux`, pgx v5 through `database/sql`, gomock (`go tool mockgen`).
- **MFE**: React + TypeScript strict + Vite, Vitest + Testing Library.
- **QA**: Cucumber 13.3.0 + Playwright 1.65.0 + tsx in `hubx-bender`.
- **Storage**: PostgreSQL, schema 0001 unchanged (`updates`, `shipments`, `users`).
- **New dependencies**: none.

## Constitution Check

| Principle | How this plan complies |
|---|---|
| I. Immutable ledger | The assignment date and the "not delivered" filter are read from the canonical `updates` table only through a new read method on `LedgerRepository`; nothing is written. |
| II. Gateway | MFE calls `/api/assigned-shipments` on the gateway; the API keeps `RequireGatewaySecret` and its own JWT check (`RequireAuth`). The gateway already validates JWT on every non-excluded route, so it needs no change. |
| III. Stack | No new dependency, stdlib only. |
| IV. Modular design | Domain logic in `internal/shipment` (no `net/http`), handler in `internal/handlers/shipment`, ledger query in `internal/ledger`; constructor injection; mocks regenerated with `go generate`; tests in `test/`. |
| V. Test-first | qa-seguridad writes `HU07_view-assigned.feature` (backlog Gherkin verbatim + one `@e2e`) red before code; backend-go and frontend-react write failing unit tests first. |
| VI. Security | Role check on the server; partner scope from JWT `sub` (no partner parameter); parameterized queries; flat 403 reveals no data; OWASP API Top 10 review in verify. |
| VII. Language | English identifiers, messages and UI text. |

No violations; no complexity to justify.

## Design

### Data flow

```text
MFE Dashboard (ground partner) -> GET /api/assigned-shipments?offset=N -> gateway (JWT, strip /api, secret)
  -> API RequireAuth -> handler (principal from claims) -> shipment.AssignedLister.List(principal, offset)
     -> ledger.LedgerRepository.PendingAssignments(partnerID, limit 21, offset)  [updates table]
     -> shipment.ShipmentRepository.FindByIDs(ids)                               [shipments table]
```

### hubx-api

- `internal/ledger`:
  - New type `Assignment { ShipmentID string; UpdateID int64; AssignedAt time.Time }`.
  - `LedgerRepository.PendingAssignments(ctx, partnerID string, limit, offset int) ([]Assignment, error)` with one
    parameterized query:

    ```sql
    SELECT u.shipment_id, u.id, u.recorded_at
    FROM updates u
    WHERE u.stage = 'Assigned to partner' AND u.assigned_partner_id = $1
      AND NOT EXISTS (SELECT 1 FROM updates d WHERE d.shipment_id = u.shipment_id AND d.stage = 'Delivered')
    ORDER BY u.recorded_at ASC, u.id ASC
    LIMIT $2 OFFSET $3
    ```

    Existing index `UNIQUE (shipment_id, sequence)` covers the `NOT EXISTS` probe; volume in this version does not need
    a new index (no migration).
- `internal/shipment`:
  - `ShipmentRepository.FindByIDs(ctx, ids []string) ([]Shipment, error)` (`WHERE id = ANY($1)`), returns code,
    origin, destination, flight, weight, stage.
  - New `AssignedLister` interface and constructor `NewAssignedLister(ledger.LedgerRepository, ShipmentRepository)`:
    `List(ctx, principal Principal, offset int) (AssignedPage, error)`. Rules: role must be `ground partner`, else
    sentinel `ErrNoPermission`; asks the ledger for `PageSize+1` rows to compute `NextOffset`; keeps ledger order;
    joins shipment data by id. `PageSize = 20`, `MaxOffset = 100000` (sentinel `ErrInvalidOffset`).
- `internal/handlers/shipment`:
  - `RegisterAssigned(mux, requireAuth, lister, principals)` mounts `GET /assigned-shipments`.
  - Parses `offset` (digits only, default 0), maps `ErrNoPermission` -> 403 `FORBIDDEN`, `ErrInvalidOffset` -> 400
    `INVALID_REQUEST`, other errors -> 500 with `slog` log. Writes `message` only for an empty list.
  - Contract validation order (role before `offset`): an unparsable `offset` is passed to the domain as `-1`, and the
    domain checks the role first, then the offset range, so a customer with a bad `offset` still gets 403.
- `cmd/api/main.go`: wire `RegisterAssigned` with `authMiddleware.RequireAuth` (flat 401, HU-01/HU-02/HU-04 style).
- Mocks regenerated by `go -C hubx-api generate ./...` (the interfaces of `ledger` and `shipment` change).

### hubx-gw

No change. `/api/assigned-shipments` is not in the excluded paths and not in `usesSessionErrorContract`, so a bad
session gets the flat `UNAUTHORIZED` body that the API also returns.

### hubx-mfe

- `src/services/shipmentService.ts`: `listAssignedShipments(offset?: number)` through the existing HTTP client
  (`/api` base, in-memory JWT); typed response; a 401 surfaces as a session-expired error.
- New feature folder `src/features/assigned-shipments/` with `AssignedShipments.tsx` (presentational) and a container:
  a `region` named "My assigned shipments" (heading level 2) with a table (code, stage, origin, destination, flight,
  weight, assigned date), the empty message "You have no assigned shipments", a "Show next 20" button when
  `nextOffset` is not null, and an accessible error `alert`. On session expiry it calls `onSessionExpired`.
- `Dashboard.tsx` (current UI-refresh layout: `AppShell`, `page-head`, `workspace` with `RecordUpdateContainer` and
  `ViewTraceabilityContainer`): render the section inside `workspace` only when `user.role === 'ground partner'`,
  reusing the existing CSS classes (no new styling system). Locators stay scoped to the region (HU-04 lesson).
- Unit tests first (Vitest): list rendering in order, empty message, next page, 401 handling, not rendered for other
  roles.

### hubx-bender

- `features/HU07_view-assigned.feature`: the four backlog scenarios verbatim, plus one `@e2e` scenario: a ground
  partner signs in through the web application and sees their pending shipments, oldest assignment first.
- `features/steps/HU07_view-assigned.steps.ts`, reusing `common.steps.ts` helpers (`prepareShipment`,
  `postUpdate`, sign-in). Actors: first partner `rapido@hubx.co` (Transportes Rápido), another partner
  `express@hubx.co` (Logística Express). New actor phrases ("the first partner", "the partner", "a customer") must be
  mapped explicitly (memory lesson from HU-04).
- **Fixture decision (no new data)**: migration 0001 has only two ground partners and the test database accumulates
  assignments across scenarios (HU-03 assigns Transportes Rápido). To establish "only their three shipments" and "no
  shipment has been assigned", the Given first drains the partner's pending list through the gateway: the partner
  records Out for delivery and Delivered on each pending shipment (the real HU-03 flow). The list then contains only
  what the scenario creates, and the drained shipments also exercise the Delivered exclusion. No users are seeded and
  no test bypasses the gateway.
- `@e2e` Given builds the assigned shipments through the API (HU-04 precedent accepted by the human) and the browser
  part never calls the API.

## Test strategy

| Level | Where | What |
|---|---|---|
| Unit (Go, TDD) | `internal/shipment/test/assigned_test.go`, `internal/handlers/shipment/test/assigned_handler_test.go`, `internal/ledger/test/postgres_test.go` | role check, page size + 1, next offset, order preserved, empty list, offset bounds, status/body mapping, query parameters (sqlmock-free: the repository tests follow the existing `DB` interface mock) |
| Unit (MFE, TDD) | `src/features/assigned-shipments/*.test.tsx`, `src/services/shipmentService.test.ts`, `src/pages/Dashboard.test.tsx` | rendering, empty message, paging, 401, role visibility |
| Acceptance (API) | `HU07_view-assigned.feature` | four backlog scenarios (five with the outline examples) through the gateway |
| Acceptance (`@e2e`) | same feature | ground partner happy path in Chromium through MFE, gateway, API |
| Security | verify | `gosec ./...`, `govulncheck` (api, gw), OWASP API Top 10 |

## Deployment

**Deployment validation (lead step 6d) is skipped at the human's explicit request** for HU-07. devops is not asked to
deploy the story branch; `deploy.md` and `pr.md` record the skip. Staging still deploys the release
branch when this PR is merged into it (constitution 1.5.0, Environments).

## Risks

- Resolved: the release branch was first cut from v0.3.4; the human re-based it on v0.4.0, so the merge conflicts
  this plan predicted (`Dashboard.tsx`, `common.steps.ts`, `cmd/api/main.go`) no longer apply. The story branch
  already contains `main` at v0.4.0.
- HU-04 and HU-05 steps share `common.steps.ts`; the HU-07 edit there must keep the HU-04/HU-05 scenarios green.
- The fixture drain changes the meaning of "no shipment has been assigned" to "no pending shipment is assigned" in the
  test data; the observable result (empty list and message) is the one the scenario asks for.

# Implementation Plan: Create a shipment

**Branch**: `002-create-shipment` | **Date**: 2026-11-24 | **Spec**: [spec.md](./spec.md)

**Source**: backlog HU-02 | **Release**: v0.3.0 (`release/v0.3.0-create-shipment`)

## Summary

An operator creates a shipment (origin, destination, flight, weight, customer email). The API validates it, generates
a unique code, stores a row in `shipments` with stage `Created` and records **no** update in the ledger. Ground
partners and customers get `403`. The API attaches its own JWT middleware to this first protected route, closing HU-01
deviation 9. The MFE adds a "Create shipment" page for operators.

## Technical Context

- **Language/Version**: Go 1.27.4 (hubx-api), React + TypeScript strict + Vite (hubx-mfe), PostgreSQL.
- **Dependencies**: none new. hubx-api uses pgx v5 (`database/sql`; `pgconn.PgError` for the unique-violation code
  `23505`), golang-jwt v5 (through the existing `internal/auth`), `go.uber.org/mock` for tests. hubx-mfe: existing
  vitest + Testing Library. hubx-bender: existing Cucumber + Playwright.
- **Storage**: existing `shipments` table from `0001_baseline_schema.sql`. **No migration.** Its constraints already
  enforce the rules as defense in depth: `code UNIQUE`, `weight_kg numeric(6,1) CHECK (BETWEEN 0.1 AND 5000)`,
  `CHECK (lower(btrim(origin)) <> lower(btrim(destination)))`, non-blank text checks, `stage DEFAULT 'Created'`,
  `customer_id` and `created_by` FK to `users`.
- **Testing**: Go unit tests in `test/` sub-packages with gomock; MFE vitest; Cucumber feature
  `hubx-bender/features/HU02_create_shipment.feature` through the gateway, plus one `@e2e`.
- **Affected services**: hubx-api (new domain + handler, wiring), hubx-mfe (new page + service), hubx-bender (feature
  + steps). hubx-gw: **no code change** (it already validates the JWT on every non-public path and proxies `/api/*`).

## HTTP contract

`contracts/shipments-create.md` (v1.0): `POST /api/shipments` -> `201` with the shipment; errors `400`
(`MISSING_REQUIRED_FIELDS`, `SAME_ORIGIN_DESTINATION`, `WEIGHT_OUT_OF_RANGE`), `401`, `403 FORBIDDEN`, `422`
(`CUSTOMER_NOT_FOUND`, `CUSTOMER_NOT_CUSTOMER_ROLE`). Validation order is fixed in the contract: session, role, body,
origin/destination, weight, customer.

## Data model

See [data-model.md](data-model.md). Insert only into `shipments`:
`INSERT INTO shipments (code, origin, destination, flight, weight_kg, customer_id, created_by) VALUES ($1..$7)
RETURNING created_at`. Customer lookup: `SELECT id, name, role FROM users WHERE email = $1`.

## Technical decisions

| # | Decision | Principle |
|---|---|---|
| D1 | Domain `hubx-api/internal/shipment` (package `shipment`, no `net/http`): `Shipment`, `NewShipment`, `Service` interface (`Create(ctx, CreateInput) (Shipment, error)`), `Repository` interface (`FindUserByEmail`, `Insert`), `CodeGenerator` interface, sentinel errors (`ErrForbidden`, `ErrMissingData`, `ErrSameOriginDestination`, `ErrWeightOutOfRange`, `ErrCustomerNotFound`, `ErrNotACustomer`, `ErrDuplicateCode`). Constructors return interfaces. | IV |
| D2 | Handlers `hubx-api/internal/handlers/shipment` (package `shipment`, domain alias `domain`), `Register(mux, db, requireAuth func(http.Handler) http.Handler)` registers `POST /shipments` wrapped in `RequireAuth`. Sentinel errors map to the contract statuses in the handler. | IV, II |
| D3 | The API wires `RequireAuth` on `POST /shipments` (first protected API route) and a unit test proves the route rejects a missing/expired token without the gateway. Closes HU-01 deviation 9. | II, VI |
| D4 | Authorization from the validated claims: `role == "operator"` or `ErrForbidden`, checked **before** decoding the body. `created_by` = claim `sub`. | VI (OWASP API5, API3) |
| D5 | Weight decoded as `json.Number`; the service checks range `[0.1, 5000]` and at most one decimal by parsing the literal, so no float rounding is involved. Stored as `numeric(6,1)`. | VI (server-side validation) |
| D6 | Code `HBX-` + 8 upper-case hex chars from `crypto/rand`. On `23505` for the code constraint the service retries with a new code, at most 3 attempts, then returns an internal error. Generator behind `CodeGenerator` so tests are deterministic. | IV, VI |
| D7 | Creating a shipment does not touch `updates` and does not use `LedgerRepository` (none exists yet; it arrives with HU-03). Stage `Created` comes from the column default and is returned as a constant. | I |
| D8 | Body limited to 16 KiB with `http.MaxBytesReader`; strings trimmed; customer email normalized (`strings.ToLower(strings.TrimSpace)`) like HU-01. | VI (OWASP API4) |
| D9 | MFE: `src/services/shipmentService.ts` (`createShipment` through the existing `httpClient`, base `/api`), `src/pages/CreateShipment.tsx` at route `/shipments/new` (protected). Labels: `Origin`, `Destination`, `Flight`, `Weight (kg)`, `Customer email`; button `Create shipment`. Success: heading `Shipment created`, text `Code: <code>` and `Stage: Created`. Errors in a `role="alert"` region with the contract `message`. Dashboard shows a `Create shipment` button only to operators. Non-operators on `/shipments/new` see `You do not have permission to create shipments`. Client-side checks only for empty fields; the server is the authority. | II, III |
| D10 | Bender: the backlog Gherkin verbatim; the `@e2e` happy path is the scenario "Shipment created with valid data" repeated verbatim with the `@e2e` tag (the untagged copy keeps the API coverage, as HU-01 did with its second `Examples:`). Fixtures: "Distribuciones Andina" = `andina@hubx.co`, ground partner = `rapido@hubx.co`, unregistered = `nobody@hubx.co`. "The shipment is not created" is asserted by the non-201 status and the absence of a code (no read endpoint exists until HU-04). | V |
| D11 | Interface language English. Error messages as in the contract. | VII |

## Constitution Check

| Principle | Status |
|---|---|
| I Ledger | Pass: no update recorded on creation; `updates` untouched; stage column is a projection that starts at `Created` |
| II Architecture | Pass: MFE -> gateway -> API; API validates JWT itself (D3) and keeps the gateway-secret guard |
| III Stack | Pass: no new dependency, no migration, stdlib `ServeMux` pattern `"POST /shipments"` |
| IV Modular | Pass: `internal/shipment` + `internal/handlers/shipment`, `test/` + `mocks/` via `go tool mockgen` |
| V Test-first | Pass: QA red first; Go unit tests red before code; `@e2e` happy path |
| VI Security | Pass: role check server-side, parameterized queries, body cap, no mass assignment |
| VII Language | Pass: English identifiers and messages |

**Gate**: PASS, no complexity to justify.

## Project structure

```text
contracts/shipments-create.md                         # NEW (hubx-lead)
hubx-api/internal/shipment/                           # NEW domain
  shipment.go  service.go  repository.go  code.go  errors.go
  mocks/  (generated)   test/service_test.go  test/repository_test.go  test/code_test.go
hubx-api/internal/handlers/shipment/                  # NEW handlers
  handler.go  register.go  test/handler_test.go
hubx-api/cmd/api/main.go                              # wire shipment.Register with authMiddleware.RequireAuth
hubx-mfe/src/services/shipmentService.ts (+ .test.ts) # NEW
hubx-mfe/src/pages/CreateShipment.tsx (+ .test.tsx)   # NEW
hubx-mfe/src/App.tsx, src/pages/Dashboard.tsx         # route + operator-only button
hubx-bender/features/HU02_create_shipment.feature     # NEW
hubx-bender/features/steps/HU02_create_shipment.steps.ts
```

## Test strategy

- **Acceptance (QA)**: 5 backlog scenarios / 13 examples via API through the gateway + 1 `@e2e`. All red before
  implementation for the right reason (route missing -> 404/405, page missing).
- **Unit (API)**: service (each sentinel, validation order, retry on duplicate code, no repository call when forbidden
  or invalid), handler (status/body per sentinel, 401 without token, 403 before body decode, body cap), code generator
  format. The PostgreSQL repository has no new dependency for mocking SQL (no sqlmock); it is covered by the
  acceptance scenarios against the real database, as in HU-01.
- **Unit (MFE)**: service request/response mapping and error mapping; page renders labels, submits, shows code and
  stage, shows server message, hides form for non-operators.
- **Security (QA verify)**: gosec, govulncheck (both modules), OWASP API Top 10 review in `qa-report.md`.

## Implementation order

1. qa-seguridad `red` (feature + steps). 2. backend-go and frontend-react in parallel (no shared files).
3. qa-seguridad `verify` + `qa-report.md`. 4. devops: verify `make test-integration` and CI need no change.

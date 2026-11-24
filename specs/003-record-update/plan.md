# Implementation Plan: Record a shipment update

**Branch**: `003-record-update` | **Date**: 2026-11-24 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-record-update/spec.md`

## Summary

Enable the party responsible for each stage to advance a shipment and append a new immutable update to its traceability
chain. The API will validate the caller's role and visibility, enforce the strict stage sequence, require a registered
ground partner for the assignment update, compute the SHA-256 fingerprint of each new update from the previous one and
persist the update together with the shipment's projected stage in one transaction. All ledger access goes through a
`LedgerRepository` interface. The MFE will offer an accessible form to record the update, reached only through the
gateway. Acceptance is proven by the HU-03 Cucumber feature (backlog scenarios verbatim plus an `@e2e` browser happy
path) and by black-box Go unit tests with gomock.

## Technical Context

**Language/Version**: Go 1.27.4 (`go.mod`, `go.work`); TypeScript 7.0.2 strict, React 19.3.1, Vite 8.3.6.

**Primary Dependencies**: no new dependency. `net/http` + `ServeMux` (stdlib), `log/slog` JSON, `database/sql` with
`github.com/jackc/pgx/v5/stdlib` v5.12.0, `go.uber.org/mock` v0.6.0 (test only, via `go tool mockgen`), and the JWT /
bcrypt libraries already approved for HU-01. QA: Cucumber 13.3.0 + Playwright 1.65.0 + tsx 4.23.15.

**Storage**: PostgreSQL, managed by the local stack. The existing applied migration `0001_baseline_schema.sql` already
defines `users`, `shipments` and `updates` (with `sequence`, `previous_fingerprint`, `fingerprint`, the immutability
trigger and the CHECKs for chain and assignment). **HU-03 needs no new migration**: the schema is sufficient. Migration
`0002_seed_ledger` stays reserved for its owner and is not touched here. Test shipments in later stages are reached by
calling HU-03's own update endpoint through the gateway, never by direct DB writes from QA.

**Testing**: `go -C hubx-api test ./...`, `go -C hubx-gw test ./...`, `npm --prefix hubx-mfe run build` (the MFE has no
test script or test dependency; validation is typecheck/build plus the browser scenarios), `make test-integration`
(full stack: PostgreSQL + API + gateway + MFE + Cucumber API and `@e2e` scenarios), `make security` (`gosec`,
`govulncheck` for both Go services).

**Target Platform**: Linux containers (Docker Compose). Browser: Playwright Chromium for `@e2e`.

**Project Type**: web service (hubx-api), reverse-proxy gateway (hubx-gw), SPA (hubx-mfe), QA workspace (hubx-bender):
four independently built components in one git repository, no code imports between services.

**Performance Goals**: no domain-specific throughput or latency target in the backlog. Keep a single indexed lookup by
shipment code, one row lock and one insert per update; no N+1 queries.

**Constraints**: strict journey sequence; immutable updates; only the assigned partner records delivery; privacy must
not reveal whether a shipment exists; all queries parameterized; secrets only via environment; no new dependencies;
all artifacts, messages and code in English.

**Scale/Scope**: one HTTP operation to record an update, one read operation to list ground partners for assignment, one
MFE view, one Cucumber feature with the 8 backlog scenario groups and one `@e2e` happy path. No incident, traceability
view, verification, list or evidence work (HU-04 to HU-10).

## Constitution Check

*GATE: checked before Phase 0 and re-checked after Phase 1 design (see "Constitution Check — post-design" below).*

| Principle | Requirement | Plan decision | Result |
|---|---|---|---|
| I — Immutable ledger | Verifiable SHA-256 chain, responsible party per update, all access through `LedgerRepository`, no modification/deletion | `ledger.Repository` is the only interface touching `updates`; `Append` is insert-only; `previous_fingerprint` and the canonical fingerprint formula are defined in `contracts/hu03-record-update.md`; the 0001 trigger stays as defence in depth | PASS |
| II — Three-tier + mandatory gateway | API only behind gateway; MFE never calls API directly; no cross-service imports; API validates JWT and shared secret | New routes are served under `/api` through the existing proxy; MFE uses one client with base `/api` and JWT in memory; no imports between services; JWT/shared-secret validation remains HU-01's responsibility and is assumed integrated upstream | PASS |
| III — Stack and restrictions | Go stdlib `net/http`, PostgreSQL, no new deps, per-service `application.properties`, secrets env only | No new dependency or property key; existing pgx driver and config reader reused; no secret in files | PASS |
| IV — Modular design / constructor injection | Domain in `internal/<domain>` without `net/http`; handlers in `internal/handlers/<domain>` with `Register`; `test/` and generated `mocks/` subpackages; sentinel errors mapped in handlers | New domains `internal/ledger` and `internal/shipment`; handler package `internal/handlers/shipment`; every tested package gets `test/` + `mocks/` via `go generate`; constructors return interfaces; domain sentinel errors mapped in `errors.go` of the handler | PASS |
| V — Test-first | Backlog Gherkin verbatim in `features/HU03_record-update.feature`, one `@e2e` browser happy path, all scenarios before implementation, unit tests red-green-refactor | QA writes and runs the feature in `red` before any backend/frontend work; black-box unit tests use gomock; full suite runs with `make test-integration` | PASS |
| VI — Security | Parameterized queries, authorization to own resources, indistinguishable answers, bcrypt/JWT, gateway controls, clean gosec/govulncheck | All SQL parameterized; role and per-shipment visibility enforced in `shipment.Recorder`; missing/invisible shipment and unauthorized caller yield the same "not responsible" rejection (FR-012); no secret logged; OWASP API Security Top 10 used in QA review | PASS |
| VII — Language | English in specs, code, messages and commits; glossary from backlog | Spec and this plan in English; UI text in English (clarified 2026-11-24); domain identifiers use shipment, stage, update, fingerprint, responsible party, ground partner | PASS |

No principle is violated, so no complexity justification is required.

## Project Structure

### Documentation (this feature)

```text
specs/003-record-update/
├── spec.md                          # WHAT and WHY (done, clarified)
├── plan.md                          # This file
├── research.md                      # Phase 0: decisions and alternatives
├── data-model.md                    # Phase 1: entities, states, validation rules
├── contracts/
│   └── hu03-record-update.md        # Phase 1: HTTP contract + canonical fingerprint formula
├── quickstart.md                    # Phase 1: runnable validation guide
├── tasks.md                         # Phase 2 output (NOT created by /speckit.plan)
├── qa-report.md                     # QA deliverable
└── pr.md                            # closing deliverable
```

Repository-level contract (shared with HU-05/HU-10): `contracts/hu03-record-update.md` is the single source of the
fingerprint formula; `contracts/` is created by this story.

### Source Code (repository root)

Existing paths are marked `existing`; paths created or modified by HU-03 are marked `new` or `modified`.

```text
hubx-api/
├── cmd/api/main.go                          [modified] wiring only: config, pgx pool, repositories, Recorder,
│                                                       auth middleware and shared-secret check from HU-01, handler Register
├── cmd/migrate/main.go                      [existing] unchanged
├── application.properties                   [existing] unchanged: no new configuration key
├── migrations/
│   ├── 0001_baseline_schema.sql             [existing] unchanged: already contains users/shipments/updates
│   └── embed.go                             [existing] unchanged
├── internal/
│   ├── config/                              [existing] unchanged
│   ├── health/                              [existing] unchanged
│   ├── migrate/                             [existing] unchanged
│   ├── ledger/                              [new] domain — no net/http import
│   │   ├── update.go                        Update, AppendInput, canonical fingerprint function, sentinel errors
│   │   ├── repository.go                    LedgerRepository interface (Last, Append)
│   │   ├── postgres.go                      pgx implementation: row lock, optimistic sequence check, insert-only
│   │   ├── mocks/                           [new] generated mock of LedgerRepository (go:generate mockgen)
│   │   └── test/                            [new] black-box unit tests: fingerprint, chain and append behavior
│   ├── shipment/                            [new] domain — no net/http import
│   │   ├── shipment.go                      Shipment, Stage constants and transition table
│   │   ├── recorder.go                      Record(): visibility, responsibility, sequence, fingerprint chaining
│   │   ├── repository.go                    ShipmentRepository and PartnerRepository interfaces
│   │   ├── postgres.go                      parameterized pgx implementations
│   │   ├── errors.go                        domain sentinel errors
│   │   ├── mocks/                           [new] generated mocks (go:generate mockgen)
│   │   └── test/                            [new] black-box unit tests of Recorder and stage rules
│   └── handlers/
│       └── shipment/                        [new] HTTP transport package
│           ├── handler.go                   Register(mux, ...); POST /shipments/{code}/updates, GET /ground-partners
│           ├── errors.go                    sentinel errors -> HTTP status and English message
│           ├── mocks/                       [new] generated mocks for injected interfaces
│           └── test/                        [new] black-box handler tests with gomock
└── go.mod / go.sum                          [existing] unchanged: no new dependency

hubx-gw/
├── cmd/gw/main.go                           [existing] unchanged: no domain logic added to the gateway
├── internal/proxy/proxy.go                  [existing] unchanged: already strips /api and forwards Authorization
├── internal/proxy/proxy_test.go             [existing] unchanged (in-package legacy test; HU-03 adds no gateway test)
├── application.properties                   [existing] unchanged
└── go.mod / go.sum                          [existing] unchanged

hubx-mfe/
├── src/
│   ├── App.tsx                              [modified] render the update-recording view for the signed-in role
│   ├── api/client.ts                        [new] the single /api client; JWT held in memory, never localStorage
│   └── features/shipment-updates/
│       └── RecordUpdate.tsx                 [new] accessible form: shipment code, stage select, ground-partner select,
│                                                  submit button, role/label locators for @e2e; all text in English
├── package.json                             [existing] unchanged: no new dependency or script
├── index.html / vite.config.ts              [existing] unchanged
└── Dockerfile                               [existing] unchanged

hubx-bender/
├── cucumber.mjs                             [existing] unchanged
├── features/health.feature                  [existing] unchanged
├── features/HU03_record-update.feature      [new] backlog Gherkin verbatim + one @e2e happy path
├── features/steps/HU03_record-update.steps.ts [new] story-specific steps
├── features/steps/common.steps.ts           [modified] sign-in through the gateway, shipment fixture via HU-03 updates,
│                                                       POST update helper and response assertions
├── features/support/world.ts                [existing] unchanged
└── features/support/hooks.ts                [existing] unchanged

contracts/
└── hu03-record-update.md                    [new] shared HTTP + fingerprint contract (lead-owned)
```

**Structure Decision**: keep the four existing components and their ownership boundaries. HU-03 adds two Go domains
because the ledger chain and the shipment stage rules are distinct responsibilities (Principle I vs. the stage
transition rules); the handler package owns transport only. The gateway needs no change at all: its existing proxy
forwards any method, including POST, and the Cucumber API scenarios exercise that path. Adding a new in-package gateway
test would perpetuate a layout that contradicts Principle IV, so none is added. The MFE keeps a single API client and
one feature folder. No new file is created in `migrations/`,
`hubx-gw/internal/security` or `contracts/` beyond the contract document listed above.

**Upstream dependency (accepted 2026-11-24)**: HU-01 (sign-in/JWT/roles) and HU-02 (shipment creation) are assumed
integrated before this story. HU-03 does not implement their domains. Its handler depends on a small `PrincipalSource`
interface it defines itself, so the package is testable in isolation; `cmd/api/main.go` wires the HU-01 implementation.
QA obtains a token from the HU-01 sign-in endpoint through the gateway, using the demo users in migration 0001.

## Complexity Tracking

> No Constitution Check violation: this section is not applicable.

## Constitution Check — post-design

Re-evaluated after `research.md`, `data-model.md`, `contracts/hu03-record-update.md` and `quickstart.md`:

| Principle | Post-design verification | Result |
|---|---|---|
| I | Fingerprint input is canonical and documented in the shared contract; `Append` is insert-only and rejects a stale expected sequence; the shipment projection is updated in the same transaction; no code path edits an update | PASS |
| II | Every new endpoint is reached as `/api/...` through the existing proxy; the MFE issue no direct API call; no service imports another; the API's own JWT/shared-secret validation remains the HU-01 wiring | PASS |
| III | Dependency list and property keys are unchanged; secrets stay in the environment; no new toolchain version | PASS |
| IV | Domain packages import neither `net/http` nor a handler; `Register` is the only route registration; each tested package has `test/` and generated `mocks/`; handlers translate sentinel errors | PASS |
| V | The feature file is produced in `red` before implementation and contains all backlog scenarios verbatim plus one `@e2e` path; Go unit tests are black-box with gomock; evidence is the QA report and `make test-integration` | PASS |
| VI | Parameterized SQL, role and ownership checks in the Recorder, indistinguishable FR-012 rejection, no secret or credential fingerprint logged, OWASP API Security Top 10 checklist in the QA report | PASS |
| VII | Spec, plan, contract, code identifiers, user-facing messages and commit titles are English, using the backlog glossary | PASS |

Evidence of the re-check: `research.md` (7 decisions with rationale and alternatives), `data-model.md` (entities,
transition table, visibility and error catalog), `contracts/hu03-record-update.md` (HTTP contract and canonical
fingerprint formula) and `quickstart.md` (validation commands and expected results were reviewed against all seven
principles). The only earlier deviation detected in this pass — a proposed new in-package gateway test — was removed
from the plan so that no new file breaks the Principle IV test layout.

Final gate status: **PASS**. No deviation from the constitution was introduced by the design.

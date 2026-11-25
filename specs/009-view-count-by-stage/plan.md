# Implementation Plan: View the shipment count by stage

**Branch**: `009-view-count-by-stage` | **Date**: 2026-11-24 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/009-view-count-by-stage/spec.md`

## Summary

Give an operator a single aggregate view of how many shipments sit in each of the six journey stages, so pile-ups are
visible at a glance. The API adds one read-only endpoint that authorizes the `operator` role, counts shipments grouped
by the `shipments.stage` projection (the fast-read column already maintained by HU-02/HU-03 and backed by the existing
`shipments_stage_created_at_idx` index), and always returns the six stages in journey order, each with its count,
filling absent stages with zero. A ground partner or customer gets `403` with no totals. The gateway forwards the new
route under `/api` with its existing perimeter controls; it gains no domain logic. The MFE adds an operator-only
"Shipment count by stage" view that reads the endpoint through the single `/api` client. Acceptance is proven by the
HU-09 Cucumber feature (backlog scenarios verbatim plus one `@e2e` browser happy path) and by black-box Go unit tests
with gomock. No new migration, no new dependency, no ledger write.

## Technical Context

**Language/Version**: Go 1.27.4 (`go.mod`, `go.work`); TypeScript 7.0.2 strict, React 19.3.1, Vite 8.3.6.

**Primary Dependencies**: no new dependency. `net/http` + `ServeMux` (stdlib), `log/slog` JSON, `database/sql` with
`github.com/jackc/pgx/v5/stdlib`, `go.uber.org/mock` (test only, via `go tool mockgen`), and the JWT / bcrypt
libraries already approved for HU-01. QA: Cucumber + Playwright + tsx, already present in hubx-bender.

**Storage**: PostgreSQL, managed by the local stack. The applied migration `0001_baseline_schema.sql` already defines
`shipments` with the `stage` projection column, its CHECK constraint on the six valid stages, and the index
`shipments_stage_created_at_idx ON shipments (stage, created_at DESC)`. **HU-09 needs no new migration**: a single
`SELECT stage, COUNT(*) ... GROUP BY stage` over the existing projection is sufficient. The six-stage, journey-order,
zero-filled shape is produced in the Go domain, not in SQL, so the result is deterministic regardless of which stages
have rows.

**Testing**: `go -C hubx-api test ./...`, `go -C hubx-gw test ./...`, `npm --prefix hubx-mfe run test` and
`npm --prefix hubx-mfe run build`, `make test-integration` (full stack: PostgreSQL + API + gateway + MFE + Cucumber
API and `@e2e` scenarios), `make security` (`gosec`, `govulncheck` for both Go services).

**Target Platform**: Linux containers (Docker Compose). Browser: Playwright Chromium for `@e2e`.

**Project Type**: web service (hubx-api), reverse-proxy gateway (hubx-gw), SPA (hubx-mfe), QA workspace (hubx-bender):
four independently built components in one git repository, no code imports between services.

**Performance Goals**: no domain-specific target in the backlog. One indexed aggregate query per request
(`GROUP BY stage`), covered by `shipments_stage_created_at_idx`; no N+1, no per-shipment read, no ledger access.

**Constraints**: operator-only; the six stages always returned in journey order; absent stages shown as zero; an empty
system still returns six zero counts; refusal to non-operators reveals no totals; all queries parameterized (this one
takes no parameters); secrets only via environment; no new dependency; all artifacts, messages and code in English.

**Scale/Scope**: one read HTTP operation, one MFE view, one Cucumber feature with the four HU-09 scenario groups and
one `@e2e` happy path. No list pagination (HU-06), assigned view (HU-07), incidents (HU-08) or evidence (HU-10).

## Constitution Check

*GATE: checked before Phase 0 and re-checked after Phase 1 design (see "Constitution Check — post-design" below).*

| Principle | Requirement | Plan decision | Result |
|---|---|---|---|
| I — Immutable ledger | Verifiable SHA-256 chain, responsible party per update, all ledger access through `LedgerRepository`, no modification/deletion | HU-09 is read-only and never touches `updates`; it reads the `shipments.stage` projection only. No ledger interface is used or needed, so no immutability surface is added | PASS |
| II — Three-tier + mandatory gateway | API only behind gateway; MFE never calls API directly; no cross-service imports; API validates JWT and shared secret | The new route is served under `/api` through the existing proxy; the MFE reads it through the single `/api` client; no imports between services; JWT and shared-secret validation remain the HU-01 wiring, reused here | PASS |
| III — Stack and restrictions | Go stdlib `net/http`, PostgreSQL, no new deps, per-service `application.properties`, secrets env only | No new dependency or property key; the existing pgx driver and config reader are reused; no secret in files | PASS |
| IV — Modular design / constructor injection | Domain in `internal/<domain>` without `net/http`; handlers in `internal/handlers/<domain>` with `Register`; `test/` and generated `mocks/` subpackages; sentinel errors mapped in handlers | A `StageCounter` service and a `StageCountRepository` interface are added to the existing `internal/shipment` domain (no `net/http`); the HTTP transport goes in the existing `internal/handlers/shipment` package via a new `RegisterStageCounts`; each tested package keeps its `test/` + generated `mocks/`; the sole domain error maps to HTTP in the handler | PASS |
| V — Test-first | Backlog Gherkin verbatim in `features/HU09_stage-counts.feature`, one `@e2e` browser happy path, all scenarios before implementation, unit tests red-green-refactor | QA writes and runs the feature in `red` before any backend/frontend work; black-box unit tests use gomock for the counter and the handler; the full suite runs with `make test-integration` | PASS |
| VI — Security | Parameterized queries, function-level authorization, indistinguishable answers where applicable, gateway controls, clean gosec/govulncheck | The single aggregate query takes no user input; the operator role is checked from the validated JWT; a non-operator gets `403` with no totals; gateway keeps CORS, headers and rate limit; OWASP API Security Top 10 used in QA review | PASS |
| VII — Language | English in specs, code, messages and commits; glossary from backlog | Spec, plan and contract in English; UI text in English; identifiers use shipment, stage; stage names are the exact backlog glossary strings | PASS |

No principle is violated, so no complexity justification is required.

## Project Structure

### Documentation (this feature)

```text
specs/009-view-count-by-stage/
├── spec.md                          # WHAT and WHY (done)
├── plan.md                          # This file
├── research.md                      # Phase 0: decisions and alternatives
├── data-model.md                    # Phase 1: entities, result shape, validation rules
├── quickstart.md                    # Phase 1: runnable validation guide
├── checklists/
│   └── requirements.md              # Spec quality checklist (from /speckit.specify)
├── tasks.md                         # Phase 2 output (NOT created by /speckit.plan)
├── qa-report.md                     # QA deliverable
└── pr.md                            # closing deliverable
```

The HTTP contract is repository-level: `contracts/hu09-stage-counts.md` (lead-owned), the single source of the
endpoint's shape and authorization rules.

### Source Code (repository root)

Existing paths are marked `existing`; paths created or modified by HU-09 are marked `new` or `modified`.

```text
hubx-api/
├── cmd/api/main.go                              [modified] wiring only: build StageCountRepository and StageCounter,
│                                                           call shipment.RegisterStageCounts with RequireAuth
├── application.properties                       [existing] unchanged: no new configuration key
├── migrations/
│   └── 0001_baseline_schema.sql                 [existing] unchanged: shipments.stage + stage index already present
├── internal/
│   ├── shipment/                                [existing] domain — no net/http import
│   │   ├── stage.go                             [existing] unchanged: the six Stage constants and journey order
│   │   ├── stage_count.go                       [new] StageCount type, JourneyOrder(), StageCounter service,
│   │   │                                               six-stage zero-filled assembly in journey order
│   │   ├── stage_count_repository.go            [new] StageCountRepository interface (CountByStage)
│   │   ├── stage_count_postgres.go              [new] pgx implementation: SELECT stage, COUNT(*) GROUP BY stage
│   │   ├── mocks/                               [existing] extended: generated mock of StageCountRepository
│   │   └── test/                                [existing] extended: black-box unit tests of StageCounter
│   │       └── stage_count_test.go              [new] zero-fill, order, totals, repository error propagation
│   └── handlers/
│       └── shipment/                            [existing] HTTP transport package
│           ├── stage_counts_handler.go          [new] RegisterStageCounts(mux, requireAuth, counter);
│           │                                           GET /shipments/stage-counts, operator-only, JSON response
│           ├── stage_counts_errors.go           [new] domain/role errors -> HTTP status + English message
│           ├── mocks/                           [existing] extended: generated mock of the StageCounter interface
│           └── test/                            [existing] extended
│               └── stage_counts_handler_test.go [new] 200 shape/order, 403 non-operator, 401 passthrough, 500 mapping
└── go.mod / go.sum                              [existing] unchanged: no new dependency

hubx-gw/
├── cmd/gw/main.go                               [existing] unchanged: no domain logic in the gateway
├── internal/proxy/proxy.go                      [existing] unchanged: already strips /api and forwards Authorization
└── application.properties                       [existing] unchanged

hubx-mfe/
├── src/
│   ├── pages/Dashboard.tsx                      [modified] operator-only entry to the stage-count view
│   ├── services/
│   │   ├── stageCountService.ts                 [new] typed GET /shipments/stage-counts via the single client
│   │   └── stageCountService.test.ts            [new] unit test of the service (success, 403, 401)
│   └── features/stage-counts/
│       ├── StageCounts.tsx                      [new] accessible view: six rows in journey order, stage name + count,
│       │                                               role/label locators for @e2e; all text in English
│       ├── StageCounts.test.tsx                 [new] renders six stages in order, zeros shown, non-operator hidden
│       └── StageCountsContainer.tsx             [new] container wiring the service to the view, handles session expiry
├── package.json                                 [existing] unchanged: no new dependency or script
└── Dockerfile                                   [existing] unchanged

hubx-bender/
├── features/HU09_stage-counts.feature           [new] backlog Gherkin verbatim + one @e2e happy path
├── features/steps/HU09_stage-counts.steps.ts    [new] story-specific steps
├── features/steps/common.steps.ts               [modified, if needed] reuse sign-in + stage fixtures via HU-02/HU-03
├── features/support/world.ts                    [existing] unchanged
└── features/support/hooks.ts                    [existing] unchanged

contracts/
└── hu09-stage-counts.md                         [new] shared HTTP contract (lead-owned)
```

**Structure Decision**: keep the four existing components and their ownership boundaries. HU-09 reuses the existing
`internal/shipment` domain and `internal/handlers/shipment` transport package rather than creating new ones, because
the count is a read over the same shipment projection those packages already own; adding a parallel domain would be
unjustified complexity (constitution "any added complexity MUST be explicitly justified"). The gateway needs no change:
the new route is just another `/api` path behind the existing proxy and perimeter controls. The MFE adds one
operator-gated view and one service, both going through the single `/api` client (constitution II).

**Upstream dependency (accepted 2026-11-24)**: HU-01 (sign-in/JWT/roles) is a prerequisite and provides the
`RequireAuth` middleware and the `operator` role claim used for authorization. HU-02 and HU-03 populate the
`shipments.stage` projection across stages; HU-09 only reads it and implements none of their domains. QA obtains a
token from the HU-01 sign-in endpoint through the gateway using the demo users in migration 0001, and reaches the
stages a scenario needs by creating shipments (HU-02) and advancing them (HU-03) through the gateway, never by direct
DB writes.

## Complexity Tracking

> No Constitution Check violation: this section is not applicable.

## Constitution Check — post-design

Re-evaluated after `research.md`, `data-model.md`, `contracts/hu09-stage-counts.md` and `quickstart.md`:

| Principle | Post-design verification | Result |
|---|---|---|
| I | The design reads the `shipments.stage` projection only; no code path reads, writes or edits `updates`, so immutability is untouched and no `LedgerRepository` is involved | PASS |
| II | The endpoint is reached as `/api/shipments/stage-counts` through the existing proxy; the MFE issues no direct API call; no service imports another; the API's own JWT and shared-secret validation remain the HU-01 wiring | PASS |
| III | Dependency list and property keys are unchanged; no new migration; secrets stay in the environment | PASS |
| IV | The domain addition imports neither `net/http` nor a handler; `RegisterStageCounts` is the only route registration; the tested packages keep `test/` and generated `mocks/`; the handler translates the one domain/role error | PASS |
| V | The feature file is produced in `red` before implementation with the four backlog scenario groups verbatim plus one `@e2e` path; Go unit tests are black-box with gomock; evidence is the QA report and `make test-integration` | PASS |
| VI | The aggregate query takes no user input and is parameter-free; function-level authorization uses the validated JWT role; a non-operator gets `403` with no totals; gateway perimeter controls are unchanged; OWASP API Security Top 10 checklist in the QA report | PASS |
| VII | Spec, plan, contract, identifiers, user-facing messages and commit titles are English, using the backlog glossary stage names | PASS |

Final gate status: **PASS**. No deviation from the constitution is introduced by the design.

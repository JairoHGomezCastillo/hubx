# Implementation Plan: Verify the integrity of a shipment's chain

**Branch**: `005-verify-integrity` (from `origin/release/v0.4.0-ledger`) | **Date**: 2026-11-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/005-verify-integrity/spec.md`

## Summary

Add a read-only endpoint `GET /api/shipments/{code}/integrity` that loads every update of a visible shipment through
`LedgerRepository`, recomputes each fingerprint with the HU-03 formula and reports whether the chain is intact, how many
updates were checked and the position of the first altered update. Unknown and invisible shipments get the same 404.
The MFE adds a "Verify integrity" view. QA proves the altered case by tampering one update directly in PostgreSQL with
the documented tamper procedure (trigger bypassed on purpose), only in the local isolated stack.

## Technical Context

**Language/Version**: Go 1.27.4; TypeScript strict, React, Vite (versions unchanged from HU-03).

**Primary Dependencies**: none new. hubx-api: stdlib `net/http`, `database/sql` + pgx v5.12.0, `go.uber.org/mock`
v0.6.0 (tests). hubx-bender: Cucumber 13.3.0, Playwright 1.65.0, tsx 4.23.15; the tamper step uses Node's built-in
`node:child_process` (`execFile`, no shell) to run `docker compose exec -T postgres psql`, so no PostgreSQL client
library is added.

**Storage**: PostgreSQL, existing `updates` table from `0001_baseline_schema.sql`. **No migration.** `0002_seed_ledger`
stays reserved and untouched.

**Testing**: `go -C hubx-api test ./...`, `go -C hubx-gw test ./...`, `npm --prefix hubx-mfe run test`,
`npm --prefix hubx-mfe run build`, `make test-integration`, `make security`.

**Target Platform**: Linux containers (Docker Compose); Chromium for `@e2e`.

**Performance Goals**: one indexed shipment lookup plus one query of the shipment's updates (no N+1); verification is
linear in the number of updates (at most six per shipment in this version).

**Constraints**: read-only; privacy rule 4; parameterized SQL; no new dependency; English everywhere.

**Scale/Scope**: one GET route, one MFE view, one Cucumber feature (the 5 backlog scenario groups verbatim, 7 scenarios
after expanding the outline) plus one `@e2e` happy path. No alerts (HU-12), no traceability view (HU-04).

## Research and decisions

| # | Decision | Rationale | Alternatives rejected |
|---|---|---|---|
| R1 | Verification is a pure function `ledger.Verify([]Update) Result` in the ledger domain, reading through a new `LedgerRepository.List(ctx, shipmentID)` | Principle I: all ledger access through `LedgerRepository`; the formula already lives in `internal/ledger` (`Fingerprint`) | A SQL-side check (would duplicate the formula); a new package (splits the formula owner) |
| R2 | Visibility in a new `shipment.Verifier` reusing `ShipmentRepository.FindByCode` and the existing `canSee` rules | Same visibility rule as HU-03 (Principle VI); one place for role rules | Re-implementing visibility in the handler (handlers translate errors only, Principle IV) |
| R3 | Altered position = first 1-based position `p` where the sequence, the link to the previous update or the recomputed fingerprint does not match (contract "Verification rule") | Detects field edits, seal edits, broken links, deleted or reordered entries; the reported position stays meaningful even if `sequence` itself was edited | Reporting the stored `sequence` (wrong when the sequence is tampered) |
| R4 | Flat error body `{"error","message"}` and HU-01 `RequireAuth` (401 `UNAUTHORIZED`) | Matches HU-01/HU-02 and the gateway's default 401 body for every non-HU-03 route, so **no gateway change** | HU-03's nested body with `RequireSessionAuth` (would need a gateway change for one more route) |
| R5 | 404 `SHIPMENT_NOT_FOUND` for unknown and invisible shipments, identical body | Backlog wording "no shipment exists with that code" and rule 4 | 403 for invisible (reveals existence) |
| R6 | **Fix in HU-03 code:** `ledger` truncates `RecordedAt` to microseconds before computing the fingerprint (Append path) | PostgreSQL `timestamptz` keeps microseconds while Go's clock has nanoseconds: without the fix almost every stored update recomputes to a different fingerprint and HU-05 would always report the first update as altered. Clarified in `contracts/hu03-record-update.md` rule 1; the formula itself is unchanged | Comparing with tolerance (breaks reproducibility for third parties, HU-10); storing ns as text (needs a migration) |
| R7 | The tamper scenario edits the second update's `stage` in a fresh shipment with `SET LOCAL session_replication_role = replica` via `docker compose -p "$BENDER_COMPOSE_PROJECT" exec -T postgres psql`; tagged `@needs-db` and `@needs-secret` | Uses the procedure recorded in memory.md; only the isolated `hubx-it` stack is touched; deployments already exclude `@needs-secret`, so staging never runs it | An API to alter updates (violates Principle I); a `pg` npm dependency (new dependency) |

**Known risk (R6):** updates already stored in staging/production before the fix were fingerprinted over nanosecond
instants; verification will report them as altered at position 1. The local and CI stacks start from an empty ledger,
so tests are unaffected. Reported to the human in `pr.md`; no data is rewritten (Principle I).

## Constitution Check

| Principle | Plan decision | Result |
|---|---|---|
| I — Immutable ledger | Read-only `List` on `LedgerRepository`; recomputation with the existing `ledger.Fingerprint`; no update written; tamper only in tests through the documented bypass | PASS |
| II — Gateway | Route reached as `/api/shipments/{code}/integrity` through the existing proxy; API keeps `RequireAuth` and the shared secret; MFE uses the single `/api` client | PASS |
| III — Stack | No dependency, no property key, no migration | PASS |
| IV — Modular design | Domain in `internal/ledger` (`Verify`, `List`) and `internal/shipment` (`Verifier`); handler in `internal/handlers/shipment`; `test/` + generated `mocks/`; sentinel `ErrShipmentNotFound` mapped in the handler | PASS |
| V — Test-first | QA red first with the backlog Gherkin verbatim + `@e2e`; Go unit tests red-green with gomock | PASS |
| VI — Security | Visibility enforced in the domain; identical 404 for invisible/unknown; parameterized SQL; OWASP API Top 10 review (API1 BOLA focus) | PASS |
| VII — Language | English in specs, code, messages and UI | PASS |

## Data model

No schema change. Entities used (from `0001_baseline_schema.sql` and `internal/ledger/update.go`):

- `updates(id, shipment_id, sequence, stage, responsible_party_id, assigned_partner_id, recorded_at,
  previous_fingerprint, fingerprint)` read ordered by `sequence ASC`.
- `ledger.VerificationResult{Intact bool, UpdatesChecked int, FirstAlteredUpdate int}` (0 = none), computed, not stored.
- Visibility: operator any; customer `shipments.customer_id = caller`; ground partner
  `shipments.assigned_partner_id = caller`.

## Project Structure

```text
specs/005-verify-integrity/   spec.md, plan.md, quickstart.md, tasks.md, qa-report.md (QA), pr.md
contracts/hu05-verify-integrity.md            [new] HTTP contract (references the HU-03 formula)
contracts/hu03-record-update.md               [modified] rule 1 clarifies microsecond truncation (R6)

hubx-api/internal/ledger/
  repository.go        [modified] LedgerRepository gains List(ctx, shipmentID) ([]Update, error)
  postgres.go          [modified] List (parameterized, ORDER BY sequence ASC); Append truncates RecordedAt to µs (R6);
                                  DB interface gains a multi-row query method
  verify.go            [new]      Verify(updates []Update) VerificationResult
  mocks/               [regenerated]
  test/                [modified/new] verify_test.go, list and truncation tests
hubx-api/internal/shipment/
  verifier.go          [new]      Verifier.Verify(ctx, code, principal) (*Verification, error); ErrShipmentNotFound
  mocks/, test/        [regenerated/new]
hubx-api/internal/handlers/shipment/
  integrity_handler.go [new]      RegisterIntegrity(mux, requireAuth, verifier, principals); GET /shipments/{code}/integrity
  mocks/, test/        [regenerated/new]
hubx-api/cmd/api/main.go  [modified] wiring only

hubx-mfe/src/features/integrity/   [new] VerifyIntegrity view (code input, "Verify integrity" button, result text)
hubx-mfe/src/App.tsx, pages/Dashboard.tsx  [modified] route/link to the view

hubx-bender/features/HU05_verify-integrity.feature      [new] backlog Gherkin verbatim + @e2e
hubx-bender/features/steps/HU05_verify-integrity.steps.ts [new]
hubx-bender/features/support/tamper.ts                   [new] execFile docker compose psql helper

scripts/test-integration.sh   [modified by devops] export BENDER_COMPOSE_PROJECT=hubx-it to bender
```

hubx-gw: no change (its JWT middleware already validates every `/api` route and returns the flat body here).

## Complexity Tracking

Not applicable: no principle is violated.

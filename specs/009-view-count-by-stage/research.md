# Phase 0 Research: View the shipment count by stage (HU-09)

The spec left no `[NEEDS CLARIFICATION]` markers. Research confirms the design choices against the existing codebase,
the constitution and AGENTS.md. Each decision records what was chosen, why, and the alternatives rejected.

## D1 — Source of the count: the `shipments.stage` projection, not the ledger

- **Decision**: count shipments by reading the `stage` column of the `shipments` table, grouped by stage.
- **Rationale**: migration `0001_baseline_schema.sql` already keeps `shipments.stage` as a fast-read projection of the
  newest ledger update, with a CHECK restricting it to the six valid stages and the index
  `shipments_stage_created_at_idx ON shipments (stage, created_at DESC)`. A single `GROUP BY stage` aggregate is one
  indexed pass, no per-shipment reads, no N+1. The projection is exactly what HU-06 (list by stage) also reads, so
  HU-09 stays consistent with the rest of the platform.
- **Alternatives rejected**:
  - Re-derive each shipment's current stage from the `updates` ledger at request time — correct but needlessly
    expensive (one chain walk per shipment) and couples a read-only count to the ledger, which HU-09 has no reason to
    touch (constitution Principle I keeps ledger access behind `LedgerRepository`; introducing that here adds surface
    for no benefit).
  - A materialized counter table updated by triggers — premature optimization; the backlog sets no scale target and
    the indexed `GROUP BY` is sufficient.

## D2 — Six stages, journey order, zero-filled: assembled in Go, not in SQL

- **Decision**: the SQL returns only the stages that have rows; the Go `StageCounter` maps those into the fixed
  six-element journey-order slice, filling missing stages with zero.
- **Rationale**: `GROUP BY stage` omits stages with no shipments. The spec requires all six stages, always present,
  always in journey order (FR-002, FR-003, FR-005). Building the canonical shape in the domain keeps the result
  deterministic and independent of the data, makes it unit-testable with a mocked repository, and keeps ordering logic
  out of SQL where it would be fragile. The journey order is already defined by `shipment.stage.go`; `StageCounter`
  reuses it as the single source of order.
- **Alternatives rejected**:
  - A SQL `LEFT JOIN` against a `VALUES` list of the six stages with `ORDER BY` a position column — works, but moves
    the canonical stage list and its order into SQL, duplicating what `stage.go` already owns and making the behavior
    harder to unit-test without a database.

## D3 — Reuse the existing `shipment` domain and `handlers/shipment` package

- **Decision**: add `StageCounter`, `StageCountRepository` and the postgres implementation to `internal/shipment`, and
  the HTTP transport to `internal/handlers/shipment` via a new `RegisterStageCounts`.
- **Rationale**: the count is a read over the same shipment projection these packages already own (HU-02 creation,
  HU-03 stage rules). Constitution Principle IV organizes code by domain; a shipment-count read belongs to the
  shipment domain. Creating a new domain or handler package would be added complexity the constitution requires to be
  justified, and there is no justification here.
- **Alternatives rejected**:
  - A dedicated `internal/reporting` domain — reasonable if more aggregate reports were planned in this story, but
    HU-09 is a single count; YAGNI. HU-06 and future reports can still live beside the shipment projection they read.

## D4 — Authorization: operator-only, from the validated JWT role

- **Decision**: the handler authorizes by the `role` claim (via the same `auth.GetClaims`/`RequireAuth` pattern used
  by HU-02's shipment create handler); non-operator -> `403 FORBIDDEN` with the backlog message and no totals.
- **Rationale**: the backlog "Who can do what" table restricts "view counts" to the operator. Constitution Principle VI
  requires function-level authorization on the server from the validated JWT, never from the request. HU-02 already
  established this exact pattern (`claims.Role != "operator"` -> `403`), so HU-09 follows it for consistency.
- **Alternatives rejected**:
  - Enforcing the role only at the gateway — violates Principle II (the API must validate and authorize on its own)
    and OWASP API5 (broken function-level authorization).

## D5 — No new migration, no new dependency, gateway unchanged

- **Decision**: ship with the existing schema, drivers and gateway proxy.
- **Rationale**: the `shipments.stage` column, its CHECK and its index already exist (D1). The endpoint is a plain
  `/api` GET that the existing reverse proxy forwards with its CORS, security-header and rate-limit controls; the
  gateway needs no route-specific code. No library beyond the already-approved stdlib + pgx + mock set is required.
  Constitution Principle III forbids new Go dependencies without human approval, and none is needed.
- **Alternatives rejected**:
  - Adding a gateway allow-list entry or per-route handler — unnecessary; the proxy already forwards every `/api`
    path and validates the JWT, and the API authorizes the role independently.

## D6 — Response shape and error contract

- **Decision**: `200` returns `{ "stages": [ {stage, count} x6 ], "total" }`; errors use the HU-02 body shape
  `{"error","message"}` with `401 UNAUTHORIZED`, `403 FORBIDDEN`, `500 INTERNAL_SERVER_ERROR`. See
  `contracts/hu09-stage-counts.md`.
- **Rationale**: HU-02's `contracts/shipments-create.md` already defines the flat `{"error","message"}` body and the
  `401`/`403`/`500` codes for shipment routes; reusing it keeps the shipment endpoints consistent and lets the MFE and
  QA reuse their existing error handling. The `403` message mirrors the backlog wording ("you do not have permission to
  see that information") and reveals no totals. `total` is a convenience sum the client may ignore.
- **Alternatives rejected**:
  - The nested `{"error":{"code","message"}}` body used by HU-03 — rejected to stay consistent with the other
    `shipments` endpoints (HU-02); mixing the two shapes on the same resource family would complicate the MFE client.
  - Returning a bare array of counts — loses the explicit six-stage, zero-filled contract and the self-describing
    stage names.

## Open questions

None. All spec requirements map to the decisions above; no clarification is outstanding.

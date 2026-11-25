# Phase 1 Data Model: View the shipment count by stage (HU-09)

HU-09 is read-only. It adds no table, column, migration or ledger entry. It reads the existing `shipments.stage`
projection and shapes the result in the domain. This document describes the entities as the feature sees them and the
rules the `StageCounter` enforces.

## Entities

### Stage (existing)

One of the six points of the journey, in a fixed order. Defined by `internal/shipment/stage.go`:

| Order | Stage constant | Glossary name |
|---|---|---|
| 1 | `StageCreated` | `Created` |
| 2 | `StageInFlight` | `In flight` |
| 3 | `StageAtDestinationAirport` | `At destination airport` |
| 4 | `StageAssignedToPartner` | `Assigned to partner` |
| 5 | `StageOutForDelivery` | `Out for delivery` |
| 6 | `StageDelivered` | `Delivered` |

The journey order is the single source of ordering for the response. The database CHECK on `shipments.stage` already
guarantees stored stages are exactly these six strings.

### Shipment (existing, read only)

Only the `stage` column matters here. It is the projection of the shipment's newest ledger update (or `Created` for a
shipment with no updates). HU-09 reads it and changes nothing.

### StageCount (new, in-memory)

The pairing of a stage with the number of shipments currently in it.

| Field | Type | Rule |
|---|---|---|
| `Stage` | `shipment.Stage` | one of the six stages |
| `Count` | `int` | `>= 0`; number of shipments whose current stage equals `Stage` at the request instant |

### StageCountReport (new, in-memory)

The full result: exactly six `StageCount` entries in journey order, plus the total.

| Field | Type | Rule |
|---|---|---|
| `Stages` | `[]StageCount` | length is always 6, in journey order, one entry per stage, missing stages filled with `Count = 0` |
| `Total` | `int` | sum of the six counts; equals the number of shipments in HUBX |

## Repository interface (new)

In `internal/shipment` (no `net/http` import):

```
// StageCountRepository returns the raw per-stage tallies from the shipments
// projection. It returns only the stages that have at least one shipment.
type StageCountRepository interface {
    CountByStage(ctx context.Context) (map[Stage]int, error)
}
```

The postgres implementation runs a single parameter-free aggregate:

```sql
SELECT stage, COUNT(*) AS n FROM shipments GROUP BY stage;
```

covered by `shipments_stage_created_at_idx`. It returns a map keyed by stage; absent stages are simply not in the map.

## Service rules (StageCounter)

`StageCounter.Count(ctx)` builds the `StageCountReport`:

1. Call `CountByStage`. On repository error, return it unwrapped to the handler (mapped to `500`).
2. For each of the six stages in journey order, emit a `StageCount` with the map value or `0` if absent.
3. The result always has exactly six entries, in order, even when the map is empty (no shipments -> six zeros).
4. `Total` is the sum of the six counts.

The service performs no authorization: the role check lives in the handler (function-level authorization, Principle
VI), keeping the domain free of transport concerns.

## Authorization and visibility

- Only `operator` may read the report. A ground partner or customer is rejected by the handler with `403` and receives
  no `StageCountReport` (FR-006). This is a system-wide aggregate, not a per-shipment resource, so no object-level
  (per-owner) visibility rule applies; the whole report is operator-only.
- The session cross-cutting rule (backlog rule 1) is enforced by the reused HU-01 `RequireAuth` middleware before the
  handler runs (FR-007); an expired session yields `401` and no report.

## Result-to-contract mapping

| Domain | Contract field (`contracts/hu09-stage-counts.md`) |
|---|---|
| `StageCount.Stage` (glossary name) | `stages[i].stage` |
| `StageCount.Count` | `stages[i].count` |
| `StageCountReport.Total` | `total` |
| journey order of `Stages` | fixed order of the `stages` array |

## Error catalog

| Condition | Domain signal | HTTP (contract) |
|---|---|---|
| expired/invalid/missing session | middleware rejects before handler | `401 UNAUTHORIZED` |
| caller role is not `operator` | handler role check | `403 FORBIDDEN` (no totals) |
| repository/database failure | error from `CountByStage` | `500 INTERNAL_SERVER_ERROR` |

There is no `400`/`404`/`422` path: the endpoint takes no input and the report always exists (six stages, possibly all
zero).

# Data Model: Download the traceability evidence

## No storage changes

This story adds no table, column, migration or recorded business event. It reads `shipments`, `updates` and `users`
from migration 0001. All ledger reads go through `LedgerRepository.List` (Constitution I) and all SQL is parameterized
(Constitution VI).

## Entities in memory

### Evidence

The assembled document content, produced only when requested and never persisted.

| Field | Type | Source | Rule |
|---|---|---|---|
| `Shipment` | ShipmentData | `ShipmentRepository.FindByCode` | Must pass `shipment.CanSee` for the caller |
| `Updates` | []UpdateEvidence | `LedgerRepository.List` ordered by sequence ascending | Empty slice is valid and must be stated in the document |
| `GeneratedAt` | timestamp (UTC, RFC3339) | moment of assembly | Not the date of the last update |
| `Instructions` | string | canonical HU-03 fingerprint recipe | Must be self-contained for a third party |

### ShipmentData

| Field | Type | Source | Rule |
|---|---|---|---|
| `Code` | string | `shipments.code` | Non-empty; unique |
| `Origin` | string | `shipments.origin` | Included per FR-012 |
| `Destination` | string | `shipments.destination` | Included per FR-012 |
| `Flight` | string | `shipments.flight` | Included per FR-012 |
| `WeightKg` | string | `shipments.weight_kg` | Included per FR-012; rendered as a decimal number |
| `CustomerName` | string | joined user name | Included per FR-012 |
| `Stage` | string | `shipments.stage` | Current stage, one of the six backlog stages |

### UpdateEvidence

| Field | Type | Source | Rule |
|---|---|---|---|
| `Sequence` | int | `updates.sequence` | 1-based chain position; document shows ascending order |
| `Stage` | string | `updates.stage` | Backlog glossary stage |
| `ResponsibleParty` | string | joined user name | Party that recorded the update |
| `RecordedAt` | timestamp (UTC, RFC3339Nano) | `updates.recorded_at` | Microsecond precision, as recorded |
| `Fingerprint` | string | `updates.fingerprint` | SHA-256 from the canonical HU-03 formula |
| `PreviousFingerprint` | string or absent | `updates.previous_fingerprint` | Absent for the first update; rendered as `-` |

## Validation and error rules

- The caller must have a valid session (401 flat `UNAUTHORIZED` otherwise).
- The caller must be an operator (403 flat `FORBIDDEN` otherwise, no document delivered).
- An unknown code and a shipment the caller cannot see both produce 404 flat `SHIPMENT_NOT_FOUND` with the same
  message; the document is never partially assembled for an invisible shipment.
- The document states that it proves consistency with itself only, not that HUBX did not rebuild the whole history.
- Empty chain: the document includes a sentence stating that the shipment has no updates yet.

## State transitions

None. The endpoint is read-only and does not move a shipment to a new stage or append an update.

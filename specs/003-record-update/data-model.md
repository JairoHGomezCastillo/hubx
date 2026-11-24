# Data model: Record a shipment update (HU-03)

Branch: `003-record-update` | Date: 2026-11-24

The physical tables already exist in the applied migration `hubx-api/migrations/0001_baseline_schema.sql`. This document
describes the domain view used by HU-03; it does not redefine the schema or add a migration.

## Entities

### Shipment (read model + projection)

| Field | Type | Source | Notes |
|---|---|---|---|
| id | string (UUID) | `shipments.id` | internal identity |
| code | string | `shipments.code` | unique, used in the URL |
| stage | Stage | `shipments.stage` | read projection of the last update; Created for a new shipment |
| customerID | string (UUID) | `shipments.customer_id` | used for customer visibility |
| assignedPartnerID | string (UUID) or null | `shipments.assigned_partner_id` | set by the assignment update, never replaced |
| createdBy | string (UUID) | `shipments.created_by` | operator who created it |

Validation: `code` must be non-empty; a shipment that does not exist is handled as FR-012 (indistinguishable rejection).

### Update (ledger entry, immutable)

| Field | Type | Source | Notes |
|---|---|---|---|
| id | int64 | `updates.id` | identity of the inserted row |
| shipmentID | string (UUID) | `updates.shipment_id` | owner of the chain |
| sequence | int | `updates.sequence` | starts at 1, strictly increasing, unique per shipment |
| stage | Stage | `updates.stage` | never Created (a new shipment has no update) |
| responsiblePartyID | string (UUID) | `updates.responsible_party_id` | from the authenticated principal |
| assignedPartnerID | string (UUID) or null | `updates.assigned_partner_id` | non-null only for Assigned to partner |
| recordedAt | time (UTC) | `updates.recorded_at` | date recorded in the update |
| previousFingerprint | string (64 hex) or null | `updates.previous_fingerprint` | null only when sequence = 1 |
| fingerprint | string (64 hex) | `updates.fingerprint` | SHA-256 of the canonical input in `contracts/hu03-record-update.md` |

Validation (enforced by the database and the domain):
- `(sequence = 1) ⇔ (previousFingerprint is null)`;
- `(stage = 'Assigned to partner') ⇔ (assignedPartnerID is not null)`;
- no UPDATE or DELETE is ever issued; the 0001 trigger rejects attempts to tamper with a ledger row.

### Principal (from HU-01, not owned by this story)

| Field | Type | Notes |
|---|---|---|
| userID | string (UUID) | identity of the signed-in caller |
| role | `operator` \| `ground partner` \| `customer` | from the validated JWT |
| expiresAt | time | session lasts one hour (cross-cutting rule 1) |

## Stage transitions

| Current stage | Requested stage | Allowed actor | Partner required | Resulting projection |
|---|---|---|---|---|
| Created | In flight | operator | no | stage = In flight |
| In flight | At destination airport | operator | no | stage = At destination airport |
| At destination airport | Assigned to partner | operator | yes, a registered ground partner | stage = Assigned to partner, assignedPartnerID = partner |
| Assigned to partner | Out for delivery | the assigned ground partner | no | stage = Out for delivery |
| Out for delivery | Delivered | the assigned ground partner | no | stage = Delivered |
| Delivered | any | nobody | — | terminal; rejected as out of sequence |

Any other pair is rejected as out of sequence (FR-006). A caller whose role or assignment does not match the allowed
actor is rejected as not responsible (FR-007/FR-012). The assigned partner is never replaced (FR-009).

## Visibility rules applied before an update

| Role | Can see |
|---|---|
| operator | any shipment |
| customer | shipments whose `customer_id` equals the principal's `userID` |
| ground partner | shipments whose `assigned_partner_id` equals the principal's `userID` |

An invisible or unknown shipment produces the same FR-012 rejection as a visible shipment with the wrong actor.

## Validation and error catalog

| Domain sentinel | Trigger | HTTP | English message |
|---|---|---|---|
| `ErrSessionExpired` | missing/invalid/expired token | 401 | Session expired; sign in again. |
| `ErrPartnerRequired` | Assigned to partner without `assignedPartnerId` | 400 | You must name the ground partner responsible for the delivery. |
| `ErrInvalidRequest` | malformed JSON or unknown stage value | 400 | The update data is invalid. |
| `ErrNotGroundPartner` | named user is not a registered ground partner | 403 | Only a ground partner can be responsible for the delivery. |
| `ErrNotResponsible` | wrong role, wrong partner, invisible or unknown shipment | 403 | You are not responsible for that update. |
| `ErrOutOfSequence` | requested stage does not follow the current one | 409 | That update does not follow the shipment's sequence. |
| `ErrConcurrentUpdate` | stale expected sequence / unique violation | 409 | The shipment changed while recording the update; try again. |
| `ErrInternal` | unexpected persistence failure | 500 | The update could not be recorded. |

A rejected attempt creates no update and changes no projection field.

## Relationships

- One shipment has zero or more updates, ordered by `sequence`; the chain is the ordered list of those updates.
- Each update identifies one responsible party (a user) and, for the assignment stage, one assigned ground partner.
- The shipment's `stage` and `assignedPartnerID` are projections of the newest update, never an independent source of
  truth (constitution Principle I).

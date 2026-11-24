# HTTP contract: HU-04 View a shipment's traceability

Branch: `004-view-traceability` | Date: 2026-11-25 | Status: agreed for implementation

The browser calls `/api/...`; the gateway validates the JWT, strips `/api` and forwards to hubx-api with the shared
secret. hubx-api validates the JWT on its own (constitution Principle II). This operation is read-only: it never
writes to `updates` or `shipments`.

## View the traceability of a shipment

```http
GET /api/shipments/{code}/traceability
Authorization: Bearer <JWT>
```

`{code}` is the public shipment code (e.g. `HBX-SEED-0001`, `HBX-1A2B3C4D`). It is matched exactly; no trimming or
case folding.

### Success response — 200 OK

```json
{
  "shipmentCode": "HBX-1A2B3C4D",
  "stage": "Assigned to partner",
  "updates": [
    {
      "sequence": 1,
      "stage": "In flight",
      "responsibleParty": {"id": "00000000-0000-4000-8000-000000000001", "name": "Laura Operator", "role": "operator"},
      "assignedPartner": null,
      "recordedAt": "2026-11-25T10:15:30.123456789Z",
      "previousFingerprint": null,
      "fingerprint": "<64 lowercase hex>"
    },
    {
      "sequence": 2,
      "stage": "At destination airport",
      "responsibleParty": {"id": "00000000-0000-4000-8000-000000000001", "name": "Laura Operator", "role": "operator"},
      "assignedPartner": null,
      "recordedAt": "2026-11-25T12:00:00.5Z",
      "previousFingerprint": "<fingerprint of sequence 1>",
      "fingerprint": "<64 lowercase hex>"
    },
    {
      "sequence": 3,
      "stage": "Assigned to partner",
      "responsibleParty": {"id": "00000000-0000-4000-8000-000000000001", "name": "Laura Operator", "role": "operator"},
      "assignedPartner": {"id": "00000000-0000-4000-8000-000000000002", "name": "Transportes Rápido"},
      "recordedAt": "2026-11-25T12:30:00Z",
      "previousFingerprint": "<fingerprint of sequence 2>",
      "fingerprint": "<64 lowercase hex>"
    }
  ]
}
```

| Field | Notes |
|---|---|
| shipmentCode | the requested code |
| stage | current stage of the shipment (projection); `Created` when there is no update |
| updates | every update of the shipment ordered by `sequence` ascending (oldest first); `[]` (never `null`) when none |
| updates[].responsibleParty | `id`, registered `name` and `role` of the party stored in the update |
| updates[].assignedPartner | `{"id","name"}` only on the `Assigned to partner` update; `null` otherwise |
| updates[].recordedAt | stored instant in UTC, RFC3339Nano (same value used by the fingerprint formula in `hu03-record-update.md`) |
| updates[].previousFingerprint / fingerprint | stored values, unchanged; `previousFingerprint` is `null` only for `sequence = 1` |

### Visibility (cross-cutting rules 4 and 6)

| Role | Can view |
|---|---|
| `operator` | any shipment |
| `customer` | shipments whose `customer_id` is the caller |
| `ground partner` | shipments whose `assigned_partner_id` is the caller |

A shipment the caller cannot see gets **exactly** the same response as an unknown code: same status, same body, no
extra header.

### Error responses

| Status | Body | When |
|---|---|---|
| 401 | `{"error": "UNAUTHORIZED", "message": "..."}` (HU-01 flat body, from the gateway or the API's `RequireAuth`) | missing, invalid or expired JWT |
| 404 | `{"error": {"code": "SHIPMENT_NOT_FOUND", "message": "No shipment exists with that code."}}` | unknown code, or a shipment the caller cannot see |
| 500 | `{"error": {"code": "INTERNAL_ERROR", "message": "The traceability could not be retrieved."}}` | unexpected failure |

The 401 body is the HU-01 flat body because the gateway only emits the nested `SESSION_EXPIRED` body on the HU-03
routes; HU-04 needs no gateway change. Error bodies for 404/500 use the nested shape of `hu03-record-update.md`.

## Upstream assumptions

- HU-01: sign-in and JWT middleware (`RequireAuth`). HU-02: shipment creation (`POST /api/shipments`). HU-03: recording
  updates (`POST /api/shipments/{code}/updates`). QA fixtures create fresh shipments and advance them through these
  endpoints, never by direct database writes.

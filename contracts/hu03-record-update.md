# HTTP contract: HU-03 Record a shipment update

Branch: `003-record-update` | Date: 2026-11-24 | Status: agreed for implementation

All paths below are seen by the browser as `/api/...`. The gateway strips the `/api` prefix before forwarding to
hubx-api. The API validates the JWT and the gateway shared secret on its own (constitution Principle II).

This document is the single source of the fingerprint formula used by HU-05 and HU-10.

## Record an update

```http
POST /api/shipments/{code}/updates
Authorization: Bearer <JWT>
Content-Type: application/json
```

### Request body

```json
{
  "stage": "In flight",
  "assignedPartnerId": null
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| stage | string | yes | one of the five update stages (never `Created`) |
| assignedPartnerId | string (UUID) or null | only for `Assigned to partner` | must identify a registered user with role `ground partner`; forbidden for other stages |

The MFE resolves the partner name to this id through `GET /api/ground-partners`.

### Success response — 201 Created

```json
{
  "id": 1,
  "shipmentCode": "HBX-SEED-0001",
  "sequence": 1,
  "stage": "In flight",
  "responsibleParty": {
    "id": "00000000-0000-4000-8000-000000000001",
    "name": "Laura Operator",
    "role": "operator"
  },
  "assignedPartner": null,
  "recordedAt": "2026-11-24T10:15:30Z",
  "previousFingerprint": null,
  "fingerprint": "9f2c... (64 lowercase hex characters)"
}
```

`assignedPartner` is `{"id": "...", "name": "..."}` for the assignment update and `null` otherwise.
`previousFingerprint` is `null` only for `sequence = 1`.

### Error responses

| Status | Code | Message |
|---|---|---|
| 401 | `SESSION_EXPIRED` | Session expired; sign in again. |
| 400 | `INVALID_REQUEST` | The update data is invalid. |
| 400 | `PARTNER_REQUIRED` | You must name the ground partner responsible for the delivery. |
| 403 | `NOT_GROUND_PARTNER` | Only a ground partner can be responsible for the delivery. |
| 403 | `NOT_RESPONSIBLE` | You are not responsible for that update. |
| 409 | `OUT_OF_SEQUENCE` | That update does not follow the shipment's sequence. |
| 409 | `CONCURRENT_UPDATE` | The shipment changed while recording the update; try again. |
| 500 | `INTERNAL_ERROR` | The update could not be recorded. |

Error body:

```json
{"error": {"code": "NOT_RESPONSIBLE", "message": "You are not responsible for that update."}}
```

`NOT_RESPONSIBLE` is deliberately returned for an unknown shipment code and for a shipment the caller cannot see, so
the answer never reveals existence (FR-012).

## List ground partners

```http
GET /api/ground-partners
Authorization: Bearer <JWT>
```

### Success response — 200 OK

```json
[
  {"id": "00000000-0000-4000-8000-000000000002", "name": "Transportes Rápido"},
  {"id": "00000000-0000-4000-8000-000000000003", "name": "Logística Express"}
]
```

The list contains every registered user whose role is `ground partner`, ordered by name. Any authenticated role may
read it; it reveals only that these partners exist, not which shipments are assigned to them. The same error body and
authorization rules apply for 401 and 500.

## Fingerprint formula (canonical, reproducible)

The fingerprint is `hex(SHA-256(canonicalInput))` with lowercase hex digits. The canonical input is UTF-8 text made of
these seven lines in this exact order, joined by `\n` with **no trailing newline**:

```text
shipment_id=<shipment UUID>
sequence=<positive decimal integer, no leading zeros>
stage=<exact stage name>
responsible_party_id=<UUID of the authenticated caller>
assigned_partner_id=<UUID, or - when there is no assigned partner>
recorded_at=<RFC3339Nano timestamp in UTC, e.g. 2026-11-24T10:15:30.123456789Z>
previous_fingerprint=<64 lowercase hex digits, or - for the first update>
```

Rules:

1. `recorded_at` is the exact instant stored in `updates.recorded_at`, rendered in UTC with nanosecond precision
   (RFC3339Nano). The API returns the same instant in its `recordedAt` field; a third party uses that value to
   reproduce the fingerprint.
2. Null values are the single character `-`, never an empty string and never the literal `null`.
3. Stage names are the exact English strings from the backlog glossary: `In flight`, `At destination airport`,
   `Assigned to partner`, `Out for delivery`, `Delivered`.
4. The first update of a shipment has `sequence = 1` and `previous_fingerprint = -`; every later update has the
   lowercase 64-hex fingerprint of the immediately preceding update.
5. Recomputing any stored update with the same inputs must yield its stored fingerprint. Altering any field, including
   the date or the predecessor link, breaks the match (basis for HU-05 and HU-10).

## Upstream assumptions

- HU-01 provides `POST /api/auth/sign-in` and the JWT validation middleware. The demo users in migration 0001 are used
  by QA; their password is the documented demo password and is never hard-coded in the MFE.
- HU-02 provides shipment creation; QA fixtures start from the seed shipments in migration 0001 and advance them
  through this endpoint to reach the stages required by the scenarios.

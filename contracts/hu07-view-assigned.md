# HTTP contract: HU-07 View my assigned shipments

Branch: `007-view-assigned` | Date: 2026-11-24 | Status: agreed for implementation

The browser calls `/api/...` on the gateway; the gateway validates the JWT, strips `/api`, injects
`X-Gateway-Secret` and forwards to hubx-api, which validates the JWT again on its own (constitution Principle II).
No gateway change: this route uses the gateway's default flat 401 body.

## List my assigned shipments

```http
GET /api/assigned-shipments?offset=0
Authorization: Bearer <JWT>
```

The ground partner is always the JWT `sub`. There is no parameter to name a partner, so a partner cannot ask for
another partner's list (constitution VI, OWASP API1).

| Query parameter | Type | Required | Rule |
|---|---|---|---|
| `offset` | decimal integer | no (default `0`) | `0 <= offset <= 100000`, digits only; anything else is `400 INVALID_REQUEST` |

### Selection and order

- A shipment is listed when the ledger holds an `Assigned to partner` update whose `assigned_partner_id` is the caller
  and the shipment has no `Delivered` update. With the HU-03 sequence rules this is exactly the shipments whose stage is
  `Assigned to partner` or `Out for delivery`.
- Order: `assignedAt` ascending (oldest assignment first, backlog HU-07), then the assignment update id ascending as a
  stable tie-break.
- Page size: 20 (cross-cutting rule 5). `nextOffset` is `offset + 20` when at least one more shipment exists after this
  page, otherwise `null`.

### 200 OK

```json
{
  "shipments": [
    {
      "code": "HBX-7F3A9C21",
      "stage": "Assigned to partner",
      "origin": "Bogotá",
      "destination": "Miami",
      "flight": "AV-204",
      "weightKg": 120,
      "assignedAt": "2026-11-24T10:15:30.123456Z"
    }
  ],
  "nextOffset": null,
  "message": null
}
```

| Field | Notes |
|---|---|
| `shipments` | always an array (`[]` when empty), at most 20 items |
| `stage` | `Assigned to partner` or `Out for delivery` (read from the shipment projection) |
| `weightKg` | JSON number, same representation as HU-02 |
| `assignedAt` | RFC 3339 UTC, the `recorded_at` of the shipment's `Assigned to partner` update |
| `nextOffset` | integer or `null` |
| `message` | `"You have no assigned shipments"` when `shipments` is empty, otherwise `null` |

The customer of each shipment is not returned (spec clarification: not needed to plan the delivery).

### Errors

Flat body, same shape as HU-02: `{"error": "<CODE>", "message": "<text>"}`.

| Status | `error` | `message` | When |
|---|---|---|---|
| 401 | `UNAUTHORIZED` | see `contracts/auth-sign-in.md` (`Session expired. Please sign in again.` for an expired token) | missing, invalid or expired JWT (gateway or API `RequireAuth`) |
| 403 | `FORBIDDEN` | `You do not have permission to see that information` | role is `operator` or `customer` |
| 400 | `INVALID_REQUEST` | `The offset must be a whole number between 0 and 100000` | bad `offset` (checked after the role) |
| 500 | `INTERNAL_SERVER_ERROR` | `An unexpected error occurred` | storage failure; details only in the log |

Validation order: session (401), role (403), `offset` (400). A 403 or 400 returns no shipment data.

## Example

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in -H 'Content-Type: application/json' \
  -d '{"email":"rapido@hubx.co","password":"Hubx-Demo-2026"}' | jq -r '.token')
curl -s http://localhost:8080/api/assigned-shipments -H "Authorization: Bearer $TOKEN"
```

## Security

- Read-only: no update is recorded, no shipment changes (FR-010).
- Parameterized queries only; `offset` and the page size are bound parameters.
- Function-level authorization on the server from the validated JWT role (OWASP API5); object-level scope from the
  JWT `sub` (OWASP API1). The ledger is read only through `LedgerRepository` (constitution I).

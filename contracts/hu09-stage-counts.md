# HTTP Contract: GET /api/shipments/stage-counts

**Story**: HU-09 View the shipment count by stage
**Version**: 1.0
**Status**: Implemented (v0.7.0)

The MFE calls `GET /api/shipments/stage-counts` on the gateway. The gateway validates the JWT, strips `/api`, injects `X-Gateway-Secret` and forwards
`GET /shipments/stage-counts` to the API. The API validates the JWT again on its own (constitution Principle II) and
authorizes by the `role` claim.

This is a read-only aggregate over the `shipments.stage` projection. It records nothing and never touches the ledger.

## Request

```http
GET /api/shipments/stage-counts
Authorization: Bearer <JWT>
```

No query parameters, no request body. Any body or query string sent is ignored.

## Validation order

The first failing check answers.

1. Session: missing, invalid or expired token -> `401 UNAUTHORIZED` (bodies in `contracts/auth-sign-in.md`).
2. Role: `role != "operator"` -> `403 FORBIDDEN`. No totals are returned.

## Responses

### 200 OK

```json
{
  "stages": [
    { "stage": "Created",                "count": 0 },
    { "stage": "In flight",              "count": 3 },
    { "stage": "At destination airport", "count": 0 },
    { "stage": "Assigned to partner",    "count": 0 },
    { "stage": "Out for delivery",       "count": 2 },
    { "stage": "Delivered",              "count": 5 }
  ],
  "total": 10
}
```

- `stages` MUST contain exactly the six stages, always in journey order: `Created`, `In flight`,
  `At destination airport`, `Assigned to partner`, `Out for delivery`, `Delivered`.
- A stage with no shipments appears with `"count": 0`; stages are never omitted.
- When HUBX holds no shipments, all six entries have `"count": 0` and `total` is `0`.
- `count` is a non-negative integer equal to the number of shipments whose current stage equals that stage at the
  instant of the request. `total` is the sum of the six counts (a convenience field; the client may ignore it).
- Stage names are the exact English strings from the backlog glossary.

### Errors

All error bodies have the shape `{"error": "<CODE>", "message": "<text>"}`, consistent with HU-02's contract.

| Status | `error` | `message` |
|---|---|---|
| 401 | `UNAUTHORIZED` | see `contracts/auth-sign-in.md` (missing, invalid, `Session expired. Please sign in again.`) |
| 403 | `FORBIDDEN` | `You do not have permission to see that information` |
| 500 | `INTERNAL_SERVER_ERROR` | `An unexpected error occurred` |

The `403` message is deliberately generic ("that information"), matching the backlog wording for a role that may not
see the count. It does not disclose any total.

## Examples

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}' | jq -r '.token')

curl -s http://localhost:8080/api/shipments/stage-counts \
  -H "Authorization: Bearer $TOKEN"
# 200, six stages in journey order, each with its count

# A ground partner is refused:
PARTNER=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"rapido@hubx.co","password":"Hubx-Demo-2026"}' | jq -r '.token')
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/api/shipments/stage-counts \
  -H "Authorization: Bearer $PARTNER"
# 403
```

## Security

- The single query is a parameterless aggregate; no user input reaches SQL (OWASP API8/API3 — no injection surface).
- Function-level authorization on the server (OWASP API5): the role comes from the validated JWT, never from the
  request. A non-operator receives `403` with no totals, so the aggregate view is never disclosed.
- No object-level authorization is needed: the count is a system-wide aggregate exposed only to operators; it returns
  no shipment identity, code or customer data (OWASP API1/API3).
- Rate limiting, security headers and CORS belong to the gateway (constitution Principle VI); it does not implement
  them yet (open finding, `memory.md` Pending).

## Related stories

- HU-01 (session and roles), HU-02 and HU-03 (populate `shipments.stage` across stages), HU-06 (lists shipments by
  stage, same projection). HU-09 reads the projection only; it writes nothing.

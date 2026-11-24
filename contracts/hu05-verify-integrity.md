# HTTP contract: HU-05 Verify the integrity of a shipment's chain

Branch: `005-verify-integrity` | Date: 2026-11-25 | Status: agreed for implementation

All paths below are seen by the browser as `/api/...`. The gateway strips the `/api` prefix before forwarding to
hubx-api. The API validates the JWT and the gateway shared secret on its own (constitution Principle II).

The fingerprint recomputed here is the one defined in
[`contracts/hu03-record-update.md`, "Fingerprint formula (canonical, reproducible)"](./hu03-record-update.md#fingerprint-formula-canonical-reproducible).
This contract does not define another formula.

## Verify a shipment's chain

```http
GET /api/shipments/{code}/integrity
Authorization: Bearer <JWT>
```

Read-only: it records, modifies and deletes nothing, and does not change the shipment's stage.

### Success response — 200 OK (intact chain)

```json
{
  "shipmentCode": "HBX-1A2B3C4D",
  "intact": true,
  "updatesChecked": 3,
  "firstAlteredUpdate": null
}
```

A shipment without updates returns `"intact": true`, `"updatesChecked": 0`, `"firstAlteredUpdate": null`.

### Success response — 200 OK (altered chain)

```json
{
  "shipmentCode": "HBX-1A2B3C4D",
  "intact": false,
  "updatesChecked": 3,
  "firstAlteredUpdate": 2
}
```

| Field | Type | Notes |
|---|---|---|
| shipmentCode | string | the code as stored |
| intact | boolean | `true` when every update matches |
| updatesChecked | integer >= 0 | number of updates read for the shipment |
| firstAlteredUpdate | integer >= 1 or null | 1-based position, oldest first, of the first update that does not match; `null` when intact |

### Verification rule

Read every update of the shipment ordered by `sequence` ascending and walk it with a 1-based position `p`. The update
at position `p` is **altered** when any of these holds:

1. its stored `sequence` is not `p` (a missing or reordered entry);
2. its stored `previous_fingerprint` is not `null` for `p = 1`, or is not the stored `fingerprint` of the update at
   `p - 1` for `p > 1`;
3. the fingerprint recomputed with the HU-03 formula from its stored `shipment_id`, `sequence`, `stage`,
   `responsible_party_id`, `assigned_partner_id`, `recorded_at` and `previous_fingerprint` is not its stored
   `fingerprint`.

`firstAlteredUpdate` is the smallest such `p`. Only the first one is reported.

### Error responses

Error body (same shape as HU-01 and HU-02): `{"error": "<CODE>", "message": "<text>"}`.

| Status | `error` | `message` |
|---|---|---|
| 401 | `UNAUTHORIZED` | see `contracts/auth-sign-in.md` (missing, invalid, `Session expired. Please sign in again.`) |
| 404 | `SHIPMENT_NOT_FOUND` | `No shipment exists with that code` |
| 500 | `INTERNAL_SERVER_ERROR` | `An unexpected error occurred` |

`SHIPMENT_NOT_FOUND` is returned for an unknown code and for a shipment the caller cannot see (operator: any shipment;
customer: own shipments; ground partner: shipments assigned to them), with an identical body, so the answer never
reveals existence (cross-cutting rule 4).

## Upstream assumptions

- HU-01 provides sign-in and `RequireAuth` (flat `UNAUTHORIZED` 401 body). The gateway keeps validating the JWT on
  every `/api` route; no gateway change is needed for this route.
- HU-03 provides the ledger, its fingerprint formula and the update endpoint QA uses to build fixtures.

# HTTP contract: Download the traceability evidence (HU-10)

**Branch**: `010-download-evidence` | **Date**: 2026-11-25 | **Status**: agreed for implementation

Reachable only through the gateway (`/api` prefix stripped). The API validates the JWT on its own and requires the
gateway shared secret. This route is read-only and uses the same flat 401 family as HU-02/HU-05; the gateway needs no
change.

## Request

```http
GET /api/shipments/{code}/evidence HTTP/1.1
Authorization: Bearer <jwt>
Accept: application/pdf
```

`{code}` is the shipment code (e.g. `HBX-SEED-0001`). No query parameters and no request body.

## Success response

```http
HTTP/1.1 200 OK
Content-Type: application/pdf
Content-Disposition: attachment; filename="evidence-HBX-SEED-0001.pdf"
Cache-Control: no-store
```

The body is a self-contained PDF document in English. It contains, in readable text:

1. Heading `HUBX traceability evidence`.
2. Shipment data: code, origin, destination, flight, weight, customer and current stage.
3. `Updates:` followed by each update in chain order with its sequence, stage, responsible party, date (UTC),
   fingerprint and previous fingerprint (`-` for the first one). If the chain is empty, the text
   `The shipment has no updates yet.` appears instead.
4. `Generated at:` followed by the UTC date on which the document was generated.
5. `How to verify the fingerprints:` followed by the canonical recipe from
   [hu03-record-update.md](./hu03-record-update.md) §"Fingerprint formula (canonical, reproducible)": the seven
   `key=value` lines in order, joined by `\n` with no trailing newline, `-` for null values and `recorded_at`
   truncated to microseconds before computing SHA-256.
6. `Scope:` a statement that the document proves the history is consistent with itself and does not by itself prove
   that HUBX did not rebuild the whole history.

The PDF is generated from current records; no file is stored. `updates` are never reordered, edited or deleted.

## Errors

All error bodies are flat JSON (`Content-Type: application/json`) and no PDF is delivered.

| Status | Code | Message | When |
|---|---|---|---|
| 401 | `UNAUTHORIZED` | see `contracts/auth-sign-in.md` (missing header: `Authorization header required`; invalid or expired token as defined there) | No JWT, expired JWT or invalid JWT |
| 403 | `FORBIDDEN` | `You do not have permission to download evidence` | Signed-in ground partner or customer |
| 404 | `SHIPMENT_NOT_FOUND` | `No shipment exists with that code` | Unknown code or shipment invisible to the caller |
| 500 | `INTERNAL_SERVER_ERROR` | `An unexpected error occurred` | Unexpected failure while assembling the document |

Privacy: an unknown code and an invisible shipment are indistinguishable — same status, code and message.

## Visibility

| Caller | Result |
|---|---|
| Operator | Any shipment |
| Ground partner | 403 |
| Customer | 403 |
| No session | 401 |

## Upstream assumptions

- `ShipmentRepository.FindByCode` returns the full shipment projection, including origin, destination, flight,
  weight, customer name and creation time.
- `LedgerRepository.List` returns the updates ordered by sequence ascending, with all fingerprint fields.
- `shipment.CanSee` is the single visibility rule shared with HU-03/HU-04/HU-05.
- The canonical fingerprint formula in `contracts/hu03-record-update.md` remains the source of truth; this contract
  references it instead of duplicating it.
- The gateway validates the JWT before proxying, so for a missing or invalid token the client receives the gateway's
  flat 401 (`contracts/auth-sign-in.md`) and the request never reaches this route. The API's `RequireAuth` is the
  second validation (Constitution II), not the first; hence this contract does not fix a 401 message of its own.

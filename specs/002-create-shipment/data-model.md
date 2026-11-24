# Data model: Create a shipment (HU-02)

No migration. HU-02 uses the `shipments` table created by `hubx-api/migrations/0001_baseline_schema.sql`.

## shipments (existing)

| Column | Type | Set by HU-02 | Rule |
|---|---|---|---|
| `id` | uuid | default `gen_random_uuid()` | internal, never exposed |
| `code` | text UNIQUE | API (`HBX-` + 8 hex) | unique; retry on `23505` |
| `origin` | text | request, trimmed | non-blank |
| `destination` | text | request, trimmed | non-blank; `lower(btrim(origin)) <> lower(btrim(destination))` |
| `flight` | text | request, trimmed | non-blank; not unique |
| `weight_kg` | numeric(6,1) | request | `BETWEEN 0.1 AND 5000` |
| `customer_id` | uuid FK users | lookup by normalized email | user role must be `customer` (checked in the service) |
| `stage` | text | default `'Created'` | projection of the ledger; never set by HU-02 |
| `assigned_partner_id` | uuid | NULL | set by HU-03 |
| `created_by` | uuid FK users | JWT `sub` | the operator |
| `created_at` | timestamptz | default `now()` | returned as `createdAt` |

## updates (existing)

Not written. A new shipment has zero updates; HU-03 records the first (`In flight`, `sequence = 1`,
`previous_fingerprint` NULL).

## Domain types (hubx-api/internal/shipment)

```go
type CreateInput struct {
    Role, OperatorID                    string // from JWT claims
    Origin, Destination, Flight         string
    WeightKg                            string // json.Number literal
    CustomerEmail                       string
}

type Shipment struct {
    Code, Origin, Destination, Flight string
    WeightKg                          string // decimal literal, one decimal max
    CustomerEmail, CustomerName       string
    Stage                             string // "Created"
    CreatedAt                         time.Time
}
```

Exact field names are the backend's choice; the contract in `contracts/shipments-create.md` is binding.

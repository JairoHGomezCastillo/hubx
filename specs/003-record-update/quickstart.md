# Quickstart: validate HU-03 Record a shipment update

Branch: `003-record-update` | Date: 2026-11-24

This is a validation guide, not implementation code. See `contracts/hu03-record-update.md` for field definitions and the
fingerprint formula, and `data-model.md` for the state table.

## Prerequisites

- Docker, Node 22+, Go 1.27.4, `make`, `curl`.
- One-time setup: `npm --prefix hubx-bender ci && npm --prefix hubx-bender run browsers`.
- HU-01 (sign-in) and HU-02 (shipment creation) assumed integrated upstream; the demo users of migration 0001 exist.

## 1. Full acceptance suite (authoritative)

```bash
make test-integration
```

Starts an isolated stack (`hubx-it`), applies migrations, builds the MFE, runs every Cucumber scenario from
`hubx-bender/features/HU03_record-update.feature` against the gateway and runs the `@e2e` scenario in Chromium, then
tears the stack down. Expected: all scenarios green, including `@e2e`. A red `@e2e` blocks the story even if every API
scenario passes.

## 2. Unit and build checks

```bash
go -C hubx-api test ./...
go -C hubx-gw test ./...
go -C hubx-api generate ./...   # mocks must be reproducible; no generated file edited by hand
npm --prefix hubx-mfe run build # strict typecheck + production build
make security                   # gosec + govulncheck on hubx-api and hubx-gw, no findings
```

Expected: no failing test, no diff in generated mocks after `go generate`, clean security scan.

## 3. Manual happy path against a running stack

```bash
make up
```

1. **Sign in** (HU-01) as `operator@hubx.co` with the documented demo password through the browser at
   http://localhost:3000. The MFE keeps the token in memory and navigates inside the app.
2. **Open a seed shipment** `HBX-SEED-0001` (stage Created, no updates) in the update form.
3. **Record `In flight`**. Expected: the view shows the new stage, the responsible party `Laura Operator`, the recorded
   date and a 64-hex fingerprint; `previousFingerprint` is null and `sequence` is 1.
4. **Record `At destination airport`**, then **assign `Transportes Rápido`**. Expected: the shipment reaches
   `Assigned to partner`, the update names the operator as responsible party and the partner as responsible for
   delivery, and each new fingerprint's previous value is the previous update's fingerprint.
5. **Sign in as `rapido@hubx.co`** in a new browser session and record `Out for delivery`, then `Delivered`. Expected:
   both updates are accepted from the assigned partner only.
6. **Negative checks**: retrying `In flight` from `In flight` returns 409 `OUT_OF_SEQUENCE`; assigning a customer
   returns 403 `NOT_GROUND_PARTNER`; assigning without a partner returns 400 `PARTNER_REQUIRED`; an unassigned partner
   or an unknown code returns 403 `NOT_RESPONSIBLE`. In every rejected case the stage and the chain are unchanged.

## 4. Verifying the fingerprint by hand

Take the `recordedAt`, `stage`, `responsibleParty.id`, `assignedPartner.id`, `shipmentCode`, `sequence` and
`previousFingerprint` from the API response. Build the seven-line canonical string exactly as specified in
`contracts/hu03-record-update.md` and compute:

```bash
printf '%s' '<canonical string>' | sha256sum
```

Expected: the hex digest equals the `fingerprint` returned by the API. This is the same check HU-05 automates.

## 5. Stop

```bash
make down
```

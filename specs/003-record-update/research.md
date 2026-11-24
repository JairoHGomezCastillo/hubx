# Research: Record a shipment update (HU-03)

Branch: `003-record-update` | Date: 2026-11-24

All unknowns were resolved from the constitution, the backlog, the existing schema and the two clarifications recorded in
`spec.md`. No new dependency is required.

## 1. Canonical fingerprint input

**Decision**: compute the fingerprint as lowercase hex SHA-256 of a canonical UTF-8 string joined by `\n` with no
trailing newline:

```text
shipment_id=<uuid>
sequence=<decimal, no leading zeros>
stage=<exact stage name>
responsible_party_id=<uuid>
assigned_partner_id=<uuid or ->
recorded_at=<RFC3339Nano, UTC>
previous_fingerprint=<64 lowercase hex or ->
```

**Rationale**: HU-05 (verify integrity) and HU-10 (download evidence) need a formula that a third party can reproduce
without HUBX; the backlog only stores the result. A fixed field order, explicit `-` for null values and a fixed time
format remove ambiguity. Using `recorded_at` from the stored row makes a reproduced fingerprint sensitive to any
alteration of the date.

**Alternatives considered**: hashing the raw database row (not reproducible outside PostgreSQL and sensitive to column
order); hashing a JSON object (key order and number formatting are ambiguous unless canonicalized); omitting the date
(would let an altered date pass verification). The canonical string above was chosen.

**Follow-up**: this contract is shared by HU-05 and HU-10 and is the pending item that was blocking `0002_seed_ledger`.

## 2. Atomicity and concurrent updates

**Decision**: `LedgerRepository.Append` runs in one transaction: lock the `shipments` row with `SELECT ... FOR UPDATE`,
re-read the last update, check that the caller's `ExpectedSequence` still matches, insert the new update, then update
`shipments.stage` and `shipments.assigned_partner_id` in the same transaction. A stale `ExpectedSequence` or a unique
violation on `(shipment_id, sequence)` yields a conflict error mapped to HTTP 409.

**Rationale**: two simultaneous requests must not append two updates with the same sequence or break the chain. The
existing `UNIQUE (shipment_id, sequence)` constraint is the last line of defence; the row lock gives a deterministic
error before the insert.

**Alternatives considered**: no transaction (risks a split chain and a stale projection); optimistic retry in the
handler (hides a concurrency bug from the caller and complicates the API contract); database advisory lock per shipment
(extra mechanism already used by the migration runner, but harder to reason about than a row lock).

## 3. Identifying the ground partner to assign

**Decision**: the update request carries `assignedPartnerId` (UUID). A new `GET /ground-partners` endpoint lists the
registered ground partners (`id`, `name`) so the MFE can render a labelled `<select>` and API steps can resolve the name
used in the Gherkin ("Transportes Rápido") to its id.

**Rationale**: the Gherkin names the partner, but the ledger stores `responsible_party_id` and
`assigned_partner_id` as UUIDs (migration 0001). A read-only lookup endpoint keeps the contract explicit, gives the
browser an accessible control label, and avoids sending mutable names as identifiers.

**Alternatives considered**: accepting the partner name in the update request (breaks when a name changes and leaks a
presentation concern into the ledger); hard-coding the demo partner UUIDs in the MFE (couples the UI to seed data);
reusing an HU-01 user-list endpoint (not in HU-03's contract and may not exist).

## 4. Privacy versus the responsibility message

**Decision** (user clarification 2026-11-24): an update attempt on an unknown shipment code, or on a shipment the caller
cannot see, returns the same 403 `NOT_RESPONSIBLE` as an attempt on a visible shipment by the wrong caller. The response
never reveals existence.

**Rationale**: satisfies FR-012, cross-cutting rule 4 and constitution Principle VI while keeping the literal Gherkin
outcome "you are told you are not responsible for that update" for every responsibility example, including the
non-assigned ground partner.

**Alternatives considered**: 404 for invisible/unknown shipments (constitution-compliant but contradicts the scenario
wording); 403 for all attempts including unknown codes but with a different message for invisible shipments (reveals
existence through the message).

## 5. No new migration

**Decision**: HU-03 adds no SQL migration.

**Rationale**: migration 0001 already contains `users`, `shipments`, `updates`, the CHECK constraints, the immutability
trigger and the indexes needed by this story. Seeds for later stages belong to the reserved `0002_seed_ledger`, now
unblocked by the fingerprint contract but outside HU-03's scope. QA prepares shipments in later stages by calling
HU-03's own endpoint through the gateway, which also exercises the feature under test.

**Alternatives considered**: a test-only migration that inserts chained updates (would duplicate the fingerprint
formula in SQL, couple tests to the DB and pre-empt another owner's reserved migration); direct DB writes from QA
(forbidden: QA goes through the gateway).

## 6. Gateway scope

**Decision**: no change at all in `hubx-gw`, including no new test file.

**Rationale**: the existing reverse proxy already strips `/api` and forwards Authorization for any HTTP method, which is
all the new route needs (constitution Principle II). The Cucumber API scenarios exercise `POST /api/...` through the
gateway in every run. Security headers, CORS and rate limiting already live there. The only existing gateway test is an
in-package `proxy_test.go` that predates the `test/`+`mocks/` layout required by Principle IV; adding another test in
that file would extend a deviation instead of fixing it, so gateway test refactoring is out of HU-03's scope.

**Alternatives considered**: adding per-route logic to the gateway (would introduce business rules, forbidden by
`hubx-gw/AGENTS.md`).

## 7. MFE validation without a test runner

**Decision**: the MFE has no `test` script and no test dependency. Validate it with `npm --prefix hubx-mfe run build`
(strict typecheck + production build) and with the `@e2e` Cucumber scenario in Chromium.

**Rationale**: adding a component-test framework needs a new dependency and human approval (constitution Principle III);
the mandatory browser happy path already covers the observable UI behavior.

**Alternatives considered**: adding Vitest/Testing Library (new dependencies, out of scope); relying only on unit tests
( HU-01's sign-in bug showed that the browser path must be exercised).

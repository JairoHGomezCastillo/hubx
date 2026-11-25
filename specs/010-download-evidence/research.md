# Research: Download the traceability evidence

## Decision 1: PDF generation with the Go standard library only

- **Decision**: implement a small PDF writer in `internal/evidence/pdf.go`. It emits a valid single-page PDF with
  uncompressed text streams (Helvetica, 10–12 pt), one stream per page and a standard xref table. No external package
  is imported.
- **Rationale**: FR-011 fixes PDF as the deliverable. Constitution III limits Go dependencies to the PostgreSQL
  driver, JWT, bcrypt and `go.uber.org/mock`, and requires human approval for any new one; no approval exists for a
  PDF library. A stdlib writer keeps `go.mod` unchanged and `govulncheck`/`gosec` scope unchanged.
- **Alternatives considered**:
  - Add `gopdf`/`fpdf`: rejected, needs a human-approved dependency and expands the supply chain.
  - Return HTML and let the browser "Print to PDF": rejected, the API would not deliver a PDF and the artifact would
    depend on the recipient's browser; it also makes the `@e2e` download assertion impossible.
  - Pre-render a static PDF template: rejected, the content (updates, fingerprints, generation date) is dynamic.
- **Consequence**: text must be Latin-1-safe. All document content is English and uses ASCII names/codes from the
  backlog fixtures, so no font embedding is needed. The writer escapes `(`, `)` and `\` in PDF strings. Uncompressed
  streams are deliberate: a third party (and the acceptance test) can read the document bytes directly.

## Decision 2: reuse one visibility rule, exported from `shipment`

- **Decision**: export the existing unexported `canSee` as `shipment.CanSee(current *Shipment, principal Principal) bool`
  and update HU-03/HU-04/HU-05 callers. The evidence reader calls `FindByCode`, then `CanSee` before assembling anything.
- **Rationale**: Constitution VI requires that a caller who cannot see a shipment gets the same answer as if it did not
  exist; four features would otherwise carry four copies of the rule.
- **Alternatives considered**: a separate `CanSee` in `internal/evidence`: rejected, duplicated business rule that can
  drift. Letting the traceability reader decide visibility: rejected, it would couple evidence to a presentation view
  and lose the shipment data needed for FR-012.

## Decision 3: complete shipment projection from one parameterized query

- **Decision**: extend `ShipmentRepository.FindByCode` to select `code, origin, destination, flight, weight_kg,
  customer_id, customer_name, stage, assigned_partner_id, created_at` (plus the existing fields) with the same
  parameterized `WHERE code = $1`.
- **Rationale**: FR-012 requires the creation data; today those fields are only set in memory during creation, so a
  reloaded shipment would produce an incomplete document.
- **Alternatives considered**: add a second repository method just for evidence: rejected, two queries for the same
  row. Read from the traceability response: rejected, it does not carry origin/destination/flight/weight/customer.

## Decision 4: verification instructions are the canonical fingerprint recipe

- **Decision**: the document's instructions reproduce, in plain English, the canonical formula already documented in
  `contracts/hu03-record-update.md` §"Fingerprint formula (canonical, reproducible)": the seven `key=value` lines in
  order, joined by `\n`, no trailing newline, `-` for nulls, `recorded_at` truncated to microseconds, SHA-256 over
  that literal string. The document lists each update's inputs alongside its fingerprint so the check needs no HUBX
  access.
- **Rationale**: HU-10's Conversation requires a third party to check fingerprints without HUBX; reusing the canonical
  formula guarantees agreement with what HU-03 recorded and HU-05 verifies (including the microsecond fix).
- **Alternatives considered**: writing "trust us, the chain is valid": rejected, not verifiable. Sending a link to
  HU-05: rejected, the third party has no access to HUBX.
- **Scope boundary**: the document MUST state that it proves internal consistency only, not that HUBX did not rebuild
  the whole history (FR-007).

## Decision 5: error mapping and route wiring

- **Decision**: `GET /api/shipments/{code}/evidence` is guarded by the existing flat `RequireAuth` (same as HU-04/HU-05),
  so no gateway change is needed. Errors: 401 flat `UNAUTHORIZED`, 403 flat `FORBIDDEN` for roles without permission,
  404 flat `SHIPMENT_NOT_FOUND` for unknown or invisible shipments, 500 flat `INTERNAL_SERVER_ERROR`.
- **Rationale**: matches the existing flat-body family on HU-02/HU-05; the gateway emits the nested session body only
  for HU-03 routes and must not be changed for this story.
- **Alternatives considered**: nested HU-04-style 404: rejected, would break the shared Gherkin step dispatch for
  "no shipment exists with that code" and diverge from HU-05.

## Decision 6: MFE download without a new dependency

- **Decision**: `evidenceService` calls `fetchWithAuth`, checks `response.ok`, turns the body into a `Blob`, creates an
  object URL and clicks a temporary `<a download="evidence-<code>.pdf">`. The URL is revoked after the click.
- **Rationale**: `fetchWithAuth` already returns the raw `Response` and handles 401 by clearing the in-memory JWT and
  navigating to sign-in; no new npm package is required.
- **Alternatives considered**: navigate to the URL in a new tab: rejected, it would not save the PDF and the JWT is
  in memory. Server-side file path: rejected, the MFE is served by nginx and has no access to API internals.

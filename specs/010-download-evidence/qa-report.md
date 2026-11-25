# QA Report: HU-10 Download the traceability evidence

**Story**: HU-10  
**Agent**: qa-seguridad  
**Phase**: verify (final, after gosec fix)  
**Date**: 2026-11-25  
**Branch**: `010-download-evidence`  
**Verified commit**: `f81a551` (`[agent:backend-go] HU-10 close gosec findings without changing PDF output`; tree clean before this report)  
**Contract**: `contracts/hu10-download-evidence.md`

---

## Verdict

**PASSED.** On commit `f81a551` the full integration suite is green (`66 scenarios (66 passed)`, `424 steps (424 passed)`),
including all 7 HU-10 scenarios and the `@e2e` browser PDF download. `gosec ./...` now reports `Issues : 0` in both
`hubx-api` and `hubx-gw` (exit 0), and both `govulncheck` scans report no vulnerabilities. The two previously open HU-10
gosec findings are resolved. No critical or high finding is open; only the pre-existing gateway gap (security headers,
restrictive CORS and rate limiting), not introduced by HU-10, remains recorded. QA changed no production code, tests,
contract or task file and made no commit or push.

---

## 1. Integration suite (full stack, API + browser)

**Command**: `make test-integration` (clean isolated `hubx-it` stack: PostgreSQL + migrations + API + gateway + MFE,
then Cucumber with `strict: true` at `hubx-bender/cucumber.mjs:8`).  
**Exit code**: 0.  
**Literal final summary of the executed run**:

```
1 hook (1 passed)
66 scenarios (66 passed)
424 steps (424 passed)
1m 5.472s (1m 5.232s executing your code)
INTEGRATION_EXIT=0
```

`strict: true` (`hubx-bender/cucumber.mjs:8`) makes an undefined/pending step fail the suite; all 424 steps passed.

The HU-05 ledger tamper regression ran in this same suite. Literal line from the run output:

```
Tampered update 2 of HBX-317323CB to stage "Out for delivery"
```

### Gherkin coverage (scenario → test → result)

Per-scenario statuses were read from the Cucumber JSON report written by this run
(`hubx-bender/reports/cucumber.json`) with a read-only Node script:

```
SCENARIO line=7 | Scenario: Evidence for a shipment with updates | tags=[@e2e] | steps=9 | passed
SCENARIO line=15 | Scenario: Verification by a third party | tags=[] | steps=5 | passed
SCENARIO line=20 | Scenario: Evidence for a shipment without updates | tags=[] | steps=7 | passed
SCENARIO line=27 | Scenario: Code that matches no shipment | tags=[] | steps=5 | passed
SCENARIO line=39 | Scenario Outline: Role without permission to download evidence | tags=[] | steps=5 | passed
SCENARIO line=40 | Scenario Outline: Role without permission to download evidence | tags=[] | steps=5 | passed
SCENARIO line=42 | Scenario: Person who has not signed in | tags=[] | steps=6 | passed
```

| # | Scenario (verbatim from `hubx-bender/features/HU10_download-evidence.feature`) | Feature line | Type | Result |
|---|---|---|---|---|
| 1 | Evidence for a shipment with updates (operator, 3 updates) | 7 (`@e2e`) | @e2e browser download via MFE | passed |
| 2 | Verification by a third party (recomputes fingerprints from the PDF only) | 15 | API fixture + PDF-only verification | passed |
| 3 | Evidence for a shipment without updates | 20 | API through gateway | passed |
| 4 | Code that matches no shipment | 27 | API through gateway | passed |
| 5 | Role without permission — `ground partner` | 39 (Examples row) | API through gateway | passed |
| 6 | Role without permission — `customer` | 40 (Examples row) | API through gateway | passed |
| 7 | Person who has not signed in | 42 | API through gateway | passed |

**Regression**: the other 59 of 66 scenarios (health + HU-01…HU-09) passed in the same run; the literal summary above
is the evidence. No older story regressed.

---

## 2. `@e2e` download evidence

The critical happy path runs in Chromium against the MFE; the step definition uses role/label locators only and a real
browser download event (`hubx-bender/features/steps/HU10_download-evidence.steps.ts:167-180`):

```
167:    const link = this.ui.getByRole('link', { name: /download evidence/i });
168:    await expect(link, 'operator-only "Download evidence" link on the Dashboard').toBeVisible({ timeout: 5000 });
169:    await link.click();
170:    await expect(this.ui).toHaveURL(/\/download-evidence$/);
172:    const region = this.ui.getByRole('region', { name: /download evidence/i });
173:    await region.getByLabel(/shipment code/i).fill(code);
174:    const downloadPromise = this.ui.waitForEvent('download');
175:    await region.getByRole('button', { name: /download evidence/i }).click();
176:    this.download = await downloadPromise;
178:    this.evidencePath = join(tmpdir(), `hubx-evidence-${code}.pdf`);
179:    await this.download.saveAs(this.evidencePath);
180:    this.pdfBytes = await readFile(this.evidencePath);
```

The browser download's suggested filename is asserted against `^evidence-HBX-[0-9A-F]{8}\.pdf$`
(`HU10_download-evidence.steps.ts:215-218`), followed by `%PDF-` magic bytes, heading, shipment data and scope
statements (`:225-239`), three updates in chain order with responsible party, dates and fingerprints (`:242-262`), and
a parseable `Generated at:` plus the verification instructions (`:265-272`). All passed (scenario line 7 above).

Literal evidence from the PDF saved by the browser download in this final run:

```
/tmp/hubx-evidence-HBX-6118F6AD.pdf 3814 bytes
magic=%PDF-
(HUBX traceability evidence) Tj
(Code: HBX-6118F6AD) Tj
(Updates:) Tj
(1. In flight) Tj
(Fingerprint: f63658b3e19dd4da989ea9900f0c12adcbfe7367bb35ed0d6ad0b154b7af667e) Tj
(2. At destination airport) Tj
(Fingerprint: 94accefb4fe2c8592d3174a54efb6ab65519e1dc649c42da8879c71fa4a24105) Tj
(3. Assigned to partner) Tj
(Fingerprint: 4b1bc8c62caf688a02cb56637d65ad6e690f39bdc2f8df7ac08570d73a8dbd9c) Tj
(Generated at: 2026-11-25T01:14:21Z) Tj
(How to verify the fingerprints:) Tj
(Scope: This evidence proves that the shipment history is consistent with itself. It does not by itself prove that HUBX did not rebuild the whole history.) Tj
```

`/tmp/hubx-evidence-…pdf` is the QA workspace's `saveAs` target; the suggested filename is asserted separately by the
steps quoted above.

---

## 3. Third-party fingerprint verification

Scenario at feature line 15 passed. The step drops the JWT (`HU10_download-evidence.steps.ts:283`), makes **zero HTTP
calls**, extracts the three seven-line canonical blocks printed in the PDF, recomputes each SHA-256 digest, checks that
each digest is printed in the document and that `previous_fingerprint` of update *n* equals the digest of update *n-1*
(`:285-303`), then asserts the printed fingerprints equal the recomputed ones in chain order (`:307-315`). The literal
PDF lines in §2 show the three digests for the three updates, and the scenario result above is `passed`.

---

## 4. Security scans (literal output)

### 4.1 `gosec ./...` in `hubx-api` — passed (exit 0)

**Command**: `NO_COLOR=1 gosec ./...` with cwd `hubx-api` (colour disabled for a literal excerpt; same findings as the
mandated `gosec ./...`, which also exited 0; `gosec -quiet ./...` exited 0 with no findings).  
**Literal output summary**:

```
Results:


Summary:
  Gosec  : dev
  Files  : 64
  Lines  : 6213
  Nosec  : 1
  Issues : 0

GOSEC_API_EXIT=0
```

### 4.2 `gosec ./...` in `hubx-gw` — passed (exit 0)

**Command**: `gosec ./...` with cwd `hubx-gw`. Literal summary:

```
Results:

Summary:
  Gosec  : dev
  Files  : 3
  Lines  : 294
  Nosec  : 0
  Issues : 0

GOSEC_GW_EXIT=0
```

### 4.3 `govulncheck -C hubx-api ./...` — passed (exit 0)

```
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 1
vulnerability in modules you require, but your code doesn't appear to call these
vulnerabilities.
Use '-show verbose' for more details.
GOVULN_API_EXIT=0
```

### 4.4 `govulncheck -C hubx-gw ./...` — passed (exit 0)

```
No vulnerabilities found.
GOVULN_GW_EXIT=0
```

### 4.5 Previously open HU-10 findings — RESOLVED

| Previous finding | Resolution in `f81a551` | Closing evidence |
|---|---|---|
| HIGH `G115` at `hubx-api/internal/evidence/pdf.go:150` (`rune -> byte` conversion) | `pdfString` no longer converts a rune to byte; it emits `out.WriteRune(r)` at `pdf.go:151`. Check: `grep -n 'byte(r)' hubx-api/internal/evidence/pdf.go` → `no byte(r) conversion`. | `gosec ./...` in hubx-api: `Issues : 0`, `GOSEC_API_EXIT=0` (§4.1). |
| MEDIUM `G705` at `hubx-api/internal/handlers/evidence/handler.go:81` (`ResponseWriter.Write`) | The generated PDF is now served with `io.Copy(w, bytes.NewReader(pdfBytes))` at `handler.go:85`; headers were already `application/pdf`, `attachment` and `no-store` at `:79-81`. Check: `grep -n 'io.Copy\|ResponseWriter.Write\|w.Write(' hubx-api/internal/handlers/evidence/handler.go` → `85: _, _ = io.Copy(w, bytes.NewReader(pdfBytes))` (the other `w.Write` at `:129` is the flat JSON error body, not flagged). | `gosec ./...` in hubx-api: `Issues : 0`, `GOSEC_API_EXIT=0` (§4.1). |

PDF bytes, response headers and error bodies are unchanged by the fix, as demonstrated by the green suite in §1 and
the downloaded PDF content in §2.

---

## 5. OWASP API Security Top 10 (2023) review

| OWASP category | Test or evidence (literal output / `file:line`) | Result |
|---|---|---|
| **API1** Broken Object Level Authorization | Scenario line 27 ("Code that matches no shipment") passes with the flat 404. Reader returns `ErrShipmentNotFound` for unknown **or** invisible shipments (`hubx-api/internal/evidence/reader.go:71-72`) and `ErrForbidden` before any lookup for non-operators (`:63-65`). | passed |
| **API2** Broken Authentication | Scenario line 42 passes: 401 `UNAUTHORIZED`, `Authorization header required`, no PDF (`HU10_download-evidence.steps.ts:332-340`). Route wrapped in `RequireAuth` (`handler.go:59`); API validates JWT independently (`internal/handlers/auth/middleware.go:33-44`); gateway validates every route except `/health`, `/api/health`, `/api/auth/sign-in` (`hubx-gw/cmd/gw/main.go:58-65`). | passed |
| **API3** Broken Object Property Level Authorization | Only contract fields are rendered (`hubx-api/internal/evidence/pdf.go:44-72`; literal PDF lines in §2); non-operators refused before lookup (`reader.go:63-65`); flat error body only (`handler.go:111-123`). | passed |
| **API4** Unrestricted Resource Consumption | Single shipment, no pagination or size parameter (`contracts/hu10-download-evidence.md:17`); gateway rate limiting still unimplemented (`hubx-gw/cmd/gw/main.go:44-70`). | open pre-existing finding (not introduced by HU-10) |
| **API5** Broken Function Level Authorization | Scenario Outline lines 39-40 pass with 403 `FORBIDDEN` and no PDF for both `ground partner` and `customer` (`HU10_download-evidence.steps.ts:324-330`). | passed |
| **API6** Unrestricted Access to Sensitive Business Flows | Read-only GET; no write or ledger mutation (`reader.go:16-19`, `:62-136`). Rate limiting still absent at the gateway. | open pre-existing finding |
| **API7** Server Side Request Forgery | No user-supplied URL is fetched: the API reads its own DB through injected repositories (`reader.go:67-89`); the MFE calls only `/api/…` (`hubx-mfe/src/services/evidenceService.ts:62-67`). | not applicable (no external URL input) |
| **API8** Security Misconfiguration | Both mandated `gosec` scans are clean (`Issues : 0`, §4.1-4.2) and govulncheck found nothing (§4.3-4.4). Gateway still has no security headers, restrictive CORS or rate limiting (`hubx-gw/cmd/gw/main.go:44-70`; previously recorded in `specs/002-create-shipment/qa-report.md:227` and `specs/005-verify-integrity/qa-report.md:242-246`). | passed for scans; open pre-existing gateway gap |
| **API9** Improper Inventory Management | Route documented in `contracts/hu10-download-evidence.md:12` and registered once at `hubx-api/cmd/api/main.go:115-116`; no undocumented route added. API version header was **not verified** in this run. | passed for the new route; version header not verified |
| **API10** Unsafe Consumption of APIs | No external API consumed; the MFE uses its own gateway path through `fetchWithAuth` (`evidenceService.ts:62-67`, `httpClient.ts:28-33`). | not applicable |

Additional controls checked:

- **Parameterized SQL**: every query used by the reader is parameterized (`hubx-api/internal/shipment/update_postgres.go:19-32`, including `WHERE code = $1` and `WHERE id = ANY($1)`); ledger access only via `LedgerRepository` (`reader.go:75`).
- **Server-side validation**: role and visibility are checked in the API, not only in the UI (`reader.go:62-73`, `handler.go:62-74`).
- **Secrets**: no secret value in tracked `application.properties`, only comments pointing to env vars (`hubx-api/application.properties:1-2`, `hubx-gw/application.properties:1`); gateway requires `JWT_SECRET` and `GATEWAY_SECRET` from the environment (`hubx-gw/cmd/gw/main.go:29-42`).
- **Bcrypt**: unchanged from HU-01; migration 0001 documents cost 12 (`hubx-api/migrations/0001_baseline_schema.sql:71`). Not re-verified by HU-10 tests.
- **Content-Disposition safety**: `evidenceFilename` allowlists `[A-Za-z0-9_-]` (`handler.go:95-109`); `Cache-Control: no-store` and `application/pdf` set at `:79-81`.

---

## 6. Boundary review

| Boundary | Evidence | Result |
|---|---|---|
| No imports between services | `grep -rn '"hubx-api/' hubx-gw --include='*.go'` → `no hubx-api imports in hubx-gw`; `grep -rn '"hubx-gw/' hubx-api --include='*.go'` → `no hubx-gw imports in hubx-api`. | passed |
| No business logic in gateway | `hubx-gw/cmd/gw/main.go:44-70` only wires `/health`, proxy and JWT middleware; proxy strips `/api` and injects the shared secret (`hubx-gw/internal/proxy/proxy.go:41-57`). | passed |
| MFE only through `/api` | `hubx-mfe/src/services/evidenceService.ts:62-67` calls `/api/shipments/{code}/evidence` via `fetchWithAuth`; no direct API URL found in `hubx-mfe/src` by `grep -rnE 'localhost:8081|VITE_GATEWAY_URL|https?://[^" ]+' hubx-mfe/src`. | passed |
| Ledger immutability / tamper regression | HU-05 tamper scenario ran and passed in this suite: `Tampered update 2 of HBX-317323CB to stage "Out for delivery"` (§1). HU-10 reader only calls `LedgerRepository.List` (`reader.go:75`). | passed |

---

## 7. Findings

### Critical

None.

### High

None. The previously open HIGH `G115` at `pdf.go:150` is resolved and the final `gosec` scan shows `Issues : 0` (§4.1, §4.5).

### Medium

The previously open MEDIUM `G705` at `handler.go:81` is resolved (§4.5).

### Pre-existing (open, not introduced by HU-10)

1. **Gateway security headers, restrictive CORS and rate limiting not implemented**  
   **OWASP**: API4, API8  
   **Location**: `hubx-gw/cmd/gw/main.go:44-70`  
   **Description**: no headers/CORS/limiter in the gateway wiring; already recorded as open in
   `specs/002-create-shipment/qa-report.md:227` and `specs/005-verify-integrity/qa-report.md:242-246`, and still listed in
   `memory.md` ("Pending"). Requires its own story. `hubx-gw` was untouched by `f81a551`.

---

## 8. Blockers

None. All 66 scenarios passed, both `gosec` scans reported `Issues : 0` with exit 0, both `govulncheck` scans found no
vulnerabilities, and no critical or high finding remains open. The only open finding is the pre-existing gateway gap
listed in §7.

---

## 9. Files changed in this phase

- `specs/010-download-evidence/qa-report.md` (this file, overwritten with the final verification). No production code,
  feature, step definition, contract, `tasks.md`, `spec.md`, `plan.md` or `pr.md` was edited; no commit, push or merge
  was performed.

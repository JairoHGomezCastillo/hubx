# QA Report: HU-10 Download the traceability evidence

**Story**: HU-10  
**Agent**: qa-seguridad  
**Phase**: verify  
**Date**: 2026-11-25  
**Branch**: `010-download-evidence`  
**Implementation under test**: `2f9c9ae` (hubx-api), `bdf93ec` (hubx-mfe)  
**Contract**: `contracts/hu10-download-evidence.md`

---

## Verdict

**FAILED for merge, not for tests.** The complete integration suite is green — `66 scenarios (66 passed)`,
`424 steps (424 passed)`, including all 7 HU-10 scenarios and the `@e2e` browser download — and both `govulncheck`
scans and `gosec` in `hubx-gw` are clean. However, `gosec ./...` in `hubx-api` **exits 1 with 2 findings**, one of them
**HIGH severity (G115)**. Constitution VI requires `gosec` to pass without findings before merging, so this story
cannot be marked `ok` until `backend-go` resolves or documents an accepted exception for the HIGH finding. No
production code, test, contract or task file was changed by QA in this phase.

---

## 1. Integration suite (full stack, API + browser)

**Command**: `make test-integration` (clean isolated `hubx-it` stack: PostgreSQL + migrations + API + gateway + MFE,
then Cucumber with `strict: true` at `hubx-bender/cucumber.mjs:8`).  
**Exit code**: 0.  
**Literal final summary (from the executed run)**:

```
1 hook (1 passed)
66 scenarios (66 passed)
424 steps (424 passed)
1m 7.813s (1m 7.523s executing your code)
INTEGRATION_EXIT=0
```

`strict: true` (`hubx-bender/cucumber.mjs:8`) means an undefined or pending step fails the suite; all 424 steps
passed, so no step was skipped.

The ledger tamper regression from HU-05 also executed in this run. Literal line from the suite output:

```
Tampered update 2 of HBX-01C0CCCE to stage "Out for delivery"
```

### Gherkin coverage (scenario → test → result)

Per-scenario statuses were read from the Cucumber JSON report written by the same run
(`hubx-bender/reports/cucumber.json`) with a read-only Node script; every HU-10 step has `result.status: "passed"`.

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

**Regression**: the remaining 59 scenarios of the 66 (health + HU-01…HU-09) all passed in the same run; the suite
summary above is the literal evidence. No older story regressed.

---

## 2. `@e2e` download evidence

The critical happy path runs in Chromium against the MFE. The step definition uses role/label locators only and a
real browser download event (`hubx-bender/features/steps/HU10_download-evidence.steps.ts:167-180`):

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

The same scenario text asserts the browser download's suggested filename matches
`^evidence-HBX-[0-9A-F]{8}\.pdf$` (`HU10_download-evidence.steps.ts:215-218`), then asserts `%PDF-` magic bytes, heading,
shipment code, origin, destination, flight, weight, customer, current stage and the internal-consistency scope
statement (`:225-239`), the three updates in chain order with party, dates and fingerprints (`:242-262`), and a
parseable `Generated at:` value plus the verification instructions (`:265-272`). All passed (scenario line 7 above).

Literal evidence from the saved browser download (the file left by `download.saveAs` at the end of the run):

```
PDF_FILE=/tmp/hubx-evidence-HBX-5ABDFE96.pdf
/tmp/hubx-evidence-HBX-5ABDFE96.pdf 3814 bytes
magic=%PDF-
```

Literal lines from the uncompressed PDF text stream of that downloaded file:

```
(HUBX traceability evidence) Tj
(Code: HBX-5ABDFE96) Tj
(Origin: Bogot\341) Tj
(Destination: Miami) Tj
(Flight: AV-204) Tj
(Weight: 100.0 kg) Tj
(Customer: Distribuciones Andina) Tj
(Current stage: Assigned to partner) Tj
(Updates:) Tj
(1. In flight) Tj
(Responsible party: Laura Operator) Tj
(Recorded at: 2026-11-25T01:08:05.300531Z) Tj
(2. At destination airport) Tj
(3. Assigned to partner) Tj
(Generated at: 2026-11-25T01:08:07Z) Tj
(How to verify the fingerprints:) Tj
(Scope: This evidence proves that the shipment history is consistent with itself. It does not by itself prove that HUBX did not rebuild the whole history.) Tj
```

```
--- hex64 count ---
3
```

The generation date (`2026-11-25T01:08:07Z`) is distinct from every update's `Recorded at`
(`2026-11-25T01:08:05…Z`), matching FR-005. The filename assertion above applies to the browser's suggested
filename; `/tmp/hubx-evidence-…pdf` is the QA workspace's `saveAs` target, not the suggested filename.

---

## 3. Third-party fingerprint verification

Scenario at feature line 15 passed (steps at `HU10_download-evidence.steps.ts:278-315`). The step drops the JWT
(`:283`), makes **zero HTTP calls**, extracts the three seven-line canonical blocks printed in the PDF, recomputes
each SHA-256 digest, checks that each is printed in the document, and checks that `previous_fingerprint` of update
*n* equals the digest of update *n-1* (`:285-303`), then asserts the printed fingerprints equal the recomputed ones
in chain order (`:307-315`). Literal PDF lines confirm the chain and the three digests:

```
(Fingerprint: 04a80178d748644ac65f0ca040f367fc1f09e6a9731f4c31d445a1174d742f61) Tj
(previous_fingerprint=04a80178d748644ac65f0ca040f367fc1f09e6a9731f4c31d445a1174d742f61) Tj
(Fingerprint: 751fc462a476b5e56b5ed4edbd3ae60351fd93bd9b32ccb8caa2c09cb4308ddd) Tj
(Fingerprint: 5915718952b8466ab2d523f97e2614e907dffe5655b8adeeeda6c35e0e6cf3eb) Tj
```

---

## 4. Security scans (literal output)

### 4.1 `gosec ./...` in `hubx-api` — FAILED (exit 1)

**Command**: `gosec ./...` with cwd `hubx-api` → `GOSEC_API_EXIT=1`. A repeat with `NO_COLOR=1 gosec -quiet ./...`
produced the same findings without terminal colour codes; literal output:

```
Results:


[/root/.local/share/opencode/worktree/2ba5b3/hu10-demo/hubx-api/internal/evidence/pdf.go:150] - G115 (CWE-190): integer overflow conversion rune -> byte (Confidence: MEDIUM, Severity: HIGH)
    149: 		if r <= 255 {
  > 150: 			c = byte(r)
    151: 		}

Autofix: 

[/root/.local/share/opencode/worktree/2ba5b3/hu10-demo/hubx-api/internal/handlers/evidence/handler.go:81] - G705 (CWE-79): XSS via taint analysis: (net/http.ResponseWriter).Write (Confidence: HIGH, Severity: MEDIUM)
    80: 	w.WriteHeader(http.StatusOK)
  > 81: 	_, _ = w.Write(pdfBytes)
    82: }

Autofix: 

Summary:
  Gosec  : dev
  Files  : 64
  Lines  : 6211
  Nosec  : 1
  Issues : 2

GOSEC_API_QUIET_EXIT=1
```

Both findings are in HU-10 code (`internal/evidence/pdf.go`, `internal/handlers/evidence/handler.go`); QA did not
modify them. See Findings below.

### 4.2 `gosec ./...` in `hubx-gw` — passed (exit 0)

**Command**: `gosec ./...` with cwd `hubx-gw` → `GOSEC_GW_EXIT=0`. Literal summary:

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

---

## 5. OWASP API Security Top 10 (2023) review

| OWASP category | Test or evidence (literal output / `file:line`) | Result |
|---|---|---|
| **API1** Broken Object Level Authorization | Scenario line 27 ("Code that matches no shipment") passes with the contract's flat 404. Reader returns `ErrShipmentNotFound` for unknown **or** invisible shipments (`hubx-api/internal/evidence/reader.go:71-72`) and `ErrForbidden` before any lookup for non-operators (`:63-65`). | passed |
| **API2** Broken Authentication | Scenario line 42 passes: 401 `UNAUTHORIZED`, `Authorization header required`, no PDF (`HU10_download-evidence.steps.ts:332-340`). Route is wrapped in `RequireAuth` (`hubx-api/internal/handlers/evidence/handler.go:59`), API validates JWT independently (`internal/handlers/auth/middleware.go:33-44`), gateway validates on every route except `/health`, `/api/health`, `/api/auth/sign-in` (`hubx-gw/cmd/gw/main.go:58-65`). | passed |
| **API3** Broken Object Property Level Authorization | Only contract fields are rendered (`pdf.go:44-72`; literal PDF lines in §2). Non-operators are refused before a shipment lookup (`reader.go:63-65`); no extra JSON fields on errors (`handler.go:111-123`). | passed |
| **API4** Unrestricted Resource Consumption | Single shipment, no pagination or user-controlled size parameter (`contracts/hu10-download-evidence.md:17`); gateway rate limiting is still not implemented (`hubx-gw/cmd/gw/main.go:44-70`). | open pre-existing finding (not introduced by HU-10) |
| **API5** Broken Function Level Authorization | Scenario Outline lines 39-40 pass with 403 `FORBIDDEN` and no PDF for both `ground partner` and `customer` (`HU10_download-evidence.steps.ts:324-330`). | passed |
| **API6** Unrestricted Access to Sensitive Business Flows | Read-only GET; no write, no ledger mutation (`reader.go:16-19`, `:62-136`). Rate limiting still absent at the gateway. | open pre-existing finding |
| **API7** Server Side Request Forgery | No user-supplied URL is fetched: the API reads its own DB through injected repositories (`reader.go:67-89`); the MFE calls only `/api/…` (`hubx-mfe/src/services/evidenceService.ts:62-67`). | not applicable (no external URL input) |
| **API8** Security Misconfiguration | Gateway has no security headers, restrictive CORS or rate limiting (`hubx-gw/cmd/gw/main.go:44-70`; also recorded in `specs/005-verify-integrity/qa-report.md:242-246`). Additionally, `gosec ./...` in hubx-api exits 1 with HIGH G115 and MEDIUM G705 (§4.1). | open pre-existing + 2 open HU-10 findings |
| **API9** Improper Inventory Management | Route documented in `contracts/hu10-download-evidence.md:12` and registered once at `hubx-api/cmd/api/main.go:115-116`; no undocumented route added. API version header was **not verified** in this run. | passed for inventory of the new route; version header not verified |
| **API10** Unsafe Consumption of APIs | No external API is consumed; the MFE uses its own gateway path through `fetchWithAuth` (`evidenceService.ts:62-67`, `httpClient.ts:28-33`). | not applicable |

Additional controls checked:

- **Parameterized SQL**: every query used by the reader is parameterized (`hubx-api/internal/shipment/update_postgres.go:19-32`, including `WHERE code = $1` and `WHERE id = ANY($1)`; ledger access only via `LedgerRepository`, `reader.go:75`).
- **Server-side validation**: role and visibility are checked in the API, never only in the UI (`reader.go:62-73`; `handler.go:62-74`).
- **Secrets**: no secret value in tracked `application.properties` — only comments pointing to env vars (`hubx-api/application.properties:2`, `hubx-gw/application.properties:1`); the gateway requires `JWT_SECRET` and `GATEWAY_SECRET` from the environment (`hubx-gw/cmd/gw/main.go:29-42`).
- **Bcrypt**: unchanged from HU-01; migration 0001 documents cost 12 (`hubx-api/migrations/0001_baseline_schema.sql:71`). Not re-verified by HU-10 tests.
- **Content-Disposition safety**: `evidenceFilename` allowlists `[A-Za-z0-9_-]` (`hubx-api/internal/handlers/evidence/handler.go:95-109`); `Cache-Control: no-store` and `application/pdf` set at `:77-79`.

---

## 6. Boundary review

| Boundary | Evidence | Result |
|---|---|---|
| No imports between services | `grep -rn '"hubx-api/' hubx-gw --include='*.go'` → `no hubx-api imports in hubx-gw`; `grep -rn '"hubx-gw/' hubx-api --include='*.go'` → `no hubx-gw imports in hubx-api`. | passed |
| No business logic in gateway | `hubx-gw/cmd/gw/main.go:44-70` only wires `/health`, proxy and JWT middleware; proxy strips `/api` and injects the shared secret (`hubx-gw/internal/proxy/proxy.go:41-57`). | passed |
| MFE only through `/api` | `hubx-mfe/src/services/evidenceService.ts:62-67` calls `/api/shipments/{code}/evidence` via `fetchWithAuth`; no direct API URL was found in `hubx-mfe/src` by `grep -rnE 'localhost:8081|VITE_GATEWAY_URL|https?://[^" ]+' hubx-mfe/src`. | passed |
| Ledger immutability / tamper regression | HU-05 tamper scenario ran and passed in this suite: `Tampered update 2 of HBX-01C0CCCE to stage "Out for delivery"` (§1). The HU-10 reader only calls `LedgerRepository.List` (`reader.go:75`). | passed |

---

## 7. Findings

### Critical

None.

### High

1. **`gosec` G115 (CWE-190), integer overflow conversion `rune -> byte`**  
   **OWASP**: API8 (security misconfiguration / static analysis gate)  
   **Location**: `hubx-api/internal/evidence/pdf.go:150`  
   **Description**: `gosec ./...` in hubx-api exits 1 with `G115 … Severity: HIGH`. The surrounding code guards with
   `if r <= 255` at `pdf.go:149`, so this looks like a false positive, **but it is not waived in the repository and
   Constitution VI requires `gosec` to pass with no findings before merging**. Owner: `backend-go`; either make the
   conversion provably safe to the analyser or record a reviewed `#nosec G115` justification. QA did not edit code.

### Medium

2. **`gosec` G705 (CWE-79), XSS via `ResponseWriter.Write`**  
   **OWASP**: API8  
   **Location**: `hubx-api/internal/handlers/evidence/handler.go:81`  
   **Description**: taint analysis flags the PDF write. The response sets `Content-Type: application/pdf`,
   `Content-Disposition: attachment` and `Cache-Control: no-store` at `handler.go:77-79`, so browser HTML execution
   is unlikely, but the finding is open and makes the mandated scan fail. Owner: `backend-go`.

### Pre-existing (open, not introduced by HU-10)

3. **Gateway security headers, restrictive CORS and rate limiting not implemented**  
   **OWASP**: API4, API8  
   **Location**: `hubx-gw/cmd/gw/main.go:44-70`  
   **Description**: no headers/CORS/limiter in the gateway wiring (`hubx-gw/cmd/gw/main.go:44-70`); already recorded as
   open in `specs/002-create-shipment/qa-report.md:227` and `specs/005-verify-integrity/qa-report.md:242-246`, and still
   listed in `memory.md` ("Pending"). Requires its own story.

---

## 8. Blockers

1. **Open HIGH `gosec` finding G115 in HU-10 code** (`hubx-api/internal/evidence/pdf.go:150`); `gosec ./...` in
   hubx-api exits 1. Constitution VI blocks merge while this is open. No critical finding was found.
2. Medium `gosec` G705 (`hubx-api/internal/handlers/evidence/handler.go:81`) is also open and contributes to the
   failed scan.

There are **no red scenarios**: all 7 HU-10 scenarios and the 59 regression scenarios passed in the executed suite.

---

## 9. Files changed in this phase

- `specs/010-download-evidence/qa-report.md` (this file) — the only file written by QA in verify. No production code,
  feature file, step definition, contract or `tasks.md` was edited, and no commit, push or merge was performed.

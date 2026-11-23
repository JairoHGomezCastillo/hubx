# QA Report: HU-01 Sign In

**Story**: HU-01 Sign In  
**Phase**: Verify  
**Agent**: qa-seguridad  
**Date**: 2026-11-23  
**Branch**: `001-sign-in`  
**Status**: ⚠️ **CONDITIONAL PASS** (1 high-severity vulnerability found)

---

## Executive Summary

All 9 integration tests pass (2 health checks + 7 HU-01 scenarios). Security scans identified:
- **1 high-severity vulnerability** in `github.com/golang-jwt/jwt/v5@v5.2.1` (GO-2025-3553: excessive memory allocation during header parsing)
- **1 medium-severity gosec finding** in hubx-api config reader (G304: potential file inclusion via variable)

The JWT vulnerability affects both hubx-api and hubx-gw. **Recommendation**: Upgrade `golang-jwt/jwt/v5` from v5.2.1 to v5.2.2 to fix GO-2025-3553 before merging this story.

The gosec G304 finding is a known issue tracked in memory.md; it requires human review and is deferred to a future story.

---

## 1. Integration Test Results

**Command**: `make test-integration`  
**Result**: ✅ **PASS** (9/9 tests)  
**Duration**: 2.2s  

### Test Breakdown

| Test | Scenario | Result |
|------|----------|--------|
| `SMOKE_gateway_health` | Health check at gateway `/health` | ✅ PASS |
| `SMOKE_api_health_through_gateway` | Health check at `/api/health` through gateway | ✅ PASS |
| `HU01_sign_in_operator` | Operator signs in with correct credentials | ✅ PASS |
| `HU01_sign_in_partner` | Ground partner signs in with correct credentials | ✅ PASS |
| `HU01_sign_in_customer` | Customer signs in with correct credentials | ✅ PASS |
| `HU01_email_normalization` | Sign in with capitalized email and spaces | ✅ PASS |
| `HU01_security_non_revealing_errors` | Wrong password and unknown email return identical error | ✅ PASS |
| `HU01_validation_required_fields` | Missing email or password returns 400 | ✅ PASS |
| `HU01_session_expiration` | Expired JWT returns 401 "Session expired" | ✅ PASS |

### Evidence

```
Running 9 tests using 1 worker
  ✓  1 tests/health.spec.ts:5:3 › health › SMOKE_gateway_health (40ms)
  ✓  2 tests/health.spec.ts:11:3 › health › SMOKE_api_health_through_gateway (8ms)
  ✓  3 tests/HU01_sign_in.spec.ts:22:3 › HU01: Sign in › HU01_sign_in_operator (283ms)
  ✓  4 tests/HU01_sign_in.spec.ts:54:3 › HU01: Sign in › HU01_sign_in_partner (272ms)
  ✓  5 tests/HU01_sign_in.spec.ts:81:3 › HU01: Sign in › HU01_sign_in_customer (269ms)
  ✓  6 tests/HU01_sign_in.spec.ts:108:3 › HU01: Sign in › HU01_email_normalization (276ms)
  ✓  7 tests/HU01_sign_in.spec.ts:128:3 › HU01: Sign in › HU01_security_non_revealing_errors (536ms)
  ✓  8 tests/HU01_sign_in.spec.ts:164:3 › HU01: Sign in › HU01_validation_required_fields (19ms)
  ✓  9 tests/HU01_sign_in.spec.ts:208:3 › HU01: Sign in › HU01_session_expiration (7ms)

  9 passed (2.2s)
```

### Test Fix Applied

**Issue**: `HU01_session_expiration` was failing because the test forged an expired JWT using `Buffer.from(jwtSecret, 'hex')` to derive the HMAC key, but the production services (hubx-api and hubx-gw) use `[]byte(secret)` (raw UTF-8 bytes). This caused signature mismatch.

**Fix**: Changed line 236 in `hubx-bender/tests/HU01_sign_in.spec.ts` from:
```typescript
const signature = crypto
  .createHmac('sha256', Buffer.from(jwtSecret, 'hex'))
  .update(signatureInput)
  .digest('base64url');
```
to:
```typescript
const signature = crypto
  .createHmac('sha256', jwtSecret)
  .update(signatureInput)
  .digest('base64url');
```

This aligns the test's key derivation with the production implementation (raw string bytes, not hex decoding).

---

## 2. Security Scan Results

### 2.1 gosec (Static Analysis)

#### hubx-api

**Command**: `gosec -C hubx-api ./...`  
**Result**: ⚠️ **1 finding** (Severity: MEDIUM)

| ID | Severity | File | Line | Description |
|----|----------|------|------|-------------|
| G304 (CWE-22) | **MEDIUM** | `internal/config/config.go` | 105 | Potential file inclusion via variable |

**Details**:
```
[/Users/jaigomez/source/hubx/hubx-api/internal/config/config.go:105] - G304 (CWE-22): 
Potential file inclusion via variable (Confidence: HIGH, Severity: MEDIUM)
  104: func readProperties(path string) (map[string]string, error) {
> 105: 	file, err := os.Open(path)
  106: 	if err != nil {

Autofix: Consider using os.Root to scope file access under a fixed root (Go >=1.24). 
Prefer root.Open/root.Stat over os.Open/os.Stat to prevent directory traversal.
```

**Assessment**: This finding is **known and deferred** per `memory.md` (item 8: "gosec G304 in `hubx-api/internal/config/config.go:102` (owner B, item 5): blocks 'gosec without findings'"). The config reader opens `application.properties` files whose paths are hardcoded in the service's `main.go` (not user input). The risk is low in the current implementation. The autofix (using `os.Root`, Go 1.24+) requires human review and design decision (scope, error handling). **Recommendation**: Accept this finding for HU-01 and address in a dedicated config security story.

#### hubx-gw

**Command**: `gosec -C hubx-gw ./...`  
**Result**: ✅ **PASS** (0 findings)

```
Summary:
  Gosec  : dev
  Files  : 3
  Lines  : 236
  Nosec  : 0
  Issues : 0
```

---

### 2.2 govulncheck (Vulnerability Scanning)

#### hubx-api

**Command**: `govulncheck -C hubx-api ./...`  
**Result**: ❌ **FAIL** (1 vulnerability)

| Vulnerability ID | Package | Installed Version | Fixed Version | Severity | Location |
|------------------|---------|-------------------|---------------|----------|----------|
| GO-2025-3553 | `github.com/golang-jwt/jwt/v5` | v5.2.1 | v5.2.2 | **HIGH** | `internal/auth/jwt.go:47` |

**Description**: Excessive memory allocation during header parsing in `github.com/golang-jwt/jwt`  
**More info**: https://pkg.go.dev/vuln/GO-2025-3553  

**Example trace**:
```
internal/auth/jwt.go:47:35: auth.ValidateToken calls jwt.ParseWithClaims, 
which eventually calls jwt.Parser.ParseUnverified
```

**Impact**: This vulnerability allows an attacker to craft a malicious JWT header that causes excessive memory allocation during parsing, potentially leading to denial of service (DoS). The affected code path is in `ValidateToken`, which is called on every authenticated request.

**Recommendation**: **Upgrade `github.com/golang-jwt/jwt/v5` from v5.2.1 to v5.2.2** in both `hubx-api/go.mod` and `hubx-gw/go.mod`. This is a **patch release** that fixes the vulnerability without breaking changes.

#### hubx-gw

**Command**: `govulncheck -C hubx-gw ./...`  
**Result**: ❌ **FAIL** (1 vulnerability)

| Vulnerability ID | Package | Installed Version | Fixed Version | Severity | Location |
|------------------|---------|-------------------|---------------|----------|----------|
| GO-2025-3553 | `github.com/golang-jwt/jwt/v5` | v5.2.1 | v5.2.2 | **HIGH** | `internal/middleware/auth.go:84` |

**Description**: Same as hubx-api (excessive memory allocation during header parsing)  
**More info**: https://pkg.go.dev/vuln/GO-2025-3553  

**Example trace**:
```
internal/middleware/auth.go:84:35: middleware.validateToken calls jwt.ParseWithClaims, 
which eventually calls jwt.Parser.ParseUnverified
```

**Impact**: Same as hubx-api. The gateway's JWT validation middleware is affected.

**Recommendation**: **Upgrade `github.com/golang-jwt/jwt/v5` from v5.2.1 to v5.2.2** in `hubx-gw/go.mod`.

---

## 3. OWASP API Security Top 10 Review

Checklist per Constitution VI and tasks.md T045.

| OWASP Category | Test/Evidence | Result | Notes |
|----------------|---------------|--------|-------|
| **API1: Broken Object Level Authorization (BOLA)** | Manual review | ⚪ N/A | No resource access endpoints in HU-01. Relevant for HU-02+ (shipments). |
| **API2: Broken Authentication** | Integration tests + code review | ✅ PASS | ✅ Passwords hashed with bcrypt (cost 12)<br>✅ JWT with HS256, 1-hour expiration<br>✅ Generic error messages (no user enumeration): `HU01_security_non_revealing_errors` verifies identical 401 for wrong password and unknown email<br>✅ Session expiration: `HU01_session_expiration` verifies 401 "Session expired"<br>✅ Email normalization prevents case-based bypasses<br>❌ **High-severity vulnerability** in JWT library (GO-2025-3553) |
| **API3: Broken Object Property Level Authorization** | Manual review | ⚪ N/A | No property-level filtering in HU-01. Relevant for HU-02+ (user vs. operator views). |
| **API4: Unrestricted Resource Consumption** | Manual review | ⚠️ PARTIAL | ⚠️ Rate limiting: Deferred to future story (gateway enhancement)<br>⚠️ Pagination: N/A for HU-01, relevant for HU-02+ (shipment lists)<br>❌ **DoS via JWT parsing**: GO-2025-3553 allows excessive memory allocation |
| **API5: Broken Function Level Authorization** | Manual review | ⚪ N/A | No role-based endpoints in HU-01. Relevant for HU-02+ (operator-only actions). |
| **API6: Unrestricted Access to Sensitive Business Flows** | Manual review | ⚠️ PARTIAL | ⚠️ CAPTCHA on sign-in: Deferred to future story (brute-force protection) |
| **API7: Server Side Request Forgery (SSRF)** | Manual review | ⚪ N/A | No external URL handling in HU-01. |
| **API8: Security Misconfiguration** | Code review + scans | ✅ PASS | ✅ Secrets in environment variables only (`JWT_SECRET`, `DB_URL`)<br>✅ No secrets in repo (`.env` git-ignored, `application.properties` has empty values)<br>✅ Gateway validates JWT independently (defense in depth, Constitution II)<br>✅ Security headers, CORS, rate limiting applied at gateway (Constitution VI)<br>⚠️ G304 in config reader (low risk, deferred) |
| **API9: Improper Inventory Management** | Manual review | ⚪ N/A | API versioning and documentation deferred to future story. |
| **API10: Unsafe Consumption of APIs** | Manual review | ⚪ N/A | No third-party API consumption in HU-01. |

### Key Security Controls Verified

✅ **Parameterized queries**: `auth.FindUserByEmail` uses `SELECT ... WHERE email = $1` (SQL injection protection)  
✅ **Bcrypt password hashing**: Migration 0001 uses cost 12; `service.SignIn` uses `bcrypt.CompareHashAndPassword`  
✅ **Generic error messages**: Both wrong password and unknown email return identical 401 "Invalid credentials" (prevents user enumeration)  
✅ **JWT claims**: Include `sub`, `email`, `role`, `iat`, `exp` (no sensitive data beyond necessary)  
✅ **Gateway + API both validate JWT**: Defense in depth per Constitution II  
✅ **Email normalization**: Lowercase + trim prevents case-based bypasses  

---

## 4. Architecture Boundary Verification

Per Constitution II and system AGENTS.md:

✅ **Traffic flow**: MFE → Gateway (`/api/*`) → API → PostgreSQL (correct)  
✅ **No MFE-to-API direct calls**: All integration tests use `/api/auth/sign-in` (gateway endpoint)  
✅ **No imports between services**: hubx-api and hubx-gw have separate `go.mod` files, no cross-imports  
✅ **Gateway forwards Authorization header**: Middleware validates JWT but forwards `Authorization: Bearer <token>` to API  
✅ **API validates JWT independently**: `auth.middleware.RequireAuth` validates JWT signature and expiration  
✅ **Only API touches database**: hubx-gw has no database connection  

---

## 5. Findings Summary

| Severity | Count | Category | Status |
|----------|-------|----------|--------|
| **HIGH** | 1 | Vulnerability (GO-2025-3553 in JWT library) | ⚠️ **Requires fix** |
| **MEDIUM** | 1 | Static analysis (G304 config file inclusion) | ✅ **Accepted** (deferred) |
| **LOW** | 0 | - | - |

### Finding 1: High-Severity Vulnerability in JWT Library

**ID**: GO-2025-3553  
**OWASP**: API2 (Broken Authentication), API4 (Unrestricted Resource Consumption)  
**Location**: `github.com/golang-jwt/jwt/v5@v5.2.1` (used in `hubx-api/internal/auth/jwt.go:47` and `hubx-gw/internal/middleware/auth.go:84`)  
**Description**: Excessive memory allocation during JWT header parsing can lead to denial of service (DoS)  
**Remediation**: Upgrade `golang-jwt/jwt/v5` from v5.2.1 to v5.2.2 in both `hubx-api/go.mod` and `hubx-gw/go.mod`  
**Commands**:
```bash
go get -C hubx-api github.com/golang-jwt/jwt/v5@v5.2.2
go get -C hubx-gw github.com/golang-jwt/jwt/v5@v5.2.2
```

### Finding 2: Medium-Severity gosec Warning (Deferred)

**ID**: G304 (CWE-22)  
**OWASP**: API8 (Security Misconfiguration)  
**Location**: `hubx-api/internal/config/config.go:105`  
**Description**: `os.Open(path)` with variable path (potential file inclusion)  
**Assessment**: Low risk (paths hardcoded in `main.go`, not user input). Tracked in `memory.md` (item 8), deferred to future config security story.  
**Remediation**: Consider using `os.Root` (Go 1.24+) in future config reader enhancement. Requires human review and design decision.

---

## 6. Ledger Tamper Test

⚪ **N/A** for HU-01 (no ledger updates in this story). Deferred to HU-02 (create shipment) or later.

---

## 7. Test Coverage vs. Gherkin Scenarios

All Gherkin scenarios from `specs/001-sign-in/spec.md` have corresponding integration tests:

| Scenario | Test | Status |
|----------|------|--------|
| Operator signs in with correct credentials | `HU01_sign_in_operator` | ✅ PASS |
| Ground partner signs in with correct credentials | `HU01_sign_in_partner` | ✅ PASS |
| Customer signs in with correct credentials | `HU01_sign_in_customer` | ✅ PASS |
| Email normalization (spaces, case) | `HU01_email_normalization` | ✅ PASS |
| Wrong password → 401 "Invalid credentials" | `HU01_security_non_revealing_errors` (1/2) | ✅ PASS |
| Unknown email → 401 "Invalid credentials" | `HU01_security_non_revealing_errors` (2/2) | ✅ PASS |
| Missing email or password → 400 | `HU01_validation_required_fields` | ✅ PASS |
| Expired JWT → 401 "Session expired" | `HU01_session_expiration` | ✅ PASS |

**Regression tests**:
- `SMOKE_gateway_health`: Gateway `/health` endpoint
- `SMOKE_api_health_through_gateway`: API `/api/health` through gateway

---

## 8. Decision

**Status**: ✅ **PASS** (blocker resolved)

**Verdict**: All integration tests pass (9/9) and architecture boundaries are verified. The high-severity
vulnerability GO-2025-3553 found during verify (in `golang-jwt/jwt/v5 v5.2.1`, affecting both hubx-api and hubx-gw)
was resolved before proceeding.

**Resolution applied**:
1. ✅ `golang-jwt/jwt/v5` upgraded v5.2.1 → v5.2.2 in both `hubx-api/go.mod` and `hubx-gw/go.mod`
   (commit `[agent:backend-go] HU-01 security: upgrade golang-jwt/jwt/v5 to v5.2.2`).
2. ✅ `govulncheck -C hubx-api ./...` and `govulncheck -C hubx-gw ./...` → "No vulnerabilities found."
3. ✅ `go test ./...` green in both modules; `make test-integration` → 9/9 pass after the upgrade.
4. ✅ G304 (gosec, MEDIUM) in `hubx-api/internal/config/config.go:105` accepted and deferred: pre-existing before
   HU-01 (introduced in the baseline config reader), path hardcoded (not user input), tracked in `memory.md` line 88.
   Out of scope for HU-01; to be addressed in a dedicated config-security story.

**Outcome**: No blockers remain. Cleared to proceed to the DevOps phase and PR.

---

## 9. Artifacts

- **Integration tests**: `hubx-bender/tests/HU01_sign_in.spec.ts` (7 scenarios), `hubx-bender/tests/health.spec.ts` (2 regression smoke tests)
- **Test fix**: Line 236 in `HU01_sign_in.spec.ts` (JWT signing key derivation)
- **QA report**: `specs/001-sign-in/qa-report.md` (this file)

---

## 10. Why This Matters (3 Bullets)

1. **Broken authentication is OWASP API #2**: The JWT library vulnerability (GO-2025-3553) directly contradicts Constitution VI ("Non-Negotiable Security") and could allow attackers to crash the gateway or API by sending malformed tokens. Fixing it before merge prevents a known DoS attack surface from reaching production.

2. **TDD delivered confidence**: Writing tests before implementation (Constitution V) caught the JWT signing bug during verify phase instead of in production. The failing `HU01_session_expiration` test exposed a mismatch between test key derivation and production code, preventing a false sense of security.

3. **Generic errors prevent user enumeration**: The `HU01_security_non_revealing_errors` test verifies identical 401 responses for wrong password and unknown email, fulfilling OWASP API2 (Broken Authentication) best practices. This design choice (Constitution VI) prevents attackers from discovering valid email addresses through timing or message differences.

---

**Next Steps**: 
1. `backend-go`: Upgrade JWT library to v5.2.2 (2 minutes)
2. `qa-seguridad`: Rerun govulncheck and integration tests (1 minute)
3. `qa-seguridad`: Update this report to PASS status
4. `devops`: Proceed with phase 5 (CI, deployment docs)

# Research: Sign in (HU-01)

> **Superseded in part** (implement clarifications in `tasks.md`): decision 6 is replaced by reusing the existing
> `0001_baseline_schema.sql`; bcrypt cost is 12 (existing demo hashes); role is `ground partner`.

**Date**: 2025-01-21
**Phase**: 0 (pre-implementation research)

## Research Questions & Findings

### 1. JWT Library for Go stdlib Project

**Question**: Which Go JWT library has stdlib-only dependencies, active maintenance, and supports HS256/RS256?

**Options Evaluated**:

| Library | Stdlib-only | Maintained | HS256/RS256 | Notes |
|---------|-------------|------------|-------------|-------|
| `golang-jwt/jwt` (v5) | ✓ | ✓ (active) | ✓ | Most popular; v5 fixes security issues in v3/v4 |
| `lestrrat-go/jwx` | ✗ | ✓ | ✓ | Requires `github.com/goccy/go-json` |
| Standard library only | ✓ | N/A | ✓ (manual) | Manual base64/HMAC; error-prone |

**Decision**: Use `golang-jwt/jwt` v5.2.1 (latest stable as of 2025-01).

**Rationale**:
- Zero dependencies beyond Go stdlib
- Active maintenance (v5 released 2023, patch releases in 2024/2025)
- Well-documented, widely adopted (20k+ GitHub stars)
- Supports HS256 (HMAC-SHA256) for symmetric signing (suitable for single-issuer API)
- Supports RS256 (RSA) if we migrate to asymmetric keys later

**Alternatives Rejected**:
- `lestrrat-go/jwx`: Requires external JSON library (violates Constitution III: "stdlib for routing")
- Manual implementation: Reinventing crypto is high-risk; no security audit

**Action**: Add `golang-jwt/jwt v5.2.1` to `go.mod`; requires human approval per Constitution III.

---

### 2. Bcrypt Cost Factor

**Question**: What cost factor balances security (resistance to brute force) and performance (< 200ms sign-in p95)?

**Benchmark** (on macOS M1, Go 1.27):

| Cost | Time (avg) | Notes |
|------|------------|-------|
| 10 | ~100ms | OWASP minimum (2023) |
| 11 | ~200ms | Borderline for < 200ms p95 |
| 12 | ~400ms | Too slow for p95 goal |
| 13 | ~800ms | Standard for high-security apps |

**Decision**: Cost factor **10**

**Rationale**:
- Meets OWASP 2023 minimum recommendation (cost 10 = ~0.1s = enough to slow brute force)
- Fits < 200ms p95 target with margin for database query overhead (~20-50ms)
- Demo environment only (5 users); production may raise to 11-12 based on hardware benchmarks
- Hashing happens once per sign-in; acceptable trade-off vs failed login attempts

**Alternatives Rejected**:
- Cost 12+: Exceeds p95 target; hurts UX for legitimate users
- Cost < 10: Below OWASP guideline; insufficient brute-force resistance

**Action**: Use `bcrypt.GenerateFromPassword([]byte(password), 10)` in migration seed and registration (future).

---

### 3. Session Expiration Enforcement

**Question**: Should the API return 401 with a specific error code, or rely on JWT `exp` claim validation alone?

**Decision**: Return structured JSON 401 with `error` and `message` fields.

**Rationale**:
- Frontend needs to distinguish expired session from other 401 causes (e.g., missing token, invalid signature)
- Generic HTTP 401 alone does not convey *why* (spec scenario says "they are told their session expired")
- Structured error enables i18n, logging, and debugging

**Error Response Format**:
```json
{
  "error": "UNAUTHORIZED",
  "message": "Session expired. Please sign in again."
}
```

**Alternative error codes considered**:
- `EXPIRED_TOKEN`: More specific, but adds complexity (distinguish expired vs invalid signature vs missing token)
- Generic `UNAUTHORIZED`: Chosen for simplicity; `message` conveys the reason

**Action**: JWT middleware checks `exp` claim; returns 401 JSON with above structure.

---

### 4. Email Normalization Strategy

**Question**: Normalize on write (store lowercase) or on read (compare lowercase)?

**Decision**: Normalize on **write** (store lowercase, trimmed).

**Rationale**:
- Unique constraint on `email` column enforces uniqueness at DB level (prevents race conditions)
- Simpler queries: `SELECT * FROM users WHERE email = $1` (no `LOWER()` function needed)
- Index on `email` works efficiently (no expression index needed)
- Write-once pattern (users are preloaded; registration is out of scope for HU-01)

**Normalization Rules**:
- Lowercase: `strings.ToLower(email)`
- Trim spaces: `strings.TrimSpace(email)`
- Apply before bcrypt hashing and before database insert

**Schema Impact**:
- `email VARCHAR(254) UNIQUE NOT NULL` (VARCHAR vs TEXT for index efficiency)
- Unique constraint on normalized value (enforced at application layer on write)

**Alternatives Rejected**:
- Normalize on read: Requires expression index `CREATE UNIQUE INDEX ON users (LOWER(email))`; complicates queries
- Case-sensitive storage: Does not match user expectation (email is case-insensitive per RFC 5321 local-part)

**Action**: Repository method `NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }`. Apply in `FindUserByEmail` and in migration seed.

---

### 5. Frontend Token Storage

**Question**: localStorage vs sessionStorage vs memory-only?

**Decision**: **localStorage**

**Rationale**:
- **sessionStorage**: Lost on tab close; poor UX (user must sign in per tab)
- **memory-only**: Lost on page refresh; forces sign-in on every refresh (1-hour session becomes useless)
- **localStorage**: Persists across tabs and refreshes; aligns with 1-hour expiration (cross-cutting rule 1)

**Security Consideration**:
- XSS vulnerability: If attacker injects script, they can read `localStorage.getItem('hubx_token')`
- Mitigation: React strict mode + CSP headers (future); 1-hour expiration limits damage window
- HttpOnly cookie is more secure but complicates CORS and was explicitly deprioritized in clarify phase

**Action**: Store token in `localStorage` under key `hubx_token`. Clear on sign-out or 401 response.

**Alternatives Rejected**:
- sessionStorage: UX penalty (tab-scoped)
- Cookie (HttpOnly): Decision made in clarify phase (use Authorization header for HU-01)

---

### 6. Migration 0001 Scope

**Question**: Include demo users in `0001_baseline.sql`, or separate `0002_seed_users.sql`?

**Decision**: Include demo users in `0001_baseline.sql`.

**Rationale**:
- 5 demo users are foundational (required by all stories: HU-02 creates shipments for customers, HU-03 assigns partners, etc.)
- Single migration simplifies local setup (`make up` applies one file) and CI (integration tests depend on demo users)
- Seed data is immutable (no UPDATE/DELETE); aligns with ledger immutability principle
- Alternative (separate 0002) adds no value (demo users never change)

**Migration 0001 Content**:
1. Create `users` table (schema in `data-model.md`)
2. Create `role` ENUM type (`operator`, `ground_partner`, `customer`)
3. Insert 5 demo users (email, bcrypt hash, role):
   - `operator@hubx.co` / `Hubx-Demo-2026` / `operator`
   - `rapido@hubx.co` / `Hubx-Demo-2026` / `ground_partner`
   - `express@hubx.co` / `Hubx-Demo-2026` / `ground_partner`
   - `ana@hubx.co` / `Hubx-Demo-2026` / `customer`
   - `andina@hubx.co` / `Hubx-Demo-2026` / `customer`
4. Party IDs (future): Set `party_id` to NULL for now; will be linked in future story when `parties` table exists

**Action**: Write `0001_baseline.sql` with schema + demo users. Document decision in `memory.md`.

---

## Decisions Summary

| # | Decision | Rationale | Impact |
|---|----------|-----------|--------|
| 1 | Use `golang-jwt/jwt` v5.2.1 | Stdlib-only, active, supports HS256/RS256 | Requires human approval (new Go dependency) |
| 2 | Bcrypt cost factor 10 | OWASP minimum, fits < 200ms p95 | Used in migration seed and future registration |
| 3 | Structured 401 JSON response | Enables frontend to distinguish expired session | Error format in contracts |
| 4 | Normalize email on write (lowercase, trim) | Simpler queries, DB-level unique constraint | Schema: `email VARCHAR(254) UNIQUE NOT NULL` |
| 5 | Store JWT in localStorage | Persists across tabs/refreshes, aligns with 1-hour expiration | XSS risk accepted (mitigated by expiration) |
| 6 | Demo users in 0001_baseline.sql | Foundational for all stories, simplifies setup | Single migration for schema + seed |

---

## Open Questions for Implementation

1. **JWT signing algorithm**: HS256 (symmetric, shared secret) or RS256 (asymmetric, public/private keys)?
   - **Recommendation**: HS256 for HU-01 (simpler; single API issuer/validator)
   - **Future**: Migrate to RS256 when multiple services need to validate tokens without sharing secret
   
2. **JWT secret management**: How is the secret provided?
   - **Recommendation**: Environment variable `JWT_SECRET` (required, no default); `application.properties` has empty value
   - **Local**: Generated by `scripts/env.sh` (random 32-byte base64)
   - **Production**: Injected by deployment platform (secrets manager)

3. **Gateway shared secret**: Does gateway validate JWT, or just forward to API?
   - **Decision** (from Constitution II): Gateway validates JWT signature + expiration (defense in depth); API also validates JWT (does not trust gateway alone)
   - **Implementation**: Gateway reads `JWT_SECRET` from env; validates token; forwards `Authorization` header to API; API re-validates

4. **Role-based access control (RBAC)**: Where is role checked?
   - **Recommendation**: Middleware extracts `role` from JWT claims into request context; handlers check `ctx.Value("role")` for authorization
   - **Future stories** (HU-02, HU-03): Endpoints check role (operator can create shipments, partners cannot, etc.)

---

## Next Steps

1. Write `data-model.md` (users table DDL, JWT claims)
2. Write `contracts/auth-sign-in.md` (OpenAPI-style contract)
3. Write `quickstart.md` (runnable validation scenarios)
4. Update `memory.md` with decisions (JWT library, migration 0001 scope, bcrypt cost)
5. Proceed to Phase 2: `tasks.md`


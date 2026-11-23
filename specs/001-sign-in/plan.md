# Implementation Plan: Sign in

**Branch**: `001-sign-in` | **Date**: 2025-01-21 | **Spec**: [spec.md](./spec.md)

**Source**: backlog HU-01

## Summary

Authenticate HUBX users (operator, ground partner, customer) with email and password, issuing a JWT session token valid for one hour. The system must normalize emails (lowercase, trim), use bcrypt for password validation, return generic error messages to prevent user enumeration, and enforce session expiration on all protected endpoints. Constitution Principle VI mandates bcrypt + JWT; Principle III mandates local authentication against PostgreSQL.

## Technical Context

**Language/Version**: Go 1.27.4, React 18 + TypeScript 5 strict, PostgreSQL 17

**Primary Dependencies**:
- Backend: `github.com/jackc/pgx/v5` (approved), `golang.org/x/crypto/bcrypt`, JWT library (to be chosen in research)
- Frontend: React, TypeScript strict, Vite
- Gateway: Go stdlib only

**Storage**: PostgreSQL (managed free instance); only hubx-api accesses it

**Testing**:
- Unit: `go test` (backend), `npm run test` (frontend)
- Integration: Playwright Test (API mode) in hubx-bender, via gateway

**Target Platform**: Linux containers (Docker Compose locally, cloud deployment)

**Project Type**: Microservices (hubx-api, hubx-gw, hubx-mfe in one git repository)

**Performance Goals**: < 200ms p95 for sign-in; session validation < 10ms p95

**Constraints**: 
- One-hour session expiration (cross-cutting rule 1)
- Generic error messages (Constitution VI, prevent user enumeration)
- No imports between services; all traffic through the gateway

**Scale/Scope**: 5 demo users preloaded; production scale TBD (design for 1000+ users)

## Constitution Check

*Phase 0 (pre-research)*:

| Principle | Requirement | Status |
|-----------|-------------|--------|
| I. Immutable Ledger | N/A for authentication | ✓ Pass |
| II. Three-Tier Architecture | Frontend → Gateway → API; no direct MFE-API calls | ✓ Pass |
| III. Technology Stack | Go stdlib + approved pgx driver; React strict TypeScript; bcrypt + JWT; `application.properties` config | ✓ Pass |
| IV. Modular Design | Auth module under `internal/auth`; constructor injection; slog JSON logging | ✓ Pass |
| V. Test-First | Integration tests in hubx-bender (red first); unit tests in `*_test.go` | ✓ Pass |
| VI. Security | Parameterized queries; bcrypt passwords; JWT sessions; gateway validates JWT + shared secret; roles in JWT | ✓ Pass |
| VII. Language | English code, identifiers, messages; interface language TBD (spec says "functions that correspond to my role" without specifying UI language) | ✓ Pass |

**Gate Result**: PASS. No violations.

*Phase 1 (post-design): Re-check after contracts defined.*

## Project Structure

### Documentation (this feature)

```text
specs/001-sign-in/
├── spec.md              # Completed (clarified)
├── plan.md              # This file
├── research.md          # Phase 0: JWT library choice, password hashing best practices
├── data-model.md        # Phase 1: users table schema, JWT claims structure
├── contracts/           # Phase 1: HTTP contracts (POST /api/auth/sign-in, Authorization header)
│   └── auth-sign-in.md
├── quickstart.md        # Phase 1: validation scenarios
├── tasks.md             # Phase 2: breakdown by agent
└── qa-report.md         # Verify phase: findings and security scan results
```

### Source Code (HUBX repository)

```text
hubx-api/
├── internal/
│   ├── auth/              # NEW: authentication module
│   │   ├── service.go     # SignIn(email, password) → (token, user, error)
│   │   ├── service_test.go
│   │   ├── repository.go  # FindUserByEmail(email) → (*User, error)
│   │   ├── repository_test.go
│   │   ├── jwt.go         # GenerateToken, ValidateToken
│   │   ├── jwt_test.go
│   │   ├── handler.go     # POST /auth/sign-in
│   │   ├── handler_test.go
│   │   ├── middleware.go  # RequireAuth middleware (validates JWT, injects claims)
│   │   ├── middleware_test.go
│   │   └── register.go    # auth.Register(mux, db, config)
│   ├── config/            # Existing (from 000-smoke-config)
│   └── migrate/           # Existing
├── migrations/
│   └── 0001_baseline.sql  # users table, bcrypt password, role enum
├── cmd/
│   └── api/
│       └── main.go        # Wire auth.Register
└── application.properties # jwt.secret.path, jwt.expiration.hours

hubx-gw/
├── internal/
│   └── middleware/        # NEW or MODIFY: JWT validation, shared secret check
└── application.properties # api.shared.secret.path, cors.allowed.origins

hubx-mfe/
├── src/
│   ├── pages/
│   │   └── SignIn.tsx     # NEW: sign-in form
│   ├── services/
│   │   └── authService.ts # NEW: signIn(email, password), getToken, clearToken
│   ├── contexts/
│   │   └── AuthContext.tsx # NEW: user state, role, sign-out
│   └── App.tsx            # Wire SignIn route
└── package.json

hubx-bender/
├── tests/
│   ├── HU01_sign_in_operator.spec.ts
│   ├── HU01_sign_in_partner.spec.ts
│   ├── HU01_sign_in_customer.spec.ts
│   ├── HU01_email_normalization.spec.ts
│   ├── HU01_security_non_revealing_errors.spec.ts
│   ├── HU01_validation_required_fields.spec.ts
│   └── HU01_session_expiration.spec.ts
└── playwright.config.ts

contracts/
└── auth-sign-in.md        # NEW: POST /api/auth/sign-in contract

memory.md                  # UPDATE: record JWT library choice, migration 0001 decision
```

**Structure Decision**: HUBX uses Option 2 (web application) with three separate services in one repository. The auth module is a new subdirectory under `hubx-api/internal/`. Gateway gains JWT validation middleware. Frontend adds auth service, context and sign-in page. Each service already has its own test structure; we extend them with auth tests.

## Complexity Tracking

No constitution violations.

---

## Phase 0: Research

### Research Questions

1. **JWT library for Go stdlib project**: Which Go JWT library has stdlib-only dependencies, active maintenance, and supports HS256/RS256?
2. **Bcrypt cost factor**: What cost factor balances security (resistance to brute force) and performance (< 200ms sign-in p95)?
3. **Session expiration enforcement**: Should the API return 401 with a specific error code, or rely on JWT `exp` claim validation alone?
4. **Email normalization**: Normalize on write (store lowercase) or on read (compare lowercase)?
5. **Frontend token storage**: localStorage vs sessionStorage vs memory-only? Constitution prefers security; cross-cutting rule 1 says one-hour expiration.
6. **Migration 0001 scope**: Include demo users in 0001_baseline.sql, or separate 0002_seed_users.sql?

### Research Tasks

| Task | Owner | Deliverable |
|------|-------|-------------|
| Evaluate Go JWT libraries (`golang-jwt/jwt`, `lestrrat-go/jwx`, others) against stdlib-only constraint | backend-go | Recommendation in research.md |
| Determine bcrypt cost factor (benchmark on target platform) | backend-go | Cost factor + rationale |
| Define 401 error response format for expired/invalid JWT | backend-go, qa-seguridad | Contract in research.md |
| Decide email normalization strategy (write vs read, unique constraint) | backend-go | Decision + schema impact |
| Assess frontend token storage options (security vs UX) | frontend-react | Recommendation |
| Decide migration scope (seed users in 0001 or later) | backend-go | Decision |

---

## Phase 1: Design & Contracts

### Data Model (`data-model.md`)

**Entity: User**

| Field | Type | Constraints | Notes |
|-------|------|-------------|-------|
| `id` | UUID | PK | Generated on insert |
| `email` | VARCHAR(254) | UNIQUE, NOT NULL | Normalized (lowercase, trimmed) on write |
| `password_hash` | TEXT | NOT NULL | bcrypt hash |
| `role` | ENUM('operator', 'ground_partner', 'customer') | NOT NULL | |
| `party_id` | UUID | NULL | FK to parties table (future); NULL for operators |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

**Validation Rules**:
- Email: RFC 5321 format (up to 254 chars), unique after normalization
- Password: Minimum 8 characters (enforced only on user creation; demo users already exist)

**State Transitions**: N/A (users are preloaded; registration is out of scope)

**JWT Claims Structure**:

```json
{
  "sub": "user-uuid",
  "email": "user@hubx.co",
  "role": "operator|ground_partner|customer",
  "party_id": "optional-uuid",
  "iat": 1234567890,
  "exp": 1234571490
}
```

### HTTP Contracts (`contracts/auth-sign-in.md`)

**Endpoint**: `POST /api/auth/sign-in`

**Request**:
```json
{
  "email": "user@hubx.co",
  "password": "Hubx-Demo-2026"
}
```

**Responses**:

- **200 OK** (success):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "user@hubx.co",
    "role": "operator"
  }
}
```

- **400 Bad Request** (missing fields):
```json
{
  "error": "MISSING_REQUIRED_FIELDS",
  "message": "Email and password are required"
}
```

- **401 Unauthorized** (invalid credentials):
```json
{
  "error": "INVALID_CREDENTIALS",
  "message": "Invalid credentials"
}
```

**Protected Endpoints**: All endpoints except `/auth/sign-in`, `/health` require `Authorization: Bearer <token>` header.

**401 Unauthorized** (expired/invalid token):
```json
{
  "error": "UNAUTHORIZED",
  "message": "Session expired. Please sign in again."
}
```

### Gateway Contract

**Incoming**: `Authorization: Bearer <token>` from MFE

**Validation**:
1. Check `X-Gateway-Secret` header from downstream API (shared secret)
2. Extract and validate JWT (signature, expiration)
3. Forward `Authorization` header to API

**Forwarding**: Gateway does NOT strip `Authorization` header; API validates JWT independently (Constitution II: API must validate JWT on its own).

### Quickstart Validation (`quickstart.md`)

**Prerequisites**:
- PostgreSQL running with `hubx` database
- Migration 0001 applied (users table with 5 demo users)
- API, gateway, MFE running (`make up`)

**Validation Scenarios**:

1. **Operator signs in**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/sign-in \
     -H 'Content-Type: application/json' \
     -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}'
   # Expected: 200 OK, token + user object
   ```

2. **Email normalization**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/sign-in \
     -H 'Content-Type: application/json' \
     -d '{"email":" ANA@hubx.co ","password":"Hubx-Demo-2026"}'
   # Expected: 200 OK (matches ana@hubx.co)
   ```

3. **Wrong password**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/sign-in \
     -H 'Content-Type: application/json' \
     -d '{"email":"ana@hubx.co","password":"wrong"}'
   # Expected: 401 Unauthorized, "Invalid credentials"
   ```

4. **Unregistered email**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/sign-in \
     -H 'Content-Type: application/json' \
     -d '{"email":"nobody@hubx.co","password":"Hubx-Demo-2026"}'
   # Expected: 401 Unauthorized, "Invalid credentials" (same message)
   ```

5. **Missing password**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/sign-in \
     -H 'Content-Type: application/json' \
     -d '{"email":"operator@hubx.co"}'
   # Expected: 400 Bad Request, "Email and password are required"
   ```

6. **Session expiration** (requires waiting 1 hour or manipulating JWT `exp`):
   ```bash
   # 1. Sign in, get token
   TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
     -H 'Content-Type: application/json' \
     -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}' \
     | jq -r '.token')
   
   # 2. Use expired token (simulate by advancing system clock or forging JWT with past exp)
   curl -X GET http://localhost:8080/api/shipments/HBX-SEED-0001 \
     -H "Authorization: Bearer $EXPIRED_TOKEN"
   # Expected: 401 Unauthorized, "Session expired. Please sign in again."
   ```

**Expected Outcomes**:
- All scenarios return correct HTTP status and error codes
- Generic error message for wrong email/password (no distinction)
- Email case/spaces ignored
- Integration tests (Phase 2) automate these scenarios

---

## Phase 1: Constitution Re-Check

| Principle | Design Element | Status |
|-----------|----------------|--------|
| I. Immutable Ledger | N/A | ✓ Pass |
| II. Three-Tier Architecture | MFE → GW (`/api/auth/sign-in`) → API (`/auth/sign-in`); API validates JWT independently | ✓ Pass |
| III. Technology Stack | Go stdlib + pgx + bcrypt + JWT library (from research); React strict TS; `application.properties` for non-secret config | ✓ Pass |
| IV. Modular Design | `internal/auth` module; `auth.Register(mux)`; constructor injection for repo/service | ✓ Pass |
| V. Test-First | Integration tests in hubx-bender (Phase 2 red); unit tests for service, repo, JWT, middleware | ✓ Pass |
| VI. Security | Parameterized queries (`$1`); bcrypt cost from research; JWT HS256/RS256; roles in JWT; 401 for expired sessions | ✓ Pass |
| VII. Language | All code, identifiers, messages in English | ✓ Pass |

**Gate Result**: PASS. Design complies with constitution.

---

## Implementation Strategy

**Order** (Constitution V, TDD):
1. **QA (red)**: Write failing integration tests in hubx-bender for all 7 scenarios (HU01_*.spec.ts)
2. **Backend** (green): Implement auth module (service, repo, JWT, handler, middleware); migration 0001; wire into main.go
3. **Gateway** (green): Add JWT validation middleware (or enhance existing middleware to handle Authorization header)
4. **Frontend** (green): Sign-in page, auth service, context; wire into App.tsx
5. **QA (verify)**: Run integration tests (expect green); run gosec + govulncheck; write qa-report.md
6. **DevOps**: Update CI (if needed), ensure `make up` applies migration and starts services, update deploy docs

**Parallelization**: Backend and Frontend can proceed in parallel after QA writes red tests (they share only the HTTP contract, not files).

---

## Decisions

1. **JWT library**: TBD in research (Phase 0)
2. **Bcrypt cost**: TBD in research (target < 200ms, likely 10-12)
3. **Email normalization**: Store lowercase (decided here; write in research.md)
4. **Frontend token storage**: TBD in research (lean toward localStorage given 1-hour expiration mitigates XSS risk)
5. **Migration 0001 scope**: TBD in research (lean toward including demo users to minimize migration count)
6. **Error response format**: JSON with `error` code + `message` (decided here)
7. **Session expiration message**: "Session expired. Please sign in again." (decided here)

---

## Next Steps

1. Execute Phase 0 research (resolve TBDs)
2. Write `research.md` with findings
3. Write `data-model.md` (users table DDL, JWT claims)
4. Write `contracts/auth-sign-in.md` (full OpenAPI-style contract)
5. Write `quickstart.md` (runnable curl commands)
6. Proceed to Phase 2: `tasks.md` (break down by agent: QA, API, GW, MFE, OPS)


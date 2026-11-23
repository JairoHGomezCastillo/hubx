# Tasks: Sign in (HU-01)

**Input**: spec.md, plan.md, research.md, data-model.md, contracts/auth-sign-in.md, quickstart.md

**Branch**: `001-sign-in`

**TDD Order** (Constitution V): QA red → Backend + Frontend → QA verify → DevOps

**Agent Ownership**: `[QA]`, `[API]`, `[GW]`, `[MFE]`, `[OPS]`

---

## Phase 1: QA Red (Integration Tests MUST Fail)

**Agent**: `qa-seguridad`

**Goal**: Write integration tests in hubx-bender that exercise the HTTP contract through the gateway. All tests MUST fail (API not implemented yet).

**Owned Paths**: `hubx-bender/tests/`

- [ ] **T001** [P] [QA] Write `HU01_sign_in_operator.spec.ts`: POST /api/auth/sign-in with operator@hubx.co, expect 200 + JWT with role=operator
- [ ] **T002** [P] [QA] Write `HU01_sign_in_partner.spec.ts`: POST /api/auth/sign-in with rapido@hubx.co, expect 200 + JWT with role=ground_partner
- [ ] **T003** [P] [QA] Write `HU01_sign_in_customer.spec.ts`: POST /api/auth/sign-in with ana@hubx.co, expect 200 + JWT with role=customer
- [ ] **T004** [P] [QA] Write `HU01_email_normalization.spec.ts`: POST /api/auth/sign-in with " ANA@hubx.co " (capitals, spaces), expect 200
- [ ] **T005** [P] [QA] Write `HU01_security_non_revealing_errors.spec.ts`: POST /api/auth/sign-in with wrong password AND with unknown email, expect identical 401 "Invalid credentials"
- [ ] **T006** [P] [QA] Write `HU01_validation_required_fields.spec.ts`: POST /api/auth/sign-in with missing password, expect 400 "Email and password are required"
- [ ] **T007** [P] [QA] Write `HU01_session_expiration.spec.ts`: Forge expired JWT (exp in past), GET /api/shipments/HBX-SEED-0001, expect 401 "Session expired. Please sign in again."

**Evidence**: All tests run and **fail** (API returns 404 or 500). Commit test files; report failure count.

**Checkpoint**: Red tests committed. Ready for parallel backend + frontend implementation.

---

## Phase 2: Backend Implementation (Green)

**Agents**: `backend-go` (API module + migration), `backend-go` (GW middleware)

**Goal**: Implement auth module in hubx-api, migration 0001, JWT middleware in hubx-gw. Integration tests turn green.

**Parallelization**: API and GW tasks can partially overlap (GW waits for JWT library choice from T010).

---

### Phase 2A: API (Auth Module)

**Agent**: `backend-go`

**Owned Paths**: `hubx-api/internal/auth/`, `hubx-api/migrations/`, `hubx-api/cmd/api/main.go`, `hubx-api/application.properties`, `hubx-api/go.mod`

- [ ] **T008** [API] Add `golang-jwt/jwt` v5.2.1 to `go.mod`: `go get -C hubx-api github.com/golang-jwt/jwt/v5@v5.2.1` (requires human approval per Constitution III)

#### Models & Repository

- [ ] **T009** [P] [API] Create `hubx-api/internal/auth/user.go`: User struct (id, email, passwordHash, role, partyID, createdAt, updatedAt)
- [ ] **T010** [P] [API] Create `hubx-api/internal/auth/repository.go`: Interface + PostgreSQL impl
  - `FindUserByEmail(ctx, email) (*User, error)` (parameterized query: `SELECT * FROM users WHERE email = $1`)
  - `NormalizeEmail(email) string` helper (lowercase, trim)
- [ ] **T011** [API] Write `hubx-api/internal/auth/repository_test.go`: Unit tests for FindUserByEmail (mock DB or use testcontainers)

#### JWT Logic

- [ ] **T012** [P] [API] Create `hubx-api/internal/auth/jwt.go`: JWT functions
  - `GenerateToken(user *User, secret string, expirationHours int) (string, error)` (HS256, claims: sub, email, role, party_id, iat, exp)
  - `ValidateToken(tokenString, secret string) (*Claims, error)` (verify signature, check exp)
  - `Claims struct` (sub, email, role, partyID, iat, exp)
- [ ] **T013** [API] Write `hubx-api/internal/auth/jwt_test.go`: Unit tests for GenerateToken and ValidateToken (valid token, expired token, invalid signature)

#### Service

- [ ] **T014** [API] Create `hubx-api/internal/auth/service.go`: SignIn(ctx, email, password) (token, user, error)
  - Normalize email
  - FindUserByEmail
  - bcrypt.CompareHashAndPassword (cost 10)
  - Return generic "Invalid credentials" for both user-not-found and wrong-password
  - GenerateToken on success
- [ ] **T015** [API] Write `hubx-api/internal/auth/service_test.go`: Unit tests for SignIn (success, wrong password, unknown email, email normalization)

#### HTTP Handler

- [ ] **T016** [API] Create `hubx-api/internal/auth/handler.go`: POST /auth/sign-in handler
  - Parse JSON request (email, password)
  - Validate required fields (400 if missing)
  - Call service.SignIn
  - Return 200 JSON (token, user) or 401 JSON (error, message)
- [ ] **T017** [API] Write `hubx-api/internal/auth/handler_test.go`: Unit tests for handler (httptest.NewRecorder, mock service)

#### Middleware

- [ ] **T018** [API] Create `hubx-api/internal/auth/middleware.go`: RequireAuth middleware
  - Extract Authorization header ("Bearer <token>")
  - ValidateToken
  - Return 401 JSON if missing/invalid/expired (message: "Session expired. Please sign in again." if exp < now)
  - Inject claims (sub, email, role) into request context
- [ ] **T019** [API] Write `hubx-api/internal/auth/middleware_test.go`: Unit tests for RequireAuth (valid token, missing token, expired token)

#### Registration

- [ ] **T020** [API] Create `hubx-api/internal/auth/register.go`: `Register(mux *http.ServeMux, db *sql.DB, config Config)` function
  - Register POST /auth/sign-in handler
  - Return RequireAuth middleware for use in other modules
- [ ] **T021** [API] Update `hubx-api/cmd/api/main.go`: Wire auth.Register, apply RequireAuth to future protected routes (placeholder for HU-02+)

#### Configuration

- [ ] **T022** [API] Update `hubx-api/application.properties`: Add jwt.expiration.hours=1, jwt.secret.path= (empty, must come from env)
- [ ] **T023** [API] Update `hubx-api/internal/config/config.go`: Read jwt.expiration.hours (default 1), JWT_SECRET env var (required, no default)

#### Migration

- [ ] **T024** [API] Create `hubx-api/migrations/0001_baseline.sql`:
  - CREATE TYPE role AS ENUM ('operator', 'ground_partner', 'customer');
  - CREATE TABLE users (id UUID PK, email VARCHAR(254) UNIQUE NOT NULL, password_hash TEXT NOT NULL, role role NOT NULL, party_id UUID NULL, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ);
  - CREATE INDEX idx_users_email, idx_users_role;
  - INSERT 5 demo users (email, bcrypt hash of 'Hubx-Demo-2026' cost 10, role)
  - Run in one transaction (Constitution: immutability)
- [ ] **T025** [API] Generate bcrypt hashes for demo users:
  ```bash
  # From repo root or use Go script
  for user in operator@hubx.co rapido@hubx.co express@hubx.co ana@hubx.co andina@hubx.co; do
    echo "$user: $(echo -n 'Hubx-Demo-2026' | bcrypt -c 10)"
  done
  ```
- [ ] **T026** [API] Verify migration applies cleanly: `make migrate` (or `make up` if migration runs on startup)

**Evidence**: `go test ./internal/auth/...` passes; `make up` applies migration; curl POST /api/auth/sign-in returns 200 or 401 (not 404/500).

---

### Phase 2B: Gateway (JWT Validation Middleware)

**Agent**: `backend-go`

**Owned Paths**: `hubx-gw/internal/middleware/`, `hubx-gw/cmd/gw/main.go`, `hubx-gw/application.properties`, `hubx-gw/go.mod`

- [ ] **T027** [GW] Add `golang-jwt/jwt` v5.2.1 to `hubx-gw/go.mod`: `go get -C hubx-gw github.com/golang-jwt/jwt/v5@v5.2.1`

#### JWT Validation Middleware

- [ ] **T028** [GW] Create `hubx-gw/internal/middleware/auth.go`: JWTValidation middleware (or enhance existing middleware)
  - Extract Authorization header ("Bearer <token>")
  - Validate JWT (signature, exp) using JWT_SECRET env var
  - Return 401 JSON if invalid/expired (message: "Session expired. Please sign in again." if exp < now)
  - Forward Authorization header to API (Constitution II: API validates independently)
- [ ] **T029** [GW] Write `hubx-gw/internal/middleware/auth_test.go`: Unit tests for JWTValidation (valid token, expired token, missing token)

#### Registration

- [ ] **T030** [GW] Update `hubx-gw/cmd/gw/main.go`: Apply JWTValidation middleware to all routes except /health and /api/auth/sign-in
- [ ] **T031** [GW] Update `hubx-gw/application.properties`: Read JWT_SECRET env var (same value as API)

**Evidence**: `go test ./internal/middleware/...` passes; gateway returns 401 for invalid JWT before forwarding to API.

---

## Phase 3: Frontend Implementation (Green)

**Agent**: `frontend-react`

**Goal**: Implement sign-in page, auth service, context. User can sign in via UI; token stored in localStorage.

**Owned Paths**: `hubx-mfe/src/`

**Parallelization**: Can run in parallel with Phase 2 (backend); only depends on HTTP contract, not implementation.

#### Auth Service

- [ ] **T032** [P] [MFE] Create `hubx-mfe/src/services/authService.ts`:
  - `signIn(email: string, password: string): Promise<{ token: string; user: { email: string; role: string } }>` (POST /api/auth/sign-in)
  - `getToken(): string | null` (read from localStorage key `hubx_token`)
  - `setToken(token: string): void` (write to localStorage)
  - `clearToken(): void` (remove from localStorage)
  - `getAuthHeaders(): { Authorization: string } | {}` (return Bearer header if token exists)

#### Auth Context

- [ ] **T033** [MFE] Create `hubx-mfe/src/contexts/AuthContext.tsx`:
  - `AuthProvider` component (manages user state: email, role, isAuthenticated)
  - `useAuth()` hook (returns user, isAuthenticated, signIn, signOut)
  - signIn: call authService.signIn, setToken, update user state
  - signOut: clearToken, reset user state
  - Load token from localStorage on mount (validate or discard if expired)

#### Sign-In Page

- [ ] **T034** [MFE] Create `hubx-mfe/src/pages/SignIn.tsx`:
  - Form: email input, password input, submit button
  - Call `useAuth().signIn(email, password)` on submit
  - Display error message on 400/401 (from authService)
  - Redirect to dashboard on success (future route)
  - Email normalization happens on backend (no client-side normalization needed, but trim for UX)

#### Routing

- [ ] **T035** [MFE] Update `hubx-mfe/src/App.tsx`:
  - Wrap app in `<AuthProvider>`
  - Add `/sign-in` route → `<SignIn />`
  - Add protected route guard (redirect to /sign-in if !isAuthenticated)
  - Add placeholder dashboard route (future HU-02+)

#### HTTP Client

- [ ] **T036** [MFE] Create or update `hubx-mfe/src/services/httpClient.ts`:
  - Axios/fetch wrapper that includes `Authorization: Bearer <token>` header (from authService.getAuthHeaders)
  - Intercept 401 responses → clearToken, redirect to /sign-in
  - Handle "Session expired" message (show notification, then redirect)

#### Unit Tests (Optional for HU-01)

- [ ] **T037** [P] [MFE] Write `hubx-mfe/src/services/authService.test.ts`: Unit tests for signIn (mock fetch), getToken, setToken, clearToken
- [ ] **T038** [P] [MFE] Write `hubx-mfe/src/contexts/AuthContext.test.tsx`: Unit tests for AuthProvider, useAuth hook (React Testing Library)
- [ ] **T039** [P] [MFE] Write `hubx-mfe/src/pages/SignIn.test.tsx`: Unit tests for SignIn form (submit, error display)

**Evidence**: `npm --prefix hubx-mfe run test` passes (if tests written); manual test: navigate to localhost:5173/sign-in, sign in as operator@hubx.co, token appears in localStorage.

---

## Phase 4: QA Verify (Integration Tests MUST Pass)

**Agent**: `qa-seguridad`

**Goal**: Run integration tests (now green), gosec, govulncheck. Report findings.

**Owned Paths**: `hubx-bender/`, `specs/001-sign-in/qa-report.md`

- [ ] **T040** [QA] Run integration tests: `make test-integration` (all HU01_*.spec.ts must pass)
- [ ] **T041** [QA] Run gosec on API: `cd hubx-api && gosec ./...` (expect no critical/high findings)
- [ ] **T042** [QA] Run gosec on gateway: `cd hubx-gw && gosec ./...` (expect no critical/high findings)
- [ ] **T043** [QA] Run govulncheck on API: `govulncheck -C hubx-api ./...` (expect no vulnerabilities)
- [ ] **T044** [QA] Run govulncheck on gateway: `govulncheck -C hubx-gw ./...` (expect no vulnerabilities)
- [ ] **T045** [QA] Write `specs/001-sign-in/qa-report.md`:
  - Integration test results (7 tests, all pass)
  - Security scan results (gosec, govulncheck)
  - Findings (if any): severity, file, line, recommendation
  - OWASP API Security Top 10 checklist (from Constitution VI):
    - API1: Broken Object Level Authorization (N/A for HU-01, relevant for HU-04+)
    - API2: Broken Authentication (covered: bcrypt, JWT, generic errors, session expiration)
    - API3: Broken Object Property Level Authorization (N/A for HU-01)
    - API4: Unrestricted Resource Consumption (future: rate limiting)
    - API5: Broken Function Level Authorization (N/A for HU-01, relevant for HU-02+)
    - API6: Unrestricted Access to Sensitive Business Flows (future: CAPTCHA on sign-in)
    - API7: Server Side Request Forgery (N/A)
    - API8: Security Misconfiguration (checked: secrets in env, not in repo)
    - API9: Improper Inventory Management (N/A)
    - API10: Unsafe Consumption of APIs (N/A)
  - Decision: OK to proceed, or blockers (critical/high findings require fix)

**Evidence**: qa-report.md committed; all tests green; no critical/high findings (or findings documented and accepted by human).

**Checkpoint**: If findings exist, send qa-report.md back to `backend-go` for fix (Phase 2 tasks repeat); else proceed to DevOps.

---

## Phase 5: DevOps (CI, Deployment, Docs)

**Agent**: `devops`

**Goal**: Ensure local stack works (`make up` applies migration, services start), CI runs tests, deployment docs updated.

**Owned Paths**: `Makefile`, `compose.yaml`, `Dockerfile` (if changes needed), `.github/workflows/`, `scripts/`, `README.md`, `docs/`

- [ ] **T046** [OPS] Verify `make up` applies migration 0001 and starts services (health checks pass)
- [ ] **T047** [OPS] Verify `make test-integration` runs hubx-bender tests (no manual psql setup needed)
- [ ] **T048** [OPS] Update `scripts/env.sh`: Generate `JWT_SECRET` (32-byte random, base64-encoded) if missing
- [ ] **T049** [OPS] Update `compose.yaml` (if needed): Inject JWT_SECRET env var into api and gw services
- [ ] **T050** [OPS] Update `.github/workflows/ci.yml` (if exists): Add step to run `make test-integration` (after `make test`)
- [ ] **T051** [OPS] Update `README.md`: Document sign-in feature, demo users, JWT token usage
- [ ] **T052** [OPS] Update `docs/setup.md`: Add JWT_SECRET to required env vars, mention migration 0001

**Evidence**: `make up && make test-integration` succeeds on clean checkout; CI workflow green (if exists); README updated.

---

## Dependencies & Execution Order

### Phase Dependencies

1. **Phase 1 (QA Red)**: No dependencies, start immediately
2. **Phase 2 (Backend Green)**: Depends on Phase 1 completion (tests exist)
   - **Phase 2A (API)** and **Phase 2B (GW)** can partially overlap (GW waits for JWT library choice from T008)
3. **Phase 3 (Frontend Green)**: Can run **in parallel with Phase 2** (only depends on HTTP contract, not implementation)
4. **Phase 4 (QA Verify)**: Depends on Phase 2 AND Phase 3 completion (backend + frontend implemented)
5. **Phase 5 (DevOps)**: Depends on Phase 4 completion (tests green, no blockers)

### Within Each Phase

- **Phase 1**: All 7 test tasks [P] can run in parallel (different files)
- **Phase 2A**: T009-T010 [P] can run in parallel (different files); T011 depends on T010; T012 [P] can run in parallel with T009-T011; T014 depends on T010 + T012; T016 depends on T014; etc.
- **Phase 2B**: Depends on T008 (JWT library added to both services)
- **Phase 3**: T032 [P] can run before or in parallel with T033; T034 depends on T033; T035 depends on T034; T036 can run in parallel with T032-T035
- **Phase 4**: T040-T044 can run in parallel (different scans); T045 consolidates results
- **Phase 5**: T046-T052 mostly sequential (verify before updating docs)

### Parallel Opportunities (with 2+ team members)

- **Phase 1 → Phase 2 & 3 (parallel)**:
  - Developer A (backend-go): Phase 2A + 2B (API + GW)
  - Developer B (frontend-react): Phase 3 (MFE)
  - Both can proceed after Phase 1 completes
- **Phase 2A tasks**:
  - T009, T010, T012 [P] can be split among multiple developers (different files)
- **Phase 4 tasks**:
  - T041-T044 [P] (different scans) can run concurrently

---

## Checkpoints

| Checkpoint | Validation | Proceed If |
|------------|------------|------------|
| After Phase 1 | Run `npm --prefix hubx-bender run test` | All 7 tests fail (expected) |
| After Phase 2 | Run `make test-integration` | All 7 tests pass |
| After Phase 3 | Manual UI test | Sign in succeeds, token in localStorage |
| After Phase 4 | Read `qa-report.md` | No critical/high findings, or findings accepted |
| After Phase 5 | Run `make up && make test-integration` on clean checkout | All tests pass |

---

## Task Completion Criteria

Each task is **DONE** when:
- Code written (following Constitution IV: modular design, constructor injection, slog logging)
- Unit tests pass (if unit tests exist for that task)
- Code committed with message `[agent:backend-go] HU-01 <task description>` (or appropriate agent)
- Hubx-lead marks task as `- [x]` after validating subagent evidence

---

## Notes

- **[P]** = Parallelizable (different files, no dependencies within that group)
- **[Agent]** = Owner agent (QA, API, GW, MFE, OPS)
- **Constitution compliance**: All tasks must respect Constitution principles (stdlib, TDD, parameterized queries, bcrypt, JWT, immutability, English identifiers)
- **TDD**: Phase 1 (red tests) MUST complete before Phase 2/3 (implementation)
- **Evidence**: Each subagent provides JSON output with `status`, `summary`, `evidence`, `blockers`; hubx-lead validates and commits
- **Blockers**: If Phase 4 finds critical/high security issues, Phase 2/3 tasks repeat with fixes; qa-seguridad writes new tests if needed


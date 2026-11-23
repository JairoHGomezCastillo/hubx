# Quickstart: Sign in (HU-01)

**Purpose**: Runnable validation scenarios to prove the sign-in feature works end-to-end.

**Audience**: Developers, QA, reviewers.

---

## Prerequisites

1. **Local Stack Running**:
   ```bash
   cd /path/to/hubx
   make up
   ```
   This starts PostgreSQL, runs migrations (including `0001_baseline.sql` with demo users), and launches hubx-api and hubx-gw.

2. **Health Check**:
   ```bash
   curl http://localhost:8080/health
   # Expected: {"status":"ok"}
   
   curl http://localhost:8080/api/health
   # Expected: {"status":"ok"}
   ```

3. **Demo Users Available** (from migration 0001):
   - `operator@hubx.co` / `Hubx-Demo-2026` (operator)
   - `rapido@hubx.co` / `Hubx-Demo-2026` (ground partner)
   - `express@hubx.co` / `Hubx-Demo-2026` (ground partner)
   - `ana@hubx.co` / `Hubx-Demo-2026` (customer)
   - `andina@hubx.co` / `Hubx-Demo-2026` (customer)

4. **Tools**:
   - `curl` (for HTTP requests)
   - `jq` (for JSON parsing, optional but recommended)

---

## Validation Scenarios

### Scenario 1: Operator Signs In

**Goal**: Verify operator can authenticate and receive a valid JWT.

**Command**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}'
```

**Expected Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "operator@hubx.co",
    "role": "operator"
  }
}
```

**Validation**:
- HTTP status 200
- `token` field present (JWT string, ~150-200 chars)
- `user.email` matches request email (normalized)
- `user.role` is `"operator"`

**Extract Token** (for use in protected endpoint tests):
```bash
export TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}' \
  | jq -r '.token')

echo "JWT: $TOKEN"
```

---

### Scenario 2: Ground Partner Signs In

**Goal**: Verify ground partner can authenticate with correct role.

**Command**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"rapido@hubx.co","password":"Hubx-Demo-2026"}'
```

**Expected Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "rapido@hubx.co",
    "role": "ground_partner"
  }
}
```

**Validation**:
- `user.role` is `"ground_partner"`

---

### Scenario 3: Customer Signs In

**Goal**: Verify customer can authenticate with correct role.

**Command**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"ana@hubx.co","password":"Hubx-Demo-2026"}'
```

**Expected Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "ana@hubx.co",
    "role": "customer"
  }
}
```

**Validation**:
- `user.role` is `"customer"`

---

### Scenario 4: Email Normalization (Capitals and Spaces)

**Goal**: Verify email is normalized (lowercase, trimmed) before lookup.

**Command**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":" ANA@hubx.co ","password":"Hubx-Demo-2026"}'
```

**Expected Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "ana@hubx.co",
    "role": "customer"
  }
}
```

**Validation**:
- HTTP status 200 (not 401)
- `user.email` is `"ana@hubx.co"` (normalized from `" ANA@hubx.co "`)

---

### Scenario 5: Wrong Password

**Goal**: Verify generic error message (does not reveal which field is wrong).

**Command**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"ana@hubx.co","password":"wrong-password"}'
```

**Expected Response** (401 Unauthorized):
```json
{
  "error": "INVALID_CREDENTIALS",
  "message": "Invalid credentials"
}
```

**Validation**:
- HTTP status 401
- Error message is **generic** (does not say "wrong password")

---

### Scenario 6: Email Not Registered

**Goal**: Verify generic error message (does not reveal user does not exist).

**Command**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"nobody@hubx.co","password":"Hubx-Demo-2026"}'
```

**Expected Response** (401 Unauthorized):
```json
{
  "error": "INVALID_CREDENTIALS",
  "message": "Invalid credentials"
}
```

**Validation**:
- HTTP status 401
- Error message is **identical to Scenario 5** (prevents user enumeration)

---

### Scenario 7: Missing Password

**Goal**: Verify validation error for incomplete data.

**Command**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co"}'
```

**Expected Response** (400 Bad Request):
```json
{
  "error": "MISSING_REQUIRED_FIELDS",
  "message": "Email and password are required"
}
```

**Validation**:
- HTTP status 400
- Clear message about missing field

---

### Scenario 8: Missing Email

**Goal**: Verify validation error for missing email.

**Command**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"password":"Hubx-Demo-2026"}'
```

**Expected Response** (400 Bad Request):
```json
{
  "error": "MISSING_REQUIRED_FIELDS",
  "message": "Email and password are required"
}
```

---

### Scenario 9: Protected Endpoint with Valid Token

**Goal**: Verify JWT works for authenticated requests.

**Setup** (get token from Scenario 1):
```bash
export TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}' \
  | jq -r '.token')
```

**Command** (access protected endpoint, e.g., future `/api/shipments`):
```bash
curl -X GET http://localhost:8080/api/shipments/HBX-SEED-0001 \
  -H "Authorization: Bearer $TOKEN"
```

**Expected Response**: 
- HTTP 200 (if shipment exists and endpoint is implemented)
- OR HTTP 404 (if endpoint not yet implemented)
- **NOT** HTTP 401 (token is valid)

**Note**: This scenario verifies JWT validation; actual shipment endpoints are implemented in future stories (HU-02, HU-04).

---

### Scenario 10: Protected Endpoint without Token

**Goal**: Verify 401 when Authorization header is missing.

**Command**:
```bash
curl -X GET http://localhost:8080/api/shipments/HBX-SEED-0001
```

**Expected Response** (401 Unauthorized):
```json
{
  "error": "UNAUTHORIZED",
  "message": "Authorization header required"
}
```

---

### Scenario 11: Protected Endpoint with Invalid Token

**Goal**: Verify 401 when JWT signature is invalid.

**Command**:
```bash
curl -X GET http://localhost:8080/api/shipments/HBX-SEED-0001 \
  -H 'Authorization: Bearer invalid.token.here'
```

**Expected Response** (401 Unauthorized):
```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid token"
}
```

---

### Scenario 12: Session Expiration

**Goal**: Verify 401 when JWT has expired (requires waiting 1 hour or forging expired token).

**Manual Test** (requires waiting 1 hour):
```bash
# 1. Sign in and get token
export TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}' \
  | jq -r '.token')

# 2. Wait 1 hour (or advance system clock)
sleep 3600

# 3. Use expired token
curl -X GET http://localhost:8080/api/shipments/HBX-SEED-0001 \
  -H "Authorization: Bearer $TOKEN"
```

**Expected Response** (401 Unauthorized):
```json
{
  "error": "UNAUTHORIZED",
  "message": "Session expired. Please sign in again."
}
```

**Automated Test**: Integration tests (Playwright) will forge a JWT with `exp` in the past to avoid waiting.

---

## JWT Token Inspection

**Goal**: Verify JWT structure and claims.

**Decode Token** (using https://jwt.io or CLI):
```bash
# Using jwt-cli (npm install -g jwt-cli)
jwt decode $TOKEN

# OR using jq + base64 (manual decoding)
echo $TOKEN | cut -d'.' -f2 | base64 -d | jq
```

**Expected Payload**:
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "email": "operator@hubx.co",
  "role": "operator",
  "party_id": null,
  "iat": 1737499200,
  "exp": 1737502800
}
```

**Validation**:
- `sub`: UUID (user ID)
- `email`: Matches signed-in user
- `role`: One of `operator`, `ground_partner`, `customer`
- `party_id`: NULL for operators, UUID for partners/customers (future)
- `iat`: Unix timestamp (issued at)
- `exp`: `iat + 3600` (1 hour later)

---

## Frontend Validation (MFE)

**Prerequisites**:
- MFE running (`npm --prefix hubx-mfe run dev` or via `make up`)
- Browser open at `http://localhost:5173` (Vite default port)

**Manual Test**:
1. Navigate to `/sign-in`
2. Enter `operator@hubx.co` / `Hubx-Demo-2026`
3. Click "Sign In"
4. **Expected**: Redirected to dashboard; user email and role displayed; JWT stored in localStorage (`hubx_token` key)

**Verify Token Storage**:
- Open browser DevTools → Application → Local Storage
- Key: `hubx_token`
- Value: JWT string (matches token from API response)

**Sign Out**:
- Click "Sign Out" (future)
- **Expected**: localStorage cleared; redirected to `/sign-in`

---

## Troubleshooting

### "Connection refused" Error

**Symptom**: `curl: (7) Failed to connect to localhost port 8080`

**Solution**:
```bash
# Check if services are running
make ps

# Restart stack
make down
make up

# Check logs
make logs
```

---

### "Invalid credentials" for Correct Password

**Symptom**: 401 error even with correct demo user credentials.

**Possible Causes**:
1. Migration 0001 not applied (users table empty)
2. Bcrypt hash mismatch (seed used wrong password)

**Solution**:
```bash
# Connect to database and verify users
make psql
# In psql:
SELECT email, role FROM users;
# Expected: 5 users (operator@hubx.co, rapido@hubx.co, express@hubx.co, ana@hubx.co, andina@hubx.co)

# If empty, re-apply migration
make reset  # Drops database
make up     # Recreates and applies migrations
```

---

### "Session expired" Immediately After Sign-In

**Symptom**: JWT expires immediately (even before 1 hour).

**Possible Causes**:
1. System clock mismatch (server and client clocks differ)
2. JWT `exp` claim set incorrectly (bug in token generation)

**Solution**:
```bash
# Decode token and check exp
export TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}' \
  | jq -r '.token')

echo $TOKEN | cut -d'.' -f2 | base64 -d | jq '.iat, .exp'
# iat and exp should differ by 3600 (1 hour)

# Check if exp is in the past
date +%s  # Current Unix timestamp
# If exp < current timestamp, bug in JWT generation
```

---

## Success Criteria

**All scenarios above pass** (HTTP status codes and response bodies match expectations).

**Integration Tests Green**:
```bash
make test-integration
# Expected: All HU01_*.spec.ts tests pass
```

**Security Scans Clean**:
```bash
# From repo root
gosec ./hubx-api/...
govulncheck -C hubx-api ./...
# Expected: No critical/high findings
```

---

## Next Steps

After validating quickstart scenarios:
1. Proceed to `tasks.md` (Phase 2: break down by agent)
2. Write integration tests (hubx-bender: HU01_*.spec.ts)
3. Implement backend (hubx-api: `internal/auth`)
4. Implement gateway middleware (hubx-gw: JWT validation)
5. Implement frontend (hubx-mfe: sign-in page, auth service)
6. Run QA verification (integration tests + security scans)
7. Update deployment docs (devops)


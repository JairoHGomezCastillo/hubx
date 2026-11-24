# HTTP Contract: Sign in

> **Superseded in part** by the implement clarifications in `tasks.md` and by `contracts/auth-sign-in.md` v1.1 (the
> root contract prevails): role `ground partner` (space), no `party_id`, schema from `0001_baseline_schema.sql`.

**Endpoint**: `POST /api/auth/sign-in`

**Description**: Authenticate a HUBX user with email and password, returning a JWT session token valid for one hour.

**Service**: hubx-api (exposed through hubx-gw at `/api/auth/sign-in`)

---

## Request

**Method**: POST

**URL**: `/api/auth/sign-in` (gateway strips `/api` prefix, forwards to API `/auth/sign-in`)

**Headers**:
```
Content-Type: application/json
```

**Body** (JSON):

```json
{
  "email": "user@hubx.co",
  "password": "Hubx-Demo-2026"
}
```

**Fields**:

| Field | Type | Required | Validation |
|-------|------|----------|------------|
| `email` | string | ✓ | Max 254 chars; RFC 5321 email format; normalized (lowercase, trimmed) before validation |
| `password` | string | ✓ | Min 1 char (demo users already exist; password strength enforced only on future registration) |

**Example Requests**:

1. **Operator sign-in**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}'
```

2. **Email with capitals and spaces** (normalized):
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":" ANA@hubx.co ","password":"Hubx-Demo-2026"}'
```

3. **Missing password** (validation error):
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co"}'
```

---

## Responses

### Success: 200 OK

**Description**: User authenticated successfully; JWT token issued.

**Headers**:
```
Content-Type: application/json
```

**Body** (JSON):

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJlbWFpbCI6Im9wZXJhdG9yQGh1YnguY28iLCJyb2xlIjoib3BlcmF0b3IiLCJwYXJ0eV9pZCI6bnVsbCwiaWF0IjoxNzM3NDk5MjAwLCJleHAiOjE3Mzc1MDI4MDB9.signature",
  "user": {
    "email": "operator@hubx.co",
    "role": "operator"
  }
}
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `token` | string | JWT session token (HS256, 1-hour expiration) |
| `user.email` | string | User email (normalized) |
| `user.role` | string | User role: `operator`, `ground partner`, or `customer` |

**JWT Token Structure** (decoded):

**Header**:
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

**Payload**:
```json
{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "email": "operator@hubx.co",
  "role": "operator",
  "iat": 1737499200,
  "exp": 1737502800
}
```

**Signature**: HMAC-SHA256 with `JWT_SECRET` environment variable

---

### Client Error: 400 Bad Request

**Description**: Missing required fields or invalid JSON.

**Body** (JSON):

```json
{
  "error": "MISSING_REQUIRED_FIELDS",
  "message": "Email and password are required"
}
```

**Scenarios**:
- Missing `email` field
- Missing `password` field
- Empty `email` or `password` after trimming
- Malformed JSON

---

### Client Error: 401 Unauthorized

**Description**: Invalid credentials (wrong email or wrong password).

**Body** (JSON):

```json
{
  "error": "INVALID_CREDENTIALS",
  "message": "Invalid credentials"
}
```

**Scenarios**:
- Email not found in database
- Password does not match bcrypt hash

**Security Note**: Both scenarios return the **same error message** to prevent user enumeration (spec requirement).

---

### Server Error: 500 Internal Server Error

**Description**: Unexpected server error (database connection failure, JWT signing error, etc.).

**Body** (JSON):

```json
{
  "error": "INTERNAL_SERVER_ERROR",
  "message": "An unexpected error occurred"
}
```

**Note**: Do not expose internal error details to clients (security best practice).

---

## Protected Endpoints (Using JWT)

**All endpoints except `/auth/sign-in` and `/health` require authentication.**

**Request Header**:
```
Authorization: Bearer <token>
```

**Example**:
```bash
curl -X GET http://localhost:8080/api/shipments/HBX-SEED-0001 \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'
```

---

### Success: 200 OK (or other status)

**Description**: Request processed normally (JWT valid and not expired).

---

### Client Error: 401 Unauthorized (Missing Token)

**Description**: `Authorization` header missing.

**Body** (JSON):

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authorization header required"
}
```

---

### Client Error: 401 Unauthorized (Invalid Token)

**Description**: JWT signature invalid, claims missing, or malformed token.

**Body** (JSON):

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid token"
}
```

---

### Client Error: 401 Unauthorized (Expired Session)

**Description**: JWT `exp` claim is in the past (session expired per cross-cutting rule 1).

**Body** (JSON):

```json
{
  "error": "UNAUTHORIZED",
  "message": "Session expired. Please sign in again."
}
```

**Spec Scenario**: "Given an operator signed in more than one hour ago, When the operator tries to view a shipment, Then they are told their session expired and are asked to sign in again."

---

## Gateway Behavior

**Incoming Request** (from MFE):
```
POST /api/auth/sign-in
Authorization: (none for sign-in; present for protected endpoints)
```

**Gateway Actions**:
1. Strip `/api` prefix → forward to API as `POST /auth/sign-in`
2. For `/auth/sign-in`: Forward without token validation (unauthenticated endpoint)
3. For other endpoints: Validate JWT (signature, expiration); return 401 if invalid; forward if valid

**Forwarded Request** (to API):
```
POST /auth/sign-in
Authorization: (original header, if present)
X-Gateway-Secret: <shared-secret>
```

**API Actions**:
1. Verify `X-Gateway-Secret` header (reject if missing/wrong)
2. Validate JWT (even though gateway validated; defense in depth per Constitution II)
3. Extract `role` from JWT claims into request context
4. Process request

---

## Security Requirements

| Requirement | Implementation | Verification |
|-------------|----------------|--------------|
| Generic error messages | Same 401 response for wrong email and wrong password | QA test HU01_security_non_revealing_errors.spec.ts |
| Bcrypt password hashing | Cost 10; `bcrypt.CompareHashAndPassword` | Code review + unit test |
| Parameterized SQL queries | `db.Query("SELECT ... WHERE email = $1", email)` | Code review + gosec scan |
| JWT signature validation | `golang-jwt/jwt` v5; HS256 with `JWT_SECRET` | Unit test + integration test |
| Session expiration enforcement | Middleware checks `exp` claim; returns 401 | QA test HU01_session_expiration.spec.ts |
| Email normalization | `strings.ToLower(strings.TrimSpace(email))` before bcrypt compare | QA test HU01_email_normalization.spec.ts |
| Rate limiting | Future (gateway-level) | Out of scope for HU-01 |

---

## Examples by Role

### 1. Operator Signs In

**Request**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}'
```

**Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJlbWFpbCI6Im9wZXJhdG9yQGh1YnguY28iLCJyb2xlIjoib3BlcmF0b3IiLCJwYXJ0eV9pZCI6bnVsbCwiaWF0IjoxNzM3NDk5MjAwLCJleHAiOjE3Mzc1MDI4MDB9.abc123",
  "user": {
    "email": "operator@hubx.co",
    "role": "operator"
  }
}
```

---

### 2. Ground Partner Signs In

**Request**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"rapido@hubx.co","password":"Hubx-Demo-2026"}'
```

**Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "rapido@hubx.co",
    "role": "ground partner"
  }
}
```

---

### 3. Customer Signs In

**Request**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"ana@hubx.co","password":"Hubx-Demo-2026"}'
```

**Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "ana@hubx.co",
    "role": "customer"
  }
}
```

---

### 4. Wrong Password

**Request**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"ana@hubx.co","password":"wrong-password"}'
```

**Response** (401 Unauthorized):
```json
{
  "error": "INVALID_CREDENTIALS",
  "message": "Invalid credentials"
}
```

---

### 5. Email Not Registered

**Request**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"nobody@hubx.co","password":"Hubx-Demo-2026"}'
```

**Response** (401 Unauthorized):
```json
{
  "error": "INVALID_CREDENTIALS",
  "message": "Invalid credentials"
}
```

**Security Note**: Same response as wrong password (prevents user enumeration).

---

### 6. Email Normalization

**Request** (capitals and spaces):
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":" ANA@hubx.co ","password":"Hubx-Demo-2026"}'
```

**Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "ana@hubx.co",
    "role": "customer"
  }
}
```

**Note**: Email normalized to `ana@hubx.co` before lookup.

---

### 7. Missing Password

**Request**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co"}'
```

**Response** (400 Bad Request):
```json
{
  "error": "MISSING_REQUIRED_FIELDS",
  "message": "Email and password are required"
}
```

---

### 8. Session Expired

**Request** (using expired JWT):
```bash
curl -X GET http://localhost:8080/api/shipments/HBX-SEED-0001 \
  -H 'Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MzczOTkyMDB9.abc'
```

**Response** (401 Unauthorized):
```json
{
  "error": "UNAUTHORIZED",
  "message": "Session expired. Please sign in again."
}
```

---

## Versioning

**Version**: 1.0 (HU-01)

**Stability**: This contract is foundational and expected to remain stable. Future enhancements (token refresh, MFA) will be additive (new endpoints) rather than breaking changes.

**Deprecation Policy**: Breaking changes (e.g., changing response format) require a new API version (`/api/v2/auth/sign-in`) and 6-month deprecation notice.

---

## Related Contracts

| Contract | Story | Description |
|----------|-------|-------------|
| `auth-refresh.md` | Future | Refresh expired JWT without re-entering password |
| `auth-sign-out.md` | Future | Revoke JWT (server-side blacklist) |
| `auth-register.md` | Future | Create new user account |
| `auth-password-reset.md` | Future | Request password reset email |


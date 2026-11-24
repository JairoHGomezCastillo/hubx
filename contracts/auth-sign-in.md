# HTTP Contract: POST /api/auth/sign-in

**Story**: HU-01 Sign in
**Version**: 1.1
**Status**: Draft (pending implementation)

**1.1 changes (implement clarifications)**: role value is `ground partner` (with a space, constitution VI); no
`party_id` claim (`sub` is `users.id`); the gateway also validates the JWT and answers the same 401 bodies below
before forwarding, while the API keeps validating on its own. `GET /health`, `GET /api/health` and
`POST /api/auth/sign-in` are public. This root file prevails over `specs/001-sign-in/contracts/auth-sign-in.md`.

## Endpoint

```
POST /api/auth/sign-in
```

## Description

Authenticate a HUBX user (operator, ground partner, or customer) with email and password, returning a JWT session token valid for one hour.

## Request

**Headers**:
```
Content-Type: application/json
```

**Body**:
```json
{
  "email": "user@hubx.co",
  "password": "password"
}
```

**Fields**:
- `email` (string, required): User email (max 254 chars, RFC 5321 format)
- `password` (string, required): User password (min 1 char)

**Email Normalization**: Email is normalized (lowercase, trimmed) before validation.

## Responses

### 200 OK (Success)

User authenticated successfully.

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "email": "user@hubx.co",
    "role": "operator"
  }
}
```

**JWT Structure** (HS256, 1-hour expiration):
```json
{
  "sub": "user-uuid",
  "email": "user@hubx.co",
  "role": "operator|ground partner|customer",
  "iat": 1737499200,
  "exp": 1737502800
}
```

### 400 Bad Request

Missing required fields or malformed JSON.

```json
{
  "error": "MISSING_REQUIRED_FIELDS",
  "message": "Email and password are required"
}
```

### 401 Unauthorized

Invalid credentials (wrong email or wrong password).

```json
{
  "error": "INVALID_CREDENTIALS",
  "message": "Invalid credentials"
}
```

**Security**: Same error message for both wrong email and wrong password (prevents user enumeration).

### 500 Internal Server Error

Unexpected server error.

```json
{
  "error": "INTERNAL_SERVER_ERROR",
  "message": "An unexpected error occurred"
}
```

## Protected Endpoints

All endpoints except `/auth/sign-in` and `/health` require authentication.

**Request Header**:
```
Authorization: Bearer <token>
```

**401 Responses**:

- Missing token:
```json
{
  "error": "UNAUTHORIZED",
  "message": "Authorization header required"
}
```

- Invalid token:
```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid token"
}
```

- Expired session (JWT `exp` < now):
```json
{
  "error": "UNAUTHORIZED",
  "message": "Session expired. Please sign in again."
}
```

## Examples

**Operator sign-in**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}'
```

**Email with capitals/spaces** (normalized):
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":" ANA@hubx.co ","password":"Hubx-Demo-2026"}'
# Returns 200 OK (matches ana@hubx.co)
```

**Wrong password**:
```bash
curl -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"ana@hubx.co","password":"wrong"}'
# Returns 401, "Invalid credentials"
```

**Use token in protected endpoint**:
```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}' \
  | jq -r '.token')

curl -X GET http://localhost:8080/api/shipments/HBX-SEED-0001 \
  -H "Authorization: Bearer $TOKEN"
```

## Security Requirements

- Parameterized SQL queries (prevent injection)
- Bcrypt password hashing (demo hashes in migration 0001 use cost 12)
- Generic error messages (prevent user enumeration)
- JWT signature validation (HS256 with `JWT_SECRET`)
- Session expiration enforcement (1 hour per cross-cutting rule 1)

## Related Stories

- HU-02: Create shipment (requires operator role)
- HU-03: Record update (requires operator or assigned partner role)
- HU-04: View traceability (role-based visibility)

---

**Full specification**: `specs/001-sign-in/contracts/auth-sign-in.md`


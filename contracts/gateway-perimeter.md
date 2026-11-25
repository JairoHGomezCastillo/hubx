# HTTP Contract: gateway perimeter

**Story**: 011 gateway hardening. Applies to every response of `hubx-gw`. Items marked (A#) are assumptions pending human
confirmation in `specs/011-gateway-hardening/spec.md`.

## Security headers (every response, exactly once each) (A5)

| Header | Value |
|---|---|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Content-Security-Policy` | `default-src 'none'; frame-ancestors 'none'` |
| `Referrer-Policy` | `no-referrer` |
| `Strict-Transport-Security` | `max-age=31536000` |
| `Cache-Control` | `no-store` |

Values from the API for these headers and any `Access-Control-*` are replaced: the gateway is authoritative.

## CORS (A3, A4)

- Allow-list: `cors.allowed-origins` (comma separated, exact match). `*` stops the start. `Vary: Origin` always.
- Preflight = `OPTIONS` + `Origin` + `Access-Control-Request-Method`; answered by the gateway, no JWT, never forwarded,
  never counted. Allowed origin: `204` with `Access-Control-Allow-Origin: <origin>`, `Access-Control-Allow-Methods: GET, POST`,
  `Access-Control-Allow-Headers: Authorization, Content-Type`. Other origin: `403`, no `Access-Control-*`.
- Other requests: allowed origin gets `Access-Control-Allow-Origin`; another origin gets none and is processed normally.
- `Access-Control-Allow-Credentials` is never sent. An `OPTIONS` that is not a preflight still needs the JWT.

## Rate limiting (A1, A2, A6, A7, A9)

| Traffic | Key | Configuration | Default |
|---|---|---|---|
| Validated JWT | `sub` | `ratelimit.requests-per-minute` | 120 |
| `POST /api/auth/sign-in` | email, trimmed and lowercased | `ratelimit.sign-in-requests-per-minute` | 120 |
| `/health`, `/api/health`, preflight | none | - | unlimited |

Refusal, not forwarded to the API:

```
429 Too Many Requests
Retry-After: <whole seconds, 1..60>
Content-Type: application/json

{"error":"RATE_LIMITED","message":"Too many requests. Please try again later."}
```

Sign-in body over 8192 bytes: `413 {"error":"PAYLOAD_TOO_LARGE","message":"Request body too large"}`.
Order: security headers, CORS, JWT, limiter, proxy. An invalid JWT gets its existing 401 and consumes nothing. A valid
JWT with an empty `sub` is answered `401 UNAUTHORIZED` `Invalid token`.

State is in memory, per instance.

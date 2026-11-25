# Implementation Plan: Gateway perimeter hardening

**Branch**: `011-gateway-hardening` | **Spec**: `spec.md` | **Contract**: `contracts/gateway-perimeter.md`

## Summary
`hubx-gw` gains its own configuration reader and middlewares for security headers, CORS and rate limiting. No domain,
API, MFE or database change; no migration; no new dependency.

## Services and owners

| Zone | Change | Owner |
|---|---|---|
| `hubx-gw/` | `internal/config` (own copy), `internal/ratelimit`, `internal/middleware` (headers, CORS, limits), proxy header cleanup, wiring | backend-go |
| `hubx-bender/` | feature, steps; HU-09 `@e2e` totals | qa-seguridad |
| `compose.yaml`, `scripts/`, `.github/workflows/deploy.yml`, `deploy/` | test limits, limited gateway, staging env, tag exclusion | devops |

## Design

- Chain: `SecurityHeaders -> CORS -> JWTValidation -> RateLimit -> mux` (Contract order). JWT middleware puts the validated
  `sub` in the request context; `RateLimit` reads only that. `Authorization` is forwarded untouched (II).
- `ratelimit.Limiter` interface: `Allow(key string) (ok bool, retryAfter time.Duration)`; constructor takes limit, window,
  max keys and a clock (injected, IV). Mutex, per-key timestamps of accepted requests (sliding window, at most `limit`
  each), LRU list capped at max keys, expired entries dropped. No background goroutine. Two independent instances.
- Sign-in: only exact `POST /api/auth/sign-in`; read at most 8192 bytes, decode with `json.NewDecoder(...).Decode` into
  `{Email string json:"email"}` plus `Password`, the same way as the API, so both see the same email (no parser
  differential: duplicate keys, case-insensitive keys, trailing data); restore body and length byte for byte. Key =
  SHA-256 of the normalised email (fixed size, no personal data kept). Emails are never logged.
- `proxy.ModifyResponse` removes contract headers and `Access-Control-*` that come from the API, so each appears once.
- Configuration: `server.port` (alias `PORT`), `api.url`, `cors.allowed-origins`, `ratelimit.requests-per-minute`,
  `ratelimit.sign-in-requests-per-minute`; secrets only from the environment. Non-positive or non-numeric limits, `*` or
  malformed origins stop the start. Precedence of AGENTS.md section 6.
- Acceptance: limits are proven against `gw-limited` (compose profile `test`, `BENDER_LIMITED_BASE_URL`, default
  `http://localhost:8090`; 5 user / 3 sign-in), a real gateway in front of the same API. Everything else uses the main
  gateway with 10000. `@needs-limited-gateway` never runs on deployed environments.
- 2026-11-25 (human): sign-in default 10, CORS default empty. Only `gw-limited` sets `CORS_ALLOWED_ORIGINS`
  (`http://localhost:5173`), so the main test gateway behaves like production. Vite proxies `/api` (`vite.config.ts`).
- The one-minute recovery and the defaults (120 and 10) are proven by unit tests with an injected clock and the real properties file.

## Constitution check
II: gateway only, JWT still validated twice. III: stdlib, own reader, no new dependency. IV: constructors, `test/` and
`mocks/` sub-packages (gomock). V: red before implementation, one `@e2e`. VI: closes headers, CORS, rate limiting. VII: ok.

## Risks
The staging Blueprint reads `main`, so before the first smoke a human sets the two `RATELIMIT_*` variables to 10000 by
hand on `hubx-gw-stg`. Deploy validation is pending.

## Extra scope (human, 2026-11-25)
HU-10: `hubx-api/internal/evidence` states that names are outside the fingerprint (`contracts/hu10-download-evidence.md`).
`docs/diagramas` is brought to the current state (hubx-lead).

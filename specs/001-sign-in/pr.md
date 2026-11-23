# HU-01 · Sign in

**Source**: backlog HU-01 · **Branch**: `001-sign-in` → `release/sdd-hubx-system`

## What this delivers

HUBX users (operator, ground partner, customer) sign in with email and password and receive a one-hour JWT session.
The token is validated at the gateway and again at the API (defense in depth, Constitution II), and protected routes
reject expired or missing tokens.

Endpoint (through the gateway): `POST /api/auth/sign-in` → `200 {token, user:{email, role}}`.

## Behavior (acceptance scenarios covered)

- **Each role signs in** with correct credentials and gets a JWT carrying its role (`operator`, `ground partner`,
  `customer`).
- **Email normalization**: `" ANA@hubx.co "` is lowercased and trimmed before lookup.
- **Non-revealing errors**: wrong password and unknown email both return an identical `401 Invalid credentials`,
  so neither reveals which field failed (bcrypt compared against a dummy hash on unknown email to equalize timing).
- **Incomplete data**: missing email or password returns `400 Email and password are required`.
- **Expired session**: an expired JWT on a protected route returns `401 Session expired. Please sign in again.`

## Changes by service

| Service | Change |
|---|---|
| `hubx-api` | New `internal/auth` module: handler, service, repository, JWT, `RequireAuth` middleware; wired in `main.go`; `jwt.expiration.hours=1`, `JWT_SECRET` from env. |
| `hubx-gw` | `internal/middleware/auth.go`: JWT validation on all routes except `GET /health`, `GET /api/health`, `POST /api/auth/sign-in`; `Authorization` forwarded untouched. |
| `hubx-mfe` | Sign-in page, auth context, HTTP client; **JWT kept in memory** (not localStorage), per `hubx-mfe/AGENTS.md`. |
| `hubx-bender` | `tests/HU01_sign_in.spec.ts`: 7 scenarios through the gateway. |
| infra | `compose.yaml` injects `JWT_SECRET` into the gateway; `scripts/test-integration.sh` passes it to bender; new `.github/workflows/ci.yml`. |

Migration `0001_baseline_schema.sql` (already on the base branch) seeds the demo users; no new migration.

## Evidence

- **Integration**: `make test-integration` → **9/9 pass** (7 HU-01 scenarios + 2 health smoke tests).
- **Unit**: `go -C hubx-api test ./...` and `go -C hubx-gw test ./...` green; `npm --prefix hubx-mfe run test` → 28 pass;
  MFE `typecheck` and `build` clean (TypeScript strict).
- **Security**: `govulncheck` clean on both Go modules after upgrading `golang-jwt/jwt/v5` v5.2.1 → **v5.2.2**
  (fixes GO-2025-3553, HIGH). `gosec` on `hubx-gw` clean; `hubx-api` has one pre-existing MEDIUM G304 in the baseline
  config reader (hardcoded path, not user input), tracked in `memory.md` and out of scope for HU-01.
- Full QA write-up: [`specs/001-sign-in/qa-report.md`](qa-report.md).

## Decisions and deviations

Recorded in `specs/001-sign-in/tasks.md` (Analyze section):

1. One bender test file for the whole story (per `hubx-bender/AGENTS.md`), not seven.
2. MFE token **in memory** (human decision, honoring `hubx-mfe/AGENTS.md` rule 3 over the task-list wording).
3. Role is `ground partner` (with a space), no `party_id` — supersedes the obsolete examples in the contract body.
4. JWT HS256 key derived from the raw `JWT_SECRET` string; the session-expiration test was aligned to match.
5. New dev-only MFE test dependencies (`vitest`, Testing Library, `jsdom`) approved by the human.
6. `golang-jwt/jwt/v5` bumped to v5.2.2 to clear GO-2025-3553.

## Notes for the reviewer

- No secrets in the repo: `JWT_SECRET` is generated locally in a git-ignored `.env` and injected via environment.
- No imports between services; the MFE talks only to `/api` through the gateway.
- Agents do not merge: this PR is for human review and merge.

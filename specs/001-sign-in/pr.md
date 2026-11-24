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
| `hubx-mfe` | Sign-in page, auth context, HTTP client; **JWT kept in memory** (not localStorage), per `hubx-mfe/AGENTS.md`; client-side history-API navigation (no full-page reload). |
| `hubx-bender` | `features/HU01_sign_in.feature` (backlog Gherkin verbatim) + `features/steps/HU01_sign_in.steps.ts`: the HU-01 scenarios through the gateway plus one `@e2e` browser happy path. |
| infra | `compose.yaml` injects `JWT_SECRET` into the gateway; `scripts/test-integration.sh` passes it to bender; new `.github/workflows/ci.yml`. |

Migration `0001_baseline_schema.sql` (already on the base branch) seeds the demo users; no new migration.

## Evidence

- **Integration**: `make test-integration` → **12 scenarios / 65 steps pass** (Cucumber + Playwright): the HU-01
  scenarios (the "Each role signs in" outline over operator / ground partner / customer, email normalization, wrong
  password, unknown email, incomplete data, expired session), the two health smoke scenarios, and one `@e2e` browser
  happy path (operator signs in through the MFE and sees the dashboard with their email and `Role: operator`).

  ```
  12 scenarios (12 passed)
  65 steps (65 passed)
  ```

- **Unit**: `go -C hubx-api test ./...` and `go -C hubx-gw test ./...` green (black-box `test/` subpackages);
  `npm --prefix hubx-mfe run test` → 28 pass; MFE strict `tsc -b` typecheck and `build` clean.
- **Security**: `gosec ./...` → **0 issues** (1 justified `#nosec G304` in the baseline config reader: path from
  internal config, not request input). `govulncheck -C hubx-gw ./...` → **No vulnerabilities found**;
  `govulncheck -C hubx-api ./...` → **0 vulnerabilities in code**, with 1 vulnerability reported in a required-but-not-
  called module. Earlier upgrades stand: `golang-jwt/jwt/v5` v5.2.1 → v5.2.2 (GO-2025-3553) and `golang.org/x/crypto`
  v0.32.0 → v0.58.0 (GO-2026-6355 / 6354 / 6303).
- Full QA write-up: [`specs/001-sign-in/qa-report.md`](qa-report.md).

## Decisions and deviations

Recorded in `specs/001-sign-in/tasks.md` (Analyze section):

1. One bender test file for the whole story (per `hubx-bender/AGENTS.md`), not seven.
2. MFE token **in memory** (human decision, honoring `hubx-mfe/AGENTS.md` rule 3 over the task-list wording).
3. Role is `ground partner` (with a space), no `party_id` — supersedes the obsolete examples in the contract body.
4. JWT HS256 key derived from the raw `JWT_SECRET` string; the session-expiration test was aligned to match.
5. New dev-only MFE test dependencies (`vitest`, Testing Library, `jsdom`) approved by the human.
6. `golang-jwt/jwt/v5` bumped to v5.2.2 to clear GO-2025-3553.

## Review round (hardening applied before merge)

Changes made after the first review of this PR:

- **Shared gateway↔API secret** implemented: the gateway injects `X-Gateway-Secret` on every proxied request and the
  API validates it on all routes with a constant-time compare (`crypto/subtle`); both services refuse to start without
  `GATEWAY_SECRET`. Closes the defense-in-depth gap (Constitution II/VI, OWASP API8).
- **`golang.org/x/crypto` upgraded** v0.32.0 → **v0.58.0** (human-approved), fixing GO-2026-6355 / 6354 / 6303.
- **G304** in the config reader justified with a precise `#nosec G304` (path is internal config, not request input);
  gosec not disabled. Now 0 issues / 1 justified nosec.
- **Spec hygiene**: technology decisions (JWT, bcrypt, token shape) moved from `spec.md` to `plan.md`; `ground_partner`
  corrected to `ground partner` in plan, quickstart and contract; `RequireAuth`-not-wired recorded as deviation 9.

Final state (after the review round): `govulncheck` clean on both Go modules; `gosec` 0 issues on both; Go + MFE unit
tests green. See [`qa-report.md`](qa-report.md) (verdict PASS).

## Constitution 1.4.0 adaptation (before merge)

Constitution 1.4.0 landed on the base branch after HU-01's first delivery. This round adapts HU-01 to it without
changing behavior (the acceptance scenarios stayed the source of truth). Tasks T100–T109 in `tasks.md`.

- **Acceptance tests → Cucumber.** `hubx-bender/tests/HU01_sign_in.spec.ts` became `features/HU01_sign_in.feature`
  (backlog Gherkin verbatim) + `features/steps/HU01_sign_in.steps.ts`. The "Each role signs in" outline has a plain
  `Examples:` (3 roles, API) and an `@e2e Examples:` (operator); the same steps run in the browser when a page exists
  and via the API otherwise.
- **Mandatory `@e2e` + the navigation bug it caught.** The new `@e2e` operator happy path was **red** while every API
  scenario and all 28 MFE unit tests were green: after sign-in the MFE changed route with `window.location` (a full
  page reload), which dropped the **in-memory** JWT, so the app bounced back to `/sign-in` and never showed the
  dashboard. `frontend-react` fixed it by navigating client-side with the history API (keeping the token in memory per
  `hubx-mfe/AGENTS.md`); the `@e2e` is now green. This is deviation 10 in `tasks.md` and the concrete payoff of the
  1.4.0 `@e2e` requirement.
- **1.4.0 package layout.** `hubx-api/internal/auth` was split into a domain package (no `net/http`) and HTTP transport
  in `internal/handlers/auth`; the health handler moved to `internal/handlers/health`. Every package with tests now has
  a `test/` black-box subpackage (`package test`), and `mocks/` for `Repository`/`Service` are generated with gomock
  via `go tool mockgen` (`go.uber.org/mock v0.6.0`, human-approved). `hubx-gw` got the same `test/` layout (no mocks: no
  injected interface). `go mod tidy -diff` is empty for both modules.

Commits this round: `d1606e5`, `b0d40ea` (`qa-seguridad` red), `85c3f66`, `55a683a` (`backend-go` layout + tidy),
`12d2778` (`frontend-react` navigation fix), `9b5b037` (`hubx-lead` spec/pr/tasks closeout) and `5013eda`
(`qa-seguridad` verify). Final: `make test-integration` **12 scenarios / 65 steps green** including the `@e2e`.

> **Note on `qa-report.md` authorship.** In commit `9b5b037` the lead had rewritten `qa-report.md` to the 1.4.0 state;
> that overstepped ownership (the QA report is `qa-seguridad`'s artifact). `qa-seguridad` then re-ran the verify with
> its own evidence (`make test-integration` 12/65, `gosec`, `govulncheck`), corrected the feature-line numbers and
> timestamps to its own run, signed the header, and committed the report itself in `5013eda`
> (`[agent:qa-seguridad] HU-01 verify (1.4.0): PASS`).

## Notes for the reviewer

- No secrets in the repo: `JWT_SECRET` and `GATEWAY_SECRET` are generated locally in a git-ignored `.env` and injected
  via environment.
- No imports between services; the MFE talks only to `/api` through the gateway.
- `auth.RequireAuth` (API JWT middleware) is not attached to any route yet: sign-in is the only API endpoint in HU-01
  and it is public. Protected-route enforcement is deferred to HU-02 (deviation 9 in `tasks.md`); in HU-01 the gateway
  validates the JWT and the shared secret protects the API.
- Agents do not merge: this PR is for human review and merge.

# hubx-gw

Owner agent: `backend-go`. Read the root `AGENTS.md` first; the constitution prevails.

1. **Purpose.** Single entry point of the system. Reverse proxy to hubx-api; no domain logic.
2. **Layout.** `cmd/gw/main.go` (wiring), `internal/proxy` (`httputil` reverse proxy, `StripPrefix /api`), `internal/middleware` (JWT validation, future: headers, CORS, rate limit).
3. **Conventions.** Go stdlib + approved `golang-jwt/jwt/v5` for JWT validation only (never issuance) and `go.uber.org/mock` for tests. Same test layout as hubx-api: every package with tests has `test/` and generated `mocks/` (`go -C hubx-gw generate ./...`). `log/slog` JSON. `ReadHeaderTimeout: 5s`. Own `GET /health`; `GET /api/health` is proxied to the API.
4. **Config.** `application.properties` in this folder. Keys: `server.port` (8080), `api.url` (http://localhost:8081), `cors.allowed-origins` (http://localhost:5173), `ratelimit.requests-per-minute`. Secrets `jwt.secret` and `gateway.secret` env only (`JWT_SECRET`, `GATEWAY_SECRET`). Current code still reads `PORT`/`API_URL` directly; migrate to its own copy of the config reader through a spec.
5. **Commands.** `go -C hubx-gw test ./...`, `go -C hubx-gw build ./...`, `gosec ./...`, `govulncheck -C hubx-gw ./...`.
6. **Contract.** Validates JWT on all routes except `/health`, `/api/health` and `/api/auth/sign-in`; applies security headers, restrictive CORS and rate limiting; injects the shared secret toward the API; forwards `Authorization` untouched (API validates independently, Constitution II).
7. **Never.** Business rules, JWT issuance, database access, imports from hubx-api.

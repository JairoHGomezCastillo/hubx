# hubx-gw

Owner agent: `backend-go`. Read the root `AGENTS.md` first; the constitution prevails.

1. **Purpose.** Single entry point of the system. Reverse proxy to hubx-api; no domain logic.
2. **Layout.** `cmd/gw/main.go` (wiring), `internal/proxy` (`httputil` reverse proxy, `StripPrefix /api`), future `internal/security` (headers, CORS, rate limit).
3. **Conventions.** Go stdlib only. `log/slog` JSON. `ReadHeaderTimeout: 5s`. Own `GET /health`; `GET /api/health` is proxied to the API.
4. **Config.** `application.properties` in this folder. Keys: `server.port` (8080), `api.url` (http://localhost:8081), `cors.allowed-origins` (http://localhost:5173), `ratelimit.requests-per-minute`. Secret `gateway.secret` env only (`GATEWAY_SECRET`). Current code still reads `PORT`/`API_URL` directly; migrate to its own copy of the config reader through a spec.
5. **Commands.** `go -C hubx-gw test ./...`, `go -C hubx-gw build ./...`, `gosec ./...`, `govulncheck -C hubx-gw ./...`.
6. **Contract.** Applies security headers, restrictive CORS and rate limiting; injects the shared secret toward the API; forwards `Authorization` untouched.
7. **Never.** Business rules, JWT issuance, database access, imports from hubx-api.

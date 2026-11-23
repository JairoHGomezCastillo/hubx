# hubx-api

Owner agent: `backend-go`. Read the root `AGENTS.md` first; the constitution prevails.

1. **Purpose.** Domain API: auth, shipments, ledger of chained updates, PostgreSQL access. Only service that touches the DB.
2. **Layout** (constitution IV). `cmd/api/main.go` (wiring only). Domain in `internal/<domain>/` (no `net/http`);
   HTTP handlers in `internal/handlers/<domain>/` exposing `Register(mux, ...)`. Every package with tests has `test/`
   (package `test`, black-box) and `mocks/` (package `mocks`, generated: `//go:generate go tool mockgen ...`, run
   `go -C hubx-api generate ./...`). Infrastructure packages: `config`, `migrate`.
3. **Conventions.** Go latest stable, stdlib `net/http` + `ServeMux` (`"GET /path"` patterns). Constructor injection returning interfaces. Sentinel errors mapped to HTTP in handlers. `log/slog` JSON. `http.Server` with `ReadHeaderTimeout: 5s`. Ledger access only through `LedgerRepository`. Queries always parameterized.
4. **Config.** `application.properties` in this folder; reader in `internal/config` (stdlib only). Keys: `server.port` (default 8081), `db.url` (env only), `jwt.secret` (env only), `gateway.secret` (env only). Run from this folder or pass the dir to `config.Load`.
5. **Commands.** `go -C hubx-api test ./...`, `go -C hubx-api build ./...`, `gosec ./...`, `govulncheck -C hubx-api ./...`.
6. **Contract.** Reachable only through the gateway (`/api` prefix stripped); validates the JWT itself and requires the gateway shared secret. Contracts live in `contracts/`.
7. **Never.** Import another service; add dependencies beyond PG driver, JWT, bcrypt and `go.uber.org/mock` (tests) without approval; hand-write mocks or edit generated ones; log secrets or fingerprints of credentials; edit or delete ledger updates.

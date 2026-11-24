# Tasks: Create a shipment (HU-02)

**Input**: spec.md, plan.md, data-model.md, quickstart.md, `contracts/shipments-create.md`

**Branch**: `002-create-shipment` | **Release**: v0.3.0 (`release/v0.3.0-create-shipment`)

**TDD order** (constitution V): QA red -> API + MFE in parallel -> QA verify -> DevOps

**Agents**: `[QA]` qa-seguridad, `[API]`/`[GW]` backend-go, `[MFE]` frontend-react, `[OPS]` devops. Only hubx-lead
marks `- [x]`, citing the subagent evidence.

---

## Phase 1: QA red

**Owned paths**: `hubx-bender/features/HU02_create_shipment.feature`, `hubx-bender/features/steps/HU02_create_shipment.steps.ts`,
`hubx-bender/features/steps/common.steps.ts`, `hubx-bender/features/support/`

- [ ] **T001** [QA] `features/HU02_create_shipment.feature`: the HU-02 Gherkin from `backlog/backlog.md` verbatim (5
  scenarios, 13 examples), plus the scenario "Shipment created with valid data" repeated verbatim with the `@e2e` tag
  (plan D10).
- [ ] **T002** [QA] `features/steps/HU02_create_shipment.steps.ts`: API steps through the gateway against
  `contracts/shipments-create.md` (statuses, `error` codes, `message`, code format `^HBX-[0-9A-F]{8}$`, stage
  `Created`, two different codes on `AV-204`); reuse HU-01 sign-in steps where they match.
- [ ] **T003** [QA] `@e2e` steps: sign in through the MFE as operator@hubx.co, press `Create shipment`, fill the
  labeled fields, submit, see `Shipment created`, `Code: HBX-...` and `Stage: Created`. Locate by role/label only; no
  API calls.
- [ ] **T004** [QA] Run `make test-integration`: HU-01 and health stay green; every HU-02 scenario red for the right
  reason (route missing, page missing), none undefined.

## Phase 2A: API (backend-go)

**Owned paths**: `hubx-api/internal/shipment/`, `hubx-api/internal/handlers/shipment/`, `hubx-api/cmd/api/main.go`

- [ ] **T010** [API] Red: `internal/shipment/test/service_test.go` (gomock `Repository` and `CodeGenerator`): forbidden
  role makes no repository call; each missing field; same origin/destination ignoring case/spaces; weight 0, 0.1,
  5000, 5001, 0.15, non-number; customer not found; ground partner as customer; success returns stage `Created` and
  the generated code; duplicate code retried, third collision returns an error.
- [ ] **T011** [API] `internal/shipment`: types, `Service`, `Repository` (PostgreSQL, parameterized, insert into
  `shipments` only), `CodeGenerator` (`crypto/rand`, `HBX-` + 8 upper hex), sentinel errors; `//go:generate go tool
  mockgen` into `mocks/`; `test/code_test.go` for the format. No `net/http` import.
- [ ] **T012** [API] Red: `internal/handlers/shipment/test/handler_test.go` (mock `Service`): 201 body per contract;
  each sentinel to its status/`error`/`message`; malformed JSON and body > 16 KiB -> 400 `MISSING_REQUIRED_FIELDS`;
  non-operator gets 403 without the service seeing the body; through `RequireAuth`: missing and expired token -> 401.
- [ ] **T013** [API] `internal/handlers/shipment`: handler + `Register(mux, db, requireAuth)` for `POST /shipments`.
- [ ] **T014** [API] `cmd/api/main.go`: wire `shipment.Register` with `authMiddleware.RequireAuth` (closes HU-01
  deviation 9; remove the `_ = authMiddleware` placeholder).
- [ ] **T015** [API] Evidence: `go -C hubx-api generate ./...` clean diff, `go -C hubx-api test ./...`,
  `go -C hubx-api vet ./...`, `gosec ./...` on hubx-api.

## Phase 2B: Gateway

- [ ] **T020** [GW] No code change: confirm `POST /api/shipments` is proxied and JWT-validated by the existing
  middleware (not in the excluded paths). Evidence: `go -C hubx-gw test ./...` green, no diff under `hubx-gw/`.

## Phase 3: MFE (frontend-react, parallel with Phase 2A)

**Owned paths**: `hubx-mfe/src/`

- [ ] **T030** [MFE] Red: `src/services/shipmentService.test.ts` (request to `/api/shipments` with the JSON body,
  201 mapping, error `message` surfaced).
- [ ] **T031** [MFE] `src/services/shipmentService.ts` using the existing `httpClient`.
- [ ] **T032** [MFE] Red: `src/pages/CreateShipment.test.tsx` (labels, submit, success shows heading `Shipment
  created`, `Code: ...`, `Stage: Created`; server error in `role="alert"`; non-operator sees `You do not have
  permission to create shipments` and no form).
- [ ] **T033** [MFE] `src/pages/CreateShipment.tsx`; route `/shipments/new` in `App.tsx` under `ProtectedRoute`;
  `Create shipment` button in `Dashboard.tsx` for operators only; history-API navigation (`navigateTo`).
- [ ] **T034** [MFE] Evidence: `npm --prefix hubx-mfe run test`, `run lint`, `run build` (includes `tsc -b`).

## Phase 4: QA verify

**Owned paths**: `hubx-bender/`, `specs/002-create-shipment/qa-report.md`

- [ ] **T040** [QA] `make test-integration`: all HU-01, HU-02 (API + `@e2e`) and health scenarios green.
- [ ] **T041** [QA] `gosec ./...` (hubx-api, hubx-gw), `govulncheck -C hubx-api ./...`, `govulncheck -C hubx-gw ./...`.
- [ ] **T042** [QA] `qa-report.md`: scenario -> feature:line -> API/E2E -> result; OWASP API Top 10 (API1, API3, API4,
  API5 relevant); findings with severity.

## Phase 5: DevOps

**Owned paths**: `Makefile`, `compose.yaml`, `*/Dockerfile`, `.github/workflows/`, `scripts/`, `README.md`, `docs/`

- [ ] **T050** [OPS] Confirm `make test-integration` and CI need no change for HU-02 (no new env var, migration or
  service); document `POST /api/shipments` in `README.md` if the README lists endpoints.

## Dependencies

Phase 1 -> (2A, 2B, 3 in parallel; no shared files) -> 4 -> 5.

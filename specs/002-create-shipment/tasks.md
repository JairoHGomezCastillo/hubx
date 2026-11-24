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

- [x] **T001** [QA] `features/HU02_create_shipment.feature`: the HU-02 Gherkin from `backlog/backlog.md` verbatim (5
  scenarios, 12 examples), plus the scenario "Shipment created with valid data" repeated verbatim with the `@e2e` tag
  (plan D10). **Evidence**: commit `e321e77`; feature lines 6 (API) and 13 (`@e2e`).
- [x] **T002** [QA] `features/steps/HU02_create_shipment.steps.ts`: API steps through the gateway against
  `contracts/shipments-create.md` (statuses, `error` codes, `message`, code format `^HBX-[0-9A-F]{8}$`, stage
  `Created`, two different codes on `AV-204`). **Evidence**: commit `e321e77`; status and `error` assertions added in
  verify, commit `bd52222`.
- [x] **T003** [QA] `@e2e` steps: sign in through the MFE as operator@hubx.co, press `Create shipment`, fill the
  labeled fields, submit, see `Shipment created`, `Code: HBX-...` and `Stage: Created`. Locate by role/label only; no
  API calls. **Evidence**: commit `e321e77`.
- [x] **T004** [QA] Run `make test-integration`: HU-01 and health stay green; every HU-02 scenario red for the right
  reason. **Evidence** (qa-seguridad): `25 scenarios (12 passed, 13 failed)`, `146 steps (126 passed, 7 skipped, 13
  failed)`; API scenarios 404 on `POST /api/shipments`, `@e2e` timeout waiting for the `Create shipment` button; 0
  undefined.

## Phase 2A: API (backend-go)

**Owned paths**: `hubx-api/internal/shipment/`, `hubx-api/internal/handlers/shipment/`, `hubx-api/cmd/api/main.go`

- [x] **T010** [API] Red: `internal/shipment/test/service_test.go` (gomock `Repository` and `CodeGenerator`).
  **Evidence**: commit `ea974a4`; weight cases 5e3, 1e-1, -1, 0.15 and the 3-collision path added in the fix round.
  The red run was not shown by the agent (see deviation 3).
- [x] **T011** [API] `internal/shipment`: types, `Service`, `Repository`, `CodeGenerator`, sentinel errors, generated
  `mocks/`. **Evidence**: commit `ea974a4`; repository writes only `shipments`; `ErrDuplicateCode` on 23505 for
  `shipments_code_key`.
- [x] **T012** [API] Red: `internal/handlers/shipment/test/handler_test.go` (mock `Service`). **Evidence**: commit
  `ea974a4`; includes JSON-string weight -> `WEIGHT_OUT_OF_RANGE`.
- [x] **T013** [API] `internal/handlers/shipment`: handler + `Register` for `POST /shipments`. **Evidence**: commit
  `ea974a4`.
- [x] **T014** [API] `cmd/api/main.go`: `shipment.Register` with `authMiddleware.RequireAuth` (closes HU-01 deviation
  9). **Evidence**: commit `ea974a4` (`main.go` +3/-2).
- [x] **T015** [API] Evidence: `go -C hubx-api test -count=1 ./...` all `ok` (run by hubx-lead, including
  `internal/shipment/test` and `internal/handlers/shipment/test`); `go -C hubx-api vet ./...` clean (backend-go);
  `gosec ./...` 0 issues (qa-seguridad). `gofmt -l` was denied to backend-go; it ran `go fmt` on its packages instead.

## Phase 2B: Gateway

- [x] **T020** [GW] No code change. **Evidence** (backend-go): `go -C hubx-gw test ./...` ok; `git diff --stat
  hubx-gw/` empty.

## Phase 3: MFE (frontend-react, parallel with Phase 2A)

**Owned paths**: `hubx-mfe/src/`

- [x] **T030** [MFE] Red: `src/services/shipmentService.test.ts`. **Evidence**: commit `01c242a`, 8 tests.
- [x] **T031** [MFE] `src/services/shipmentService.ts` using the existing `httpClient`. **Evidence**: commit `01c242a`.
- [x] **T032** [MFE] Red: `src/pages/CreateShipment.test.tsx`. **Evidence**: commit `01c242a`, 7 tests.
- [x] **T033** [MFE] `CreateShipment.tsx`, route `/shipments/new`, operator-only `Create shipment` button.
  **Evidence**: commit `01c242a`.
- [x] **T034** [MFE] Evidence (run by hubx-lead after the fix round): `npm --prefix hubx-mfe run test` -> `Test Files
  5 passed (5)`, `Tests 43 passed (43)`; `run build` -> `✓ built in 136ms`; `run lint` -> 0 errors, 3 pre-existing
  warnings in HU-01 files.

## Phase 4: QA verify

**Owned paths**: `hubx-bender/`, `specs/002-create-shipment/qa-report.md`

- [x] **T040** [QA] `make test-integration`: `25 scenarios (25 passed)`, `146 steps (146 passed)` (qa-seguridad;
  rerun by hubx-lead with the same result, 6.783s).
- [x] **T041** [QA] `gosec ./...` 0 issues, 1 nosec (pre-existing G304); `govulncheck -C hubx-api ./...` 0 affecting
  (1 in a required module, not called); `govulncheck -C hubx-gw ./...` no vulnerabilities.
- [x] **T042** [QA] `qa-report.md`: commit `bd52222`; 1 low finding (validation order), 0 high/medium.

## Phase 5: DevOps

- [x] **T050** [OPS] No change needed. **Evidence** (devops): compose, Dockerfiles, `scripts/env.sh`, `ci.yml` and
  Makefile inspected; no new env var, migration, service or port; README has no endpoint list. `git status` showed no
  devops changes.

## Dependencies

Phase 1 -> (2A, 2B, 3 in parallel; no shared files) -> 4 -> 5.

---

## Analyze: deviations and decisions

Contrast of spec, plan, tasks, contract and code at the end of implement.

1. **Validation order (low, accepted by QA).** `hubx-api/internal/handlers/shipment/handler.go` rejects a JSON-string
   `weightKg` before the required-fields check, so a body missing `flight` and carrying `"weightKg":"120"` answers
   `WEIGHT_OUT_OF_RANGE` instead of `MISSING_REQUIRED_FIELDS` (contract step 3 before step 5). No security impact; fix
   in a later story or during PR review.
2. **MFE fix round.** frontend-react's first delivery reported its test, lint and build commands as `not_run` (shell
   denied). hubx-lead ran them: build failed (TS1484 `FormEvent` type import) and 3 unit tests failed, including a
   success view whose `<strong>Code:</strong>` split would also have broken the `@e2e` locator. Sent back as `fix`;
   second delivery green and re-verified by hubx-lead.
3. **TDD evidence for the API is partial.** backend-go did not paste the red run for its first delivery or for the fix
   round, and `gofmt -l` and `gosec` were denied to it. Tests exist and pass; the red step is not evidenced.
4. **API review round.** hubx-lead's review sent back: duplicate-code detection by string comparison (now sentinel
   `ErrDuplicateCode` on constraint `shipments_code_key`), exponent and string weights bypassing the one-decimal rule
   (now `^[0-9]+(\.[0-9])?$`), formatting and a leftover comment.
5. **"The shipment is not created"** is asserted by status and the absence of `code` in the body; no read endpoint
   exists until HU-04, and hubx-bender does not read the database.
6. **QA report inaccuracy.** `qa-report.md` section 3 (API8) says security headers, CORS and rate limiting are applied
   at the gateway; `hubx-gw/AGENTS.md` lists them as future work and `hubx-gw/cmd/gw/main.go` applies only JWT
   validation. Constitution VI requires them; this predates HU-02 and is not introduced by it. The report also says
   "ready to merge", which is a human decision.
7. **Gateway-secret and body-cap checks** were verified by code review in QA, not by an acceptance scenario.
8. **HU-01 deviation 9 closed**: the API's `RequireAuth` now guards `POST /shipments`.
9. **Clarification Q2 (weight with more than one decimal)**: closed by a human decision on 2026-11-24 (Option A,
   rejected with the weight-range message), not deduced from the schema. The code already implements it
   (`service.go` `validWeightRegex` + `ErrWeightOutOfRange`; `service_test.go` "two decimals invalid" `0.15`); no
   code or test change.
10. **Spec count corrected**: HU-02 has 12 examples plus 1 `@e2e` (13 HU-02 scenarios run), not 13 examples.

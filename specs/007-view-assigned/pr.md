## Story
HU-07 View my assigned shipments. Spec: `specs/007-view-assigned/`

Release branch: `release/v0.6.0-demo-07`

A signed-in ground partner sees, on the Dashboard, the shipments assigned to them whose delivery is still pending
(stages Assigned to partner and Out for delivery), the one assigned the longest ago first, 20 at a time with
"Show next 20". Delivered shipments never appear. With nothing pending the list is empty and says "You have no
assigned shipments". Operators and customers are told "You do not have permission to see that information". The
partner is always the session's user; there is no parameter to ask for another partner's list (cross-cutting rule 4).
New route `GET /api/assigned-shipments?offset=N` (`contracts/hu07-view-assigned.md`), read-only, no migration, no
gateway change, no new dependency.

**Base note for the reviewer:** `release/v0.6.0-demo-07` was first cut at v0.3.4. The human re-based it locally on
`main` at v0.4.0 (`8c47690`), but `origin/release/v0.6.0-demo-07` is still at v0.3.4 (`853944c`) when this PR opens:
agents never push to an existing release branch (constitution, Development Process). Until a human runs
`git push origin origin/main:refs/heads/release/v0.6.0-demo-07` (a fast-forward), this PR's diff also shows the
v0.4.0 commits (HU-04, HU-05, UI refresh). After that the diff is only HU-07 (stat below).

## Evidence
- make test-integration (run by hubx-lead, qa-seguridad's shell was denied; HU-04 precedent):
  - Red, before implementation (after qa-seguridad's `fix`): `65 scenarios (59 passed, 6 failed)`. The five API
    scenarios failed with `GET /api/assigned-shipments status Expected: 200 Received: 404` (403 expected for the two
    role examples); the `@e2e` failed with `getByRole('region', { name: /my assigned shipments/i }) ... element(s)
    not found`. A first red run had three scenarios failing for the wrong reason (`function has 0 arguments, should
    have 1`, step regex capture groups without parameters); qa-seguridad fixed them.
  - Green, after implementation (tree of 5f8816b): `1 hook (1 passed)`, `65 scenarios (65 passed)`,
    `421 steps (421 passed)`, `1m 28.911s`, exit 0. `cucumber.mjs` runs `strict: true`.
- go test / npm test (rerun by hubx-lead):
  - `go -C hubx-api test -count=1 ./...`: every test package `ok` (auth, config, handlers/auth, handlers/health,
    handlers/shipment, ledger, migrate, shipment).
  - `go -C hubx-gw test -count=1 ./...`: `ok hubx-gw/internal/middleware/test`, `ok hubx-gw/internal/proxy/test`.
  - `npm --prefix hubx-mfe run test`: `Test Files  14 passed (14)`, `Tests  105 passed (105)`. `run lint`: exit 0,
    4 pre-existing warnings, none in HU-07 files. `run build`: `tsc -b && vite build` ok.
- gosec / govulncheck: **NOT RUN.** qa-seguridad's shell was denied and the lead's allowed commands do not include
  them. HU-07 adds no dependency. **A human must run `gosec ./...` (hubx-api, hubx-gw), `govulncheck -C hubx-api
  ./...` and `govulncheck -C hubx-gw ./...` before merging (constitution VI).**
- make agents-zones: **NOT RUN** (not in the lead's allowed commands). Expected to flag 1ace486, see deviation 1.
- deploy: **Deployment validation skipped at the human's explicit request** (demo mode, lead step 6d). The story
  branch was not deployed to staging and devops was not delegated; `specs/007-view-assigned/deploy.md` records it.
  Staging is deployed and smoke-tested automatically when this PR is merged into the release branch.
- speckit.analyze (spec, plan, tasks, contract against the code): FR-001..FR-006, FR-008, FR-010 map to code and to
  a passing scenario; FR-007 (20 per page, `nextOffset`) and the `offset` 400 are covered by unit tests
  (`internal/shipment/test/assigned_test.go`, `internal/handlers/shipment/test/assigned_handler_test.go`,
  `AssignedShipments.test.tsx`), not by a backlog scenario; FR-009 (one-hour session) comes from the shared
  `RequireAuth` and gateway JWT check, with no HU-07-specific scenario in the backlog. Deviations:
  1. Commit 1ace486 `[agent:hubx-lead] HU-07 red WIP` (a pre-rebase checkpoint, made before this session) contains
     qa-seguridad's bender files, outside the lead's zone. qa-seguridad took ownership in its red delivery (8e6275e),
     but the tag on 1ace486 stays wrong and `make agents-zones` / CI should reject it. Needs a human decision: add an
     exception line to `scripts/agent-zones-exceptions.txt` (devops/human path), or rewrite the branch history
     before merge. The lead did neither on its own.
  2. Artifacts were drafted on the v0.3.4 base. Reconciled with v0.4.0 in b947601: base, risks and assumptions
     updated; the merge-conflict risk is gone; the plan names the existing `LedgerRepository`/`ShipmentRepository`
     methods and the UI-refresh Dashboard layout. The contract needed no change: the gateway's
     `usesSessionErrorContract` does not match `/api/assigned-shipments`, so the flat 401 holds.
  3. The handler package declares its own `AssignedLister` interface (with a generated mock) besides the domain's;
     same pattern as HU-04's `traceability_reader_mock.go`.
  4. `weightKg` is converted from the stored decimal literal to a JSON number in the handler (`parseWeightJSON`,
     falls back to the string if parsing ever fails).
  5. The MFE shows the assignment date as `YYYY-MM-DD`; "Show next 20" appends the next page to the visible list.
  6. Rework after rerunning sub-agent evidence: backend-go's first delivery did not compile (missing import, unused
     variable, stale mocks); frontend-react's first delivery replaced 7 HU-02 `createShipment` tests with one weaker
     test. Both went back as `fix` and are corrected in their single commits.
- qa-report.md: `specs/007-view-assigned/qa-report.md` (verdict PASS with pre-existing finding: gateway security
  headers, CORS and rate limiting, API8, medium, needs its own story).

## Checklist
- [x] Every backlog scenario has a test (`hubx-bender/features/HU07_view-assigned.feature`, Gherkin verbatim + one
      `@e2e`)
- [x] No secrets, no cross-service imports, MFE only via /api
- [ ] Only my own paths changed: true for every commit except 1ace486 (deviation 1); `make agents-zones` not run
- [x] Backlog status updated (HU-07 In review; HU-03/04/05 reconciled to the v0.4.0 tag)
- [ ] gosec / govulncheck clean: not run, human to run before merge
- [x] Deploy validation skipped at the human's explicit request (deploy.md)

Commits: 5171925 `[agent:hubx-lead]` spec, plan, tasks, contract; 1ace486 `[agent:hubx-lead]` red WIP checkpoint
(see deviation 1); a3a692b merge of the re-based release branch; b947601 `[agent:hubx-lead]` reconcile with v0.4.0;
8e6275e `[agent:qa-seguridad]` red; 74a93b3 `[agent:hubx-lead]` red task marks; 1afa4c9 `[agent:backend-go]` API
route; 5f8816b `[agent:frontend-react]` Dashboard section; da7103b `[agent:qa-seguridad]` verify and qa-report;
closing `[agent:hubx-lead]` commit with tasks, backlog, memory, README and this file.

<details><summary>git diff --stat origin/main...HEAD (before the closing commit)</summary>

```
 backlog/backlog.md                         |   8 +-
 contracts/hu07-view-assigned.md            |  90 ++++
 hubx-api/cmd/api/main.go                   |   5 +
 .../handlers/shipment/assigned_handler.go  | 161 +++++++
 .../mocks/assigned_handler_mock.go         |  57 +++
 .../shipment/mocks/repository_mock.go      |  15 +
 .../shipment/test/assigned_handler_test.go | 390 ++++++++++++++++
 .../ledger/mocks/repository_mock.go        |  15 +
 hubx-api/internal/ledger/postgres.go       |  40 ++
 hubx-api/internal/ledger/repository.go     |  17 +-
 .../internal/ledger/test/postgres_test.go  | 220 +++++++++
 hubx-api/internal/shipment/assigned.go     | 133 ++++++
 .../mocks/ledger_repository_mock.go        |  15 +
 .../shipment/mocks/repository_mock.go      |  15 +
 hubx-api/internal/shipment/repository.go   |   2 +
 .../shipment/test/assigned_test.go         | 292 ++++++++++++
 .../internal/shipment/update_postgres.go   |  48 ++
 .../features/HU07_view-assigned.feature    |  43 ++
 .../steps/HU07_view-assigned.steps.ts      | 366 +++++++++++++++
 hubx-bender/features/steps/common.steps.ts |  20 +-
 .../AssignedShipments.test.tsx             | 240 ++++++++++
 .../AssignedShipments.tsx                  | 136 ++++++
 .../AssignedShipmentsContainer.tsx         |  14 +
 hubx-mfe/src/pages/Dashboard.test.tsx      |  57 ++-
 hubx-mfe/src/pages/Dashboard.tsx           |   5 +
 .../src/services/shipmentService.test.ts   | 289 +++++++-----
 hubx-mfe/src/services/shipmentService.ts   |  33 +-
 specs/007-view-assigned/deploy.md          |  10 +
 specs/007-view-assigned/plan.md            | 147 ++++++
 specs/007-view-assigned/qa-report.md       | 255 ++++++++++
 specs/007-view-assigned/spec.md            | 155 ++++++
 specs/007-view-assigned/tasks.md           |  60 +++
 32 files changed, 3227 insertions(+), 126 deletions(-)
```
`pr.md`, `memory.md` and `README.md` are added by the closing lead commit on top of this stat.
</details>

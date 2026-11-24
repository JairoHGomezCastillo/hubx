# hubx-bender

Owner agent: `qa-seguridad`. QA workspace, not a running component. Constitution Principle V applies.

1. **Purpose.** BDD acceptance tests written before implementation: every backlog scenario automated against the API
   through the gateway, plus at least one `@e2e` scenario per story (its critical happy path in a real browser).
2. **Tooling.** Cucumber (`@cucumber/cucumber`) + Playwright, TypeScript loaded with `tsx`. Config `cucumber.mjs`.
3. **Layout.**
   - `features/HUxx_<name>.feature`: one per story, scenarios copied **verbatim** from `backlog/backlog.md`.
   - `features/steps/HUxx_<name>.steps.ts`: that story's steps; `features/steps/common.steps.ts`: reusable steps.
   - `features/support/world.ts` (`HubxWorld`: `api` = Playwright request to the gateway, `ui` = browser page in
     `@e2e`), `features/support/hooks.ts` (contexts, screenshot on failure).
   - `features/health.feature`: platform smoke (gateway, API through the gateway, browser through the MFE).
4. **E2E.** Tag `@e2e`; drive the MFE with `this.ui`, locating by role and label (`getByRole`, `getByLabel`), never by
   CSS. An `@e2e` scenario must not call the API directly.
5. **Run.** `make test-integration` (full stack + all scenarios). Against a running `make up`:
   `npm --prefix hubx-bender test`, `run test:api`, `run test:e2e` (`HEADED=1` to watch the browser). Setup once:
   `npm --prefix hubx-bender ci && npm --prefix hubx-bender run browsers`. URLs: `BENDER_BASE_URL` (gateway, default
   `http://localhost:8080`), `BENDER_UI_URL` (MFE, default `http://localhost:3000`). Reports in `reports/` (ignored).
6. **Never.** Fix production code, bypass the gateway, hard-code secrets, add report files other than
   `specs/NNN-*/qa-report.md`, new npm dependencies without human approval.

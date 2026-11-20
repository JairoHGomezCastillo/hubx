# hubx-bender

Owner agent: `qa-seguridad`. QA workspace, not a running component. Constitution Principle V applies.

1. **Purpose.** API integration tests, one per Gherkin scenario, written before implementation.
2. **Tooling.** Playwright Test in API mode (`request`); always through the gateway (`/api/...`). Browser E2E optional.
3. **Naming.** `HUxx_<scenario>` per test. Test data and credentials from environment variables.
4. **Run.** `make test-integration` (devops owns the target). Setup once: `npm --prefix hubx-bender ci`. Gateway URL
   from `BENDER_BASE_URL` (default `http://localhost:8080`). One file per story in `tests/` (`HU02_*.spec.ts`);
   `tests/health.spec.ts` is the regression smoke. New npm dependencies need human approval.
5. **Never.** Fix production code, bypass the gateway, hard-code secrets.

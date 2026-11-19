# hubx-bender

Owner agent: `qa-seguridad`. QA workspace, not a running component. Constitution Principle V applies.

1. **Purpose.** API integration tests, one per Gherkin scenario, written before implementation.
2. **Tooling.** Playwright Test in API mode (`request`); always through the gateway (`/api/...`). Browser E2E optional.
3. **Naming.** `HUxx_<scenario>` per test. Test data and credentials from environment variables.
4. **Run.** `make test-integration` (devops owns the target). Setup once: `npm --prefix hubx-bender init -y` and install `@playwright/test` (asks for approval).
5. **Never.** Fix production code, bypass the gateway, hard-code secrets.

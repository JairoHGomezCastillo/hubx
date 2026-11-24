# hubx-mfe

Owner agent: `frontend-react`. Read the root `AGENTS.md` first; the constitution prevails.

1. **Purpose.** Web UI for operator, ground partner and customer.
2. **Layout.** `src/` (feature folders, components, an API client module), Vite config at root.
3. **Conventions.** React + TypeScript `strict`. All HTTP goes through one API client module using base path `/api` on the gateway. JWT kept in memory (never localStorage), so navigation after sign-in stays inside the app (history API), never `window.location`. Accessible markup (labels, buttons, headings): the `@e2e` scenarios locate elements by role and label. UI text language as defined in each spec; code and identifiers in English with the backlog glossary.
4. **Config.** Build-time `VITE_GATEWAY_URL` (public, no secrets). Dev server on 5173, matching the gateway CORS origin. In the local stack (`make up`) the build is served by nginx on http://localhost:3000 (`Dockerfile`), with `/api` proxied to the gateway.
5. **Commands.** `npm --prefix hubx-mfe run dev|build|lint|test`; typecheck via `tsc -b`.
6. **Contract.** Consumes only the HTTP contract in `contracts/`; never calls hubx-api directly.
7. **Never.** Secrets, direct API calls, `any` to silence the compiler, new dependencies without justification in the plan.

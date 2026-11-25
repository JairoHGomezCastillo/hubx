## Story
011 Gateway perimeter hardening (Source: user description; foundation story, not in the backlog). Spec:
`specs/011-gateway-hardening/`. Contract: `contracts/gateway-perimeter.md`.

Release branch: `release/v0.8.0-gateway-hardening` (confirmed by the human, created from `main` 551773b).

**Extra scope requested by the human (2026-11-25):**
- HU-09 (review of PR #41): dated correction of its `qa-report.md` and an `@e2e` that checks the totals.
- HU-10 (review of PR #43): the PDF states that fingerprints cover the party identifiers and not the printed names.
- `docs/diagramas`: cherry-pick of de033a1 (04894bd, author Andres Castañeda) and, separately, its update to the current state.

**Deploy validation: PENDING.** Nothing was pushed or deployed. Before the first staging smoke a human sets
`RATELIMIT_REQUESTS_PER_MINUTE` and `RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE` to 10000 by hand on `hubx-gw-stg`
(`deploy/README.md`). Demo mode was not requested.

## Human decisions
Limit per validated `sub`, 120/min; sign-in by email, 10/min (A1, lockout risk accepted); never by address; CORS allow-list
empty by default (A3); A2, A4-A9, empty `sub` 401, invalid port or missing secret stops the start: accepted. None pending.

## Evidence
- make test-integration (qa-seguridad): `88 scenarios (88 passed)`, `555 steps (555 passed)`; `@empty-db`
  `2 scenarios (2 passed)`, `12 steps (12 passed)`. Red runs: 9 failed of 87 (gateway); 3 failed of 88 (decisions, HU-10)
- go test hubx-api and hubx-gw (qa-seguridad): all `ok`; hubx-gw `-race` (backend-go): all `ok`. npm test: not run (hubx-mfe
  not changed)
- gosec (qa-seguridad): `Files : 83`, `Nosec : 2` (G304 in both config readers), `Issues : 0`
- govulncheck (qa-seguridad): hubx-api `Your code is affected by 0 vulnerabilities`; hubx-gw `No vulnerabilities found`
- speckit.analyze (read by hubx-lead): 12/12 FR of 011 and FR-014 of 010 have tasks, code and tests. Deviations: expired
  identities are evicted only by LRU (the bound holds); no per-scenario listing; the feature is named `S011_` (no HU number)
- qa-report.md: `specs/011-gateway-hardening/qa-report.md`. Open low: F1 (accepted), F2, F3 (no `ReadTimeout`)
- Mermaid of `docs/diagramas` reviewed by reading; not rendered by a tool

## Checklist
- [x] Every spec scenario has a test (no backlog scenarios exist for 011; HU-09 and HU-10 Gherkin unchanged)
- [x] No secrets, no cross-service imports, MFE only via /api
- [x] Each commit stays in its agent's zone (`make agents-zones`)
- [x] Backlog status: no change needed (011 is not in the backlog)

🤖 Generated with [Claude Code](https://claude.com/claude-code)

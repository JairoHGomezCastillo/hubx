## Story
011 Gateway perimeter hardening (Source: user description; foundation story, not in the backlog) and the HU-09 correction
asked by the review of PR #41. Spec: `specs/011-gateway-hardening/`. Contract: `contracts/gateway-perimeter.md`.

Release branch: `release/v0.8.0-gateway-hardening` is intended, **not confirmed by the human and not created**. Branch base:
`hotfix/v0.7.1-close-pending` (efdb5a3).

**Deploy validation: PENDING.** Nothing was pushed or deployed; it waits for the human to confirm the release. Demo mode
was not requested. A human must sync the staging Blueprint first (limits 10000).

## Evidence
- make test-integration (qa-seguridad): red `87 scenarios (78 passed, 9 failed)`; verify `87 scenarios (87 passed)`,
  `550 steps (550 passed)` and `@empty-db` `2 scenarios (2 passed)`, `12 steps (12 passed)`
- go test (qa-seguridad): hubx-api and hubx-gw all `ok`. backend-go also ran hubx-gw `-race`: all `ok`; hubx-lead reran
  hubx-gw: all `ok`. npm test: not run (hubx-mfe not changed)
- gosec (qa-seguridad): `Files : 83`, `Nosec : 2`, `Issues : 0` (both are G304 on the config readers of hubx-api and hubx-gw)
- govulncheck (qa-seguridad): hubx-api `Your code is affected by 0 vulnerabilities`; hubx-gw `No vulnerabilities found`
- make agents-zones (devops, up to 78bb8c1): `agent zones ok (origin/main..HEAD)`. The closing commit needs a rerun
- speckit.analyze (read by hubx-lead): 12/12 FR have tasks and code. Deviations: expired identities are only evicted by LRU,
  not swept (the bound holds); red runs could not print a per-scenario list (progress formatter); the feature is named
  `S011_` because there is no HU number
- qa-report.md: `specs/011-gateway-hardening/qa-report.md`

## Assumptions pending human confirmation
A1-A9 in `spec.md`. The ones with impact: A1 sign-in 120/min per email (QA finding F1, medium: weak against guessing), A2 429
body, A3 production origins unset (default `http://localhost:5173`), A4 CORS behaviour, A5 header values, A6 8 KiB / 413.

## Checklist
- [x] Every spec scenario has a test (no backlog scenarios exist for this story)
- [x] No secrets, no cross-service imports, MFE only via /api
- [x] Only my own paths changed (git diff --stat below, before the closing commit)
- [x] Backlog status: no change needed (story not in the backlog; rows already reconciled)

<details><summary>git diff --stat</summary>

```
 34 files changed, 3960 insertions(+), 83 deletions(-)
 deploy/compose/CI: .github/workflows/deploy.yml compose.yaml deploy/README.md deploy/render.staging.yaml scripts/{review-story,test-integration}.sh
 hubx-gw: AGENTS.md application.properties cmd/gw internal/{config,middleware,proxy,ratelimit,server}
 hubx-bender: S011 feature and steps, HU09 steps
 specs/009-view-count-by-stage/qa-report.md, specs/011-gateway-hardening/, contracts/gateway-perimeter.md
```
</details>

🤖 Generated with [Claude Code](https://claude.com/claude-code)

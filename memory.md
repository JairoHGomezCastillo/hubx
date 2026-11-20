# Project memory (hubx-lead maintains; humans may edit)

## Decisions
- Verifiable hash-chained ledger instead of on-chain blockchain; identity from session; signing is a future extension.
- PostgreSQL (managed free instance); only hubx-api touches it.
- Configuration via `application.properties` per service (constitution Principle III); stdlib reader, one copy per service.
- Backlog and all artifacts in English; guide and conversation with the team in Spanish.
- Branching: release/feature/hotfix; the Spec Kit story branch is the feature branch.
- Integration branch is `release/sdd-hubx-system` (the team guide calls it `release/v1`); story PRs target it, `main` receives it by PR for deploy.

## Conventions
- Commit format: `[agent:role] HU-xx short summary`; one commit per subagent delivery; no agent pushes.
- Test names: `HUxx_<scenario>`.
- Tasks in `tasks.md` are marked `- [x]` only by hubx-lead, citing the subagent evidence.

## Lessons
- Subagent delivery must be compared with `git status --short` and `git diff --stat` before and after, not by file names only.

## Pending
- hubx-api `main.go` still reads `PORT` directly; wire `internal/config`.
- hubx-gw has no config reader yet (reads `PORT`/`API_URL`).
- `Makefile` / `make test-integration` and `hubx-bender` project (npm init + Playwright) not created yet.

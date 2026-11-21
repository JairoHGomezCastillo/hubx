# HUBX

Air cargo traceability platform with ground partners. The operator, the ground partner and the customer of a shipment
share one verifiable history: every update is recorded by its responsible party and chained to the previous one with a
SHA-256 fingerprint, so any later alteration is detected (a verifiable ledger, not an on-chain blockchain).

Built with Spec-Driven Development (Spec Kit) by a team of five AI agents in opencode, orchestrated by `hubx-lead`.

## Architecture

```
hubx-mfe (React) --> hubx-gw (gateway, /api/*) --> hubx-api (domain, ledger, auth) --> PostgreSQL
                                                     ^
hubx-bender (Playwright, API mode) ------------------+ through the gateway only
```

| Service | Role | Stack |
|---|---|---|
| [`hubx-gw`](hubx-gw/) | Single entry point: reverse proxy, CORS, security headers, rate limit | Go, standard library |
| [`hubx-api`](hubx-api/) | Domain API, ledger, auth; the only service that touches the database | Go `net/http`, pgx |
| [`hubx-mfe`](hubx-mfe/) | Frontend | React, TypeScript strict, Vite |
| [`hubx-bender`](hubx-bender/) | QA workspace: API integration tests through the gateway | Playwright Test |

Services never import each other's code and talk only over HTTP. Each folder has its own `AGENTS.md`.

## Quick start

Requirements: Git, Go 1.27.4+, Node LTS, Docker with Compose, make. Step-by-step setup for macOS and Windows, known
problems and demo users: [`docs/setup.md`](docs/setup.md).

```bash
git clone https://github.com/JairoHGomezCastillo/hubx.git && cd hubx
git switch release/sdd-hubx-system
npm --prefix hubx-bender ci && npm --prefix hubx-mfe ci

make up                                   # PostgreSQL, migrations, API and gateway
curl -s http://localhost:8080/health      # gateway
curl -s http://localhost:8080/api/health  # API through the gateway
make test-integration                     # isolated clean stack + hubx-bender, then tear down
make down
```

`make help` lists every target (`test`, `security`, `psql`, `logs`, `reset`...). Local secrets are generated at random
in a git-ignored `.env` on first run; nothing has to be configured by hand.

## How we work

| Document | What it holds |
|---|---|
| [`docs/GUIA-EQUIPO-HUBX.md`](docs/GUIA-EQUIPO-HUBX.md) | Team guide (Spanish): roles, calendar, the per-story cycle, conflict rules, demo |
| [`backlog/backlog.md`](backlog/backlog.md) | User stories HU-01 to HU-10, business language and cross-cutting rules |
| [`.specify/memory/constitution.md`](.specify/memory/constitution.md) | Constitution: the highest authority of the project |
| [`AGENTS.md`](AGENTS.md) | System context for every agent |
| [`memory.md`](memory.md) | Decisions, conventions, lessons and pending items |
| [`specs/`](specs/) | One folder per story: `spec.md`, `plan.md`, `tasks.md`, `qa-report.md`, `deploy.md`, `pr.md` |

Authority order: constitution > `AGENTS.md` > service `AGENTS.md` > `memory.md`.

Per story: `specify -> clarify -> plan -> tasks -> implement -> analyze -> pr`, on its own branch `NNN-name`. The story
ends with a PR to `release/sdd-hubx-system`: `hubx-lead` writes `specs/NNN-name/pr.md`, you push, `hubx-lead` opens the
PR; the branch is deleted automatically on merge. `main` receives the release by PR for deployment. Commits are one
per agent delivery, tagged `[agent:role] HU-xx summary`; agents never push.

## Multi-agent architecture

Single entry point: `hubx-lead` (default agent of [`opencode.jsonc`](opencode.jsonc)). Definitions in
[`.opencode/agents/`](.opencode/agents/).

| Agent | Function | Edits only |
|---|---|---|
| `hubx-lead` | Orchestrates the Spec Kit flow, delegates, checks evidence and scope, proposes commits, opens the PR | `specs/`, `backlog/`, `contracts/`, `memory.md` |
| `backend-go` | Implements Go modules with TDD | `hubx-api/`, `hubx-gw/` |
| `frontend-react` | Implements the frontend with TDD | `hubx-mfe/` |
| `qa-seguridad` | Integration tests first (red), then verification and security report | `hubx-bender/`, `qa-report.md` |
| `devops` | Makefile, Dockerfiles, Compose, CI, deployment | build and deploy files |

<!-- Owner B (guide, section 5): the lead's flow in detail and the agent tests carried out. -->

## Deployment

Pending (guide, section 9): public URL and smoke test evidence will be recorded here and in `specs/NNN-*/deploy.md`.

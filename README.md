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

## Authentication (HU-01)

Sign in through the gateway with a preloaded demo user to obtain a one-hour JWT:

```bash
curl -s -X POST http://localhost:8080/api/auth/sign-in \
  -H 'Content-Type: application/json' \
  -d '{"email":"operator@hubx.co","password":"Hubx-Demo-2026"}'
# -> {"token":"<jwt>","user":{"email":"operator@hubx.co","role":"operator"}}
```

The email is normalized (lowercased and trimmed) before lookup. Wrong password and unknown email return the same
`401 Invalid credentials` so neither reveals which field failed. Use the token on protected endpoints with
`Authorization: Bearer <token>`; once it expires the gateway and API answer `401 Session expired. Please sign in again.`

Demo users seeded by migration `0001` (all with password `Hubx-Demo-2026`):

| Email | Role |
|---|---|
| `operator@hubx.co` | operator |
| `rapido@hubx.co` | ground partner |
| `ana@hubx.co` | customer |

The JWT is signed with `JWT_SECRET` (generated locally in `.env`); it is never stored in the repository. The MFE keeps
the token in memory only (never in `localStorage`). See [`docs/setup.md`](docs/setup.md) for the full demo user list.

Continuous integration ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) builds and unit-tests all services,
runs `gosec` and `govulncheck`, and then runs `make test-integration` on a clean stack.

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
ends with a PR to `release/sdd-hubx-system`: `hubx-lead` commits, pushes the story branch and opens the PR with
`specs/NNN-name/pr.md` as body; you review and merge (agents never merge); the branch is deleted automatically on merge. `main` receives the release by PR for deployment. Commits are one
per agent delivery, tagged `[agent:role] HU-xx summary`.

## Multi-agent architecture

Single entry point: `hubx-lead` (default agent of [`opencode.jsonc`](opencode.jsonc)). Definitions in
[`.opencode/agents/`](.opencode/agents/).

The same five agents run in **opencode, Kiro or Claude Code**; use whichever you have credits for.

| | opencode (team default) | Kiro | Claude Code |
|---|---|---|---|
| Start | `opencode` (starts on `hubx-lead`) | `kiro-cli chat --agent hubx-lead` or the IDE agent picker | `claude --agent hubx-lead` |
| Agents | `.opencode/agents/<name>.md` (**source**) | `.kiro/agents/<name>.json` | `.claude/agents/<name>.md` (generated) |
| Spec Kit | `/speckit.*` (`.opencode/commands/`) | `.kiro/prompts/speckit.*` | `/speckit-*` skills (`.claude/skills/`) |
| Model per role | none in the repo; optional personal override in `~/.config/opencode/opencode.jsonc` (`"agent": {"backend-go": {"model": "..."}}`) | in the agent JSON: Sonnet 4.5 (1.30x) for lead, QA, backend, frontend; Haiku 4.5 (0.40x) for devops | in the agent: `opus` for the lead, `sonnet` for QA, backend and frontend, `haiku` for devops |
| Permissions | agent front matter | agent JSON `permissions.rules` (deny-overrides, so "all but X" is a `deny` with `exclude`) | the same Kiro rules, enforced by the `scripts/agent-guard.mjs` hook, plus `deny` rules in `.claude/settings.json` |

- **One source.** Edit agents only in `.opencode/agents/`, then run `make agents-sync`: it writes `.kiro/agent-prompts/`
  and `.claude/agents/`. `make agents-check` fails if anything drifts. Per-role permissions are edited once, in
  `.kiro/agents/<name>.json`, and apply to Kiro and Claude Code.
- **Claude Code trust.** The guard hooks only run after the folder trust dialog is accepted (run `claude` in the repo
  once and accept it). Bash rules are not a security boundary in Claude Code; the hard barrier against merges is the
  GitHub ruleset (1 approval on `main` and release).
- Available models depend on your plan: `opencode models`, `kiro-cli chat --list-models`.

| Agent | Function | Edits only |
|---|---|---|
| `hubx-lead` | Orchestrates the Spec Kit flow, delegates, checks evidence and scope, commits, pushes the story branch, opens the PR (never merges) | `specs/`, `backlog/`, `contracts/`, `memory.md` |
| `backend-go` | Implements Go modules with TDD | `hubx-api/`, `hubx-gw/` |
| `frontend-react` | Implements the frontend with TDD | `hubx-mfe/` |
| `qa-seguridad` | Integration tests first (red), then verification and security report | `hubx-bender/`, `qa-report.md` |
| `devops` | Makefile, Dockerfiles, Compose, CI, deployment | build and deploy files |

<!-- Owner B (guide, section 5): the lead's flow in detail and the agent tests carried out. -->

## Deployment

Pending (guide, section 9): public URL and smoke test evidence will be recorded here and in `specs/NNN-*/deploy.md`.

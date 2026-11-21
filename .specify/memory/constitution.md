# Hubx Constitution

## Core Principles

### I. Immutable Ledger as a Source of Truth
The system is an air cargo traceability platform with ground partners whose canonical state is a ledger of
updates chained via SHA-256 fingerprints. Each update MUST include the SHA-256 fingerprint of the preceding update
and identify the responsible party that records it, so that the chain is verifiable from end to end. Every update
MUST be immutable: it is NOT modified or deleted; corrections, when supported, are recorded as new updates. The
system MUST use a verifiable ledger instead of an on-chain blockchain during the current scope of the project, and
ALL access to the ledger MUST go through the LedgerRepository interface to enable future migration to a live network.
Known limitation: the identity of the responsible party comes from the authenticated session; cryptographic
per-party signatures and distributed consensus are future extensions.

*Rationale*: Immutability and an identified responsible party per update ensure end-to-end auditability; the
repository abstraction prevents coupling the domain to a specific persistence technology.

### II. Three-Tier Architecture and Mandatory Gateway
The system MUST consist of hubx-gw (gateway), hubx-api (base API), and hubx-mfe (frontend). Each service MUST be
built, tested and deployed independently, MUST NOT import code from another service, and communicates only through
the HTTP contract. The frontend MUST NOT communicate directly with the API: ALL traffic MUST pass through the
gateway. The API MUST require the gateway's shared secret and MUST validate the JWT on its own. The hubx-bender
automation workspace MUST remain outside the three components: it is a QA tool, not a running component.

*Rationale*: the gateway concentrates perimeter control; the API's own JWT validation avoids delegating
authentication to a single point of failure.

### III. Technology Stack and Restrictions
The backend MUST use the most recent stable version of Go with the standard library (net/http with ServeMux) for
routing; external web frameworks MUST NOT be introduced. The frontend MUST use React with TypeScript in strict mode.

**Database.** Data persistence MUST use PostgreSQL (free managed instance). Only hubx-api accesses the database.

**Dependencies.** Go external dependencies are limited to the PostgreSQL driver, JWT and bcrypt (and gomock only if
adopted). Every new dependency MUST be justified in the plan and approved by a human.

**Configuration.** Each service MUST read `application.properties` with default values containing NO secrets.
Precedence: environment variable, then `application-<profile>.properties` (profile in `APP_PROFILE`), then
`application.properties`, then the code default. The environment variable derives from the key in uppercase with
`.` and `-` replaced by `_` (`server.port` becomes `SERVER_PORT`); `PORT` is an alias of `server.port`. Secrets
(`db.url`, `jwt.secret`) MUST be empty in files and supplied only through the environment.
`application-local.properties` MUST be git-ignored. A missing file falls back to defaults with a warning; a
malformed line MUST stop the service from starting. The reader MUST use only the standard library, and each service
has its own copy. `backend-go` owns the reader and the files; `devops` only injects environment variables.

*Rationale*: Limiting the stack reduces dependencies and standardizes maintenance, testing and the integration of
new agents; layered configuration keeps secrets out of the repository.

### IV. Modular Design and Constructor Injection
Go code MUST be organized into modules by functionality under internal/<module>, exposing a Register(mux) function
that registers its routes. Dependencies MUST be injected manually through constructors that return interfaces.
Domain sentinel errors MUST be translated to HTTP responses in the handlers. Logging MUST use log/slog in JSON format.

*Rationale*: inversion of dependencies through interfaces makes the domain testable and decouples the HTTP
transport from the business logic.

### V. Test-First (NON-NEGOTIABLE)
TDD is mandatory for unit tests, focused on the observable behavior of the component; the red-green-refactor cycle
MUST be respected. Go unit tests MUST live next to the code they test, in internal/<module>/*_test.go. Every
Gherkin scenario (success, edge and error) MUST have an integration test against the API, always through the
gateway, written before implementation (Playwright Test in API mode). Integration tests live in the hubx-bender
workspace defined in Principle II and run with `make test-integration`. Browser E2E tests with Playwright are
OPTIONAL and added when time allows. No story is done until all its automated tests pass.

*Rationale*: Writing tests first establishes the expected behavior and keeps coverage from becoming a post-delivery
activity; a single QA workspace isolates Node dependencies from the backend and gives agents a predictable
location to write and run tests.

### VI. Non-Negotiable Security
ALL requests MUST be validated on the server, and ALL data queries MUST be parameterized. Authorization MUST use
JWT with the roles "operator", "ground partner" and "customer", and ALL access to resources MUST be restricted to
one's own resources. A caller who cannot see a resource gets the same answer as if it did not exist. Secrets MUST
be provided exclusively via environment variables. Passwords MUST be hashed with bcrypt. Security headers,
restrictive CORS and rate limiting MUST be applied at the gateway. gosec and govulncheck MUST pass without findings
before merging. Security reviews MUST use the OWASP API Security Top 10 as checklist.

*Rationale*: These controls cover critical attack surfaces, and their automated verification prevents silent
regressions.

### VII. Language
Backlog, specifications, code, identifiers, messages and commits MUST be written in English, using the backlog
glossary (shipment, stage, update, fingerprint, responsible party, ground partner). The interface language is
defined in each specification.

*Rationale*: one shared vocabulary prevents agents and people from mixing terms between code, tests and specs.

## Development Process

User stories MUST be written in the "As a / I want / so that" format, be as small as possible while maintaining
value, comply with the 3Cs and INVEST, and include BDD Gherkin acceptance criteria with success, edge and error
cases, written in business language without technical or interface terms.

ALL functionality MUST go through the phases: specify, clarify, plan, tasks, implement and analyze. Artifacts MUST
be located in specs/NNN-name/. Specifications MUST describe the WHAT and the WHY, NOT the HOW, and open with their
source (`Source: backlog HU-xx` or `Source: user description`). Cross-cutting rules and the glossary live once in the
backlog and are referenced, not copied.

**Agents.** The team MUST be organized into five agents, each with its own folders: hubx-lead (orchestrator),
backend-go, frontend-react, qa-seguridad and devops. Commits MUST be tagged `[agent:role]`, one per subagent
delivery. QA MUST write the integration tests before implementation and MUST review before integration. Only
hubx-lead may commit, push the story's feature branch and open its pull request; it does so without asking, because
the pull request is the human review point. No agent merges: merging a feature branch into the release branch, and a
release or hotfix branch into master/main, MUST be done by a human after review. No agent force-pushes, deletes
branches or pushes to master/main, release or hotfix branches.

## Integration and Deployment

Each feature MUST be developed on its own branch with a mandatory pull request. CI MUST be green before any merge,
and master/main MUST always remain deployable.

The branching model MUST follow this flow: a release branch is created from master; feature branches MUST be merged
into that release branch; the release branch MUST be merged into master; hotfixes MUST originate from master and be
merged into master. The per-story branch created by Spec Kit is the feature branch of this flow. Configuration MUST
differ by environment using the `application.properties` mechanism defined in Principle III.

## Governance

This charter takes precedence over any other project practice; in the event of a conflict, the provisions defined
here prevail. Service-level AGENTS.md files add particulars and MUST NOT contradict it.

Amendments MUST be documented in a proposal, approved, and include a migration plan; the impact MUST be recorded in
the Sync Impact Report in this file.

Versioning MUST follow Semantic Versioning: MAJOR for removals or incompatible redefinitions of principles; MINOR
for new principles or sections and material extensions; PATCH for clarifications, corrections and non-semantic
refinements.

ALL PRs and reviews MUST verify compliance with this constitution. Any added complexity MUST be explicitly justified.

**Version**: 1.3.0 | **Ratified**: 2026-11-17 | **Last Amended**: 2026-11-21

<!-- Sync Impact Report (1.2.1 -> 1.3.0, MINOR): Development Process / Agents: hubx-lead commits, pushes the story
branch and opens the PR without human confirmation; no agent merges (feature -> release, release/hotfix -> main),
force-pushes, deletes branches or pushes to protected branches. Proposal: PR #7. Migration: hubx-lead permissions,
AGENTS.md section 8, README, team guide and memory.md updated in the same PR; release ruleset raised to 1 approval so
GitHub enforces the human merge.
(1.2.0 -> 1.2.1, PATCH): VI now requires the OWASP API Security Top 10 as review checklist.
(1.1.0 -> 1.2.0, MINOR)
- III: MySQL replaced by PostgreSQL; added Database, Dependencies and Configuration subsections.
- V: translated to English; browser E2E made optional; API integration tests via gateway; make test-integration.
- I and VI: "event/hash/actor" aligned to backlog glossary; roles renamed; signing clarified as future extension.
- VII Language: new principle. Development Process: Agents paragraph, source line in specs.
- Templates to review: plan, spec and tasks templates must mention these principles.
-->

# Hubx Constitution

## Core Principles

### I. Immutable Ledger as a Source of Truth
The system is an air cargo traceability platform with ground partners whose
canonical state is a ledger of events chained via SHA-256 hashes and signed by the
party that issues them. Each event MUST include the SHA-256 hash of the preceding event,
so that the chain is verifiable from end to end. Every event MUST be immutable:
it is NOT updated or deleted; corrections are recorded as new events. The
system MUST use a verifiable ledger instead of an on-chain blockchain during the
current scope of the project, and ALL access to the ledger MUST be performed through the
LedgerRepository interface to enable future migration to a live network.

*Rationale*: Immutability and per-actor signing ensure end-to-end
auditability; the repository abstraction prevents coupling the domain to a specific
persistence technology.

### II. Three-Tier Architecture and Mandatory Gateway
The system MUST consist of hubx-gw (gateway), hubx-api (base API), and hubx-mfe
(frontend). The frontend MUST NOT communicate directly with the API: ALL traffic
MUST pass through the gateway. The API MUST require the gateway’s shared secret and MUST
validate the JWT on its own, independently. The hubx-bender automation workspace
MUST remain outside the three components: it is a QA tool, not a
running component.

*Rationale*: the gateway concentrates the perimeter control; the own JWT validation in
the API avoids delegating the authentication to a single point of failure.

### III. Technology Stack and Restrictions
The backend MUST be implemented using the most recent stable version of Go with the
standard library (net/http with ServeMux) for routing; external web frameworks MUST NOT be
introduced. Data persistence MUST use MySQL. The frontend MUST use React with
TypeScript in strict mode. Configuration MUST be managed in the
application.properties file, with environment-specific values as outlined in the Integration and
Deployment section. External dependencies unrelated to routing MUST be limited to those
required by this specification (MySQL driver, bcrypt, JWT) and MUST be justified.

*Rationale*: Limiting the stack reduces the scope of dependencies and standardizes
maintenance, testing, and the integration of new agents.

### IV. Modular Design and Constructor Injection
Go code MUST be organized into modules by functionality under internal/<module>,
exposing a Register(mux) function that registers its routes. Dependencies MUST
be injected manually through constructors that return interfaces. Domain sentinel errors MUST
be translated to HTTP responses in the handlers. Logging MUST be performed with log/slog in JSON format.

*Rationale*: the inversion of dependencies through interfaces makes the domain testable and
decouples the HTTP transport from the business logic.

### V. Test-First (NO NEGOCIABLE)
TDD is mandatory for unit tests, focused on the observable behavior
of the component; the red-green-refactor cycle MUST be respected. Unit tests
for Go DEBEN residir junto al código que prueban, en internal/<modulo>/*_test.go. Cada
escenario Gherkin (exitoso, borde y error) DEBE tener una prueba de integración contra la
API. Ninguna historia se da por terminada sin pruebas automatizadas de extremo a extremo
con Playwright en TypeScript, y todas DEBEN pasar en verde. Las pruebas de integración y
las pruebas E2E DEBEN residir en el workspace de automatización hubx-bender definido en
el principio II.

*Rationale*: Writing tests first establishes the expected behavior and prevents
test coverage from becoming an activity that takes place after delivery; a single QA workspace
keeps Node dependencies isolated from the backend and provides agents with a predictable
location to write and run tests.

### VI. Non-Negotiable Security
ALL requests MUST be validated on the server, and ALL data queries MUST be
parameterized. Authorization MUST use JWT with the roles “operator,” “partner,” and “client,” and
ALL access to resources MUST be restricted to one’s own resources. Secrets MUST
be provided exclusively via environment variables. Passwords MUST
be hashed with bcrypt. Security headers, restrictive CORS, and rate
limiting MUST be applied at the gateway. gosec and govulncheck MUST pass without any findings before merging.

*Rationale*: These controls cover critical attack surfaces, and their automated verification prevents silent regressions.

## Development Process

User stories MUST be written in the "How/I Want/To" format, be as
small as possible while maintaining value, comply with the 3Cs and INVEST, and include
BDD Gherkin acceptance criteria with success, edge, and error cases.

ALL functionality MUST go through the phases: specify, clarify, plan, tasks, implement, and
analyze. Artifacts MUST be located in specs/NNN-name/. Specifications MUST
describe the WHAT and the WHY, NOT the HOW.

The team MUST be organized into five agents, each with their own folder: leader/orchestrator,
backend-go, frontend-react, qa-security, and devops. Commits MUST be tagged
[agent:role]. QA MUST write the integration tests before implementation and
MUST review them before integration.

## Integration and Deployment

Each feature MUST be developed on its own branch with a mandatory pull request. CI MUST be
green before any merge, and master/main MUST always remain deployable.

The branching model MUST follow this flow: a release branch is created from master; feature
branches MUST be merged into that release branch; the release branch MUST be merged into master;
hotfixes MUST originate from master and be merged into master. Configuration MUST differ by
environment using the application.properties mechanism defined in Principle III.

## Governance

This charter takes precedence over any other project practice; in the event of a
conflict, the provisions defined here prevail.

Amendments MUST be documented in a proposal, approved, and include a
migration plan; the impact MUST be recorded in the Sync Impact Report in this file.

Versioning MUST follow Semantic Versioning: MAJOR for removals or incompatible
redefinitions of principles; MINOR for new principles or sections and material
extensions; PATCH for clarifications, corrections, and non-semantic refinements.

ALL PRs and reviews MUST verify compliance with this constitution. Any added complexity
MUST be explicitly justified.

**Version**: 1.1.0 | **Ratified**: 2026-11-17 | **Last Amended**: 2026-11-17

# QA Report: Gateway perimeter hardening (story 011)

**Source**: user description (no HU, no backlog Gherkin)  
**Agent**: qa-seguridad  
**Phase**: red (T003, T004). Verify (T011) not run yet.  
**Branch**: `011-gateway-hardening`

## Red phase

### Artifacts

- `hubx-bender/features/S011_gateway-hardening.feature`: the 11 scenarios of `spec.md`, verbatim, with tags. A leading
  comment says they derive from the user description and are not backlog text.
- `hubx-bender/features/steps/S011_gateway-hardening.steps.ts`.

### Design of the tests

- Limits are proven against the limited gateway (`BENDER_LIMITED_BASE_URL`, default `http://localhost:8090`; 5 requests
  per minute per user, 3 sign-ins per minute per email). Everything else uses the main gateway (`world.api`).
- Rerunnable within the same minute: every user is a minted HS256 token (`JWT_SECRET`, random UUID `sub`, role
  `ground partner`) calling `GET /api/assigned-shipments`; sign-in scenarios use random unregistered emails. The one
  shared email (`express@hubx.co`, registered) loops up to 4 attempts until a 429 so earlier attempts are tolerated.
- "signed in at the limited entry point" mints the session instead of calling sign-in, so no sign-in counter is consumed.
  Not verified: that a real sign-in through the limited gateway behaves the same.
- Header checks use `headersArray()`: each header exactly once with the exact contract value.
- The `@e2e` scenario uses only the browser (no `this.api`); it collects the page `response` events of `/api/` URLs.

### Command

`make test-integration` (pass 1 = `--tags "not @empty-db"`, pass 2 = `--tags "@empty-db"`). The make output is truncated
by the tool, so the summaries below were taken from Cucumber's `summary` formatter, written to a temporary file by a
temporary one-line change of `hubx-bender/cucumber.mjs` that has been reverted (`git diff` shows no change to it).

Pass 1, literal:

```
1 hook (1 passed)
87 scenarios (78 passed, 9 failed)
550 steps (536 passed, 5 skipped, 9 failed)
1m 39.090s (1m 38.900s executing your code)
```

Pass 2, literal:

```
1 hook (1 passed)
2 scenarios (2 passed)
12 steps (12 passed)
0m 1.193s (0m 1.162s executing your code)
```

`make test-integration` exit code: 2 (nine failures, all in the new feature). No undefined, ambiguous or pending steps
were reported. All 78 passing scenarios of pass 1 include every earlier story and the HU-09 scenarios.

### Failing scenarios and reasons (all the right reason: the perimeter does not exist)

| # | Scenario (`features/S011_gateway-hardening.feature`) | Reason, literal |
|---|---|---|
| 1 | :6 A signed-in user exceeds the request limit | `request 6: status` Expected: 429 Received: 200 |
| 2 | :13 One user's excess does not affect another user | Given step: `request 6: status` Expected: 429 Received: 200 |
| 3 | :19 Pretending to come from another address does not avoid the limit | Given step: `request 6: status` Expected: 429 Received: 200 |
| 4 | :32 Too many sign-in attempts on one email | `normalised sign-in attempt: status` Expected: 429 Received: 401 |
| 5 | :38 The limit does not reveal whether an account exists | `No 429 after 4 sign-in attempts of express@hubx.co: sign-in is not limited` |
| 6 | :47 Every answer carries the browser protections | `success (200): x-content-type-options exactly once` Expected length: 1 Received length: 0 |
| 7 | :51 An approved origin may read the answers | `preflight status` Expected: 204 Received: 401 |
| 8 | :55 An origin that is not approved may not read the answers | `preflight from an origin that is not approved` Expected: 403 Received: 401 |
| 9 | :60 An operator keeps working through the protected entry point | `POST /api/auth/sign-in (200): x-content-type-options exactly once` Expected length: 1 Received length: 0 |

Observations that confirm the test assumptions: in scenarios 1-3 requests 1 to 5 of a minted user returned 200, so
`GET /api/assigned-shipments` answers 200 for an unknown partner and the minted token is accepted by the gateway and
the API. In scenario 9 the browser part up to "they see the total for each stage in the browser" passed (sign-in, count
by stage, six numeric totals); only the protections fail.

### Scenarios that pass in red (no production code can make them red)

- :25 "An invalid session never consumes a user's limit" and :43 "Checking health is never limited": today nothing is
  limited, so they pass trivially. They become meaningful only once the limiter exists; they are regression guards
  against the implementation counting forged requests or limiting health. Reported so the caller does not read them as
  evidence of the feature.

### Pass 2

Not touched by story 011 (`@empty-db` only); 2 of 2 passed.

## Part B (HU-09 correction)

The `@e2e` branch of "they see the total for each stage" now asserts the browser totals (see
`specs/009-view-count-by-stage/qa-report.md`, section "Correction (2026-11-25)"). The HU-09 `@e2e` scenario "Count by
stage" is among the 78 passing scenarios of pass 1 of the run above (no HU-09 scenario is in the list of failures).

## Verify phase

Not run: T011 (full suite after implementation, go tests, gosec, govulncheck, OWASP table) is pending.

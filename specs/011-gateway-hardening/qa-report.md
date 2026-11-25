# QA Report: Gateway perimeter hardening (story 011)

**Source**: user description (no HU, no backlog Gherkin)  
**Agent**: qa-seguridad  
**Phase**: red (T003, T004) and verify (T011, see "Verify phase").  
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

## Verify phase (T011)

**Run by**: qa-seguridad, 2026-11-25, on branch `011-gateway-hardening` with the implementation commit `6d34d4d`.
Every command below was run by me in this session; outputs are literal excerpts.

**Verdict**: all 87 scenarios pass, no regression, security tools clean, no critical/high finding. Open items are
medium/low/informational (see Findings) and the deploy validation, pending a human.

### Integration (`make test-integration`)

Exit status: success (the target printed both summaries and no failure). Stack included the limited gateway
(`healthy: http://localhost:8090`).

Pass 1 (`--tags "not @empty-db"`), literal:

```
1 hook (1 passed)
87 scenarios (87 passed)
550 steps (550 passed)
1m 44.343s (1m 44.162s executing your code)
```

Pass 2 (`--tags "@empty-db"`), literal:

```
1 hook (1 passed)
2 scenarios (2 passed)
12 steps (12 passed)
0m 1.181s (0m 1.158s executing your code)
```

- Red run had 87 scenarios (78 passed, 9 failed); the same 87 now all pass, so the 11 story 011 scenarios (the 9 failing
  plus the 2 guards at :25 and :43) pass and nothing regressed. The per-scenario list was not printed by the progress
  formatter; the claim rests on the totals (87 = 87, 0 failed) and on the red table above. Not verified individually in
  `cucumber.json`.
- HU-09 `@e2e` "Count by stage" is inside the 87 passed (0 failed). It now asserts the browser totals (Part B).
- The 011 `@e2e` (:60) passes: it drives the browser only and checks the protection headers of the `/api/` responses the
  browser received.
- Test-only limits: main test gateway 10000/min, limited gateway 5 users and 3 sign-in per minute (spec A8).

### Unit tests

```
go -C hubx-api test -count=1 ./...  -> every package with tests: ok (auth, config, evidence, handlers/auth,
  handlers/evidence, handlers/health, handlers/shipment, ledger, migrate, shipment); no FAIL line
go -C hubx-gw test -count=1 ./...
ok  	hubx-gw/internal/config/test	0.157s
ok  	hubx-gw/internal/middleware/test	0.197s
ok  	hubx-gw/internal/proxy/test	0.210s
ok  	hubx-gw/internal/ratelimit/test	0.175s
ok  	hubx-gw/internal/server/test	0.271s
```

### Security tools

`gosec ./hubx-gw/... ./hubx-api/...` and plain `gosec ./...` (AGENTS.md form) gave the same result; no package loading
errors, the new gateway files were scanned (`Checking file: .../hubx-gw/internal/middleware/ratelimit.go`, `cors.go`,
`security.go`, `ratelimit/limiter.go`, `config/config.go`, `server/server.go`, `proxy/proxy.go`):

```
Summary:
  Gosec  : dev
  Files  : 83
  Lines  : 8293
  Nosec  : 2
  Issues : 0
```

`#nosec` count is 2 (HU-09 report had 1). One is the documented G304 in `hubx-gw/internal/config/config.go:208`; I did
not locate the second one, so I do not vouch for it.

```
govulncheck -C hubx-api ./...
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 1
vulnerability in modules you require, but your code doesn't appear to call these
vulnerabilities.

govulncheck -C hubx-gw ./...
No vulnerabilities found.
```

The one module-level vulnerability in hubx-api is the same unreachable `golang.org/x/crypto/openpgp` item recorded in
the HU-09 report; `-show verbose` was not rerun, so its identity is not re-verified here.

### Code review of the perimeter (read, not fixed)

Files read: `hubx-gw/internal/{middleware/{security,cors,auth,ratelimit}.go, ratelimit/limiter.go, server/server.go,
proxy/proxy.go, config/config.go}`, `cmd/gw/main.go`, and `hubx-api/internal/handlers/auth/handler.go` for the parser.

- **Order**: `server.go:53-57` builds SecurityHeaders -> CORS -> JWT -> RateLimit -> mux, as the contract says.
- **Path variants**: the JWT exclusions are exact string matches on `r.URL.Path` (`server.go:43-47`, `auth.go:67`).
  `/api/auth/sign-in/`, `/api//auth/sign-in`, `/api/health/` are not excluded, so they get the 401 (fail closed). No
  variant skips the JWT. The limiter's sign-in test is the same exact path and `POST` (`ratelimit.go:52`); a variant
  that reaches the API's sign-in handler would have to pass the same mux match, and the percent-encoded form decodes to
  the same `Path` in the gateway and in the API's Go mux. Not tested against a real request with `%2D`: not verified.
- **Parser differential (sign-in)**: gateway and API decode with `json.NewDecoder(...).Decode` into the same two-field
  struct (`ratelimit.go:30-33`, `handler.go:14-17`), so key case-insensitivity, duplicate keys (last wins) and trailing
  data are handled identically; both apply `TrimSpace` and lowercase for the email (the API's `NormalizeEmail` was not
  read, only the handler's `TrimSpace`; not verified). The body is restored byte for byte, with length reset
  (`ratelimit.go:77-80`). No differential found.
- **Header trust**: the user key is the `sub` set in the request context by the JWT middleware after validation
  (`auth.go:111`, `ratelimit.go:43`); no `X-Forwarded-For`, remote address or `Authorization` text is read by any
  limiter. The proxy overwrites `X-Gateway-Secret` (`proxy.go:53`).
- **Limiter**: refused requests are not stored (`limiter.go:75-77`), the window slides, mutex-protected, LRU bounded to
  10000 keys per limiter, two independent limiters. Sign-in keys are SHA-256 of the email, never logged.
- **CORS**: exact match, one `Origin` value only (multiple values -> not allowed), no credentials header anywhere, a
  preflight is answered before JWT and limiter (`cors.go:29-39`), `Vary: Origin` always. The config rejects `*`, `null`,
  paths, userinfo and non-http(s) schemes (`config.go:184-202`).
- **Headers**: set before any later handler (`security.go:21-25`) and the proxy deletes the API's copies and every
  `Access-Control-*` (`proxy.go:60-70`), so 401/404/413/429/502 carry them once.

### Findings

| # | Severity | OWASP | Description | Location |
|---|---|---|---|---|
| F1 | medium (residual risk, assumption A1 pending human) | API2/API4 | Sign-in limit is 120 attempts/minute per email: about 172,800 guesses per day per email. It is a brake against floods, not against password guessing. Changing it is configuration (`ratelimit.sign-in-requests-per-minute`); a lower value makes locking a victim out cheaper (known limitation). | `hubx-gw/application.properties`, `config.go:46-47` |
| F2 | low | API4 | Sign-in limiter forgets a count when more than 10000 distinct emails arrive between two attempts (LRU), so a determined attacker can reset one victim's counter. Documented in the spec's known limitations. | `limiter.go:89-93` |
| F3 | low | API4 | `http.Server` has only `ReadHeaderTimeout`; no `ReadTimeout`, so a client that drips a sign-in body (up to 8 KiB) holds a connection and goroutine indefinitely. Pre-existing server setup, newly relevant because the gateway now reads the sign-in body. Not exercised by a test. | `cmd/gw/main.go:29-34`, `ratelimit.go:67` |
| F4 | low | API8 | The default allowed origin is `http://localhost:5173` in every environment without an override (assumption A3). It grants read permission without credentials to a local dev page only; no effect on HUBX itself (same-origin through the MFE). | `config.go:45` |
| F5 | informational | API2 | Pre-existing, out of scope of 011 and I agree it stays out: JWT validation accepts any HMAC algorithm (HS256/HS384/HS512), not only HS256, and does not require `exp`. Both need the shared secret to forge, so no bypass. Candidate for a later hardening story. | `auth.go:128-136` |
| F6 | informational | API4 | Limits are in memory per gateway instance: they reset on restart and each instance counts separately; two instances double the effective limit. | spec "Known limitations" |

No critical or high finding. Not verified: behaviour with more than one gateway instance, real sign-in through the
limited gateway (the test mints sessions, see Red phase), slow-body behaviour (F3), `X-Forwarded-For` as received by the
API (the proxy appends the client address; the API was not checked to ignore it).

### OWASP API Security Top 10 (2023)

| Category | Test or evidence | Result |
|---|---|---|
| API1 BOLA | Not changed by 011; the gateway adds no object access. Earlier stories' scenarios (all in the 87) still pass. | Not applicable to 011 |
| API2 Broken authentication | Scenario :25 (forged session refused 401 and consumes no limit), :32/:38 (sign-in brake, normalised email), `auth.go:104` (empty `sub` -> 401). Residual: F1 (A1 value weak against guessing) and F5. | Pass with residual risk F1 |
| API3 Property level exposure | Not changed by 011; the 429 body is fixed and identical for registered and unregistered emails (scenario :38). | Pass for the gateway part |
| API4 Unrestricted resource consumption | Scenarios :6, :13, :19 (429 on request 6 of 5, per user, address header ignored), :32 (sign-in), :43 (health never limited); `limiter.go` bounded memory, `ratelimit.go:67` 8 KiB body bound. Previous open finding (HU-02 to HU-09 reports) is **closed** by this evidence. Limit is in memory, per gateway instance (F6); residual F2, F3. | Closed, with the stated limits |
| API5 Function level authorization | Not changed by 011. | Not applicable |
| API6 Sensitive business flows | Sign-in brake covers the credential-guessing flow only (F1). | Partial, see F1 |
| API7 SSRF | The gateway forwards to one configured upstream; no URL taken from the request. | Not applicable |
| API8 Security misconfiguration | Scenarios :47 (six headers exactly once, 200/401/health), :51 (approved origin: 204, that origin only, no credentials), :55 (other origin: 403, no permission), :60 (`@e2e`, headers seen by the browser); `config.go` rejects `*`, `null` and invalid values at start; `proxy.go:60-70` removes API duplicates. Previous open finding (no headers, no CORS) is **closed**. Residual F4. Scope: the MFE's own HTML pages are served outside the gateway and are not covered (out of scope per spec). | Closed for the gateway; MFE HTML out of scope |
| API9 Improper inventory | `contracts/gateway-perimeter.md` documents the perimeter; the second (limited) gateway exists only in the test stack. | Pass |
| API10 Unsafe consumption | The gateway consumes only its own API; sub-requests and responses are not interpreted. | Not applicable |

### 429 and spec assumptions

The status `429`, `Retry-After` and the flat body in the tests are the contract's assumptions A2 (and A1, A3 to A9),
**pending human confirmation**; the tests assert the contract as written and would need to change with the answers. Not
decisions of the human.

### Boundaries

No import between services was found in the files read (`hubx-gw` imports only its own packages and `golang-jwt`); no
business logic in the gateway beyond reading the sign-in email for the limit; the MFE reaches the API only through
`/api` (the `@e2e` scenarios pass through port 3000). Full import scan with a tool: not run (shell restricted).

### Deploy validation

**Pending human confirmation.** Not run by QA: staging deploy and smoke belong to `devops` and a human. Nothing in this
report states that staging was verified.

### Gherkin coverage (feature `S011_gateway-hardening.feature`)

| Scenario | Line | Mode | Result |
|---|---|---|---|
| A signed-in user exceeds the request limit | 6 | API | pass (87/87) |
| One user's excess does not affect another user | 13 | API | pass |
| Pretending to come from another address does not avoid the limit | 19 | API | pass |
| An invalid session never consumes a user's limit | 25 | API | pass |
| Too many sign-in attempts on one email | 32 | API | pass |
| The limit does not reveal whether an account exists | 38 | API | pass |
| Checking health is never limited | 43 | API | pass |
| Every answer carries the browser protections | 47 | API | pass |
| An approved origin may read the answers | 51 | API | pass |
| An origin that is not approved may not read the answers | 55 | API | pass |
| An operator keeps working through the protected entry point | 60 | `@e2e` browser | pass |

Per-scenario status is inferred from 0 failures of 87 (see above), not from a per-scenario listing.

---

## Red for the decisions of 2026-11-25 (task T016, phase `red`)

Added 2026-11-25. Nothing above was rewritten; the earlier text describes the earlier state (the 120/min sign-in
default, F1 and F4 and the "pending" notes of the assumptions predate the human's decisions A1 and A3).

### What changed in the tests

- `features/S011_gateway-hardening.feature`: synced verbatim with the spec. "An approved origin may read the answers"
  now carries `@needs-limited-gateway`; new scenario "No origin is approved unless it is configured" (line 60).
- `features/steps/S011_gateway-hardening.steps.ts`: the approved-origin preflight goes to the limited gateway
  (`gw-limited`, `CORS_ALLOWED_ORIGINS=http://localhost:5173` in `compose.yaml:70`). The new scenario sends the same
  `http://localhost:5173` preflight and a plain `GET /health` with that `Origin` to the main gateway and reuses the
  existing step "no permission is granted" (403, no `Access-Control-*` on both answers). No new ambiguous step: the
  new `When` text is unique.
- `features/steps/HU10_download-evidence.steps.ts`: the step "they receive a document with the shipment's data" (used by
  the updates and the empty-chain scenarios) now asserts that the text contains each of the three names sentences of
  `contracts/hu10-download-evidence.md` item 6, contiguous. The extractor joins PDF literal strings with a newline, so a
  sentence found whole lies in one literal string. The HU-10 feature file is untouched.
- `hubx-bender/AGENTS.md`: documents the production defaults against `make up` and the exclusion of
  `@needs-limited-gateway` there.

### Run (literal, `make test-integration`, exit 2 as expected for red)

```
Failed scenarios:
  1) Evidence for a shipment with updates # features/HU10_download-evidence.feature:7
       Then they receive a document with the shipment's data # features/steps/HU10_download-evidence.steps.ts:228
           Error: names statement: "Fingerprints cover the party identifiers, not the printed names."
  2) Evidence for a shipment without updates # features/HU10_download-evidence.feature:20
       Then they receive a document with the shipment's data # features/steps/HU10_download-evidence.steps.ts:228
           Error: names statement: "Fingerprints cover the party identifiers, not the printed names."
  3) No origin is approved unless it is configured # features/S011_gateway-hardening.feature:60
       Then no permission is granted # features/steps/S011_gateway-hardening.steps.ts:400
           Error: preflight from an origin that is not approved
           Expected: 403
           Received: 204

88 scenarios (85 passed, 3 failed)
555 steps (549 passed, 3 skipped, 3 failed)
2 scenarios (2 passed)        <- pass 2, @empty-db
```

Reasons, all the right ones: the PDF has the `Scope:` text of two sentences and none of the three names sentences
(the extracted text is quoted in the run, ending "...did not rebuild the whole history."); the gateway code default
still allows `http://localhost:5173` (the main gateway logged `"allowed_origins":1` at start), so the preflight is `204`.
The relocated approved-origin scenario passes on `gw-limited` (`"allowed_origins":1` there, set by environment). No
undefined or ambiguous step, no step error. Everything else passed.

### Not run / open

Verify phase (T019) is not part of this entry: go tests, gosec and govulncheck were not rerun. F1 and F4 above are
still described as of the earlier run; the implementation of defaults 10 and empty (T017) is pending.

---

## Final verify (2026-11-25), task T019

Supersedes the "Verify phase (T011)" section above where they differ (F1, F4, the "pending" notes of the assumptions,
the "not verified" notes on `%2D` and `NormalizeEmail`, the `#nosec` count). The earlier text is kept unchanged. Run by
qa-seguridad on branch `011-gateway-hardening` with both implementations committed (`1f0d1ce` backend-go defaults,
`b81585d` names statement). Every command below was run by me in this session; outputs are literal excerpts.

**Verdict**: all 90 scenarios pass (88 + 2), unit tests green, gosec 0 issues, govulncheck clean for both services, no
critical or high finding open. Deploy validation is still pending a human (see below).

### Integration (`make test-integration`)

Pass 1 (`--tags not @empty-db`), literal:

```
1 hook (1 passed)
88 scenarios (88 passed)
555 steps (555 passed)
1m 44.682s (1m 44.505s executing your code)
```

Pass 2 (`@empty-db`), literal:

```
1 hook (1 passed)
2 scenarios (2 passed)
12 steps (12 passed)
0m 1.201s (0m 1.181s executing your code)
```

The tamper regression line was printed in pass 1: `Tampered update 2 of HBX-45764AED to stage "Out for delivery"`.
The red run of T016 had 88 scenarios (85 passed, 3 failed); the same 88 now all pass, so the three red failures (HU-10
names statement x2, "No origin is approved unless it is configured") are green. The per-scenario list was not printed by
the progress formatter and `hubx-bender/reports/cucumber.json` holds only pass 2, so no per-scenario listing was seen:
the claim rests on the totals (88 of 88, 0 failed) and on the three named red failures. The `make` output showed
`healthy: http://localhost:8080`, `http://localhost:8090` and `ui ready: http://localhost:3000`.

### Unit tests, security tools (literal)

```
go -C hubx-api test -count=1 ./...
ok  	hubx-api/internal/auth/test	1.885s
ok  	hubx-api/internal/config/test	0.319s
ok  	hubx-api/internal/evidence/test	0.342s
ok  	hubx-api/internal/handlers/auth/test	0.383s
ok  	hubx-api/internal/handlers/evidence/test	0.369s
ok  	hubx-api/internal/handlers/health/test	0.409s
ok  	hubx-api/internal/handlers/shipment/test	0.398s
ok  	hubx-api/internal/ledger/test	0.318s
ok  	hubx-api/internal/migrate/test	0.283s
ok  	hubx-api/internal/shipment/test	0.205s

go -C hubx-gw test -count=1 ./...
ok  	hubx-gw/internal/config/test	0.253s
ok  	hubx-gw/internal/middleware/test	0.170s
ok  	hubx-gw/internal/proxy/test	0.159s
ok  	hubx-gw/internal/ratelimit/test	0.129s
ok  	hubx-gw/internal/server/test	0.226s

gosec ./...
Summary:
  Gosec  : dev
  Files  : 83
  Lines  : 8297
  Nosec  : 2
  Issues : 0

govulncheck -C hubx-api ./...
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 1
vulnerability in modules you require, but your code doesn't appear to call these
vulnerabilities.

govulncheck -C hubx-gw ./...
No vulnerabilities found.
```

No FAIL line in either `go test` run. The module-level item in hubx-api is the same unreachable one as before;
`-show verbose` was not rerun, so its identity is not re-verified here.

### Corrections to the earlier "not verified" notes (dated additions, read 2026-11-25)

- **`#nosec` count 2**: both are G304 on `os.Open(path)` in the properties readers: `hubx-gw/internal/config/config.go:208`
  and `hubx-api/internal/config/config.go:105` (read both; each comment states the path is built from a hardcoded `.`
  directory, constant filenames and the operator-set `APP_PROFILE`). The earlier "I did not locate the second one" is now
  resolved. I did not run a repository-wide search for other `#nosec` lines (shell search denied); gosec itself counts 2.
- **`%2D` path**: `hubx-gw/internal/server/test/server_test.go:364-402`
  (`TestChain_SignInPathVariantsCannotBypassTheCount`) sends `POST /api/auth/sign%2Din` through the full chain after the
  email's single allowance is used and expects 429 (`:373-378`), and sends eight other variants that must not return 200
  (`/api/auth/sign-in/`, `/api//auth/sign-in`, `//api/auth/sign-in`, `/api/auth/./sign-in`, `/api/auth/x/../sign-in`,
  `/api/auth/Sign-In`, `/api/auth/sign-in%2F`, `/API/auth/sign-in`) and check the upstream received no extra request
  (`:381-401`). It passes in `go -C hubx-gw test`. This is a unit test of the chain with `httptest`, not a request through
  a running gateway and API: that part stays not verified.
- **`NormalizeEmail`**: read at `hubx-api/internal/auth/repository.go:63-65`: `strings.ToLower(strings.TrimSpace(email))`,
  used by `FindUserByEmail` (`:36`). The gateway's limiter normalises by trim plus lowercase
  (`TestChain_SignInLimitedPerNormalisedEmail`, `server_test.go:314-338`, `"  USER@HUBX.CO "` counts with
  `user@hubx.co`). The two are the same function in effect; I did not re-read the gateway limiter source in this run.
  Unicode edge cases of `ToLower` and `TrimSpace` were not tested.

### Review of the diff `git diff 5d67510 -- hubx-gw hubx-api compose.yaml deploy`

Nine files (165 insertions, 21 deletions), read in full for the non-test files:

- `hubx-gw/internal/config/config.go:42-48`: defaults are `allowedOriginsKey: ""` and `signInPerMinuteKey: "10"`;
  `hubx-gw/application.properties` matches (`cors.allowed-origins=` empty, `ratelimit.sign-in-requests-per-minute=10`).
- `compose.yaml`: the main gateway defaults to 10 sign-ins per minute (`:53`); only the limited test gateway gets
  `CORS_ALLOWED_ORIGINS: http://localhost:5173` (`:70`). The main gateway has no allowed origin.
- `hubx-api/internal/evidence/pdf.go:136-146`: three sentences added to the scope block: "Fingerprints cover the party
  identifiers, not the printed names.", "The covered fields are responsible_party_id and assigned_partner_id.", "The names
  cannot be verified from this document alone." They are written as separate wrapped lines (`wrapLine`), so a sentence can
  be split across lines only if longer than `proseWidth`; the Cucumber step asserting them contiguously passes (88 of 88).
- New gateway tests: sign-in 11th gets 429 and recovers after a minute (`server_test.go:196`), default allows no cross
  origin and returns 403 with no `Access-Control-*` (`:224`), a same-origin style POST carrying `Origin` is still
  processed (200, no `Access-Control-Allow-Origin`, `:250-264`).
- `deploy/README.md`: now says the Blueprint change reaches staging only when the release merges into `main`; until then a
  human sets both rate variables to `10000` on `hubx-gw-stg` by hand, otherwise defaults apply and the staging smoke can
  get `429`. I did not verify Render.

### Findings, updated

| # | Severity | OWASP | State | Description |
|---|---|---|---|---|
| F1 | low (residual, accepted by the human) | API2/API4 | Reassessed | Sign-in default is now 10 per minute per email (`config.go:47`, `application.properties`, `compose.yaml:53`), about 14,400 guesses per day per email at most, down from 172,800. Residual: an attacker can lock a victim's sign-in for the window by sending 10 attempts (cheap lockout; known limitation, accepted by the human 2026-11-25). Brake only, not a lockout policy. Assumption A1 is confirmed at 10. |
| F2 | low | API4 | Open, unchanged | LRU of 10000 emails can forget a counter. Documented in the spec. |
| F3 | low | API4 | Open, unchanged | No `ReadTimeout` on the gateway server (`cmd/gw/main.go:29-34`); not exercised by a test. |
| F4 | closed | API8 | Closed | Default allowed origin is empty in code and in `application.properties` (see diff), the main test gateway grants none (scenario "No origin is approved unless it is configured" passes, and `TestChain_DefaultsAllowNoCrossOrigin`). Only an environment value can grant one. |
| F5 | informational | API2 | Open, out of scope | JWT accepts any HMAC method and does not require `exp` (`auth.go:128-136`); needs the shared secret to forge. |
| F6 | informational | API4 | Open, known limitation | Limits are in memory per gateway instance. |

No critical or high finding. Behaviour accepted by the human 2026-11-25 and recorded as such, not as QA findings: empty
`sub` -> 401, invalid port and missing secret stop the start. A1 to A9 are no longer pending: the human confirmed them
on 2026-11-25 (A1 and A3 changed; the 429 status, `Retry-After` and flat body stand as confirmed). The "pending human
confirmation" notes in the earlier sections are superseded by this.

### OWASP API Security Top 10 (2023), final

| Category | Evidence | Result |
|---|---|---|
| API1 BOLA | Not changed by 011. HU-10 reader unchanged apart from the three sentences. All earlier scenarios pass (88/88). | Not applicable to 011 |
| API2 Broken authentication | Sign-in brake at 10 (`TestChain_RealPropertiesFileSignIn11thOfOneEmailGets429AndRecoversAfterOneMinute`, scenarios "Too many sign-in attempts on one email" and "The limit does not reveal whether an account exists"); forged session never consumes a limit. Residual F1, F5. | Pass with residual risk |
| API3 Property level exposure | The 429 body is fixed and identical for registered and unregistered emails. The evidence PDF now states the names are outside the fingerprint. | Pass |
| API4 Unrestricted resource consumption | Scenarios and chain tests for the per-user limit, the sign-in limit, health never limited, 8 KiB body bound. Residual F2, F3, F6. | Closed with the stated limits |
| API5 Function level authorization | Not changed. | Not applicable |
| API6 Sensitive business flows | Sign-in brake (F1). | Partial, see F1 |
| API7 SSRF | One configured upstream, no URL from the request. | Not applicable |
| API8 Security misconfiguration | Empty CORS default (F4 closed), six headers exactly once, preflight never forwarded, config rejects `*` and `null` (unit tests green). MFE static HTML is outside the gateway (out of scope per the spec). | Closed for the gateway |
| API9 Improper inventory | `contracts/gateway-perimeter.md`; the limited gateway exists only in the test stack (`compose.yaml:61-72`). | Pass |
| API10 Unsafe consumption | The gateway consumes only its own API. | Not applicable |

### Docs `docs/diagramas/*.md` (reported, not edited)

Read `01-contexto.md`, `02-arquitectura.md`, `03-dominio-ledger.md`, `04-flujos-aplicacion.md`, `06-devsecops-cicd.md`
against the code and contract read in this run. No false statement found about the perimeter, fingerprints or routes:
`02` states the chain "security headers -> CORS -> JWT -> request limit", empty CORS by default, 120 per minute per user
and 10 per minute per email (match `config.go:42-48`); `01` and `03` state the fingerprints cover identifiers, not names,
and "seven lines" (matches `pdf.go` and the evidence text); the route table in `02` matches the stories' routes; `06`
says the smoke excludes `@needs-secret`, `@empty-db` and `@needs-limited-gateway` (matches `deploy/README.md`). Not
verified: the `429 RATE_LIMITED` code name in `04` against the contract (I did not read `contracts/gateway-perimeter.md`
in this run); `05-flujos-agenticos.md` and `README.md` were not read (outside the requested scope).

### Boundaries

No import between services in the diff files read; no business logic added to the gateway; the MFE reaches the API only
through `/api` (the `@e2e` scenarios pass through port 3000). A tool-assisted import scan was not run (shell restricted).

### Not run and denied commands

- Denied by the agent guard (`.kiro/agents/qa-seguridad.json`), not worked around: `grep -n "nosec" -r ... --include=*.go`,
  `grep -rn -i "NormalizeEmail\|2D" ...`, `grep -n -i "not verified\|pending\|%2D\|NormalizeEmail" ...`, `ls
  /Users/jaigomez/source/hubx/docs/diagramas`, `find /Users/jaigomez/source/hubx/docs -name "*.md"`, `git ls-files docs`,
  `git show --stat --format=%s 22f7484`, `git show --stat 22f7484`, `tail -c 200 ...qa-report.md`. I used Read and `git diff
  --stat` instead (allowed forms).
- Not run: staging deploy and smoke, a real request with `%2D` through the running stack, behaviour with several
  gateway instances, slow-body (F3), the tamper test beyond the regression scenario's output line.

### Deploy validation

**Still pending a human.** Not run by QA. Nothing in this report states that staging was verified; per `deploy/README.md`
the staging Blueprint change for the 10000 limits reaches Render only after the release merges into `main`, so a human
must set the variables by hand first.

### Gherkin coverage (final, feature `S011_gateway-hardening.feature`)

Twelve scenarios now (the eleven above plus "No origin is approved unless it is configured", line 60; the `@e2e` scenario
moved down). Mode and results: all API scenarios and the `@e2e` browser scenario passed within 88 of 88; "An approved
origin may read the answers" runs against the limited gateway (`@needs-limited-gateway`). Per-scenario status is inferred
from zero failures of 88 and the three named red failures turning green, not from a per-scenario listing. I did not
re-read the feature file in this run, so the line numbers other than 60 are those of the earlier sections and may have
shifted by one.

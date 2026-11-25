# Feature Specification: Gateway perimeter hardening

Source: user description

**Feature branch**: `011-gateway-hardening` | **Created**: 2026-11-25 | **Status**: In progress
**Kind**: foundation story; it is not in the backlog and has no HU number.
**Intended release**: `release/v0.8.0-gateway-hardening` (NOT confirmed by the human and not created)

## Why

Constitution VI requires security headers, restrictive cross-origin access and rate limiting at the single entry point.
None of them exists: it is an open finding (OWASP API4 and API8) carried by the QA reports since HU-02. Without them one
caller can exhaust the platform, guess passwords without a brake, or use responses from pages that are not HUBX.

## User stories

### US1 - Fair use per person (P1)
As a HUBX user, I want nobody to be able to exhaust the entry point, so that HUBX keeps answering me.

### US2 - Brake on sign-in attempts (P1)
As a HUBX user, I want repeated sign-in attempts on one email to be limited, so that my password is hard to guess.

### US3 - Protected responses and origins (P2)
As a HUBX user, I want responses to carry browser protections and only approved sites to read them, so that another
site cannot abuse my session data.

## Acceptance scenarios

Derived from the user description. They are NOT backlog text.

```gherkin
Feature: Gateway perimeter hardening

  @needs-limited-gateway
  Scenario: A signed-in user exceeds the request limit
    Given a ground partner has signed in at the limited entry point
    When they make more requests in one minute than the limit allows
    Then the requests within the limit are answered
    And the request over the limit is refused and they are told when to try again

  @needs-limited-gateway
  Scenario: One user's excess does not affect another user
    Given a user has exhausted their request limit
    When a different signed-in user makes a request
    Then that request is answered

  @needs-limited-gateway
  Scenario: Pretending to come from another address does not avoid the limit
    Given a user has exhausted their request limit
    When they repeat the request claiming a different network address
    Then the request is refused and they are told when to try again

  @needs-limited-gateway
  Scenario: An invalid session never consumes a user's limit
    Given a user has signed in at the limited entry point
    When more requests than the limit arrive with a forged session of that user
    Then every forged request is refused as not signed in
    And the user's own request is answered

  @needs-limited-gateway
  Scenario: Too many sign-in attempts on one email
    Given the sign-in attempts of one email have been exhausted
    When another attempt is made with that email in capital letters and with extra spaces
    Then the attempt is refused and they are told when to try again

  @needs-limited-gateway
  Scenario: The limit does not reveal whether an account exists
    When the sign-in attempts of a registered email and of an unregistered email are exhausted
    Then both are refused with exactly the same answer

  @needs-limited-gateway
  Scenario: Checking health is never limited
    When health is checked more times than the limit allows
    Then every check is answered

  Scenario: Every answer carries the browser protections
    When answers are requested with success, without a session and for health
    Then every answer carries the browser protections

  Scenario: An approved origin may read the answers
    When a browser on an approved origin asks for permission to call HUBX without a session
    Then the permission is granted for that origin only, without credentials

  Scenario: An origin that is not approved may not read the answers
    When a browser on an origin that is not approved asks for permission to call HUBX
    Then no permission is granted

  @e2e
  Scenario: An operator keeps working through the protected entry point
    Given an operator opens HUBX in a browser
    When the operator signs in and views the shipment count by stage
    Then they see the total for each stage in the browser
    And the answers their browser received carry the browser protections
```

## Functional requirements

- **FR-001** Requests of a signed-in user are limited per user, identified only after the session was validated.
- **FR-002** The per-user limit is 120 requests per minute, configurable.
- **FR-003** Sign-in is limited per submitted email, normalised as HU-01 does (capital letters and outer spaces ignored).
- **FR-004** No limit uses the network address or any address header.
- **FR-005** A refused request says so, says when to retry, is not forwarded, and is identical whether or not the account exists.
- **FR-006** What sign-in reads before forwarding is bounded, and the forwarded request is unchanged.
- **FR-007** Limiter memory is bounded; forgotten identities are evicted; sign-in floods cannot evict signed-in users.
- **FR-008** Every answer, including refusals, carries the browser protections.
- **FR-009** Cross-origin reading is granted only to configured origins, never to any origin, never with credentials.
- **FR-010** A cross-origin permission request needs no session; every other non-public request still does.
- **FR-011** Health answers are never limited.
- **FR-012** Limits and origins are configuration, with no secrets; invalid values stop the start.

## Success criteria

- **SC-001** Request 121 of one user in one minute is refused; another user is unaffected.
- **SC-002** All existing acceptance scenarios still pass.
- **SC-003** Finding API4/API8 of earlier reports can be closed with evidence.

## Known limitations

- Limits live in memory per gateway instance: they reset on restart and each instance counts separately.
- Requests refused as not signed in have no identity, so they are not counted.
- A person who knows an email can exhaust its sign-in attempts for a minute.
- A flood of enough distinct emails can make the gateway forget one email's count.
- The MFE's own HTML pages are served outside the gateway and are not covered.

## Clarifications

### Session 2026-11-25 (decisions confirmed by the human)

- Q: By what is the limit counted? A: Per signed-in user, taken from the session only AFTER the gateway validated it:
  120 requests/minute, the value already in `hubx-gw/application.properties`. Sign-in has no user, so it is limited by
  the submitted email. Never by IP and never from the forwarded-address header: all browser traffic arrives through
  the MFE's server and the header can be forged.
- Q: How do the tests avoid being throttled? A: Through a higher limit set by configuration (environment) for
  `make test-integration` and the staging smoke; the limit itself is proven by tests.

### Assumptions pending human confirmation

Not human decisions. Each is the proposed value and can be changed without redesign.

- **A1** Sign-in limit: 120 attempts/minute per email, the only existing repository value. It is weak against password
  guessing; a lower value protects more but makes locking a victim out cheaper.
- **A2** Refusal: `429`, `Retry-After` in seconds, flat body `{"error":"RATE_LIMITED","message":"Too many requests. Please try again later."}`
  on every route, including the HU-03 routes whose other errors are nested.
- **A3** Production/staging origins: not set. They keep the file default `http://localhost:5173`; the MFE is same-origin
  through its server, so none is needed for HUBX to work.
- **A4** Methods `GET, POST`, headers `Authorization, Content-Type`, no credentials, no preflight caching; a refused
  preflight is `403`; a non-preflight request from another origin is processed but gets no permission header (the MFE's
  own same-origin POSTs carry an `Origin`, so refusing them would break HUBX).
- **A5** Header values in the contract, notably HSTS of one year without `includeSubDomains`, and `no-store`.
- **A6** Sign-in body bound: 8 KiB, above it `413` `PAYLOAD_TOO_LARGE`.
- **A7** Sliding one-minute window; only accepted requests count. At most 10000 remembered identities per limiter, the
  least recently seen evicted first.
- **A8** Test configuration: 10000/minute on the main test gateway and staging; a second gateway with 5 (users) and 3
  (sign-in), only in the test stack, proves the limits.
- **A9** Sign-in attempts without an email, or with an unreadable body, are forwarded for HU-01's own 400 and not counted.

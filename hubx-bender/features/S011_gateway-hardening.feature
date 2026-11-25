# Story 011 (foundation story, no HU number). These scenarios derive from the user description in
# specs/011-gateway-hardening/spec.md; they are NOT backlog text.
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

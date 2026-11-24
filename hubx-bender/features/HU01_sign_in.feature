Feature: Sign in
  As a HUBX user (operator, ground partner or customer),
  I want to sign in with my email and my password,
  so that I can access the functions that correspond to my role.

  @smoke
  Scenario Outline: Each role signs in
    Given a user with the role of <role> is registered in HUBX
    When that user signs in with their correct email and password
    Then they access HUBX with the functions of their role

    Examples:
      | role           |
      | operator       |
      | ground partner |
      | customer       |

    @e2e
    Examples:
      | role     |
      | operator |

  Scenario: Email typed with capital letters and extra spaces
    Given a customer is registered with the email "ana@hubx.co"
    When the customer signs in typing " ANA@hubx.co " and their correct password
    Then the customer accesses HUBX with the functions of their role

  @smoke
  Scenario: Wrong password
    Given a customer is registered in HUBX
    When the customer tries to sign in with an incorrect password
    Then they are told the credentials are not valid, without saying which piece of data failed
    And they do not access HUBX

  Scenario: Email not registered
    Given no user exists with the email "nobody@hubx.co"
    When a person tries to sign in with that email
    Then they are told the credentials are not valid, without saying which piece of data failed
    And they do not access HUBX

  Scenario: Incomplete data
    Given an operator is registered in HUBX
    When the operator tries to sign in without entering their password
    Then they are asked to complete the missing data
    And they do not access HUBX

  # Forges an expired token with the environment's JWT_SECRET, which never leaves the deployed environment.
  @needs-secret
  Scenario: Expired session
    Given an operator signed in more than one hour ago
    When the operator tries to view a shipment
    Then they are told their session expired and are asked to sign in again
    And the lookup is not carried out

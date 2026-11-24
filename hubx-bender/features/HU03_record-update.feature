Feature: Record a shipment update
  As the party responsible for a stage of the shipment (operator or ground partner),
  I want to record the update when it is my turn,
  so that all participants share a single, reliable history of the shipment.

  Scenario Outline: Update recorded by its responsible party
    Given a shipment is in the stage "<current stage>"
    And <responsible party> has signed in
    When <responsible party> records the update "<new stage>"
    Then the shipment moves to the stage "<new stage>"
    And the update is recorded with its responsible party and the date

    Examples:
      | current stage       | responsible party                           | new stage              |
      | Created             | the operator                                | In flight              |
      | In flight           | the operator                                | At destination airport |
      | Assigned to partner | the ground partner assigned to the shipment | Out for delivery       |
      | Out for delivery    | the ground partner assigned to the shipment | Delivered              |

  Scenario: Assigning a ground partner
    Given a shipment is in the stage "At destination airport"
    And the ground partner "Transportes Rápido" is registered
    And an operator has signed in
    When the operator assigns "Transportes Rápido" as the party responsible for the delivery
    Then the shipment moves to the stage "Assigned to partner"
    And the update is recorded with the operator as the responsible party
    And "Transportes Rápido" becomes responsible for the delivery

  Scenario: Assigning someone who is not a ground partner
    Given a shipment is in the stage "At destination airport"
    And the customer "Distribuciones Andina" is registered
    And an operator has signed in
    When the operator tries to assign "Distribuciones Andina" as the party responsible for the delivery
    Then they are told that only a ground partner can be responsible for the delivery
    And the shipment keeps its stage

  Scenario: Assigning without naming the partner
    Given a shipment is in the stage "At destination airport"
    And an operator has signed in
    When the operator tries to record the update "Assigned to partner" without naming the responsible ground partner
    Then they are told they must name the ground partner responsible for the delivery
    And the shipment keeps its stage

  Scenario: First update in the chain
    Given a newly created shipment has no updates recorded
    And an operator has signed in
    When the operator records the update "In flight"
    Then the update becomes the start of the traceability chain
    And it has no previous update

  Scenario: Chaining to the previous update
    Given a shipment has the update "In flight" recorded
    And an operator has signed in
    When the operator records the update "At destination airport"
    Then the new update is chained to the previous one through the previous update's fingerprint

  Scenario Outline: Update outside the sequence
    Given a shipment is in the stage "<current stage>"
    And <responsible party> has signed in
    When <responsible party> tries to record the update "<new stage>"
    Then they are told that update does not follow the shipment's sequence
    And the shipment keeps its stage

    Examples:
      | current stage          | responsible party                           | new stage              |
      | Created                | the operator                                | At destination airport |
      | In flight              | the operator                                | In flight              |
      | At destination airport | the operator                                | In flight              |
      | Assigned to partner    | the ground partner assigned to the shipment | Delivered              |

  Scenario Outline: Update rejected because the person is not responsible for it
    Given a shipment is in the stage "<current stage>"
    And <who tries> has signed in
    When <who tries> tries to record the update "<new stage>"
    Then they are told they are not responsible for that update
    And the shipment keeps its stage

    Examples:
      | current stage       | who tries                                                      | new stage              |
      | Created             | a customer                                                     | In flight              |
      | In flight           | a ground partner                                               | At destination airport |
      | Out for delivery    | the operator                                                   | Delivered              |
      | Assigned to partner | a ground partner who is not the one assigned to the shipment   | Out for delivery       |

  @e2e
  Scenario: Critical happy path in the web application
    Given the shipment "HBX-SEED-0001" is in the stage "Created"
    And an operator has signed in through the web application
    When the operator records the update "In flight" through the web application
    Then the shipment moves to the stage "In flight"
    And the update is recorded with its responsible party and the date

  # Spec-derived edge cases (FR-011, FR-012); not part of the verbatim backlog Gherkin.
  Scenario: Update attempt with an expired session
    Given a shipment is in the stage "Created"
    And an operator has signed in
    When the operator's session expires
    And the operator tries to record the update "In flight"
    Then they are told their session expired and are asked to sign in again

  Scenario: Update attempt on a code that matches no shipment
    Given an operator has signed in
    When the operator tries to record the update "In flight" for a shipment code that does not exist
    Then they are told they are not responsible for that update

  Scenario: Update attempt on a shipment the caller cannot see
    Given a shipment is in the stage "Created"
    And a customer who does not own the shipment has signed in
    When that customer tries to record the update "In flight"
    Then they are told they are not responsible for that update

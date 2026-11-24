Feature: Verify the integrity of a shipment's chain
  As a participant in a shipment,
  I want to verify that its history has not been altered,
  so that I can trust that the information I see is what was actually recorded.

  Scenario Outline: Intact chain
    Given a shipment has three updates recorded without alterations
    And <participant> has signed in
    When <participant> requests verification of the shipment's integrity
    Then they are told that the chain is intact
    And that three updates were checked

    Examples:
      | participant                                 |
      | an operator                                 |
      | the customer who owns the shipment          |
      | the ground partner assigned to the shipment |

  Scenario: Shipment without updates
    Given a shipment has no updates recorded
    And an operator has signed in
    When the operator requests verification of the shipment's integrity
    Then they are told that the chain is intact
    And that there are no updates to check

  @needs-secret @needs-db
  Scenario: Update improperly altered
    Given the second of a shipment's three updates was improperly altered
    And an operator has signed in
    When the operator requests verification of the shipment's integrity
    Then they are told that the chain is not intact
    And that the first alteration is in the second update

  Scenario: Code that matches no shipment
    Given an operator has signed in
    When the operator requests verification of a code that matches no shipment
    Then they are told that no shipment exists with that code

  Scenario: Shipment the participant cannot see
    Given a shipment that belongs to another customer exists
    And a customer who does not own it has signed in
    When the customer requests verification of that shipment's integrity
    Then they are told that no shipment exists with that code

  @e2e
  Scenario: Verify an intact chain in the browser
    Given a shipment has three updates recorded without alterations
    And an operator has signed in
    When the operator verifies the shipment's integrity in the browser
    Then the browser shows that the chain is intact
    And the browser shows that three updates were checked

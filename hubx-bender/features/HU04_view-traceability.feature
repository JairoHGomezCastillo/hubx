Feature: View a shipment's traceability
  As a participant in a shipment (customer, operator or ground partner),
  I want to view its traceability using the shipment code,
  so that I know where it is and who has taken part, without depending on phone calls.

  Scenario Outline: Viewing by an authorized participant
    Given a shipment has three updates recorded
    And <participant> has signed in
    When <participant> views the shipment's traceability using its code
    Then they see the three updates, from the oldest to the newest
    And for each one they see its stage, its responsible party and its date

    Examples:
      | participant                                 |
      | the customer who owns the shipment          |
      | an operator                                 |
      | the ground partner assigned to the shipment |

  Scenario: Newly created shipment
    Given a shipment has no updates recorded
    And the customer who owns the shipment has signed in
    When the customer views the shipment's traceability
    Then they see that the shipment is in the stage "Created"
    And they see that it has no updates yet

  Scenario: Code that matches no shipment
    Given an operator has signed in
    When the operator views the traceability of a code that matches no shipment
    Then they are told that no shipment exists with that code

  Scenario Outline: Shipment the participant cannot see
    Given a shipment with updates recorded exists
    And <participant> has signed in
    When <participant> views the traceability of that shipment
    Then they are told that no shipment exists with that code

    Examples:
      | participant                                        |
      | a customer who does not own the shipment           |
      | a ground partner to whom the shipment was not assigned |

  @e2e
  Scenario: Viewing by an authorized participant
    Given a shipment has three updates recorded
    And the customer who owns the shipment has signed in
    When the customer views the shipment's traceability using its code
    Then they see the three updates, from the oldest to the newest
    And for each one they see its stage, its responsible party and its date

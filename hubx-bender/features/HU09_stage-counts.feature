Feature: View the shipment count by stage
  As an operator,
  I want to see how many shipments are in each stage,
  so that I can detect pile-ups and bottlenecks.

  Scenario: Count by stage
    Given there are 3 shipments in flight, 2 out for delivery and 5 delivered
    And an operator has signed in
    When the operator views the shipment count by stage
    Then they see the total for each stage
    And they see the stages in journey order

  @e2e
  Scenario: Count by stage
    Given there are 3 shipments in flight, 2 out for delivery and 5 delivered
    And an operator has signed in
    When the operator views the shipment count by stage
    Then they see the total for each stage
    And they see the stages in journey order

  @empty-db
  Scenario: Stages without shipments
    Given no shipment exists in the stage "Out for delivery"
    And an operator has signed in
    When the operator views the shipment count by stage
    Then the stage "Out for delivery" appears with a total of zero

  @empty-db
  Scenario: No shipments in HUBX
    Given no shipment exists yet
    And an operator has signed in
    When the operator views the shipment count by stage
    Then they see the six stages, all with a total of zero

  Scenario Outline: Role without permission to view the count
    Given a <role> has signed in
    When the <role> tries to view the shipment count by stage
    Then they are told they do not have permission to see that information

    Examples:
      | role           |
      | ground partner |
      | customer       |

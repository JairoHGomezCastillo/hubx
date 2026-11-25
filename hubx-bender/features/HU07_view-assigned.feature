Feature: View my assigned shipments
  As a ground partner,
  I want to see the shipments assigned to me that I still have to deliver,
  so that I can plan my deliveries.

  Scenario: Shipments pending delivery
    Given three shipments pending delivery were assigned to a ground partner
    And two shipments were assigned to another ground partner
    And the first partner has signed in
    When the partner views their assigned shipments
    Then they see only their three shipments
    And they see first the one that has been assigned the longest

  Scenario: Shipments already delivered
    Given a ground partner was assigned one shipment that is already delivered and another that is out for delivery
    And the partner has signed in
    When the partner views their assigned shipments
    Then they see only the shipment that is out for delivery

  Scenario: Partner without assigned shipments
    Given no shipment has been assigned to a ground partner
    And the partner has signed in
    When the partner views their assigned shipments
    Then they see an empty list
    And they are told that they have no assigned shipments

  Scenario Outline: Role without permission to view assigned shipments
    Given <role> has signed in
    When <role> tries to view the shipments assigned to a ground partner
    Then they are told they do not have permission to see that information

    Examples:
      | role        |
      | an operator |
      | a customer  |

  @e2e
  Scenario: Ground partner views assigned shipments through the web application
    Given three shipments pending delivery were assigned to a ground partner
    And the partner has signed in through the web application
    When the partner views their assigned shipments through the web application
    Then they see their three shipments through the web application
    And they see first the one that has been assigned the longest

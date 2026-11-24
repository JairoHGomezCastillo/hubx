Feature: Create a shipment
  As an operator,
  I want to create a shipment indicating its origin, destination, flight, weight and customer,
  so that its traceability begins.

  Scenario: Shipment created with valid data
    Given an operator has signed in
    And the customer "Distribuciones Andina" is registered
    When the operator creates a shipment from Bogotá to Miami on flight "AV-204", weighing 120 kg, for that customer
    Then the shipment is created with a unique code
    And its stage is "Created"

  @e2e
  Scenario: Shipment created with valid data
    Given an operator has signed in
    And the customer "Distribuciones Andina" is registered
    When the operator creates a shipment from Bogotá to Miami on flight "AV-204", weighing 120 kg, for that customer
    Then the shipment is created with a unique code
    And its stage is "Created"

  Scenario Outline: Shipment accepted at the weight limits
    Given an operator has signed in
    And the customer "Distribuciones Andina" is registered
    When the operator creates a shipment weighing <weight> kg
    Then the shipment is created in the stage "Created"

    Examples:
      | weight |
      | 0.1    |
      | 5000   |

  Scenario: Two shipments on the same flight
    Given an operator has signed in
    And a shipment already exists on flight "AV-204"
    When the operator creates another shipment on flight "AV-204"
    Then two shipments exist on that flight, each with its own code

  Scenario Outline: Shipment rejected because of invalid data
    Given an operator has signed in
    When the operator tries to create a shipment with <situation>
    Then the shipment is not created
    And they are told that <reason>

    Examples:
      | situation                                     | reason                                                         |
      | the same origin and destination               | the origin and the destination must be different               |
      | a weight of 0 kg                              | the weight must be between 0.1 and 5000 kg                     |
      | a weight of 5001 kg                           | the weight must be between 0.1 and 5000 kg                     |
      | a customer who is not registered              | the customer indicated does not exist                          |
      | a ground partner in the place of the customer | the customer indicated does not have the customer role         |
      | the flight left out                           | required data is missing                                       |

  Scenario Outline: Role without permission to create shipments
    Given a <role> has signed in
    When the <role> tries to create a shipment
    Then they are told they do not have permission to create shipments
    And the shipment is not created

    Examples:
      | role           |
      | ground partner |
      | customer       |

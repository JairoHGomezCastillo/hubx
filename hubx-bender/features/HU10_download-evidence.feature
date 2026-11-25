Feature: Download the traceability evidence
  As an operator,
  I want to download a shipment's traceability evidence,
  so that I can share it with third parties who need to trust its history.

  @e2e
  Scenario: Evidence for a shipment with updates
    Given a shipment has three updates recorded
    And an operator has signed in
    When the operator downloads the shipment's traceability evidence
    Then they receive a document with the shipment's data
    And with each update, its stage, its responsible party, its date and its fingerprint
    And with the date on which the document was generated

  Scenario: Verification by a third party
    Given an operator downloaded the evidence of a shipment with unaltered updates
    When a third party without access to HUBX follows the document's instructions to check the fingerprints
    Then the fingerprints they obtain match the ones in the document

  Scenario: Evidence for a shipment without updates
    Given a shipment has no updates recorded
    And an operator has signed in
    When the operator downloads the shipment's traceability evidence
    Then they receive a document with the shipment's data
    And it states that the shipment has no updates yet

  Scenario: Code that matches no shipment
    Given an operator has signed in
    When the operator tries to download the evidence of a code that matches no shipment
    Then they are told that no shipment exists with that code

  Scenario Outline: Role without permission to download evidence
    Given a <role> has signed in
    When the <role> tries to download the evidence of a shipment
    Then they are told they do not have permission to download it

    Examples:
      | role           |
      | ground partner |
      | customer       |

  Scenario: Person who has not signed in
    Given nobody has signed in
    When a person tries to download the evidence of a shipment
    Then they are asked to sign in
    And they receive no document

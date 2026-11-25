# HUBX Backlog

A platform where the participants of an air shipment (operator, ground partner and customer) share a single, reliable
and verifiable history of each shipment. Each participant records what falls to them and is identified in the record.

Every story meets INVEST and the 3 Cs: **Card** (as a / I want / so that), **Conversation** (business rules) and
**Confirmation** (scenarios). Response codes, routes and data structures do not live here: they are defined in the plan
and the contract of each story.

## Story Status

Reconciled with GitHub (merged PRs and release tags) by hubx-lead at the start of every story. Statuses: `Backlog`,
`In progress`, `In review (PR #n)`, `Done in <release branch> (PR #n)`, `Released vX.Y.Z`.

| ID | Story | Status | Branch | PR |
|---|---|---|---|---|
| HU-01 | Sign in | Released v0.2.0 | 001-sign-in | #13 |
| HU-02 | Create shipment | Released v0.3.0 | 002-create-shipment | #19 |
| HU-03 | Record update | Released v0.3.3; scenario fixes released v0.4.0 | 003-record-update-sync, 003-record-update-fix | #23, #27 |
| HU-04 | View traceability | Released v0.4.0 | 004-view-traceability | #30 |
| HU-05 | Verify integrity | Released v0.4.0 | 005-verify-integrity | #33 |
| HU-06 | List by stage | Backlog | - | - |
| HU-07 | View assigned | Released v0.6.0 | 007-view-assigned | #39, #44 |
| HU-08 | Report incident | Backlog | - | - |
| HU-09 | View count | Released v0.7.0 | 009-view-count-by-stage | #41, #53 |
| HU-10 | Download evidence | Released v0.7.0 | 010-download-evidence | #40, #43 |

## Business language

| Term | Meaning |
|---|---|
| Shipment | Air cargo that travels from an origin to a destination and must reach a customer |
| Stage | The point of the journey where a shipment is |
| Update | A change of stage, recorded by the party responsible for that stage |
| Traceability chain | The ordered set of a shipment's updates, each one chained to the previous one |
| Fingerprint | A unique seal of an update; if the update is altered, the fingerprint no longer matches |
| Responsible party | The participant who records an update and is identified in it |
| Ground partner | The company or person who makes the final delivery of a shipment |

**Stages, in order:** Created, In flight, At destination airport, Assigned to partner, Out for delivery, Delivered.

## Cross-cutting rules

1. **Session:** it lasts one hour. After that, HUBX asks the user to sign in again before any operation.
2. **Sequence:** stages always advance in the order above. There is no going back or repeating, and in this version a
   recorded update is not corrected.
3. **Immutability:** an update is never modified or deleted.
4. **Privacy:** someone who cannot see a shipment gets the same answer as if the shipment did not exist.
5. **Lists:** they are shown 20 at a time, most recent first, unless a story says otherwise.
6. **Who can do what:**

| Action | Operator | Ground partner | Customer |
|---|---|---|---|
| Create a shipment | Yes | No | No |
| Record In flight and At destination airport | Yes | No | No |
| Assign a partner (stage Assigned to partner) | Yes | No | No |
| Record Out for delivery and Delivered | No | Only the assigned partner | No |
| View traceability and verify integrity | Any shipment | Only shipments assigned to them | Only their own |
| List shipments by stage, view counts, download evidence | Yes | No | No |
| View assigned shipments, report incidents | No | Yes, on their own shipments | No |

---

# BASELINE

## HU-01 · Sign in

**Conversation:** sample users are preloaded. The email ignores capital letters and extra spaces. When credentials
fail, the message does not reveal which piece of data was wrong.

```gherkin
Feature: Sign in
  As a HUBX user (operator, ground partner or customer),
  I want to sign in with my email and my password,
  so that I can access the functions that correspond to my role.

  Scenario Outline: Each role signs in
    Given a user with the role of <role> is registered in HUBX
    When that user signs in with their correct email and password
    Then they access HUBX with the functions of their role

    Examples:
      | role           |
      | operator       |
      | ground partner |
      | customer       |

  Scenario: Email typed with capital letters and extra spaces
    Given a customer is registered with the email "ana@hubx.co"
    When the customer signs in typing " ANA@hubx.co " and their correct password
    Then the customer accesses HUBX with the functions of their role

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

  Scenario: Expired session
    Given an operator signed in more than one hour ago
    When the operator tries to view a shipment
    Then they are told their session expired and are asked to sign in again
    And the lookup is not carried out
```

## HU-02 · Create a shipment

**Conversation:** HUBX gives each shipment a unique code. The shipment starts in the stage Created. The weight must be
between 0.1 and 5000 kg. The customer indicated must be registered with the customer role. Several shipments can travel
on the same flight. Creating the shipment does not record an update yet: the first update will be "In flight".

```gherkin
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
      | situation                                     | reason                                                |
      | the same origin and destination               | the origin and the destination must be different      |
      | a weight of 0 kg                              | the weight must be between 0.1 and 5000 kg            |
      | a weight of 5001 kg                           | the weight must be between 0.1 and 5000 kg            |
      | a customer who is not registered              | the customer indicated does not exist                 |
      | a ground partner in the place of the customer | the customer indicated does not have the customer role |
      | the flight left out                           | required data is missing                              |

  Scenario Outline: Role without permission to create shipments
    Given a <role> has signed in
    When the <role> tries to create a shipment
    Then they are told they do not have permission to create shipments
    And the shipment is not created

    Examples:
      | role           |
      | ground partner |
      | customer       |
```

## HU-03 · Record a shipment update

**Conversation:** the operator records In flight and At destination airport. Assigning a ground partner is also an
operator update (Assigned to partner): from that moment that partner is responsible for the delivery. The assigned
partner records Out for delivery and Delivered. Each update is stored with its responsible party and its date, and
chained to the previous update. Once assigned, the partner is not replaced in this version. The stage Assigned to
partner is only reached by assigning a partner: there is no other way.

```gherkin
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
```

## HU-04 · View a shipment's traceability

**Conversation:** the customer sees only their own shipments, the operator sees any shipment, and the ground partner sees
the ones assigned to them. Updates are shown from the oldest to the newest, with their stage, responsible party and date.

```gherkin
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
```

## HU-05 · Verify the integrity of the chain

**Conversation:** any participant who can see a shipment can verify it. The verification checks the fingerprints of all
the updates and indicates the first altered update. The result is shown only when it is requested; automatic alerts
remain as a reserve story.

```gherkin
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
```

---

# PENDING FOR THE DEMONSTRATION

The five are independent of each other: each one adds its own functionality and only reads what the baseline left behind.

## HU-06 · List shipments by stage

**Conversation:** operator only. The list shows the most recently created shipment first and is shown 20 at a time.

```gherkin
Feature: List shipments by stage
  As an operator,
  I want to see the shipments that are in a given stage,
  so that I can quickly find the ones that need attention.

  Scenario: Filter by stage
    Given shipments exist in different stages
    And an operator has signed in
    When the operator asks to see the shipments in the stage "In flight"
    Then they see only the shipments in that stage
    And they see them from the most recently created to the oldest

  Scenario: More shipments than fit in one group
    Given 45 shipments exist in the stage "Created"
    And an operator has signed in
    When the operator asks to see the shipments in that stage
    Then they see the 20 most recent
    And they can continue with the next 20 until they have seen all 45

  Scenario: Stage without shipments
    Given no shipment exists in the stage "Delivered"
    And an operator has signed in
    When the operator asks to see the shipments in that stage
    Then they see an empty list
    And they are told that there are no shipments in that stage

  Scenario: Continuing past the last shipment
    Given 45 shipments exist in the stage "Created"
    And an operator has signed in
    When the operator asks to continue after having seen all 45
    Then they see an empty list

  Scenario: Stage that does not exist
    Given an operator has signed in
    When the operator asks to see the shipments in the stage "Flying"
    Then they are told that this stage does not exist
    And they are shown the valid stages

  Scenario Outline: Role without permission to list shipments
    Given a <role> has signed in
    When the <role> tries to see the shipments of a stage
    Then they are told they do not have permission to see that list

    Examples:
      | role           |
      | ground partner |
      | customer       |
```

## HU-07 · View my assigned shipments

**Conversation:** the ground partner sees the shipments assigned to them whose delivery is still pending (stages
Assigned to partner and Out for delivery). The shipment that has been assigned the longest appears first.

```gherkin
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
```

## HU-08 · Report an incident

**Conversation:** the description is required and allows up to 500 characters. The incident is stored with its author and
its date. It does not change the shipment's stage. Incidents are accepted from the moment the shipment has an assigned
partner until before its delivery (stages Assigned to partner and Out for delivery); they are not accepted on shipments
that have already been delivered. Letting the operator view them is a reserve story (HU-11).

```gherkin
Feature: Report an incident
  As the ground partner responsible for a shipment,
  I want to report an incident that occurred while the shipment is in my care,
  so that the problem is on record.

  Scenario Outline: Valid incident at each accepted stage
    Given a shipment is in the stage "<stage>" and assigned to a ground partner
    And that partner has signed in
    When the partner reports the incident "<description>"
    Then the incident is recorded with its author and its date
    And the shipment keeps its stage

    Examples:
      | stage               | description                                           |
      | Assigned to partner | The cargo was not ready to be picked up at the airport |
      | Out for delivery    | The recipient was not at the address                  |

  Scenario: Description at the maximum length
    Given a shipment is out for delivery and assigned to a ground partner
    And that partner has signed in
    When the partner reports an incident with a description of exactly 500 characters
    Then the incident is recorded

  Scenario: Description too long
    Given a shipment is out for delivery and assigned to a ground partner
    And that partner has signed in
    When the partner tries to report an incident with a description of 501 characters
    Then they are told that the description cannot exceed 500 characters
    And the incident is not recorded

  Scenario: Empty description
    Given a shipment is out for delivery and assigned to a ground partner
    And that partner has signed in
    When the partner tries to report an incident without a description
    Then they are asked to describe the problem
    And the incident is not recorded

  Scenario: Shipment already delivered
    Given a shipment is in the stage "Delivered" and assigned to a ground partner
    And that partner has signed in
    When the partner tries to report an incident on that shipment
    Then they are told that incidents are not accepted on delivered shipments
    And the incident is not recorded

  Scenario Outline: Someone other than the responsible partner
    Given a shipment is out for delivery and assigned to a ground partner
    And <who tries> has signed in
    When <who tries> tries to report an incident on that shipment
    Then they are told that only the ground partner responsible for the shipment can report incidents
    And the incident is not recorded

    Examples:
      | who tries                                              |
      | a ground partner to whom the shipment was not assigned |
      | an operator                                            |
      | a customer                                             |
```

## HU-09 · View the shipment count by stage

**Conversation:** operator only. The six stages are shown in journey order, each one with its total.

```gherkin
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

  Scenario: Stages without shipments
    Given no shipment exists in the stage "Out for delivery"
    And an operator has signed in
    When the operator views the shipment count by stage
    Then the stage "Out for delivery" appears with a total of zero

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
```

## HU-10 · Download the traceability evidence

**Conversation:** the evidence document lets a third party check, on their own, that the shipment's history was not
altered. It contains the shipment's data, each update with its stage, responsible party, date and fingerprint, the order
of the chain, the date the document was generated and the instructions for checking the fingerprints. It proves that the
history is consistent with itself; by itself it does not prove that HUBX did not rebuild the whole history.

```gherkin
Feature: Download the traceability evidence
  As an operator,
  I want to download a shipment's traceability evidence,
  so that I can share it with third parties who need to trust its history.

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
```

---

# RESERVE STORIES

They stay outside the ten, ready to be written to the same standard.

| ID | Story | Why it stays in reserve |
|---|---|---|
| HU-11 | View a shipment's incidents (operator) | Closes the value of HU-08, but depends on it |
| HU-12 | Receive an alert when an altered chain is detected (operator) | Adds notifications; HU-05 already shows the result on demand |
| HU-13 | Reassign a shipment's ground partner | Requires going back a stage or a correction update |
| HU-14 | Correct an update recorded by mistake | Solved with a correction update, never by deleting |

---

# DECISION TRACEABILITY

| # | Deviation | Decision | Where |
|---|---|---|---|
| 1 | Stage "Assigned to partner" versus the assignment story | The stage exists. Assigning is an operator update that identifies the responsible ground partner | HU-03 |
| 2 | Who records the delivery | The assigned partner records Out for delivery and Delivered; the operator cannot | HU-03, cross-cutting rules |
| 3 | Stages from which a partner is assigned | Only from "At destination airport"; no reassignment in this version | HU-03, HU-13 |
| 4 | Reversibility | Strict sequence, no going back and no corrections in this version | Cross-cutting rules, HU-03, HU-14 |
| 5 | Session expiry | The user is asked to sign in again and the operation is not carried out | HU-01 and cross-cutting rule 1 |
| 6 | Creation validations | Customer registered with the customer role; several shipments per flight; weight between 0.1 and 5000 kg | HU-02 |
| 7 | Traceability visibility | Customer: their own. Operator: all. Partner: the ones assigned to them | HU-04 |
| 8 | Reaction to tampering | The result is shown on request; the alert stays in reserve. The technical setup of the alteration is defined in the plan | HU-05, HU-12 |
| 9 | List order and continuation | Most recent first, 20 at a time, with continuation | HU-06 and cross-cutting rule 5 |
| 10 | Incident lifecycle | HU-08 puts the problem on record; viewing it is HU-11 | HU-08, HU-11 |
| 11 | Third-party verification | The evidence includes what is needed to check the fingerprints without access to HUBX | HU-10 |
| O1 | HU-05: the partner was missing from the intact-chain examples | The partner assigned to the shipment is added | HU-05 |
| O2 | Entry to the stage "Assigned to partner" | It is only reached by assigning a partner; if no partner is named, the update is rejected | HU-03 |
| O3 | Stages in which incidents are accepted | From when the shipment has an assigned partner until before its delivery | HU-08 |
| O4 | HU-08 card wording | The card says "while the shipment is in my care", consistent with the stages accepted | HU-08 |

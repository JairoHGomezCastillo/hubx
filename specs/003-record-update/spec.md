# Feature Specification: Record a shipment update

**Feature Branch**: `003-record-update`

**Created**: 2026-11-24

**Status**: Draft

**Input**: Backlog HU-03 (Record a shipment update)

Source: backlog HU-03

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Record the next stage update (Priority: P1)

As the party responsible for a stage of the shipment (operator or ground partner), I want to record the update when it
is my turn, so that all participants share a single, reliable history of the shipment.

**Why this priority**: it is the core value of the story. Every advance to the next stage becomes a permanent entry in
the shared history; without it there is no update to chain and nothing for later verification to rely on.

**Independent Test**: a shipment in Created with an operator signed in can be advanced to In flight and the recorded
update shows the operator and the date. A newly created shipment's first update starts the traceability chain with no
previous update; a later update is linked to it through the previous update's fingerprint.

**Acceptance Scenarios**:

```gherkin
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
```

---

### User Story 2 - Assign a ground partner (Priority: P2)

As an operator, I want to assign a ground partner once the shipment is at the destination airport, so that the partner
becomes responsible for the delivery and the handover is part of the shared history.

**Why this priority**: it is the only way to reach the stage Assigned to partner and the handover point to the partner.
Without it the assigned partner can never record Out for delivery or Delivered.

**Independent Test**: a shipment at At destination airport with a registered ground partner can be assigned by an
operator, moves to Assigned to partner, and names that partner as responsible for the delivery.

**Acceptance Scenarios**:

```gherkin
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
```

---

### User Story 3 - Reject updates that break the rules (Priority: P3)

As a participant in a shipment, I want updates that break the journey sequence or that I am not responsible for to be
rejected, so that the shared history stays reliable.

**Why this priority**: it protects the auditability of the history. It can be tested independently with prepared
shipments and a signed-in party that attempts a stage they are not entitled to record.

**Independent Test**: a shipment in a given stage with a signed-in party that is not entitled to the requested stage
gets a reason for the rejection and keeps its stage.

**Acceptance Scenarios**:

```gherkin
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

### Edge Cases

- A newly created shipment has no updates: its first update is the start of the chain and has no previous update.
- The stage Assigned to partner is reached only by naming a ground partner; a customer or a missing partner is rejected.
- Once a partner is assigned, it is not replaced in this version.
- A rejected update leaves both the shipment's stage and the traceability chain unchanged.
- A ground partner who is not the assigned one cannot record Out for delivery or Delivered.
- Cross-cutting rule 1 (session) applies before any update, and cross-cutting rule 4 (privacy) applies when the caller
  cannot see the shipment; both are referenced from the backlog and not restated here.
- Clarified 2026-11-24: when a caller attempts an update on a shipment they cannot see, or on a code that matches no
  shipment, the answer is the same "not responsible for that update" rejection, so the response never reveals whether
  the shipment exists. This reconciles the responsibility scenario with the privacy rule.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: the system MUST let an operator record In flight from Created and At destination airport from In flight.
- **FR-002**: the system MUST let an operator reach Assigned to partner only by naming a registered ground partner;
  from that moment that partner is responsible for the delivery.
- **FR-003**: the system MUST let only the assigned ground partner record Out for delivery and Delivered.
- **FR-004**: every recorded update MUST be stored with its responsible party and its date.
- **FR-005**: every update MUST be chained to the previous one through that update's fingerprint; the first update of a
  shipment has no previous update.
- **FR-006**: the system MUST reject an update that does not follow the journey sequence, tell the caller that it does
  not follow the sequence, and keep the shipment's stage.
- **FR-007**: the system MUST reject an update attempted by a party who is not responsible for it, tell that caller so,
  and keep the shipment's stage.
- **FR-008**: the system MUST reject an assignment to someone who is not a ground partner and an assignment that does
  not name a partner, with the corresponding reason, keeping the shipment's stage.
- **FR-009**: the system MUST NOT allow replacing the assigned ground partner in this version.
- **FR-010**: recorded updates MUST be immutable (cross-cutting rule 3); rejected attempts MUST NOT create an update.
- **FR-011**: the story MUST honour the session and privacy cross-cutting rules (1 and 4) defined in the backlog.
- **FR-012**: an update attempt on a shipment the caller cannot see, or with a code that matches no shipment, MUST
  return the same "not responsible for that update" rejection as an attempt on an existing visible shipment the caller
  is not responsible for, without disclosing whether the shipment exists.

### Key Entities *(include if feature involves data)*

- **Shipment**: the air cargo whose stage advances one step at a time; it holds the current stage and, once assigned,
  the ground partner responsible for the delivery.
- **Update**: an immutable record of a stage change. It carries the new stage, the responsible party, the date, the
  previous update's fingerprint (none for the first one) and its own fingerprint.
- **Responsible party**: the operator for the early stages and the assignment, or the assigned ground partner for the
  delivery stages; it is the party identified in the update.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: each of the four legitimate stage transitions can be recorded by its responsible party and leaves the
  shipment in the expected stage.
- **SC-002**: every recorded update identifies its responsible party and date, and every update after the first is
  chained to its predecessor through the predecessor's fingerprint.
- **SC-003**: every out-of-sequence or unauthorized attempt is rejected with the stated reason and leaves the stage and
  the chain unchanged.
- **SC-004**: an assignment from At destination airport results in the named ground partner being responsible for the
  delivery and in an update recorded by the operator.
- **SC-005**: all acceptance scenarios of this story pass, including its critical happy path driven through the user
  interface.

## Assumptions

- The cross-cutting rules, the glossary and the stage order are those of `backlog/backlog.md`; they are referenced and
  not copied into this specification.
- Sign-in (HU-01) and shipment creation (HU-02) are prerequisites: a signed-in party and an existing shipment in the
  expected stage are needed to record an update. They are assumed to be integrated upstream; their domains are not
  implemented by this story.
- The interface language is English for all actors (operator, ground partner and customer), including stage names,
  messages and controls, consistent with the backlog glossary (clarified 2026-11-24).

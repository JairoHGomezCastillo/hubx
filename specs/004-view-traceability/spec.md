# Feature Specification: View a shipment's traceability

**Feature Branch**: `004-view-traceability`

**Created**: 2026-11-25

**Status**: Clarified

**Input**: Backlog HU-04 (View a shipment's traceability)

Source: backlog HU-04

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View the traceability of a visible shipment (Priority: P1)

As a participant in a shipment (customer, operator or ground partner), I want to view its traceability using the
shipment code, so that I know where it is and who has taken part, without depending on phone calls.

**Why this priority**: it is the whole value of the story. Each participant reads the shared history recorded in HU-03
without asking anyone.

**Independent Test**: a shipment with three recorded updates is viewed by its owning customer, by an operator and by its
assigned ground partner; each sees the three updates from the oldest to the newest with stage, responsible party and
date. A newly created shipment shows the stage Created and no updates.

**Acceptance Scenarios**:

```gherkin
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
```

---

### User Story 2 - Do not reveal shipments the participant cannot see (Priority: P2)

As a participant, I want a code I cannot see to get the same answer as a code that does not exist, so that no one
learns about other parties' shipments.

**Why this priority**: it enforces cross-cutting rule 4 (privacy) on the first story that reads the history.

**Independent Test**: an unknown code, a shipment owned by another customer and a shipment not assigned to the
signed-in ground partner all get the same "no shipment exists with that code" answer.

**Acceptance Scenarios**:

```gherkin
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

### Edge Cases

- A newly created shipment has no updates: the view shows its current stage (Created) and states that it has no
  updates yet.
- The answer for a shipment the caller cannot see is indistinguishable from the answer for an unknown code
  (cross-cutting rule 4), including what the caller is told.
- A ground partner sees a shipment only from the moment it is assigned to them; before the assignment it is a shipment
  they cannot see.
- Viewing never changes the shipment or its history (cross-cutting rule 3).
- Cross-cutting rule 1 (session) applies before the view.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: the system MUST let a signed-in participant view a shipment's traceability by entering its code.
- **FR-002**: the operator MUST be able to view any shipment; the customer only the shipments they own; the ground
  partner only the shipments assigned to them (backlog cross-cutting rule 6, decision 7).
- **FR-003**: the view MUST list every recorded update of the shipment from the oldest to the newest.
- **FR-004**: for each update the view MUST show its stage, its responsible party and its date.
- **FR-005**: the view MUST show the shipment's current stage; when no update is recorded it MUST show the stage
  Created and say that the shipment has no updates yet.
- **FR-006**: a code that matches no shipment MUST be answered with "no shipment exists with that code".
- **FR-007**: a shipment the participant cannot see MUST get exactly the same answer as FR-006 (cross-cutting rule 4).
- **FR-008**: viewing MUST NOT create, modify or delete any update or change the shipment.
- **FR-009**: the story MUST honour the session cross-cutting rule (1) defined in the backlog.

### Key Entities *(include if feature involves data)*

- **Shipment**: identified by its code; has a current stage, an owning customer and, once assigned, a ground partner.
- **Update**: a recorded stage change with its responsible party and date (recorded by HU-03; read-only here).
- **Traceability chain**: the ordered set of a shipment's updates, oldest first.
- **Responsible party**: the participant identified in each update, shown by name.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: each of the three authorized participants sees all recorded updates of a visible shipment in recording
  order, each with stage, responsible party and date.
- **SC-002**: a newly created shipment shows the stage Created and no updates.
- **SC-003**: an unknown code and an invisible shipment produce the same answer, with no detail that tells them apart.
- **SC-004**: all acceptance scenarios of this story pass, including its critical happy path driven through the user
  interface.

## Clarifications

### Session 2026-11-25

No business rule was missing: the backlog's conversation, cross-cutting rules 1, 4 and 6 and decision 7 cover
visibility, ordering and the fields shown. The following points were resolved as interface decisions within the
existing rules and recorded here; the orchestrator may overrule them before the PR:

- Q: Interface language? → A: English for all roles, as in HU-03.
- Q: How is the responsible party shown? → A: by its registered name (the role is shown next to it).
- Q: How is the date shown? → A: date and time of the update, unambiguous (time zone stated).
- Q: Is the assigned ground partner shown on the assignment update? → A: yes, as additional context of that update;
  it does not replace the responsible party (the operator).

## Assumptions

- Cross-cutting rules, glossary and stage order are those of `backlog/backlog.md`; they are referenced, not copied.
- HU-01 (sign-in), HU-02 (shipment creation) and HU-03 (recording updates) are integrated on the release branch; this
  story only reads what they record.
- A shipment with three updates is a shipment that reached "Assigned to partner" (In flight, At destination airport,
  Assigned to partner), which is also the first point at which a ground partner can see it.

# Feature Specification: View the shipment count by stage

**Feature Branch**: `009-view-count-by-stage`

**Created**: 2026-11-24

**Status**: Draft

**Input**: Backlog HU-09 (View the shipment count by stage)

Source: backlog HU-09

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the shipment count for every stage (Priority: P1)

As an operator, I want to see how many shipments are in each stage, so that I can detect pile-ups and bottlenecks.

**Why this priority**: it is the whole value of the story. A single view of the six totals, in journey order, lets the
operator spot where shipments are accumulating. There is no smaller slice that still delivers that value.

**Independent Test**: with shipments spread across several stages and an operator signed in, requesting the count by
stage returns one total per stage, with the six stages shown in journey order, and the totals match how many shipments
sit in each stage.

**Acceptance Scenarios**:

```gherkin
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
```

---

### User Story 2 - Keep the count operator-only (Priority: P2)

As HUBX, I want to deny the count to anyone who is not an operator, so that only the role responsible for monitoring the
flow can see the aggregate view.

**Why this priority**: it protects the privacy and permission model. It is testable on its own by signing in as a ground
partner or a customer and confirming the request is refused without exposing any totals.

**Independent Test**: a ground partner or a customer signed in who requests the count by stage is told they do not have
permission to see that information, and no totals are returned.

**Acceptance Scenarios**:

```gherkin
  Scenario Outline: Role without permission to view the count
    Given a <role> has signed in
    When the <role> tries to view the shipment count by stage
    Then they are told they do not have permission to see that information

    Examples:
      | role           |
      | ground partner |
      | customer       |
```

### Edge Cases

- A stage with no shipments appears with a total of zero rather than being omitted.
- When HUBX has no shipments at all, all six stages appear with a total of zero; the view is never empty.
- The six stages are always shown, in journey order (Created, In flight, At destination airport, Assigned to partner,
  Out for delivery, Delivered), regardless of which ones have shipments.
- Cross-cutting rule 1 (session) applies before the count is shown; an expired session is asked to sign in again and
  the count is not produced. This is referenced from the backlog and not restated here.
- A ground partner or customer is refused with a permission message and receives no totals, so the aggregate view is
  never disclosed to a role that may not see it.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: the system MUST let an operator view the number of shipments in each stage.
- **FR-002**: the system MUST return a total for every one of the six stages, including the stages that have no
  shipments, which MUST appear with a total of zero.
- **FR-003**: the system MUST present the stages in journey order: Created, In flight, At destination airport, Assigned
  to partner, Out for delivery, Delivered.
- **FR-004**: each stage's total MUST equal the number of shipments currently in that stage at the time of the request.
- **FR-005**: when HUBX holds no shipments, the system MUST still return the six stages, each with a total of zero.
- **FR-006**: the system MUST refuse the count to a ground partner or a customer, telling them they do not have
  permission to see that information, and MUST NOT return any totals to them.
- **FR-007**: the story MUST honour the session cross-cutting rule (rule 1) defined in the backlog.

### Key Entities *(include if feature involves data)*

- **Shipment**: the air cargo whose current stage is counted. Only its current stage matters for this story; the count
  reads existing shipments and does not create or change any.
- **Stage**: one of the six points of the journey, in a fixed order. Each stage is paired with a total in the result.
- **Stage count**: the pairing of a stage with the number of shipments currently in it. The result is the set of all
  six stage counts, in journey order.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: an operator requesting the count always receives exactly six stage totals, one per stage, in journey
  order.
- **SC-002**: each returned total matches the number of shipments in that stage; a stage with no shipments shows zero.
- **SC-003**: when HUBX has no shipments, the operator sees all six stages with a total of zero and never an empty
  result.
- **SC-004**: a ground partner or a customer requesting the count is refused with the permission message and receives
  no totals in 100% of attempts.
- **SC-005**: all acceptance scenarios of this story pass, including its critical happy path driven through the user
  interface.

## Assumptions

- The cross-cutting rules, the glossary and the stage order are those of `backlog/backlog.md`; they are referenced and
  not copied into this specification.
- Sign-in (HU-01) and the existence of shipments across stages (HU-02, HU-03) are prerequisites: a signed-in operator
  and the shipments to count are assumed to be integrated upstream; their domains are not implemented by this story.
- The count reflects the shipments present at the moment of the request; the story defines no refresh, caching or
  real-time update behaviour beyond returning the current totals.
- The count is a single aggregate view of all shipments, not a paginated list, so cross-cutting rule 5 (lists shown 20
  at a time) does not apply to it.
- The interface language is English for all actors, including stage names, totals and messages, consistent with the
  backlog glossary.

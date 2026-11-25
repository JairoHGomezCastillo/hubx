# Feature Specification: View my assigned shipments

**Feature Branch**: `007-view-assigned`

**Release branch**: `release/v0.6.0-demo-07` (created from `main` at `853944c`, v0.3.4, with the human's confirmation)

**Created**: 2026-11-24

**Status**: Clarified

**Input**: Backlog HU-07 (View my assigned shipments)

Source: backlog HU-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the shipments I still have to deliver (Priority: P1)

As a ground partner, I want to see the shipments assigned to me that I still have to deliver, so that I can plan my
deliveries.

**Why this priority**: it is the whole story. A ground partner today can only act on a shipment whose code someone gave
them; this list tells them which deliveries are waiting for them and which one has waited the longest.

**Independent Test**: with three shipments pending delivery assigned to one ground partner and two assigned to another,
the first partner sees exactly their three shipments, the one assigned the longest ago first. A shipment of theirs
that is already delivered does not appear. A partner with nothing assigned sees an empty list and a message.

**Acceptance Scenarios** (backlog HU-07, verbatim):

```gherkin
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
```

---

### User Story 2 - Only ground partners see assigned shipments (Priority: P2)

As HUBX, I want only ground partners to see their own assigned shipments, so that no participant learns about
deliveries that are not theirs.

**Why this priority**: it protects the privacy rule; without it the list would leak other parties' work.

**Independent Test**: an operator and a customer who ask for the assigned shipments are told they do not have
permission to see that information.

**Acceptance Scenarios** (backlog HU-07, verbatim):

```gherkin
  Scenario Outline: Role without permission to view assigned shipments
    Given <role> has signed in
    When <role> tries to view the shipments assigned to a ground partner
    Then they are told they do not have permission to see that information

    Examples:
      | role        |
      | an operator |
      | a customer  |
```

---

### Edge Cases

- A shipment in the stage Assigned to partner and one in Out for delivery both count as pending delivery; a Delivered
  one never appears (backlog HU-07 conversation).
- Shipments in Created, In flight or At destination airport have no assigned partner yet and never appear.
- The ground partner whose list is shown is always the signed-in partner; a partner can never ask for another
  partner's list (cross-cutting rule 4, Privacy).
- Session older than one hour: the partner is asked to sign in again and the list is not shown (cross-cutting rule 1).
- More than 20 pending shipments: the list is shown 20 at a time and the partner can continue with the next group
  (cross-cutting rule 5; this story overrides only the order, see Clarifications).
- Two shipments assigned at the same instant: their relative order is stable between requests.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A signed-in ground partner MUST be able to view the shipments assigned to them whose delivery is still
  pending, meaning the shipment's current stage is Assigned to partner or Out for delivery.
- **FR-002**: The list MUST contain only shipments assigned to the signed-in ground partner; shipments assigned to any
  other partner MUST NOT appear, and the partner is identified by the session, never by data the partner supplies.
- **FR-003**: Shipments already Delivered MUST NOT appear.
- **FR-004**: The shipment assigned the longest ago MUST appear first (oldest assignment first). The assignment moment is
  the date of the shipment's "Assigned to partner" update. Ties keep a stable order.
- **FR-005**: For each shipment the partner MUST see its code, its current stage, its origin, its destination, its
  flight, its weight and the date it was assigned to them.
- **FR-006**: When the partner has no pending shipment, HUBX MUST show an empty list and tell them that they have no
  assigned shipments.
- **FR-007**: The list MUST be shown 20 shipments at a time, with a way to continue with the next group; continuing past
  the last shipment gives an empty group (cross-cutting rule 5).
- **FR-008**: An operator or a customer who asks for assigned shipments MUST be told they do not have permission to see
  that information, and MUST receive no shipment.
- **FR-009**: A session older than one hour MUST be rejected and the partner asked to sign in again (cross-cutting rule 1).
- **FR-010**: Viewing the list MUST NOT record any update or change any shipment (read-only).
- **FR-011**: Interface text is in English.

### Key Entities

- **Shipment** (backlog glossary): code, origin, destination, flight, weight, current stage, assigned ground partner.
- **Assignment**: the "Assigned to partner" update that names the ground partner responsible for the delivery; its date
  is the assignment moment used for the order.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In the backlog scenarios, a ground partner sees exactly their pending shipments, oldest assignment first,
  in 100% of the runs.
- **SC-002**: No operator, customer or other ground partner ever receives a shipment that is not assigned to the caller.
- **SC-003**: A ground partner reaches their pending list from the web application right after signing in, without
  typing any shipment code.

## Clarifications

### Session 2026-11-24

Resolved by the lead from the backlog, without new business rules; recorded so the human can contest them in the PR.

- Q: Does the "20 at a time" list rule apply? → A: Yes. Cross-cutting rule 5 says lists are "shown 20 at a time, most
  recent first, unless a story says otherwise"; HU-07 says otherwise only about the order ("assigned the longest
  appears first"), so the page size and continuation stay (FR-007).
- Q: What is "the one that has been assigned the longest"? → A: the shipment whose "Assigned to partner" update is the
  oldest. Partners are not reassigned in this version (backlog HU-03, decision 3), so each shipment has one assignment.
- Q: What does the partner see for each shipment? → A: the data needed to plan a delivery: code, stage, origin,
  destination, flight, weight and assignment date (FR-005). The customer's identity is not shown; it is not needed to
  plan and the story does not ask for it.

## Assumptions

- The human chose to cut `release/v0.6.0-demo-07` from `main` (v0.3.4), which has HU-01 to HU-03 but not HU-04/HU-05
  (still in the open release PR #36). HU-07 only reads what HU-03 records, so it does not depend on HU-04/HU-05.
- **Deployment validation is skipped at the human's explicit request** for this story (lead step 6d, demo mode).
  Staging still validates the release when the PR is merged into the integration branch.

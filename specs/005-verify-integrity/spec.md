# Feature Specification: Verify the integrity of a shipment's chain

**Feature Branch**: `005-verify-integrity` (cut from `origin/release/v0.4.0-ledger`)

**Created**: 2026-11-25

**Status**: Clarified

**Input**: Backlog HU-05 (Verify the integrity of the chain)

Source: backlog HU-05

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Verify an intact chain (Priority: P1)

As a participant in a shipment, I want to verify that its history has not been altered, so that I can trust that the
information I see is what was actually recorded.

**Why this priority**: it is the core value of the story. Every participant who can see a shipment gets an on-demand
answer about whether its traceability chain is intact and how many updates were checked.

**Independent Test**: a shipment with three unaltered updates is verified by an operator, by the customer who owns it
and by the ground partner assigned to it; each one is told the chain is intact and that three updates were checked. A
shipment with no updates is reported intact with no updates to check.

**Acceptance Scenarios**:

```gherkin
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
```

---

### User Story 2 - Detect the first altered update (Priority: P2)

As a participant in a shipment, I want to learn which update was the first to be altered, so that I know from which
point the history can no longer be trusted.

**Why this priority**: detecting an alteration is what makes the chain verifiable; it depends on the verification of
US1 and adds the "not intact" answer.

**Independent Test**: a shipment with three updates whose second update was improperly altered is verified by an
operator, who is told the chain is not intact and that the first alteration is in the second update.

**Acceptance Scenarios**:

```gherkin
  Scenario: Update improperly altered
    Given the second of a shipment's three updates was improperly altered
    And an operator has signed in
    When the operator requests verification of the shipment's integrity
    Then they are told that the chain is not intact
    And that the first alteration is in the second update
```

---

### User Story 3 - Keep privacy when the shipment cannot be seen (Priority: P3)

As a participant, I want an unknown code and a shipment I cannot see to give the same answer, so that verification never
reveals that someone else's shipment exists.

**Why this priority**: it protects privacy (cross-cutting rule 4) on top of the verification already delivered.

**Independent Test**: an operator verifying a code that matches no shipment, and a customer verifying a shipment owned
by another customer, both get "no shipment exists with that code".

**Acceptance Scenarios**:

```gherkin
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

### Edge Cases

- A shipment without updates is intact and reports that there are no updates to check (zero checked).
- Several altered updates: only the first one in chain order is reported.
- An alteration of any recorded data of an update (stage, responsible party, assigned partner, date, its seal or its link
  to the previous update) counts as an alteration of that update.
- A missing or out-of-order position in the chain counts as an alteration at the first position that does not match.
- A ground partner who is not the one assigned to the shipment, and a customer who does not own it, get the same answer
  as for an unknown code (cross-cutting rule 4).
- An expired session follows cross-cutting rule 1: the user is asked to sign in again and no verification is carried
  out.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: A signed-in participant MUST be able to request the verification of a shipment's chain by its code.
- **FR-002**: Verification MUST be available to the operator for any shipment, to the customer for their own shipments
  and to the ground partner for the shipments assigned to them (cross-cutting rule 6).
- **FR-003**: Verification MUST recompute the fingerprint of every update of the shipment from its recorded data using
  the existing fingerprint definition shared with HU-03, and compare it with the stored fingerprint and with the link
  to the previous update. No new fingerprint definition is introduced.
- **FR-004**: When every update matches, the result MUST say the chain is intact and how many updates were checked.
- **FR-005**: When the shipment has no updates, the result MUST say the chain is intact and that there are no updates
  to check.
- **FR-006**: When at least one update does not match, the result MUST say the chain is not intact and identify the
  first altered update by its position in the chain (first, second, ...).
- **FR-007**: An unknown code and a shipment the caller cannot see MUST give the same answer: no shipment exists with
  that code (cross-cutting rule 4).
- **FR-008**: Verification MUST only read the history: it never records, modifies or deletes an update (cross-cutting
  rule 3) and does not change the shipment's stage.
- **FR-009**: The result is shown only when it is requested; no automatic alert is sent (HU-12 stays in reserve).
- **FR-010**: The web application MUST let a signed-in participant enter a shipment code, request the verification and
  read the result. The interface language is English.

### Key Entities

- **Shipment**: identified by its code; has an owner customer and, from Assigned to partner on, an assigned ground
  partner. Its visibility decides who can verify it.
- **Update**: an entry of the traceability chain with its position, stage, responsible party, date, optional assigned
  partner, the fingerprint of the previous update and its own fingerprint (defined by HU-03).
- **Verification result**: whether the chain is intact, how many updates were checked and, when it is not intact, the
  position of the first altered update. It is computed on request and not stored.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100 % of the HU-05 backlog scenarios pass as automated acceptance tests, plus one browser happy path.
- **SC-002**: an alteration of any single recorded field of any update is reported at exactly that update's position.
- **SC-003**: the answer for an unknown code and for an invisible shipment cannot be told apart by the caller.
- **SC-004**: verifying a shipment leaves the number of its updates and its stage unchanged.

## Clarifications

### Session 2026-11-25

No question was sent to the user. The three readings below were derived by the lead from the backlog conversation
and the cross-cutting rules, which the lead judged settle every business rule. **They are lead proposals pending
human confirmation**, not confirmed answers: the human confirms (or corrects) them on PR #33 before merge. Each is
marked **PROPOSED (pending human confirmation)**.

- Q: Which participants count as "the ground partner assigned to the shipment" with three updates? → A **(PROPOSED,
  pending human confirmation)**: a shipment with three updates is in Assigned to partner (In flight, At destination
  airport, Assigned to partner), so the third update names the partner; no other path exists (HU-03).
- Q: How is the first altered update identified? → A **(PROPOSED, pending human confirmation)**: by its position in
  the chain, counted from the oldest (the "second update" of the scenario is position 2).
- Q: Interface language? → A **(PROPOSED, pending human confirmation)**: English, as in HU-03.

## Assumptions

- HU-01 (sign in) and HU-03 (record updates, fingerprint definition) are delivered in the release branch.
- HU-04 (view traceability) is not delivered yet; HU-05 offers its own verification entry in the web application and
  does not depend on HU-04.
- "Improperly altered" means a change made outside HUBX, directly on the stored history; HUBX itself never alters an
  update.

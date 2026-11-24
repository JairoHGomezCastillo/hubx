# Feature Specification: Create a shipment

**Feature Branch**: `002-create-shipment`

**Created**: 2026-11-24

**Status**: Clarified

**Source**: backlog HU-02

**Release**: v0.3.0 (`release/v0.3.0-create-shipment`), not v0.2.0.

## User Scenarios & Testing

### User Story 1 - Operator creates a shipment (Priority: P1)

As an operator, I want to create a shipment indicating its origin, destination, flight, weight and customer, so that
its traceability begins.

**Why this priority**: every later story (record an update, view traceability, verify, lists, counts, evidence) needs
shipments that were created in HUBX. Without this story the ledger has nothing to chain.

**Independent Test**: an operator signs in, creates a shipment with valid data and receives its unique code with the
stage "Created"; a second shipment on the same flight gets a different code.

**Acceptance Scenarios** (backlog HU-02, verbatim in `hubx-bender/features/HU02_create_shipment.feature`):

1. **Given** an operator has signed in **And** the customer "Distribuciones Andina" is registered, **When** the
   operator creates a shipment from Bogotá to Miami on flight "AV-204", weighing 120 kg, for that customer, **Then**
   the shipment is created with a unique code **And** its stage is "Created".
2. **Given** an operator has signed in **And** the customer "Distribuciones Andina" is registered, **When** the
   operator creates a shipment weighing 0.1 kg (and, separately, 5000 kg), **Then** the shipment is created in the
   stage "Created".
3. **Given** an operator has signed in **And** a shipment already exists on flight "AV-204", **When** the operator
   creates another shipment on flight "AV-204", **Then** two shipments exist on that flight, each with its own code.

---

### User Story 2 - Invalid data is rejected (Priority: P1)

As an operator, I want HUBX to reject a shipment whose data is not valid and tell me why, so that only shipments that
can really travel enter the traceability chain.

**Why this priority**: a wrong shipment cannot be corrected later (cross-cutting rules 2 and 3 forbid modifying the
history), so it must be stopped at creation.

**Independent Test**: each invalid situation below leaves no new shipment and shows its specific reason.

**Acceptance Scenarios**:

1. **Given** an operator has signed in, **When** the operator tries to create a shipment with the same origin and
   destination, **Then** the shipment is not created **And** they are told that the origin and the destination must be
   different.
2. ... with a weight of 0 kg -> the weight must be between 0.1 and 5000 kg.
3. ... with a weight of 5001 kg -> the weight must be between 0.1 and 5000 kg.
4. ... with a customer who is not registered -> the customer indicated does not exist.
5. ... with a ground partner in the place of the customer -> the customer indicated does not have the customer role.
6. ... with the flight left out -> required data is missing.

---

### User Story 3 - Only operators create shipments (Priority: P1)

As HUBX, I want only operators to create shipments, so that the permission matrix (cross-cutting rule 6) holds.

**Independent Test**: a ground partner and a customer each try to create a shipment and are refused.

**Acceptance Scenarios**:

1. **Given** a ground partner has signed in, **When** the ground partner tries to create a shipment, **Then** they are
   told they do not have permission to create shipments **And** the shipment is not created.
2. Same for a customer.

---

### Edge Cases

- Origin and destination that differ only in capital letters or surrounding spaces ("Bogotá" and " bogotá ") count as
  the same place and are rejected.
- Any required piece of data left out (origin, destination, flight, weight or customer) is rejected with "required
  data is missing"; a value made only of spaces counts as left out.
- A weight with more than one decimal (for example 0.15 kg) is rejected with the weight message (see clarification
  Q2).
- A session older than one hour cannot create a shipment (cross-cutting rule 1): the user is asked to sign in again and
  nothing is created.
- A person who has not signed in cannot create a shipment.
- A role without permission is refused before its data is examined: it learns nothing about which customers exist.

## Clarifications

### Session 2026-11-24 (resolved by hubx-lead from existing artifacts; confirm in the PR review)

- **Q1: How does the operator indicate the customer?** By the customer's email, as registered in HUBX (the same
  identifier used to sign in; capital letters and extra spaces are ignored, as in HU-01). The confirmation shows the
  customer's name. Choosing from a list of customers is out of scope.
- **Q2: What precision does the weight have?** Kilograms with at most one decimal; 0.1 kg is both the minimum and the
  smallest step. Source: the baseline schema stores one decimal; a weight with more decimals is rejected rather than
  silently rounded, because rounding would record a weight the operator did not enter. **Needs human confirmation.**
- **Q3: Interface language?** English, as in HU-01.

## Requirements

### Functional Requirements

- **FR-001**: Only a signed-in operator can create a shipment; a ground partner or a customer is told they do not have
  permission to create shipments and nothing is created (cross-cutting rule 6).
- **FR-002**: A shipment requires origin, destination, flight, weight in kilograms and customer; if any is missing the
  operator is told that required data is missing.
- **FR-003**: Origin and destination must be different places, ignoring capital letters and surrounding spaces.
- **FR-004**: The weight must be between 0.1 and 5000 kg, both included, with at most one decimal.
- **FR-005**: The customer indicated must be registered; otherwise the operator is told the customer does not exist.
- **FR-006**: The customer indicated must have the customer role; otherwise the operator is told the customer does
  not have the customer role.
- **FR-007**: HUBX assigns each new shipment a unique code that the operator does not choose; the code identifies the
  shipment in every later story.
- **FR-008**: A new shipment starts in the stage "Created".
- **FR-009**: Several shipments may share a flight; each keeps its own code.
- **FR-010**: Creating a shipment records no update: the traceability chain of a new shipment is empty and its first
  update will be "In flight" (HU-03).
- **FR-011**: HUBX records which operator created the shipment and when.
- **FR-012**: A rejected request creates nothing.
- **FR-013**: An expired session (older than one hour) cannot create a shipment (cross-cutting rule 1).

### Key Entities

- **Shipment**: unique code, origin, destination, flight, weight in kg, customer (a registered user with the customer
  role), stage (starts in "Created"), creating operator, creation date. Its stage is derived from its updates; with no
  updates it is "Created".
- **Customer**: a registered user with the customer role (from HU-01), indicated by email.

## Success Criteria

- **SC-001**: An operator creates a valid shipment and sees its code and the stage "Created" in a single step.
- **SC-002**: Every scenario of HU-02 (5 scenarios, 12 examples) passes against the full stack, plus one browser happy
  path.
- **SC-003**: No invalid or unauthorized request leaves a shipment behind.
- **SC-004**: Two shipments never share a code.

## Assumptions

- Demo users from migration 0001 are the participants: operator@hubx.co, andina@hubx.co ("Distribuciones Andina",
  customer), rapido@hubx.co (ground partner).
- Places and flights are free text; validating airport or flight codes against a catalogue is out of scope.
- Lists (cross-cutting rule 5) do not apply: this story shows only the shipment just created.
- Privacy (cross-cutting rule 4) is not affected: operators may see any shipment and any customer; non-operators are
  refused before any lookup.

> Technology and implementation decisions (endpoint, payload, codes, storage, code format) live in
> [`plan.md`](plan.md).

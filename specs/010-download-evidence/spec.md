# Feature Specification: Download the traceability evidence

**Source**: backlog HU-10

**Feature Branch**: `010-download-evidence`

**Created**: 2026-11-25

**Status**: Draft

## User Scenarios & Testing *(mandatory)*

Cross-cutting rules and glossary are defined once in `backlog/backlog.md`: session, sequence, immutability,
privacy, lists and who-can-do-what apply here and are referenced, not copied.

### User Story 1 - Download evidence for a shipment with updates (Priority: P1)

As an operator, I want to obtain a document that gathers a shipment's data and its whole traceability chain, so that I
can share it with third parties who need to trust its history. The document lets the recipient check, on their own,
that the history is consistent with itself: it contains the shipment's data, every update with its stage, responsible
party, date and fingerprint, the order of the chain, the date the document was generated and the instructions for
checking the fingerprints. The evidence proves internal consistency; by itself it does not prove that HUBX did not
rebuild the whole history.

**Why this priority**: it is the story's core value and the critical happy path that must work in a real browser.

**Independent Test**: sign in as an operator, choose a shipment with three recorded updates, download its evidence
and check that the document contains the shipment data, the three updates in chain order with stage, responsible
party, date and fingerprint, the generation date and the verification instructions.

**Acceptance Scenarios**:

1. **Given** a shipment has three updates recorded, **When** the operator downloads the shipment's traceability
   evidence, **Then** they receive a document with the shipment's data and with each update, its stage, its
   responsible party, its date and its fingerprint, and with the date on which the document was generated.
2. **Given** an operator downloaded the evidence of a shipment with unaltered updates, **When** a third party without
   access to HUBX follows the document's instructions to check the fingerprints, **Then** the fingerprints they obtain
   match the ones in the document.

---

### User Story 2 - Evidence for a shipment without updates (Priority: P2)

As an operator, I want the evidence to state explicitly that a newly created shipment has no updates yet, so that a
third party does not mistake an empty chain for missing information.

**Why this priority**: it covers the legitimate empty-chain case and prevents a misleading document.

**Independent Test**: sign in as an operator, download the evidence of a shipment with no updates and check that the
document contains the shipment data and states that the shipment has no updates yet.

**Acceptance Scenarios**:

1. **Given** a shipment has no updates recorded, **When** the operator downloads the shipment's traceability evidence,
   **Then** they receive a document with the shipment's data and it states that the shipment has no updates yet.

---

### User Story 3 - Clear refusal when the evidence cannot be given (Priority: P3)

As a user who asks for evidence I am not entitled to, I want a clear answer that does not reveal whether a shipment
exists, so that privacy is preserved and I know when I must sign in.

**Why this priority**: it protects privacy and access control, which the cross-cutting rules require for every
shipment operation.

**Independent Test**: ask for evidence without a session, with a role that has no permission and with a code that
matches no shipment, and check in each case that no document is delivered and the expected answer is given.

**Acceptance Scenarios**:

1. **Given** an operator has signed in, **When** the operator tries to download the evidence of a code that matches no
   shipment, **Then** they are told that no shipment exists with that code.
2. **Given** a ground partner or a customer has signed in, **When** they try to download the evidence of a shipment,
   **Then** they are told they do not have permission to download it.
3. **Given** nobody has signed in, **When** a person tries to download the evidence of a shipment, **Then** they are
   asked to sign in and they receive no document.

---

### Edge Cases

- A shipment whose chain is intact versus one whose chain was altered: the evidence reflects what is recorded; the
  verification result is out of scope here (it is HU-05).
- The empty chain: the document states there are no updates instead of omitting the chain section.
- A code that matches no shipment and a code the caller cannot see get the same answer, per the privacy rule.
- A caller who is not signed in, or whose session has expired, receives no document.
- The generation date is the moment the document is produced, not the date of the last update.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The operator MUST be able to request and obtain the evidence of any shipment.
- **FR-002**: The evidence MUST contain the shipment's data.
- **FR-003**: The evidence MUST contain every update in chain order, with its stage, responsible party, date and
  fingerprint.
- **FR-004**: The evidence MUST state when the shipment has no updates.
- **FR-005**: The evidence MUST contain the date on which the document was generated.
- **FR-006**: The evidence MUST contain instructions that allow a third party without access to HUBX to check that
  the fingerprints in the document match the recorded updates.
- **FR-007**: The evidence MUST NOT claim more than internal consistency: it does not by itself prove that HUBX did
  not rebuild the whole history.
- **FR-008**: A request from a ground partner or a customer MUST be refused without delivering a document.
- **FR-009**: A request with no valid session MUST be refused without delivering a document.
- **FR-010**: A request for a code that matches no shipment MUST be answered as if no shipment exists, without
  delivering a document.
- **FR-011**: The evidence MUST be delivered as a PDF document that the recipient can open and print without HUBX.
- **FR-012**: The shipment data MUST include the code, origin, destination, flight, weight, customer and current
  stage.
- **FR-013**: The document, including the verification instructions, MUST be written in English.

### Key Entities *(include if feature involves data)*

- **Evidence document**: the shareable artifact; contains shipment data, the ordered updates with their stage,
  responsible party, date and fingerprint, the generation date and verification instructions.
- **Shipment**: identified by its code; its data and current stage are taken from what HUBX already records.
- **Update**: one recorded change of stage, with its responsible party, date and fingerprint; the evidence shows
  them from the oldest to the newest, as the traceability chain defines.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An operator can obtain the evidence of a shipment with three updates and find all three updates, in
  chain order, with their stage, responsible party, date and fingerprint, plus the generation date.
- **SC-002**: A third party who has only the document can follow its instructions and reproduce every fingerprint
  for an unaltered chain.
- **SC-003**: The evidence of a shipment without updates explicitly states that there are no updates.
- **SC-004**: No document is delivered to a caller without permission, without a session or with an unknown code.

## Clarifications

Decisions confirmed by the human during `clarify` (2026-11-25); each fills a gap in the backlog, none invents a
business rule:

1. **Document format**: the evidence is a PDF, self-contained and printable, so a third party can open it without
   HUBX. Reason: the backlog says "document" without naming a format; PDF is the shareable artifact the story needs.
2. **Shipment data**: alongside the code and the current stage, the document includes origin, destination, flight,
   weight and customer — the data recorded when the shipment was created. Reason: the backlog says "the shipment's
   data" without enumerating it; these are the fields HUBX already records for a shipment.
3. **Language**: the document and its verification instructions are in English, like the backlog, the specs and the
   rest of the interface. Reason: Constitution VII requires the interface language to be fixed in the spec; no other
   audience language was specified.

## Assumptions

- The evidence is generated from records HUBX already keeps; this story adds no new recorded business event.
- Verification by the third party is manual, outside HUBX, and uses only what the document contains.
- Every backlog scenario is automated against the API through the gateway, and the critical happy path is also
  driven in a real browser, as required by the constitution and the acceptance-test rules.
- The interface language is English, consistent with the document language confirmed above.

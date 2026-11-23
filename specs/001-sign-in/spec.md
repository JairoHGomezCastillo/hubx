# Feature Specification: Sign in

**Feature Branch**: `001-sign-in`

**Created**: 2025-01-21

**Status**: Clarified

**Source**: backlog HU-01

## User Scenarios & Testing

### User Story 1 - Authenticate with email and password (Priority: P1)

As a HUBX user (operator, ground partner or customer), I want to sign in with my email and password, so that I can access the functions that correspond to my role.

**Why this priority**: Authentication is the foundation for all other features. Without sign-in, users cannot access any HUBX functionality.

**Independent Test**: Can be fully tested by attempting to sign in with valid credentials for each role and verifying access is granted with appropriate role-based permissions.

**Acceptance Scenarios**:

1. **Given** a user with the role of operator is registered in HUBX, **When** that user signs in with their correct email and password, **Then** they access HUBX with the functions of their role
2. **Given** a user with the role of ground partner is registered in HUBX, **When** that user signs in with their correct email and password, **Then** they access HUBX with the functions of their role
3. **Given** a user with the role of customer is registered in HUBX, **When** that user signs in with their correct email and password, **Then** they access HUBX with the functions of their role

---

### User Story 2 - Email normalization (Priority: P2)

As a HUBX user, I want the system to ignore capital letters and extra spaces in my email, so that I can sign in even if I type my email with variations.

**Why this priority**: Improves user experience by reducing sign-in failures due to formatting differences that don't affect email identity.

**Independent Test**: Can be tested by signing in with the same email address using different capitalizations and with surrounding spaces, verifying all variants succeed.

**Acceptance Scenarios**:

1. **Given** a customer is registered with the email "ana@hubx.co", **When** the customer signs in typing " ANA@hubx.co " and their correct password, **Then** the customer accesses HUBX with the functions of their role

---

### User Story 3 - Security: Non-revealing error messages (Priority: P1)

As a system administrator, I want authentication errors to not reveal which credential was incorrect, so that the system is protected against user enumeration attacks.

**Why this priority**: Critical security requirement to prevent attackers from discovering valid user emails through brute force attempts.

**Independent Test**: Can be tested by attempting sign-in with invalid email and with invalid password, verifying both return the same generic error message.

**Acceptance Scenarios**:

1. **Given** a customer is registered in HUBX, **When** the customer tries to sign in with an incorrect password, **Then** they are told the credentials are not valid, without saying which piece of data failed, **And** they do not access HUBX
2. **Given** no user exists with the email "nobody@hubx.co", **When** a person tries to sign in with that email, **Then** they are told the credentials are not valid, without saying which piece of data failed, **And** they do not access HUBX

---

### User Story 4 - Validation of required fields (Priority: P2)

As a HUBX user, I want to be notified when I haven't entered all required sign-in information, so that I can correct my input before submitting.

**Why this priority**: Basic validation that improves user experience and prevents unnecessary authentication attempts.

**Independent Test**: Can be tested by attempting to sign in with missing email or missing password and verifying appropriate validation messages appear.

**Acceptance Scenarios**:

1. **Given** an operator is registered in HUBX, **When** the operator tries to sign in without entering their password, **Then** they are asked to complete the missing data, **And** they do not access HUBX

---

### User Story 5 - Session expiration (Priority: P1)

As a system, I want sessions to expire after one hour, so that inactive accounts are protected from unauthorized access.

**Why this priority**: Core security requirement per cross-cutting rule 1. Session timeout is a fundamental security control.

**Independent Test**: Can be tested by establishing a session, waiting for expiration, and verifying that subsequent operations require re-authentication.

**Acceptance Scenarios**:

1. **Given** an operator signed in more than one hour ago, **When** the operator tries to view a shipment, **Then** they are told their session expired and are asked to sign in again, **And** the lookup is not carried out

---

### Edge Cases

- What happens when a user tries to sign in with an empty email and empty password?
- What happens when a user has multiple concurrent sessions and one expires?
- What happens when the user's password contains special characters or Unicode?
- How does the system handle extremely long email addresses (e.g., 254 characters, the RFC maximum)?
- What happens when a user tries to sign in while already signed in?

## Requirements

### Functional Requirements

- **FR-001**: System MUST authenticate users via email and password
- **FR-002**: System MUST support three distinct roles: operator, ground partner, and customer
- **FR-003**: System MUST normalize email addresses by converting to lowercase and trimming whitespace before validation
- **FR-004**: System MUST return a generic "invalid credentials" message for both incorrect email and incorrect password
- **FR-005**: System MUST validate that both email and password are provided before attempting authentication
- **FR-006**: System MUST create a session with a one-hour expiration upon successful authentication
- **FR-007**: System MUST include the user's role in the session so that subsequent requests can be authorized
- **FR-008**: System MUST reject any operation (except sign-in) when the session has expired
- **FR-009**: System MUST issue, on successful sign-in, a tamper-evident session credential that carries the user's identity and role and cannot be forged or altered by the client
- **FR-010**: System MUST never store or compare passwords in clear text; it verifies the supplied password against a stored non-reversible form
- **FR-011**: System MUST check session expiration on every protected endpoint request

### Key Entities

- **User**: Represents a registered HUBX participant with email (unique identifier), hashed password, role (operator, ground partner, or customer), and optionally an associated party ID (for ground partners and customers)
- **Session**: Represents an authenticated session identified by a session credential carrying user identity, role, issue time, and expiration time (one hour from issue)

## Success Criteria

### Measurable Outcomes

- **SC-001**: All three user roles (operator, ground partner, customer) can successfully sign in with valid credentials and access role-appropriate functions
- **SC-002**: Authentication failures (wrong email or wrong password) return identical generic error messages, preventing user enumeration
- **SC-003**: Email addresses with capitalization variations and surrounding whitespace authenticate successfully when the core email matches
- **SC-004**: Sessions expire exactly one hour after creation, requiring re-authentication for any subsequent operations
- **SC-005**: Sign-in attempts with incomplete data (missing email or password) are rejected with clear validation messages before authentication is attempted

## Assumptions

- Demo users with predefined credentials exist in the system for testing, as documented in memory.md: operator@hubx.co, rapido@hubx.co, express@hubx.co, ana@hubx.co, andina@hubx.co (all with password "Hubx-Demo-2026")
- User registration is out of scope for this story; users are preloaded in the database
- Password reset functionality is out of scope for this story
- Multi-factor authentication is out of scope for this story
- The session is carried by a self-contained credential issued to the client; server-side session storage is not required
- Email address validation (format checking) is performed but detailed email format rules are implementation details
- Account lockout after multiple failed attempts is out of scope for this story

> Technology and implementation decisions (JWT, bcrypt, token transmission and structure, response shape) live in
> [`plan.md`](plan.md), not in this spec. This document stays implementation-agnostic.

package ledger

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	// ErrInvalidUpdate means the caller supplied data that cannot be stored in the ledger.
	ErrInvalidUpdate = errors.New("invalid update")
	// ErrConcurrentUpdate means the expected chain position is stale; the caller must retry.
	ErrConcurrentUpdate = errors.New("concurrent update")
)

var fingerprintPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidateAppendInput checks the invariants that do not depend on the stored chain.
func ValidateAppendInput(in AppendInput) error {
	if in.ShipmentID == "" {
		return fmt.Errorf("%w: shipment id is required", ErrInvalidUpdate)
	}
	if in.Sequence < 1 {
		return fmt.Errorf("%w: sequence must be positive", ErrInvalidUpdate)
	}
	if !IsUpdateStage(in.Stage) {
		return fmt.Errorf("%w: unknown update stage %q", ErrInvalidUpdate, in.Stage)
	}
	if in.ResponsiblePartyID == "" {
		return fmt.Errorf("%w: responsible party id is required", ErrInvalidUpdate)
	}
	if in.RecordedAt.IsZero() {
		return fmt.Errorf("%w: recorded date is required", ErrInvalidUpdate)
	}
	if in.Sequence == 1 && in.PreviousFingerprint != nil {
		return fmt.Errorf("%w: the first update has no predecessor", ErrInvalidUpdate)
	}
	if in.Sequence > 1 && (in.PreviousFingerprint == nil || !fingerprintPattern.MatchString(*in.PreviousFingerprint)) {
		return fmt.Errorf("%w: a later update needs a 64-hex predecessor fingerprint", ErrInvalidUpdate)
	}
	if in.AssignedPartnerID != nil && *in.AssignedPartnerID == "" {
		return fmt.Errorf("%w: assigned partner id must not be empty", ErrInvalidUpdate)
	}
	if (in.Stage == "Assigned to partner") != (in.AssignedPartnerID != nil) {
		return fmt.Errorf("%w: assignment requires exactly one assigned partner", ErrInvalidUpdate)
	}
	return nil
}

// CheckSequence verifies that expected still follows the stored chain. last == nil means
// the chain is empty and only sequence 1 is acceptable.
func CheckSequence(expected int, last *Update) error {
	current := 1
	if last != nil {
		current = last.Sequence + 1
	}
	if expected != current {
		return fmt.Errorf("%w: expected sequence %d, next sequence %d", ErrConcurrentUpdate, expected, current)
	}
	return nil
}

// CheckPredecessor verifies that the caller chained to the fingerprint actually stored.
func CheckPredecessor(in AppendInput, last *Update) error {
	if last == nil {
		if in.PreviousFingerprint != nil {
			return fmt.Errorf("%w: predecessor supplied for an empty chain", ErrConcurrentUpdate)
		}
		return nil
	}
	if in.PreviousFingerprint == nil || *in.PreviousFingerprint != last.Fingerprint {
		return fmt.Errorf("%w: predecessor fingerprint does not match the stored chain", ErrConcurrentUpdate)
	}
	return nil
}

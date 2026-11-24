package ledger

// VerificationResult holds the outcome of verifying a shipment's update chain.
type VerificationResult struct {
	Intact             bool
	UpdatesChecked     int
	FirstAlteredUpdate int // 1-based position, 0 if intact
}

// Verify walks the update chain and reports whether every update matches its
// recomputed fingerprint, the correct sequence position and the predecessor
// link. The first altered update is reported by its 1-based position (oldest
// first); 0 means the chain is intact. An empty chain is intact.
func Verify(updates []Update) VerificationResult {
	if len(updates) == 0 {
		return VerificationResult{Intact: true, UpdatesChecked: 0, FirstAlteredUpdate: 0}
	}

	for i, update := range updates {
		position := i + 1

		// Check 1: sequence must match the 1-based position
		if update.Sequence != position {
			return VerificationResult{
				Intact:             false,
				UpdatesChecked:     len(updates),
				FirstAlteredUpdate: position,
			}
		}

		// Check 2: previous_fingerprint link
		if position == 1 {
			// First update must have no predecessor
			if update.PreviousFingerprint != nil {
				return VerificationResult{
					Intact:             false,
					UpdatesChecked:     len(updates),
					FirstAlteredUpdate: position,
				}
			}
		} else {
			// Later updates must link to the stored fingerprint of the predecessor
			predecessor := updates[i-1]
			if update.PreviousFingerprint == nil || *update.PreviousFingerprint != predecessor.Fingerprint {
				return VerificationResult{
					Intact:             false,
					UpdatesChecked:     len(updates),
					FirstAlteredUpdate: position,
				}
			}
		}

		// Check 3: recompute the fingerprint with the existing Fingerprint function
		input := AppendInput{
			ShipmentID:          update.ShipmentID,
			Sequence:            update.Sequence,
			Stage:               update.Stage,
			ResponsiblePartyID:  update.ResponsiblePartyID,
			AssignedPartnerID:   update.AssignedPartnerID,
			RecordedAt:          update.RecordedAt,
			PreviousFingerprint: update.PreviousFingerprint,
		}
		recomputed := Fingerprint(input)
		if recomputed != update.Fingerprint {
			return VerificationResult{
				Intact:             false,
				UpdatesChecked:     len(updates),
				FirstAlteredUpdate: position,
			}
		}
	}

	return VerificationResult{
		Intact:             true,
		UpdatesChecked:     len(updates),
		FirstAlteredUpdate: 0,
	}
}

// Package ledger implements the immutable chain of shipment updates.
// It owns the canonical fingerprint formula documented in
// contracts/hu03-record-update.md and never imports net/http.
package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

const (
	// nullMarker is the exact canonical representation of an absent field.
	nullMarker = "-"
	// updateStageCount is the number of stage names allowed in an update (Created is not one).
	updateStageCount = 5
)

// updateStages mirrors the five update stages of the backlog glossary. Shipment owns the
// transition rules; the ledger only rejects a stage that can never be stored in this table.
var updateStages = [updateStageCount]string{
	"In flight",
	"At destination airport",
	"Assigned to partner",
	"Out for delivery",
	"Delivered",
}

// Update is one immutable ledger entry.
type Update struct {
	ID                  int64
	ShipmentID          string
	Sequence            int
	Stage               string
	ResponsiblePartyID  string
	AssignedPartnerID   *string
	RecordedAt          time.Time
	PreviousFingerprint *string
	Fingerprint         string
}

// Draft describes a new update before its position in the chain is known.
type Draft struct {
	ShipmentID         string
	Stage              string
	ResponsiblePartyID string
	AssignedPartnerID  *string
	RecordedAt         time.Time
}

// AppendInput is a Draft placed at a specific position in the chain.
type AppendInput struct {
	ShipmentID          string
	Sequence            int
	Stage               string
	ResponsiblePartyID  string
	AssignedPartnerID   *string
	RecordedAt          time.Time
	PreviousFingerprint *string
}

// CanonicalInput renders the seven-line UTF-8 string whose SHA-256 digest is the
// update fingerprint. Lines are joined with '\n' and there is no trailing newline.
func CanonicalInput(in AppendInput) string {
	assignedPartnerID := nullMarker
	if in.AssignedPartnerID != nil {
		assignedPartnerID = *in.AssignedPartnerID
	}
	previousFingerprint := nullMarker
	if in.PreviousFingerprint != nil {
		previousFingerprint = *in.PreviousFingerprint
	}
	return strings.Join([]string{
		"shipment_id=" + in.ShipmentID,
		"sequence=" + strconv.Itoa(in.Sequence),
		"stage=" + in.Stage,
		"responsible_party_id=" + in.ResponsiblePartyID,
		"assigned_partner_id=" + assignedPartnerID,
		"recorded_at=" + in.RecordedAt.UTC().Format(time.RFC3339Nano),
		"previous_fingerprint=" + previousFingerprint,
	}, "\n")
}

// Fingerprint returns the lowercase hex SHA-256 digest of CanonicalInput.
func Fingerprint(in AppendInput) string {
	sum := sha256.Sum256([]byte(CanonicalInput(in)))
	return hex.EncodeToString(sum[:])
}

// IsUpdateStage reports whether stage can be stored in the ledger. A newly created
// shipment has no update, so "Created" is never a valid update stage.
func IsUpdateStage(stage string) bool {
	for _, candidate := range updateStages {
		if candidate == stage {
			return true
		}
	}
	return false
}

package test

import (
	"testing"
	"time"

	"hubx-api/internal/ledger"
)

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", s, err)
	}
	return tm.UTC()
}

func computeFingerprint(t *testing.T, sequence int, shipmentID, stage, responsiblePartyID string, assignedPartnerID *string, recordedAt string, previousFingerprint *string) string {
	t.Helper()
	tm, err := time.Parse(time.RFC3339Nano, recordedAt)
	if err != nil {
		t.Fatalf("failed to parse time %q: %v", recordedAt, err)
	}
	input := ledger.AppendInput{
		ShipmentID:          shipmentID,
		Sequence:            sequence,
		Stage:               stage,
		ResponsiblePartyID:  responsiblePartyID,
		AssignedPartnerID:   assignedPartnerID,
		RecordedAt:          tm.UTC(),
		PreviousFingerprint: previousFingerprint,
	}
	return ledger.Fingerprint(input)
}

func strPtr(s string) *string {
	return &s
}

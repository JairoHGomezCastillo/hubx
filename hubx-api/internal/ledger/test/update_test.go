package test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"hubx-api/internal/ledger"
)

const (
	shipmentID         = "11111111-1111-4111-8111-111111111111"
	responsiblePartyID = "22222222-2222-4222-8222-222222222222"
	assignedPartnerID  = "33333333-3333-4333-8333-333333333333"
	firstFingerprint   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func ptr(value string) *string { return &value }

func TestCanonicalInputAndFingerprint(t *testing.T) {
	recordedAt := time.Date(2026, 11, 24, 10, 15, 30, 123456789, time.UTC)
	first := ledger.AppendInput{
		ShipmentID:         shipmentID,
		Sequence:           1,
		Stage:              "In flight",
		ResponsiblePartyID: responsiblePartyID,
		RecordedAt:         recordedAt,
	}

	tests := []struct {
		name          string
		input         ledger.AppendInput
		wantCanonical string
		wantNoNewline bool
	}{
		{
			name:  "first update uses dashes for both null fields",
			input: first,
			wantCanonical: strings.Join([]string{
				"shipment_id=" + shipmentID,
				"sequence=1",
				"stage=In flight",
				"responsible_party_id=" + responsiblePartyID,
				"assigned_partner_id=-",
				"recorded_at=2026-11-24T10:15:30.123456789Z",
				"previous_fingerprint=-",
			}, "\n"),
			wantNoNewline: true,
		},
		{
			name: "assignment and predecessor use exact values",
			input: ledger.AppendInput{
				ShipmentID:          shipmentID,
				Sequence:            2,
				Stage:               "Assigned to partner",
				ResponsiblePartyID:  responsiblePartyID,
				AssignedPartnerID:   ptr(assignedPartnerID),
				RecordedAt:          recordedAt,
				PreviousFingerprint: ptr(firstFingerprint),
			},
			wantCanonical: strings.Join([]string{
				"shipment_id=" + shipmentID,
				"sequence=2",
				"stage=Assigned to partner",
				"responsible_party_id=" + responsiblePartyID,
				"assigned_partner_id=" + assignedPartnerID,
				"recorded_at=2026-11-24T10:15:30.123456789Z",
				"previous_fingerprint=" + firstFingerprint,
			}, "\n"),
			wantNoNewline: true,
		},
		{
			name: "non UTC instant is rendered in UTC with nanoseconds",
			input: ledger.AppendInput{
				ShipmentID:         shipmentID,
				Sequence:           1,
				Stage:              "In flight",
				ResponsiblePartyID: responsiblePartyID,
				RecordedAt:         recordedAt.In(time.FixedZone("COT", -5*3600)),
			},
			wantCanonical: strings.Join([]string{
				"shipment_id=" + shipmentID,
				"sequence=1",
				"stage=In flight",
				"responsible_party_id=" + responsiblePartyID,
				"assigned_partner_id=-",
				"recorded_at=2026-11-24T10:15:30.123456789Z",
				"previous_fingerprint=-",
			}, "\n"),
			wantNoNewline: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ledger.CanonicalInput(tt.input)
			if got != tt.wantCanonical {
				t.Errorf("canonical input =\n%q\nwant\n%q", got, tt.wantCanonical)
			}
			if strings.HasSuffix(got, "\n") {
				t.Error("canonical input must not end with a newline")
			}
			if len(strings.Split(got, "\n")) != 7 {
				t.Errorf("canonical input has %d lines, want 7", len(strings.Split(got, "\n")))
			}

			sum := sha256.Sum256([]byte(tt.wantCanonical))
			wantFingerprint := hex.EncodeToString(sum[:])
			if gotFingerprint := ledger.Fingerprint(tt.input); gotFingerprint != wantFingerprint {
				t.Errorf("fingerprint = %q, want %q", gotFingerprint, wantFingerprint)
			}
		})
	}
}

func TestFingerprintChangesWithEveryField(t *testing.T) {
	base := ledger.AppendInput{
		ShipmentID:         shipmentID,
		Sequence:           1,
		Stage:              "In flight",
		ResponsiblePartyID: responsiblePartyID,
		RecordedAt:         time.Date(2026, 11, 24, 10, 15, 30, 0, time.UTC),
	}
	baseFingerprint := ledger.Fingerprint(base)

	tests := []struct {
		name   string
		mutate func(*ledger.AppendInput)
	}{
		{name: "sequence", mutate: func(in *ledger.AppendInput) { in.Sequence++ }},
		{name: "stage", mutate: func(in *ledger.AppendInput) { in.Stage = "At destination airport" }},
		{name: "responsible party", mutate: func(in *ledger.AppendInput) { in.ResponsiblePartyID = assignedPartnerID }},
		{name: "date", mutate: func(in *ledger.AppendInput) { in.RecordedAt = in.RecordedAt.Add(time.Nanosecond) }},
		{name: "assigned partner", mutate: func(in *ledger.AppendInput) { in.AssignedPartnerID = ptr(assignedPartnerID) }},
		{name: "predecessor", mutate: func(in *ledger.AppendInput) { in.PreviousFingerprint = ptr(firstFingerprint) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := base
			tt.mutate(&changed)
			if got := ledger.Fingerprint(changed); got == baseFingerprint {
				t.Errorf("fingerprint did not change after altering %s", tt.name)
			}
		})
	}
}

func TestValidateAppendInput(t *testing.T) {
	valid := ledger.AppendInput{
		ShipmentID:         shipmentID,
		Sequence:           1,
		Stage:              "In flight",
		ResponsiblePartyID: responsiblePartyID,
		RecordedAt:         time.Date(2026, 11, 24, 10, 15, 30, 0, time.UTC),
	}
	assignment := valid
	assignment.Sequence = 2
	assignment.Stage = "Assigned to partner"
	assignment.AssignedPartnerID = ptr(assignedPartnerID)
	assignment.PreviousFingerprint = ptr(firstFingerprint)

	tests := []struct {
		name    string
		input   ledger.AppendInput
		wantErr bool
	}{
		{name: "valid first update", input: valid},
		{name: "valid assignment", input: assignment},
		{name: "empty shipment", input: merge(valid, func(in *ledger.AppendInput) { in.ShipmentID = "" }), wantErr: true},
		{name: "sequence zero", input: merge(valid, func(in *ledger.AppendInput) { in.Sequence = 0 }), wantErr: true},
		{name: "stage Created", input: merge(valid, func(in *ledger.AppendInput) { in.Stage = "Created" }), wantErr: true},
		{name: "missing responsible party", input: merge(valid, func(in *ledger.AppendInput) { in.ResponsiblePartyID = "" }), wantErr: true},
		{name: "missing date", input: merge(valid, func(in *ledger.AppendInput) { in.RecordedAt = time.Time{} }), wantErr: true},
		{name: "first with predecessor", input: merge(valid, func(in *ledger.AppendInput) { in.PreviousFingerprint = ptr(firstFingerprint) }), wantErr: true},
		{name: "later without predecessor", input: merge(valid, func(in *ledger.AppendInput) { in.Sequence = 2 }), wantErr: true},
		{name: "assignment without partner", input: merge(assignment, func(in *ledger.AppendInput) { in.AssignedPartnerID = nil }), wantErr: true},
		{name: "non assignment with partner", input: merge(valid, func(in *ledger.AppendInput) { in.AssignedPartnerID = ptr(assignedPartnerID) }), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ledger.ValidateAppendInput(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAppendInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckSequenceAndPredecessor(t *testing.T) {
	last := &ledger.Update{
		Sequence:    3,
		Fingerprint: firstFingerprint,
	}

	tests := []struct {
		name            string
		expected        int
		last            *ledger.Update
		previous        *string
		wantSequence    bool
		wantPredecessor bool
	}{
		{name: "empty chain first update", expected: 1, wantSequence: true, wantPredecessor: true},
		{name: "next sequence and matching predecessor", expected: 4, last: last, previous: ptr(firstFingerprint), wantSequence: true, wantPredecessor: true},
		{name: "stale sequence", expected: 2, last: last, previous: ptr(firstFingerprint), wantSequence: false, wantPredecessor: true},
		{name: "first expected on non empty chain", expected: 1, last: last, previous: ptr(firstFingerprint), wantSequence: false, wantPredecessor: true},
		{name: "stale predecessor", expected: 4, last: last, previous: ptr(strings.Repeat("b", 64)), wantSequence: true, wantPredecessor: false},
		{name: "missing predecessor", expected: 4, last: last, wantSequence: true, wantPredecessor: false},
		{name: "predecessor on empty chain", expected: 1, previous: ptr(firstFingerprint), wantSequence: true, wantPredecessor: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ledger.CheckSequence(tt.expected, tt.last)
			if (err == nil) != tt.wantSequence {
				t.Errorf("CheckSequence() error = %v, want success %v", err, tt.wantSequence)
			}
			in := ledger.AppendInput{PreviousFingerprint: tt.previous}
			err = ledger.CheckPredecessor(in, tt.last)
			if (err == nil) != tt.wantPredecessor {
				t.Errorf("CheckPredecessor() error = %v, want success %v", err, tt.wantPredecessor)
			}
		})
	}
}

func merge(in ledger.AppendInput, mutate func(*ledger.AppendInput)) ledger.AppendInput {
	copy := in
	mutate(&copy)
	return copy
}

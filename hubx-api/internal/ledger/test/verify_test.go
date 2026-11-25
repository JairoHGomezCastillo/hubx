package test

import (
	"testing"

	"hubx-api/internal/ledger"
)

func TestHU05_VerifyIntactChain(t *testing.T) {
	tests := []struct {
		name             string
		updates          []ledger.Update
		wantIntact       bool
		wantChecked      int
		wantFirstAltered int
	}{
		{
			name:             "empty chain is intact",
			updates:          []ledger.Update{},
			wantIntact:       true,
			wantChecked:      0,
			wantFirstAltered: 0,
		},
		{
			name: "single intact update",
			updates: []ledger.Update{
				{
					ShipmentID:          "shipment-1",
					Sequence:            1,
					Stage:               "In flight",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T10:00:00Z"),
					PreviousFingerprint: nil,
					Fingerprint:         computeFingerprint(t, 1, "shipment-1", "In flight", "user-1", nil, "2026-11-25T10:00:00Z", nil),
				},
			},
			wantIntact:       true,
			wantChecked:      1,
			wantFirstAltered: 0,
		},
		{
			name: "three intact chained updates",
			updates: func() []ledger.Update {
				u1 := ledger.Update{
					ShipmentID:          "shipment-1",
					Sequence:            1,
					Stage:               "In flight",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T10:00:00Z"),
					PreviousFingerprint: nil,
					Fingerprint:         "",
				}
				u1.Fingerprint = computeFingerprint(t, u1.Sequence, u1.ShipmentID, u1.Stage, u1.ResponsiblePartyID, u1.AssignedPartnerID, u1.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), u1.PreviousFingerprint)

				fp1 := u1.Fingerprint
				u2 := ledger.Update{
					ShipmentID:          "shipment-1",
					Sequence:            2,
					Stage:               "At destination airport",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T11:00:00Z"),
					PreviousFingerprint: &fp1,
					Fingerprint:         "",
				}
				u2.Fingerprint = computeFingerprint(t, u2.Sequence, u2.ShipmentID, u2.Stage, u2.ResponsiblePartyID, u2.AssignedPartnerID, u2.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), u2.PreviousFingerprint)

				partner := "partner-1"
				fp2 := u2.Fingerprint
				u3 := ledger.Update{
					ShipmentID:          "shipment-1",
					Sequence:            3,
					Stage:               "Assigned to partner",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   &partner,
					RecordedAt:          mustParseTime(t, "2026-11-25T12:00:00Z"),
					PreviousFingerprint: &fp2,
					Fingerprint:         "",
				}
				u3.Fingerprint = computeFingerprint(t, u3.Sequence, u3.ShipmentID, u3.Stage, u3.ResponsiblePartyID, u3.AssignedPartnerID, u3.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), u3.PreviousFingerprint)

				return []ledger.Update{u1, u2, u3}
			}(),
			wantIntact:       true,
			wantChecked:      3,
			wantFirstAltered: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ledger.Verify(tt.updates)
			if result.Intact != tt.wantIntact {
				t.Errorf("Intact = %v, want %v", result.Intact, tt.wantIntact)
			}
			if result.UpdatesChecked != tt.wantChecked {
				t.Errorf("UpdatesChecked = %d, want %d", result.UpdatesChecked, tt.wantChecked)
			}
			if result.FirstAlteredUpdate != tt.wantFirstAltered {
				t.Errorf("FirstAlteredUpdate = %d, want %d", result.FirstAlteredUpdate, tt.wantFirstAltered)
			}
		})
	}
}

func TestHU05_VerifyDetectsAlteration(t *testing.T) {
	tests := []struct {
		name             string
		updates          []ledger.Update
		wantIntact       bool
		wantChecked      int
		wantFirstAltered int
	}{
		{
			name: "altered sequence at position 1",
			updates: []ledger.Update{
				{
					ShipmentID:          "shipment-1",
					Sequence:            2, // should be 1
					Stage:               "In flight",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T10:00:00Z"),
					PreviousFingerprint: nil,
					Fingerprint:         computeFingerprint(t, 1, "shipment-1", "In flight", "user-1", nil, "2026-11-25T10:00:00Z", nil),
				},
			},
			wantIntact:       false,
			wantChecked:      1,
			wantFirstAltered: 1,
		},
		{
			name: "broken previous link at position 2",
			updates: func() []ledger.Update {
				u1 := ledger.Update{
					ShipmentID:          "shipment-1",
					Sequence:            1,
					Stage:               "In flight",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T10:00:00Z"),
					PreviousFingerprint: nil,
					Fingerprint:         "",
				}
				u1.Fingerprint = computeFingerprint(t, u1.Sequence, u1.ShipmentID, u1.Stage, u1.ResponsiblePartyID, u1.AssignedPartnerID, u1.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), u1.PreviousFingerprint)

				wrongLink := "0000000000000000000000000000000000000000000000000000000000000000"
				u2 := ledger.Update{
					ShipmentID:          "shipment-1",
					Sequence:            2,
					Stage:               "At destination airport",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T11:00:00Z"),
					PreviousFingerprint: &wrongLink, // wrong
					Fingerprint:         "",
				}
				u2.Fingerprint = computeFingerprint(t, u2.Sequence, u2.ShipmentID, u2.Stage, u2.ResponsiblePartyID, u2.AssignedPartnerID, u2.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), u2.PreviousFingerprint)

				return []ledger.Update{u1, u2}
			}(),
			wantIntact:       false,
			wantChecked:      2,
			wantFirstAltered: 2,
		},
		{
			name: "altered fingerprint at position 1",
			updates: []ledger.Update{
				{
					ShipmentID:          "shipment-1",
					Sequence:            1,
					Stage:               "In flight",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T10:00:00Z"),
					PreviousFingerprint: nil,
					Fingerprint:         "0000000000000000000000000000000000000000000000000000000000000000", // wrong
				},
			},
			wantIntact:       false,
			wantChecked:      1,
			wantFirstAltered: 1,
		},
		{
			name: "altered stage field at position 2",
			updates: func() []ledger.Update {
				u1 := ledger.Update{
					ShipmentID:          "shipment-1",
					Sequence:            1,
					Stage:               "In flight",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T10:00:00Z"),
					PreviousFingerprint: nil,
					Fingerprint:         "",
				}
				u1.Fingerprint = computeFingerprint(t, u1.Sequence, u1.ShipmentID, u1.Stage, u1.ResponsiblePartyID, u1.AssignedPartnerID, u1.RecordedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), u1.PreviousFingerprint)

				fp1 := u1.Fingerprint
				u2 := ledger.Update{
					ShipmentID:          "shipment-1",
					Sequence:            2,
					Stage:               "Delivered", // altered from "At destination airport"
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T11:00:00Z"),
					PreviousFingerprint: &fp1,
					// Fingerprint computed with the original stage
					Fingerprint: computeFingerprint(t, 2, "shipment-1", "At destination airport", "user-1", nil, "2026-11-25T11:00:00Z", &fp1),
				}

				return []ledger.Update{u1, u2}
			}(),
			wantIntact:       false,
			wantChecked:      2,
			wantFirstAltered: 2,
		},
		{
			name: "first update with non-null previous",
			updates: []ledger.Update{
				{
					ShipmentID:          "shipment-1",
					Sequence:            1,
					Stage:               "In flight",
					ResponsiblePartyID:  "user-1",
					AssignedPartnerID:   nil,
					RecordedAt:          mustParseTime(t, "2026-11-25T10:00:00Z"),
					PreviousFingerprint: strPtr("0000000000000000000000000000000000000000000000000000000000000000"), // should be nil
					Fingerprint:         computeFingerprint(t, 1, "shipment-1", "In flight", "user-1", nil, "2026-11-25T10:00:00Z", nil),
				},
			},
			wantIntact:       false,
			wantChecked:      1,
			wantFirstAltered: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ledger.Verify(tt.updates)
			if result.Intact != tt.wantIntact {
				t.Errorf("Intact = %v, want %v", result.Intact, tt.wantIntact)
			}
			if result.UpdatesChecked != tt.wantChecked {
				t.Errorf("UpdatesChecked = %d, want %d", result.UpdatesChecked, tt.wantChecked)
			}
			if result.FirstAlteredUpdate != tt.wantFirstAltered {
				t.Errorf("FirstAlteredUpdate = %d, want %d", result.FirstAlteredUpdate, tt.wantFirstAltered)
			}
		})
	}
}

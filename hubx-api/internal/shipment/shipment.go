// Package shipment implements shipment management for HUBX.
package shipment

import "time"

// Shipment represents a cargo shipment with its traceability data.
// The HU-02 creation fields coexist with the HU-03 read projection fields:
// Stage and AssignedPartnerID are projections of the newest ledger update.
type Shipment struct {
	ID                string
	Code              string
	Origin            string
	Destination       string
	Flight            string
	WeightKg          string // Decimal literal with at most one decimal place
	CustomerEmail     string
	CustomerName      string
	CustomerID        string
	Stage             Stage // StageCreated for a newly created shipment
	AssignedPartnerID *string
	CreatedBy         string
	CreatedAt         time.Time
}

// CreateInput contains the validated data for creating a new shipment.
type CreateInput struct {
	// From JWT claims
	Role       string
	OperatorID string

	// From request body
	Origin        string
	Destination   string
	Flight        string
	WeightKg      string // json.Number literal
	CustomerEmail string
}

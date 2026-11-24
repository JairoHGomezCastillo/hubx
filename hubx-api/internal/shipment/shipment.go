// Package shipment implements shipment management for HUBX.
package shipment

import "time"

// Shipment represents a cargo shipment with its traceability data.
type Shipment struct {
	Code          string
	Origin        string
	Destination   string
	Flight        string
	WeightKg      string // Decimal literal with at most one decimal place
	CustomerEmail string
	CustomerName  string
	Stage         string // Always "Created" for HU-02
	CreatedAt     time.Time
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

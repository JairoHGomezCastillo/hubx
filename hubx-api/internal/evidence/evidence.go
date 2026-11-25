// Package evidence assembles a self-contained traceability evidence document
// from records HUBX already keeps. It never writes to the ledger and never
// imports net/http: it is a domain package.
package evidence

import "time"

// Instructions is the self-contained, English recipe a third party can follow
// to recompute the fingerprints from the document alone. It restates the canonical
// formula of contracts/hu03-record-update.md without needing HUBX access.
const Instructions = `How to verify the fingerprints:
Each fingerprint is the lowercase hexadecimal SHA-256 digest of one UTF-8 string of seven key=value lines.
The lines appear verbatim with each update above and are in this exact order:
shipment_id, sequence, stage, responsible_party_id, assigned_partner_id, recorded_at, previous_fingerprint.
Join the seven lines with a single line feed and no trailing line feed.
Use the single character - for every null value, never an empty value and never the word null.
Write recorded_at in UTC with RFC3339Nano form, truncating the instant to microseconds before hashing.
Hash the seven lines of an update and compare the result with the fingerprint printed for that update.
The first update has - as its previous fingerprint; every later one repeats the preceding fingerprint.`

// ShipmentData is the creation data and current stage printed in the evidence.
type ShipmentData struct {
	Code         string
	Origin       string
	Destination  string
	Flight       string
	WeightKg     string
	CustomerName string
	Stage        string
}

// UpdateEvidence is one recorded update as shown in the evidence document.
// CanonicalInput is the exact seven-line string whose SHA-256 digest is the
// stored fingerprint, so a recipient can check it without calling HUBX.
type UpdateEvidence struct {
	Sequence            int
	Stage               string
	ResponsibleParty    string
	AssignedPartner     string
	RecordedAt          time.Time
	Fingerprint         string
	PreviousFingerprint string // "-" for the first update
	CanonicalInput      string
}

// Evidence is the complete content of one generated document. It is assembled
// on request and is never persisted.
type Evidence struct {
	Shipment     ShipmentData
	Updates      []UpdateEvidence // ascending chain order, empty is valid
	GeneratedAt  time.Time
	Instructions string
}

// Package shipment defines the shipment domain: stages, the strict transition
// table and the errors raised when an update is rejected. It never imports
// net/http or a persistence driver.
package shipment

// Stage is a point of the shipment journey, using the backlog glossary.
type Stage string

const (
	// StageCreated is the initial stage of a new shipment. It is never an update.
	StageCreated              Stage = "Created"
	StageInFlight             Stage = "In flight"
	StageAtDestinationAirport Stage = "At destination airport"
	StageAssignedToPartner    Stage = "Assigned to partner"
	StageOutForDelivery       Stage = "Out for delivery"
	StageDelivered            Stage = "Delivered"
)

// Role identifies the kind of participant recorded as a responsible party.
type Role string

const (
	RoleOperator      Role = "operator"
	RoleGroundPartner Role = "ground partner"
	RoleCustomer      Role = "customer"
)

// Principal is the signed-in party supplied by the API transport layer.
type Principal struct {
	UserID string
	Role   Role
}

// Transition describes the one legitimate way out of a stage.
type Transition struct {
	Next            Stage
	Actor           Role
	RequiresPartner bool
}

// transitions is the strict table derived from the backlog: no repetition, no
// going back, and Delivered is terminal because it has no entry here.
var transitions = map[Stage]Transition{
	StageCreated:              {Next: StageInFlight, Actor: RoleOperator},
	StageInFlight:             {Next: StageAtDestinationAirport, Actor: RoleOperator},
	StageAtDestinationAirport: {Next: StageAssignedToPartner, Actor: RoleOperator, RequiresPartner: true},
	StageAssignedToPartner:    {Next: StageOutForDelivery, Actor: RoleGroundPartner},
	StageOutForDelivery:       {Next: StageDelivered, Actor: RoleGroundPartner},
}

// NextTransition returns the only transition allowed from current. The boolean
// is false for an unknown stage and for the terminal stage Delivered.
func NextTransition(current Stage) (Transition, bool) {
	next, ok := transitions[current]
	return next, ok
}

// NextStage returns the stage that follows current in the strict sequence.
func NextStage(current Stage) (Stage, bool) {
	next, ok := NextTransition(current)
	return next.Next, ok
}

// IsUpdateStage reports whether stage can be stored in the ledger. A newly
// created shipment has no update, so Created is never a valid update stage.
func IsUpdateStage(stage Stage) bool {
	switch stage {
	case StageInFlight, StageAtDestinationAirport, StageAssignedToPartner,
		StageOutForDelivery, StageDelivered:
		return true
	default:
		return false
	}
}

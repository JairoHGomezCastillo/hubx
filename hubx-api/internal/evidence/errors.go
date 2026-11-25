package evidence

import "errors"

var (
	// ErrShipmentNotFound is returned for an unknown code and for a shipment
	// the caller cannot see, so the two cases are indistinguishable.
	ErrShipmentNotFound = errors.New("shipment not found")

	// ErrForbidden is returned when the signed-in caller is not an operator:
	// downloading evidence is operator-only.
	ErrForbidden = errors.New("forbidden: only operators can download evidence")
)

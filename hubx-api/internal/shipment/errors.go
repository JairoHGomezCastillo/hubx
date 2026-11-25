package shipment

import "errors"

var (
	// ErrForbidden is returned when a non-operator attempts to create a shipment.
	ErrForbidden = errors.New("forbidden: only operators can create shipments")

	// ErrMissingData is returned when required fields are missing or blank.
	ErrMissingData = errors.New("required data is missing")

	// ErrSameOriginDestination is returned when origin equals destination (case-insensitive).
	ErrSameOriginDestination = errors.New("origin and destination must be different")

	// ErrWeightOutOfRange is returned when weight is not in [0.1, 5000] or has more than one decimal.
	ErrWeightOutOfRange = errors.New("weight must be between 0.1 and 5000 kg with at most one decimal")

	// ErrCustomerNotFound is returned when the customer email does not exist.
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrNotACustomer is returned when the user exists but role is not 'customer'.
	ErrNotACustomer = errors.New("user is not a customer")

	// ErrDuplicateCode is returned after 3 attempts to generate a unique code.
	ErrDuplicateCode = errors.New("failed to generate unique shipment code after retries")

	// ErrShipmentNotFound is returned when a shipment code is unknown or the
	// caller cannot see it (HU-04 traceability, HU-05 integrity).
	ErrShipmentNotFound = errors.New("shipment not found")
)

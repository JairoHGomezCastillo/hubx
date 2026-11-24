package shipment

import (
	"errors"

	"hubx-api/internal/ledger"
)

var (
	// ErrOutOfSequence means the requested stage does not follow the current one.
	ErrOutOfSequence = errors.New("update does not follow the shipment's sequence")
	// ErrNotResponsible means the caller is not entitled to record this update.
	// It also covers shipments the caller cannot see or that do not exist.
	ErrNotResponsible = errors.New("not responsible for that update")
	// ErrPartnerRequired means the assignment update did not name a ground partner.
	ErrPartnerRequired = errors.New("ground partner required")
	// ErrNotGroundPartner means the named party is not a registered ground partner.
	ErrNotGroundPartner = errors.New("not a registered ground partner")
	// ErrConcurrentUpdate is re-exported from the ledger so callers of the
	// shipment domain can handle a stale chain without importing persistence.
	ErrConcurrentUpdate = ledger.ErrConcurrentUpdate
)

package shipment

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

//go:generate go tool mockgen -source=service.go -destination=mocks/service_mock.go -package=mocks

// Service defines shipment business logic.
type Service interface {
	Create(ctx context.Context, input CreateInput) (*Shipment, error)
}

// ShipmentService implements Service.
type ShipmentService struct {
	repo    Repository
	codeGen CodeGenerator
}

// NewShipmentService creates a new shipment service.
func NewShipmentService(repo Repository, codeGen CodeGenerator) *ShipmentService {
	return &ShipmentService{
		repo:    repo,
		codeGen: codeGen,
	}
}

// Create creates a new shipment after validation. Returns the created shipment
// or a sentinel error (ErrForbidden, ErrMissingData, ErrSameOriginDestination,
// ErrWeightOutOfRange, ErrCustomerNotFound, ErrNotACustomer, ErrDuplicateCode).
func (s *ShipmentService) Create(ctx context.Context, input CreateInput) (*Shipment, error) {
	// 1. Authorization: only operators can create shipments
	if input.Role != "operator" {
		return nil, ErrForbidden
	}

	// 2. Validate required fields (after trimming)
	origin := strings.TrimSpace(input.Origin)
	destination := strings.TrimSpace(input.Destination)
	flight := strings.TrimSpace(input.Flight)
	weightKg := strings.TrimSpace(input.WeightKg)
	customerEmail := strings.TrimSpace(input.CustomerEmail)

	if origin == "" || destination == "" || flight == "" || weightKg == "" || customerEmail == "" {
		return nil, ErrMissingData
	}

	// 3. Validate origin != destination (case-insensitive)
	if strings.EqualFold(origin, destination) {
		return nil, ErrSameOriginDestination
	}

	// 4. Validate weight: [0.1, 5000] with at most one decimal, plain decimal notation only
	if !validWeightRegex.MatchString(weightKg) {
		return nil, ErrWeightOutOfRange
	}

	weight, err := strconv.ParseFloat(weightKg, 64)
	if err != nil {
		return nil, ErrWeightOutOfRange
	}
	if weight < 0.1 || weight > 5000 {
		return nil, ErrWeightOutOfRange
	}

	// 5. Normalize customer email and lookup
	normalizedEmail := strings.ToLower(strings.TrimSpace(customerEmail))
	customer, err := s.repo.FindUserByEmail(ctx, normalizedEmail)
	if err != nil {
		return nil, err // ErrCustomerNotFound or database error
	}

	// 6. Verify customer role
	if customer.Role != "customer" {
		return nil, ErrNotACustomer
	}

	// 7. Generate unique code with retry on duplicate
	const maxAttempts = 3
	var code string
	var insertErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		code = s.codeGen.Generate()

		shipment := &Shipment{
			Code:          code,
			Origin:        origin,
			Destination:   destination,
			Flight:        flight,
			WeightKg:      weightKg,
			CustomerEmail: customer.Email,
			CustomerName:  customer.Name,
			Stage:         "Created",
		}

		insertErr = s.repo.Insert(ctx, shipment, input.OperatorID, customer.ID)
		if insertErr == nil {
			// Success
			return shipment, nil
		}

		// Check if error is ErrDuplicateCode using errors.Is
		if !errors.Is(insertErr, ErrDuplicateCode) {
			// Some other database error
			return nil, insertErr
		}

		// Duplicate code, retry
	}

	// All attempts exhausted: wrap as internal error (not a 4xx sentinel)
	return nil, errors.New("internal error: failed to generate unique shipment code after 3 attempts")
}

// validWeightRegex matches plain decimal literals with at most one decimal digit.
var validWeightRegex = regexp.MustCompile(`^[0-9]+(\.[0-9])?$`)

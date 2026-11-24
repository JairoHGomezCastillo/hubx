package shipment

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"hubx-api/internal/handlers/auth"
	domain "hubx-api/internal/shipment"
)

// createRequest represents the POST /shipments request body.
type createRequest struct {
	Origin        string      `json:"origin"`
	Destination   string      `json:"destination"`
	Flight        string      `json:"flight"`
	WeightKg      json.Number `json:"weightKg"`
	CustomerEmail string      `json:"customerEmail"`
}

// createResponse represents the POST /shipments success response.
type createResponse struct {
	Code        string       `json:"code"`
	Origin      string       `json:"origin"`
	Destination string       `json:"destination"`
	Flight      string       `json:"flight"`
	WeightKg    json.Number  `json:"weightKg"`
	Customer    customerInfo `json:"customer"`
	Stage       string       `json:"stage"`
	CreatedAt   string       `json:"createdAt"`
}

type customerInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// errorResponse represents an error response.
type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Handler handles shipment HTTP requests.
type Handler struct {
	service domain.Service
}

// NewHandler creates a new shipment handler.
func NewHandler(service domain.Service) *Handler {
	return &Handler{service: service}
}

// Create handles POST /shipments requests.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Extract claims from context (injected by RequireAuth middleware)
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
		return
	}

	// Check role BEFORE decoding body (plan D4, contract validation order)
	if claims.Role != "operator" {
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to create shipments")
		return
	}

	// Limit body size to 16 KiB (plan D8, contract requirement)
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

	// First, decode into a raw map to validate weightKg type
	var rawBody map[string]interface{}
	bodyBytes := new(bytes.Buffer)
	if _, err := bodyBytes.ReadFrom(r.Body); err != nil {
		h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELDS", "Required data is missing")
		return
	}

	if err := json.Unmarshal(bodyBytes.Bytes(), &rawBody); err != nil {
		h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELDS", "Required data is missing")
		return
	}

	// Check that weightKg is a number, not a string (contract requirement)
	if weightRaw, ok := rawBody["weightKg"]; ok {
		if _, isString := weightRaw.(string); isString {
			h.writeError(w, http.StatusBadRequest, "WEIGHT_OUT_OF_RANGE", "The weight must be between 0.1 and 5000 kg")
			return
		}
	}

	// Now decode the request properly
	var req createRequest
	if err := json.Unmarshal(bodyBytes.Bytes(), &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELDS", "Required data is missing")
		return
	}

	// Validate that weightKg is not empty and is numeric (contract: JSON number, not string)
	weightStr := string(req.WeightKg)
	if weightStr == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELDS", "Required data is missing")
		return
	}

	// Call service
	input := domain.CreateInput{
		Role:          claims.Role,
		OperatorID:    claims.Sub,
		Origin:        req.Origin,
		Destination:   req.Destination,
		Flight:        req.Flight,
		WeightKg:      string(req.WeightKg),
		CustomerEmail: req.CustomerEmail,
	}

	shipment, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Success: 201 Created
	resp := createResponse{
		Code:        shipment.Code,
		Origin:      shipment.Origin,
		Destination: shipment.Destination,
		Flight:      shipment.Flight,
		WeightKg:    json.Number(shipment.WeightKg),
		Customer: customerInfo{
			Email: shipment.CustomerEmail,
			Name:  shipment.CustomerName,
		},
		Stage:     string(shipment.Stage),
		CreatedAt: shipment.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

// handleServiceError maps domain errors to HTTP responses per the contract.
func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		h.writeError(w, http.StatusForbidden, "FORBIDDEN", "You do not have permission to create shipments")
	case errors.Is(err, domain.ErrMissingData):
		h.writeError(w, http.StatusBadRequest, "MISSING_REQUIRED_FIELDS", "Required data is missing")
	case errors.Is(err, domain.ErrSameOriginDestination):
		h.writeError(w, http.StatusBadRequest, "SAME_ORIGIN_DESTINATION", "The origin and the destination must be different")
	case errors.Is(err, domain.ErrWeightOutOfRange):
		h.writeError(w, http.StatusBadRequest, "WEIGHT_OUT_OF_RANGE", "The weight must be between 0.1 and 5000 kg")
	case errors.Is(err, domain.ErrCustomerNotFound):
		h.writeError(w, http.StatusUnprocessableEntity, "CUSTOMER_NOT_FOUND", "The customer indicated does not exist")
	case errors.Is(err, domain.ErrNotACustomer):
		h.writeError(w, http.StatusUnprocessableEntity, "CUSTOMER_NOT_CUSTOMER_ROLE", "The customer indicated does not have the customer role")
	default:
		// Internal server error (database, duplicate code after retries, etc.)
		slog.Error("shipment creation failed", "error", err)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
}

// writeError writes a JSON error response.
func (h *Handler) writeError(w http.ResponseWriter, status int, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   errorCode,
		Message: message,
	})
}

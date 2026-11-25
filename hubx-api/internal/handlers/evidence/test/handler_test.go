package test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	domainauth "hubx-api/internal/auth"
	"hubx-api/internal/evidence"
	handlerauth "hubx-api/internal/handlers/auth"
	evidencehandler "hubx-api/internal/handlers/evidence"
	"hubx-api/internal/handlers/evidence/mocks"
	domainshipment "hubx-api/internal/shipment"
)

var errUnexpected = errors.New("assembly failed")

func operator() domainshipment.Principal {
	return domainshipment.Principal{UserID: "33333333-3333-4333-8333-333333333333", Role: domainshipment.RoleOperator}
}

func sampleEvidence() *evidence.Evidence {
	return &evidence.Evidence{
		Shipment: evidence.ShipmentData{
			Code:         "HBX-TEST01",
			Origin:       "Bogotá",
			Destination:  "Miami",
			Flight:       "AV-204",
			WeightKg:     "100.0",
			CustomerName: "Distribuciones Andina",
			Stage:        "Assigned to partner",
		},
		Updates: []evidence.UpdateEvidence{
			{
				Sequence:         1,
				Stage:            "In flight",
				ResponsibleParty: "Laura Operator",
				RecordedAt:       time.Date(2026, 11, 21, 10, 15, 30, 0, time.UTC),
				Fingerprint:      strings.Repeat("a", 64),
				CanonicalInput:   "shipment_id=id\nsequence=1\nstage=In flight\nresponsible_party_id=id\nassigned_partner_id=-\nrecorded_at=2026-11-21T10:15:30Z\nprevious_fingerprint=-",
			},
		},
		GeneratedAt:  time.Date(2026, 11, 25, 9, 30, 0, 0, time.UTC),
		Instructions: evidence.Instructions,
	}
}

func passthrough(next http.Handler) http.Handler { return next }

func claimsAuth(principal domainshipment.Principal) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := &domainauth.Claims{Sub: principal.UserID, Role: string(principal.Role)}
			ctx := context.WithValue(r.Context(), handlerauth.ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func request(t *testing.T, principals evidencehandler.PrincipalSource, reader evidencehandler.Reader, code string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	evidencehandler.Register(mux, passthrough, reader, principals)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/shipments/"+code+"/evidence", nil))
	return rec
}

func assertFlatError(t *testing.T, rec *httptest.ResponseRecorder, status int, code, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, status, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if strings.HasPrefix(rec.Body.String(), "%PDF-") {
		t.Fatal("error response delivered a PDF document")
	}
	var body struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body is not flat JSON: %v; body = %s", err, rec.Body.String())
	}
	if body.Error != code || body.Message != message {
		t.Errorf("error body = %+v, want {%s, %s}", body, code, message)
	}
}

func TestHU10_HandlerDeliversPDFWithDownloadHeaders(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockReader(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)
	principal := operator()

	principals.EXPECT().Principal(gomock.Any()).Return(principal, nil)
	reader.EXPECT().Read(gomock.Any(), "HBX-TEST01", principal).Return(sampleEvidence(), nil)

	rec := request(t, principals, reader, "HBX-TEST01")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q, want application/pdf", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="evidence-HBX-TEST01.pdf"` {
		t.Errorf("Content-Disposition = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if !strings.HasPrefix(rec.Body.String(), "%PDF-") {
		t.Error("body is not a PDF document")
	}
	for _, want := range []string{"HUBX traceability evidence", "Updates:", "Generated at:", "How to verify the fingerprints:", "Scope:"} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("PDF does not contain %q", want)
		}
	}
}

func TestHU10_HandlerDeliversEmptyChainStatement(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockReader(ctrl)
	principals := mocks.NewMockPrincipalSource(ctrl)
	doc := sampleEvidence()
	doc.Updates = nil

	principals.EXPECT().Principal(gomock.Any()).Return(operator(), nil)
	reader.EXPECT().Read(gomock.Any(), "HBX-EMPTY", gomock.Any()).Return(doc, nil)

	rec := request(t, principals, reader, "HBX-EMPTY")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "The shipment has no updates yet.") {
		t.Error("PDF does not state the empty chain")
	}
}

func TestHU10_HandlerMapsErrorsToFlatJSON(t *testing.T) {
	tests := []struct {
		name    string
		readErr error
		status  int
		code    string
		message string
	}{
		{
			name:    "unknown or invisible shipment is indistinguishable",
			readErr: evidence.ErrShipmentNotFound,
			status:  http.StatusNotFound,
			code:    "SHIPMENT_NOT_FOUND",
			message: "No shipment exists with that code",
		},
		{
			name:    "non-operator is refused",
			readErr: evidence.ErrForbidden,
			status:  http.StatusForbidden,
			code:    "FORBIDDEN",
			message: "You do not have permission to download evidence",
		},
		{
			name:    "unexpected failure",
			readErr: errUnexpected,
			status:  http.StatusInternalServerError,
			code:    "INTERNAL_SERVER_ERROR",
			message: "An unexpected error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			reader := mocks.NewMockReader(ctrl)
			principals := mocks.NewMockPrincipalSource(ctrl)
			principals.EXPECT().Principal(gomock.Any()).Return(operator(), nil)
			reader.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, tt.readErr)

			rec := request(t, principals, reader, "HBX-TEST01")
			assertFlatError(t, rec, tt.status, tt.code, tt.message)
		})
	}
}

func TestHU10_HandlerRefusesMissingSessionBeforeReading(t *testing.T) {
	tests := []struct {
		name      string
		principal domainshipment.Principal
		err       error
	}{
		{
			name: "principal source failure",
			err:  errors.New("no claims"),
		},
		{
			name:      "empty principal",
			principal: domainshipment.Principal{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			reader := mocks.NewMockReader(ctrl)
			principals := mocks.NewMockPrincipalSource(ctrl)
			principals.EXPECT().Principal(gomock.Any()).Return(tt.principal, tt.err)

			rec := request(t, principals, reader, "HBX-TEST01")
			assertFlatError(t, rec, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
		})
	}
}

func TestHU10_HandlerClaimsPrincipalSourceReadsRequireAuthContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	reader := mocks.NewMockReader(ctrl)
	principal := operator()
	reader.EXPECT().Read(gomock.Any(), "HBX-TEST01", principal).Return(sampleEvidence(), nil)

	mux := http.NewServeMux()
	evidencehandler.Register(mux, claimsAuth(principal), reader, evidencehandler.NewClaimsPrincipalSource())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/shipments/HBX-TEST01/evidence", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

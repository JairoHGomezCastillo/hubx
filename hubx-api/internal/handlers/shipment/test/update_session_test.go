package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/mock/gomock"

	authdomain "hubx-api/internal/auth"
	authhandler "hubx-api/internal/handlers/auth"
	"hubx-api/internal/handlers/shipment"
	"hubx-api/internal/handlers/shipment/mocks"
	"hubx-api/internal/ledger"
	domainshipment "hubx-api/internal/shipment"
)

type sessionErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// TestHU03_UpdateRoutesSessionErrorContract exercises the exact wiring used by
// cmd/api: the HU-03 routes must use the session middleware, not the HU-01
// RequireAuth one, so a rejected JWT reaches the nested contract error.
func TestHU03_UpdateRoutesSessionErrorContract(t *testing.T) {
	const secret = "test-secret-key-32-bytes-long!!"

	validToken, err := authdomain.GenerateToken(&authdomain.User{
		ID:    handlerOperatorID,
		Email: "operator@hubx.co",
		Role:  "operator",
	}, secret, 1)
	if err != nil {
		t.Fatalf("generate valid token: %v", err)
	}
	wrongSecretToken, err := authdomain.GenerateToken(&authdomain.User{
		ID:    handlerOperatorID,
		Email: "operator@hubx.co",
		Role:  "operator",
	}, "wrong-secret-key-32-bytes-long!!", 1)
	if err != nil {
		t.Fatalf("generate wrong-secret token: %v", err)
	}
	now := time.Now()
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, authdomain.Claims{
		Sub:   handlerOperatorID,
		Email: "operator@hubx.co",
		Role:  "operator",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
		},
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}

	sessions := []struct {
		name      string
		header    string
		wantValid bool
	}{
		{name: "valid token", header: "Bearer " + validToken, wantValid: true},
		{name: "missing token", header: ""},
		{name: "wrong scheme", header: "Basic " + validToken},
		{name: "malformed token", header: "Bearer not-a-valid-jwt"},
		{name: "invalid signature", header: "Bearer " + wrongSecretToken},
		{name: "expired token", header: "Bearer " + expiredToken},
	}
	routes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "POST updates", method: http.MethodPost, path: "/shipments/" + handlerShipmentCode + "/updates", body: `{"stage":"In flight"}`},
		{name: "GET ground partners", method: http.MethodGet, path: "/ground-partners"},
	}

	for _, route := range routes {
		for _, session := range sessions {
			t.Run(route.name+"/"+session.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				recorder := mocks.NewMockRecorder(ctrl)
				directory := mocks.NewMockPartyDirectory(ctrl)
				partners := mocks.NewMockGroundPartnerRepository(ctrl)

				if session.wantValid {
					principal := domainshipment.Principal{UserID: handlerOperatorID, Role: domainshipment.RoleOperator}
					switch route.method {
					case http.MethodPost:
						update := &ledger.Update{
							ID:                 9,
							ShipmentID:         handlerShipmentID,
							Sequence:           1,
							Stage:              string(domainshipment.StageInFlight),
							ResponsiblePartyID: handlerOperatorID,
							RecordedAt:         time.Date(2026, 11, 24, 10, 15, 30, 0, time.UTC),
							Fingerprint:        handlerFingerprint,
						}
						recorder.EXPECT().Record(gomock.Any(), handlerShipmentCode, domainshipment.StageInFlight,
							(*string)(nil), principal).Return(update, nil)
						directory.EXPECT().Name(gomock.Any(), handlerOperatorID).Return("Test Operator", nil)
					case http.MethodGet:
						partners.EXPECT().List(gomock.Any()).Return([]domainshipment.GroundPartner{}, nil)
					}
				}

				mux := http.NewServeMux()
				shipment.RegisterUpdates(mux, authhandler.NewMiddleware(secret).RequireSessionAuth,
					recorder, shipment.NewClaimsPrincipalSource(), directory, partners)

				req := httptest.NewRequest(route.method, route.path, strings.NewReader(route.body))
				if session.header != "" {
					req.Header.Set("Authorization", session.header)
				}
				res := httptest.NewRecorder()
				mux.ServeHTTP(res, req)

				if session.wantValid {
					wantStatus := http.StatusOK
					if route.method == http.MethodPost {
						wantStatus = http.StatusCreated
					}
					if res.Code != wantStatus {
						t.Fatalf("status = %d, want %d (body %s)", res.Code, wantStatus, res.Body.String())
					}
					return
				}
				if res.Code != http.StatusUnauthorized {
					t.Fatalf("status = %d, want %d (body %s)", res.Code, http.StatusUnauthorized, res.Body.String())
				}
				var body sessionErrorBody
				if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
					t.Fatalf("decode error body: %v", err)
				}
				if body.Error.Code != "SESSION_EXPIRED" || body.Error.Message != "Session expired; sign in again." {
					t.Errorf("error = %+v, want SESSION_EXPIRED / Session expired; sign in again.", body.Error)
				}
			})
		}
	}
}

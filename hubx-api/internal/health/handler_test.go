package health

import (
	"net/http/httptest"
	"testing"
)

const wantJSON = `{"status":"ok","service":"hubx-api"}`

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "GET /health responde OK con estado del servicio",
			method:     "GET",
			wantStatus: 200,
			wantBody:   wantJSON,
		},
		{
			name:       "POST /health no es permitido",
			method:     "POST",
			wantStatus: 405,
			wantBody:   "Method Not Allowed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()

			NewHandler().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Body.String(); got != tt.wantBody {
				t.Errorf("body = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

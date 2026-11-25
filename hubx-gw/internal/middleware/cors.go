package middleware

import "net/http"

// CORS returns the restrictive cross-origin middleware of the perimeter
// contract. allowedOrigins are compared exactly; an empty list grants nothing.
// A preflight (OPTIONS with Origin and Access-Control-Request-Method) is
// answered here and never reaches the JWT, the limiter or the API. Other
// requests are always processed; only an allowed origin gets a permission
// header. Credentials are never allowed.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Add("Vary", "Origin")

			origin := ""
			if values := r.Header.Values("Origin"); len(values) == 1 {
				origin = values[0]
			}
			_, isAllowed := allowed[origin]
			isAllowed = isAllowed && origin != ""

			if isPreflight(r) {
				if !isAllowed {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", "GET, POST")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if isAllowed {
				h.Set("Access-Control-Allow-Origin", origin)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Origin") != "" &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

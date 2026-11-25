package middleware

import "net/http"

// securityHeaders are the browser protections of the gateway perimeter
// contract (contracts/gateway-perimeter.md). The gateway is authoritative: the
// proxy removes any copy that comes from the API.
var securityHeaders = [][2]string{
	{"X-Content-Type-Options", "nosniff"},
	{"X-Frame-Options", "DENY"},
	{"Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'"},
	{"Referrer-Policy", "no-referrer"},
	{"Strict-Transport-Security", "max-age=31536000"},
	{"Cache-Control", "no-store"},
}

// SecurityHeaders sets the contract's security headers on every response,
// including the ones produced by later middlewares, the mux and the proxy.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		for _, kv := range securityHeaders {
			h.Set(kv[0], kv[1])
		}
		next.ServeHTTP(w, r)
	})
}

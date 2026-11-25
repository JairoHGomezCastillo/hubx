// Package proxy encapsula el reverso hacia el API de HUBX.
// El gateway no conoce el dominio: solo sabe reenviar y traducir
// la ausencia del upstream en un error claro para el cliente.
package proxy

import (
	"net/http"
	httputil "net/http/httputil"
	"net/url"
	"strings"
)

// gatewayOwnedHeaders son las cabeceras de seguridad que el gateway fija en
// cada respuesta; las que llegan de la API se descartan.
var gatewayOwnedHeaders = []string{
	"X-Content-Type-Options",
	"X-Frame-Options",
	"Content-Security-Policy",
	"Referrer-Policy",
	"Strict-Transport-Security",
	"Cache-Control",
}

// GatewaySecretHeader identifica al gateway ante la API. La API rechaza
// cualquier petición que no la traiga con el valor compartido, de modo que
// el tráfico legítimo siempre entra por el gateway (constitución VI).
const GatewaySecretHeader = "X-Gateway-Secret"

// New devuelve el handler que reenvía a la API lo que llega bajo /api,
// eliminando ese prefijo antes de contactar al upstream. Inyecta en cada
// petición la cabecera del secreto compartido con la API.
func New(apiURL, gatewaySecret string) http.Handler {
	target, err := url.Parse(apiURL)
	if err != nil {
		// Una URL de upstream inválida es un error de configuración:
		// el handler resultante responde siempre 502.
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		})
	}

	rp := httputil.NewSingleHostReverseProxy(target)

	// Sin esta corrección, el proxy conserva el Host del gateway y muchos
	// upstreams (y pruebas) lo interpretan mal; el estándar es usar el
	// Host del destino.
	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		// El cliente no puede suplantar al gateway: descartamos cualquier
		// cabecera entrante y fijamos la nuestra con el secreto compartido.
		req.Header.Set(GatewaySecretHeader, gatewaySecret)
	}

	// El gateway es la autoridad de las cabeceras de seguridad y de CORS
	// (contracts/gateway-perimeter.md): las copias que envíe la API se eliminan
	// para que cada una aparezca una sola vez, con el valor del gateway.
	// Vary no se toca: conserva el Origin que añadió el gateway.
	rp.ModifyResponse = func(res *http.Response) error {
		for _, name := range gatewayOwnedHeaders {
			res.Header.Del(name)
		}
		for name := range res.Header {
			if strings.HasPrefix(strings.ToLower(name), "access-control-") {
				res.Header.Del(name)
			}
		}
		return nil
	}

	// ReverseProxy descarta en silencio los errores de transporte y escribe
	// 502 en el log interno; aquí elevamos esa condición al cliente con una
	// respuesta explícita en lugar de dejar la conexión colgando.
	rp.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
	}

	return http.StripPrefix("/api", rp)
}

// Register monta el proxy en el mux de la aplicación bajo el prefijo /api.
// La eliminación del prefijo ocurre una única vez, dentro de New.
func Register(mux *http.ServeMux, apiURL, gatewaySecret string) {
	mux.Handle("/api/", New(apiURL, gatewaySecret))
}

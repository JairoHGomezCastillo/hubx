// Package config reads hubx-gw configuration following the Configuration
// subsection of Principle III of the Hubx Constitution. It uses only the
// standard library and resolves values with the precedence: environment
// variable, then application-<profile>.properties, then application.properties,
// then the code default. Secrets are read only from the environment.
//
// Error messages name keys and never print values read from the environment, so
// a failed start cannot leak a secret into the logs.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	baseFileName  = "application.properties"
	profilePrefix = "application-"
	profileSuffix = ".properties"
	profileEnvKey = "APP_PROFILE"
	portAliasKey  = "PORT"

	serverPortKey      = "server.port"
	apiURLKey          = "api.url"
	allowedOriginsKey  = "cors.allowed-origins"
	requestsPerMinKey  = "ratelimit.requests-per-minute"
	signInPerMinuteKey = "ratelimit.sign-in-requests-per-minute"

	jwtSecretEnv     = "JWT_SECRET"
	gatewaySecretEnv = "GATEWAY_SECRET"
)

// defaults holds every non-secret key, so environment variables apply even if
// no file exists. Secrets are never here: they come only from the environment.
var defaults = map[string]string{
	serverPortKey:      "8080",
	apiURLKey:          "http://localhost:8081",
	allowedOriginsKey:  "http://localhost:5173",
	requestsPerMinKey:  "120",
	signInPerMinuteKey: "120",
}

// Config holds the resolved, validated configuration of the gateway.
type Config struct {
	Port                    string
	APIURL                  string
	AllowedOrigins          []string
	RequestsPerMinute       int
	SignInRequestsPerMinute int
	JWTSecret               string
	GatewaySecret           string
}

// Load reads configuration files from dir. getenv injects environment lookups
// so tests control the highest-precedence layer. A missing file logs a warning
// and falls back to lower-precedence values; a malformed line, an invalid value
// or a missing secret is an error that must stop the start.
func Load(dir string, getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}

	paths := []string{filepath.Join(dir, baseFileName)}
	if profile := getenv(profileEnvKey); profile != "" {
		paths = append(paths, filepath.Join(dir, profilePrefix+profile+profileSuffix))
	}

	resolved := make(map[string]string, len(defaults))
	for key, value := range defaults {
		resolved[key] = value
	}

	// Files are applied from lowest to highest precedence: base, then profile.
	// Only known keys are kept: secret keys that appear in a file are ignored.
	for _, path := range paths {
		values, err := readProperties(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				slog.Warn("configuration file not found; using lower-precedence values", "path", path)
				continue
			}
			return Config{}, err
		}
		for key, value := range values {
			if _, known := defaults[key]; known {
				resolved[key] = value
			}
		}
	}

	// PORT is an alias of server.port and is applied before the primary env
	// variable so that SERVER_PORT wins when both are set.
	if value := getenv(portAliasKey); value != "" {
		resolved[serverPortKey] = value
	}
	for key := range resolved {
		if value := getenv(envKey(key)); value != "" {
			resolved[key] = value
		}
	}

	cfg := Config{
		APIURL:        resolved[apiURLKey],
		JWTSecret:     getenv(jwtSecretEnv),
		GatewaySecret: getenv(gatewaySecretEnv),
	}

	port, err := parsePort(resolved[serverPortKey])
	if err != nil {
		return Config{}, err
	}
	cfg.Port = port

	if cfg.RequestsPerMinute, err = parseLimit(requestsPerMinKey, resolved[requestsPerMinKey]); err != nil {
		return Config{}, err
	}
	if cfg.SignInRequestsPerMinute, err = parseLimit(signInPerMinuteKey, resolved[signInPerMinuteKey]); err != nil {
		return Config{}, err
	}
	if cfg.AllowedOrigins, err = parseOrigins(resolved[allowedOriginsKey]); err != nil {
		return Config{}, err
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("%s environment variable is required", jwtSecretEnv)
	}
	if cfg.GatewaySecret == "" {
		return Config{}, fmt.Errorf("%s environment variable is required", gatewaySecretEnv)
	}
	return cfg, nil
}

// envKey derives the environment variable name from a property key: uppercase
// with '.' and '-' replaced by '_' (server.port -> SERVER_PORT).
func envKey(key string) string {
	return strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(key))
}

// parsePort validates a TCP port.
func parsePort(value string) (string, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("%s must be an integer between 1 and 65535", serverPortKey)
	}
	return strconv.Itoa(n), nil
}

// parseLimit strictly parses a positive integer limit ("12abc" fails).
func parseLimit(key, value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return n, nil
}

// parseOrigins parses a comma separated list of origins. An empty list is
// valid and grants nothing. Each origin must be scheme://host[:port] with an
// http or https scheme and no path, query, fragment or userinfo; "*" and "null"
// are never valid.
func parseOrigins(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	origins := []string{}
	if value == "" {
		return origins, nil
	}
	for i, raw := range strings.Split(value, ",") {
		origin, ok := normaliseOrigin(strings.TrimSpace(raw))
		if !ok {
			return nil, fmt.Errorf("%s: entry %d is not a valid origin (scheme://host[:port], http or https)", allowedOriginsKey, i+1)
		}
		origins = append(origins, origin)
	}
	return origins, nil
}

func normaliseOrigin(raw string) (string, bool) {
	if raw == "" || raw == "*" || strings.EqualFold(raw, "null") {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	if u.Host == "" || u.Hostname() == "" || u.User != nil || u.Opaque != "" ||
		u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" ||
		strings.ContainsAny(raw, "?#") {
		return "", false
	}
	return scheme + "://" + strings.ToLower(u.Host), true
}

// readProperties parses a properties file. Blank lines and lines starting
// with '#' or '!' are comments. Any other line without '=' or with an empty
// key is malformed and stops loading.
func readProperties(path string) (map[string]string, error) {
	// #nosec G304 -- path is not external/request input. It is built in Load from a
	// caller-supplied directory (hardcoded "." in cmd/gw) joined with internal
	// constant filenames and the operator-controlled APP_PROFILE env var. No
	// end-user data reaches this call, so directory traversal is not reachable here.
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		sep := strings.IndexByte(line, '=')
		if sep < 0 {
			return nil, fmt.Errorf("%s:%d: malformed line without '='", path, lineNo)
		}
		key := strings.TrimSpace(line[:sep])
		if key == "" {
			return nil, fmt.Errorf("%s:%d: malformed line with empty key", path, lineNo)
		}
		values[key] = strings.TrimSpace(line[sep+1:])
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return values, nil
}

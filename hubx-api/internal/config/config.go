// Package config reads hubx-api configuration following the Configuration
// subsection of Principle III of the Hubx Constitution. It uses only the
// standard library and resolves values with the precedence: environment
// variable, then application-<profile>.properties, then application.properties,
// then the code default.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	baseFileName              = "application.properties"
	profilePrefix             = "application-"
	profileSuffix             = ".properties"
	profileEnvKey             = "APP_PROFILE"
	portAliasKey              = "PORT"
	serverPortKey             = "server.port"
	jwtExpirationHoursKey     = "jwt.expiration.hours"
	defaultPort               = "8081"
	defaultJWTExpirationHours = "1"
)

// defaults holds values with no secrets. Secrets are supplied only through
// the environment and therefore never appear here.
var defaults = map[string]string{
	serverPortKey:         defaultPort,
	jwtExpirationHoursKey: defaultJWTExpirationHours,
}

// Config holds the resolved configuration values of the service.
type Config struct {
	values map[string]string
}

// Get returns the resolved value for key, or the empty string when the key
// is not part of the configuration.
func (c *Config) Get(key string) string {
	return c.values[key]
}

// Load reads configuration files from dir. getenv injects environment lookups
// so tests control the highest-precedence layer. A missing file logs a warning
// and falls back to lower-precedence values; a malformed line is an error.
func Load(dir string, getenv func(string) string) (*Config, error) {
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
	for _, path := range paths {
		values, err := readProperties(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				slog.Warn("configuration file not found; using lower-precedence values", "path", path)
				continue
			}
			return nil, err
		}
		for key, value := range values {
			resolved[key] = value
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

	return &Config{values: resolved}, nil
}

// envKey derives the environment variable name from a property key: uppercase
// with '.' and '-' replaced by '_' (server.port -> SERVER_PORT).
func envKey(key string) string {
	return strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(key))
}

// readProperties parses a properties file. Blank lines and lines starting
// with '#' or '!' are comments. Any other line without '=' or with an empty
// key is malformed and stops loading.
func readProperties(path string) (map[string]string, error) {
	// #nosec G304 -- path is not external/request input. It is built in Load from a
	// caller-supplied directory (hardcoded "." in cmd/api and cmd/migrate) joined with
	// internal constant filenames and the operator-controlled APP_PROFILE env var. No
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

// GetInt returns the configuration value as an integer. If the key is not found
// or the value cannot be parsed, returns the defaultValue.
func (c *Config) GetInt(key string, defaultValue int) int {
	value := c.Get(key)
	if value == "" {
		return defaultValue
	}
	var result int
	_, err := fmt.Sscanf(value, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

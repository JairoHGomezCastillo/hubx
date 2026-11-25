package test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"hubx-gw/internal/config"
)

const (
	testJWTSecret     = "jwt-secret-value-for-tests-0001"
	testGatewaySecret = "gateway-secret-value-for-tests-0002"
)

// env builds a getenv function over a map. The two required secrets are always
// present unless the caller removes them.
func env(extra map[string]string) func(string) string {
	values := map[string]string{
		"JWT_SECRET":     testJWTSecret,
		"GATEWAY_SECRET": testGatewaySecret,
	}
	for k, v := range extra {
		if v == "" {
			delete(values, k)
			continue
		}
		values[k] = v
	}
	return func(key string) string { return values[key] }
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestLoad_DefaultsWhenFileIsMissing(t *testing.T) {
	cfg, err := config.Load(t.TempDir(), env(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.APIURL != "http://localhost:8081" {
		t.Errorf("APIURL = %q", cfg.APIURL)
	}
	if len(cfg.AllowedOrigins) != 0 {
		t.Errorf("AllowedOrigins = %v, want empty: no cross origin is allowed by default", cfg.AllowedOrigins)
	}
	if cfg.RequestsPerMinute != 120 {
		t.Errorf("RequestsPerMinute = %d, want 120", cfg.RequestsPerMinute)
	}
	if cfg.SignInRequestsPerMinute != 10 {
		t.Errorf("SignInRequestsPerMinute = %d, want 10", cfg.SignInRequestsPerMinute)
	}
	if cfg.JWTSecret != testJWTSecret || cfg.GatewaySecret != testGatewaySecret {
		t.Errorf("secrets were not read from the environment")
	}
}

func TestLoad_EnvironmentAppliesWithoutFile(t *testing.T) {
	cfg, err := config.Load(t.TempDir(), env(map[string]string{
		"API_URL":                               "http://api:9000",
		"SERVER_PORT":                           "9090",
		"RATELIMIT_REQUESTS_PER_MINUTE":         "5",
		"RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE": "3",
		"CORS_ALLOWED_ORIGINS":                  "https://hubx.example",
	}))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIURL != "http://api:9000" || cfg.Port != "9090" {
		t.Errorf("APIURL/Port = %q/%q", cfg.APIURL, cfg.Port)
	}
	if cfg.RequestsPerMinute != 5 || cfg.SignInRequestsPerMinute != 3 {
		t.Errorf("limits = %d/%d, want 5/3", cfg.RequestsPerMinute, cfg.SignInRequestsPerMinute)
	}
	if want := []string{"https://hubx.example"}; !reflect.DeepEqual(cfg.AllowedOrigins, want) {
		t.Errorf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, want)
	}
}

func TestLoad_PortAlias(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"PORT alias", map[string]string{"PORT": "7000"}, "7000"},
		{"SERVER_PORT wins over PORT", map[string]string{"PORT": "7000", "SERVER_PORT": "7100"}, "7100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(t.TempDir(), env(tt.env))
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.Port != tt.want {
				t.Errorf("Port = %q, want %q", cfg.Port, tt.want)
			}
		})
	}
}

func TestLoad_Precedence(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "application.properties", "# comment\n! other comment\n\nserver.port=1111\napi.url=http://file:1\nratelimit.requests-per-minute=11\nunknown.key=ignored\n")
	writeFile(t, dir, "application-staging.properties", "server.port=2222\nratelimit.requests-per-minute=22\n")

	t.Run("file only", func(t *testing.T) {
		cfg, err := config.Load(dir, env(nil))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Port != "1111" || cfg.APIURL != "http://file:1" || cfg.RequestsPerMinute != 11 {
			t.Errorf("got %q %q %d", cfg.Port, cfg.APIURL, cfg.RequestsPerMinute)
		}
		if cfg.SignInRequestsPerMinute != 10 {
			t.Errorf("a key absent from the file must keep its default, got %d", cfg.SignInRequestsPerMinute)
		}
	})

	t.Run("profile overrides file", func(t *testing.T) {
		cfg, err := config.Load(dir, env(map[string]string{"APP_PROFILE": "staging"}))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Port != "2222" || cfg.RequestsPerMinute != 22 || cfg.APIURL != "http://file:1" {
			t.Errorf("got %q %q %d", cfg.Port, cfg.APIURL, cfg.RequestsPerMinute)
		}
	})

	t.Run("environment overrides profile", func(t *testing.T) {
		cfg, err := config.Load(dir, env(map[string]string{
			"APP_PROFILE":                   "staging",
			"SERVER_PORT":                   "3333",
			"RATELIMIT_REQUESTS_PER_MINUTE": "33",
		}))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Port != "3333" || cfg.RequestsPerMinute != 33 {
			t.Errorf("got %q %d", cfg.Port, cfg.RequestsPerMinute)
		}
	})

	t.Run("missing profile file falls back", func(t *testing.T) {
		cfg, err := config.Load(dir, env(map[string]string{"APP_PROFILE": "nope"}))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Port != "1111" {
			t.Errorf("Port = %q, want 1111", cfg.Port)
		}
	})
}

func TestLoad_MalformedLine(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"no separator", "server.port=8080\nthis line is wrong\n"},
		{"empty key", "=value\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "application.properties", tt.content)
			if _, err := config.Load(dir, env(nil)); err == nil {
				t.Fatal("Load() error = nil, want error")
			}
		})
	}
}

func TestLoad_SecretsOnlyFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "application.properties", "jwt.secret=from-the-file\ngateway.secret=also-from-the-file\n")

	t.Run("file values are never used", func(t *testing.T) {
		_, err := config.Load(dir, env(map[string]string{"JWT_SECRET": "", "GATEWAY_SECRET": ""}))
		if err == nil {
			t.Fatal("Load() error = nil, want error: secrets missing from the environment")
		}
		if strings.Contains(err.Error(), "from-the-file") {
			t.Errorf("error leaks a file value: %v", err)
		}
	})

	t.Run("environment wins and file is ignored", func(t *testing.T) {
		cfg, err := config.Load(dir, env(nil))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.JWTSecret != testJWTSecret || cfg.GatewaySecret != testGatewaySecret {
			t.Errorf("secrets must come from the environment only")
		}
	})
}

func TestLoad_MissingSecretStopsStart(t *testing.T) {
	tests := []struct {
		name    string
		missing string
	}{
		{"jwt secret", "JWT_SECRET"},
		{"gateway secret", "GATEWAY_SECRET"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Load(t.TempDir(), env(map[string]string{tt.missing: ""}))
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.missing) {
				t.Errorf("error %q should name %s", err, tt.missing)
			}
		})
	}
}

func TestLoad_InvalidLimits(t *testing.T) {
	keys := map[string]string{
		"ratelimit.requests-per-minute":         "RATELIMIT_REQUESTS_PER_MINUTE",
		"ratelimit.sign-in-requests-per-minute": "RATELIMIT_SIGN_IN_REQUESTS_PER_MINUTE",
	}
	values := []string{"0", "-1", "12abc", "abc", "1.5", "1 2", "+", "99999999999999999999"}

	for key, envName := range keys {
		for _, value := range values {
			t.Run("env "+envName+"="+value, func(t *testing.T) {
				_, err := config.Load(t.TempDir(), env(map[string]string{envName: value}))
				if err == nil {
					t.Fatalf("Load() error = nil for %s=%q", envName, value)
				}
				if strings.Contains(err.Error(), testJWTSecret) || strings.Contains(err.Error(), testGatewaySecret) {
					t.Errorf("error leaks a secret: %v", err)
				}
			})
			t.Run("file "+key+"="+value, func(t *testing.T) {
				dir := t.TempDir()
				writeFile(t, dir, "application.properties", key+"="+value+"\n")
				if _, err := config.Load(dir, env(nil)); err == nil {
					t.Fatalf("Load() error = nil for %s=%q", key, value)
				}
			})
		}
		t.Run("file "+key+" empty", func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "application.properties", key+"=\n")
			if _, err := config.Load(dir, env(nil)); err == nil {
				t.Fatalf("Load() error = nil for empty %s", key)
			}
		})
	}
}

func TestLoad_LimitsAreTrimmed(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "application.properties", "ratelimit.requests-per-minute =  7  \n")
	cfg, err := config.Load(dir, env(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.RequestsPerMinute != 7 {
		t.Errorf("RequestsPerMinute = %d, want 7", cfg.RequestsPerMinute)
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	for _, value := range []string{"0", "65536", "abc", "-1", "80x"} {
		t.Run(value, func(t *testing.T) {
			if _, err := config.Load(t.TempDir(), env(map[string]string{"PORT": value})); err == nil {
				t.Fatalf("Load() error = nil for port %q", value)
			}
		})
	}
}

func TestLoad_Origins(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    []string
		wantErr bool
	}{
		{name: "single", value: "http://localhost:5173", want: []string{"http://localhost:5173"}},
		{name: "trimmed list", value: " http://a.example , https://b.example:8443 ", want: []string{"http://a.example", "https://b.example:8443"}},
		{name: "empty list grants nothing", value: "", want: []string{}},
		{name: "blank list grants nothing", value: "   ", want: []string{}},
		{name: "wildcard", value: "*", wantErr: true},
		{name: "wildcard inside list", value: "http://a.example,*", wantErr: true},
		{name: "null", value: "null", wantErr: true},
		{name: "path", value: "http://a.example/app", wantErr: true},
		{name: "trailing slash", value: "http://a.example/", wantErr: true},
		{name: "query", value: "http://a.example?x=1", wantErr: true},
		{name: "fragment", value: "http://a.example#f", wantErr: true},
		{name: "userinfo", value: "http://user:pw@a.example", wantErr: true},
		{name: "ftp scheme", value: "ftp://a.example", wantErr: true},
		{name: "no scheme", value: "a.example", wantErr: true},
		{name: "no host", value: "http://", wantErr: true},
		{name: "bad port", value: "http://a.example:port", wantErr: true},
		{name: "empty entry", value: "http://a.example,,https://b.example", wantErr: true},
		{name: "trailing comma", value: "http://a.example,", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "application.properties", "cors.allowed-origins="+tt.value+"\n")
			cfg, err := config.Load(dir, env(nil))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() error = nil, want error; origins = %v", cfg.AllowedOrigins)
				}
				if strings.Contains(err.Error(), testJWTSecret) {
					t.Errorf("error leaks a secret: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if len(cfg.AllowedOrigins) != len(tt.want) {
				t.Fatalf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, tt.want)
			}
			for i := range tt.want {
				if cfg.AllowedOrigins[i] != tt.want[i] {
					t.Errorf("AllowedOrigins[%d] = %q, want %q", i, cfg.AllowedOrigins[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoad_OriginsFromEnvironmentAreValidatedToo(t *testing.T) {
	if _, err := config.Load(t.TempDir(), env(map[string]string{"CORS_ALLOWED_ORIGINS": "*"})); err == nil {
		t.Fatal("Load() error = nil, want error for wildcard from the environment")
	}
}

func TestLoad_RealPropertiesFile(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", ".."), env(nil))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "8080" || cfg.RequestsPerMinute != 120 || cfg.SignInRequestsPerMinute != 10 {
		t.Errorf("real file values = %q %d %d, want 8080 120 10", cfg.Port, cfg.RequestsPerMinute, cfg.SignInRequestsPerMinute)
	}
	if len(cfg.AllowedOrigins) != 0 {
		t.Errorf("AllowedOrigins = %v, want empty", cfg.AllowedOrigins)
	}
}

func TestLoad_EmptyEnvironmentValueCountsAsUnset(t *testing.T) {
	// env() drops empty values, so build the lookup by hand to return "" for the keys.
	getenv := func(key string) string {
		switch key {
		case "JWT_SECRET":
			return testJWTSecret
		case "GATEWAY_SECRET":
			return testGatewaySecret
		}
		return ""
	}
	dir := t.TempDir()
	writeFile(t, dir, "application.properties", "cors.allowed-origins=https://file.example\nratelimit.sign-in-requests-per-minute=4\n")
	cfg, err := config.Load(dir, getenv)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if want := []string{"https://file.example"}; !reflect.DeepEqual(cfg.AllowedOrigins, want) {
		t.Errorf("AllowedOrigins = %v, want %v", cfg.AllowedOrigins, want)
	}
	if cfg.SignInRequestsPerMinute != 4 {
		t.Errorf("SignInRequestsPerMinute = %d, want 4", cfg.SignInRequestsPerMinute)
	}
}

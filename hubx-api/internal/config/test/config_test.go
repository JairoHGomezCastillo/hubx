package test

import (
	"os"
	"path/filepath"
	"testing"

	"hubx-api/internal/config"
)

// envLookup adapts a map to the getenv function injected into Load.
func envLookup(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

// writeProperties writes a properties file into dir for the test case.
func writeProperties(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		env      map[string]string
		wantPort string
		wantErr  bool
	}{
		{
			name:     "no env and no file uses code default",
			wantPort: "8081",
		},
		{
			name:     "base file overrides code default",
			files:    map[string]string{"application.properties": "server.port=9000\n"},
			wantPort: "9000",
		},
		{
			// Explicit regression case: spaces around '=' must be trimmed from
			// both the key and the value (spec 000, acceptance criterion 3).
			name:     "base file with spaces around equals sign",
			files:    map[string]string{"application.properties": "server.port = 9300\n"},
			wantPort: "9300",
		},
		{
			name: "profile file overrides base file",
			files: map[string]string{
				"application.properties":     "server.port=9000\n",
				"application-dev.properties": "server.port=9100\n",
			},
			env:      map[string]string{"APP_PROFILE": "dev"},
			wantPort: "9100",
		},
		{
			name: "profile file ignored when APP_PROFILE unset",
			files: map[string]string{
				"application.properties":     "server.port=9000\n",
				"application-dev.properties": "server.port=9100\n",
			},
			wantPort: "9000",
		},
		{
			name:     "environment variable overrides files",
			files:    map[string]string{"application.properties": "server.port=9000\n"},
			env:      map[string]string{"SERVER_PORT": "9200"},
			wantPort: "9200",
		},
		{
			name:     "PORT alias overrides files",
			files:    map[string]string{"application.properties": "server.port=9000\n"},
			env:      map[string]string{"PORT": "9200"},
			wantPort: "9200",
		},
		{
			name:     "primary env key wins over PORT alias",
			env:      map[string]string{"SERVER_PORT": "9300", "PORT": "9200"},
			wantPort: "9300",
		},
		{
			name:     "comments and blank lines are ignored",
			files:    map[string]string{"application.properties": "# comment\n\nserver.port = 9000\n! legacy comment\n"},
			wantPort: "9000",
		},
		{
			name:    "malformed line in base file returns error",
			files:   map[string]string{"application.properties": "server.port=9000\nthis-line-has-no-equals\n"},
			wantErr: true,
		},
		{
			name: "malformed line in profile file returns error",
			files: map[string]string{
				"application.properties":      "server.port=9000\n",
				"application-prod.properties": "server.port=9100\nbroken\n",
			},
			env:     map[string]string{"APP_PROFILE": "prod"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeProperties(t, dir, name, content)
			}

			cfg, err := config.Load(dir, envLookup(tt.env))

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() error = nil, want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if got := cfg.Get("server.port"); got != tt.wantPort {
				t.Errorf("server.port = %q, want %q", got, tt.wantPort)
			}
		})
	}
}

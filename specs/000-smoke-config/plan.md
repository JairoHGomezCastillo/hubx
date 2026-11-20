# Plan 000
- Service: hubx-api only. Module: `hubx-api/internal/config`.
- Standard library only (no viper, no godotenv). Constructor `Load(dir string, getenv func(string) string) (*Config, error)`.
- No HTTP contract changes. Wiring into main.go is OUT of scope for this smoke test.
- Tests: table-driven unit tests in `hubx-api/internal/config/config_test.go`.

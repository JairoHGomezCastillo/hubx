package test

import (
	"regexp"
	"testing"

	"hubx-api/internal/shipment"
)

func TestHU02_CodeGenerator_Format(t *testing.T) {
	gen := shipment.NewCryptoCodeGenerator()

	// Generate multiple codes to verify format consistency
	pattern := regexp.MustCompile(`^HBX-[0-9A-F]{8}$`)

	for i := 0; i < 10; i++ {
		code := gen.Generate()
		if !pattern.MatchString(code) {
			t.Errorf("Generate() = %q; want format HBX-[0-9A-F]{8}", code)
		}
	}
}

func TestHU02_CodeGenerator_Uniqueness(t *testing.T) {
	gen := shipment.NewCryptoCodeGenerator()

	// Generate 100 codes and check for duplicates (statistically should not happen)
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := gen.Generate()
		if seen[code] {
			t.Errorf("Generate() produced duplicate code: %q", code)
		}
		seen[code] = true
	}
}

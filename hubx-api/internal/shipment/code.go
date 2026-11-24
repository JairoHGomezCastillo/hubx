package shipment

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

//go:generate go tool mockgen -source=code.go -destination=mocks/code_mock.go -package=mocks

// CodeGenerator generates unique shipment codes.
type CodeGenerator interface {
	Generate() string
}

// CryptoCodeGenerator generates codes using crypto/rand.
type CryptoCodeGenerator struct{}

// NewCryptoCodeGenerator creates a new code generator.
func NewCryptoCodeGenerator() *CryptoCodeGenerator {
	return &CryptoCodeGenerator{}
}

// Generate creates a shipment code: HBX- followed by 8 uppercase hex characters.
func (g *CryptoCodeGenerator) Generate() string {
	bytes := make([]byte, 4) // 4 bytes = 8 hex chars
	_, err := rand.Read(bytes)
	if err != nil {
		// crypto/rand.Read never returns an error on Unix/Windows/etc.,
		// but if it does we fall back to a zero code that will likely collide
		// and trigger the retry mechanism
		bytes = make([]byte, 4)
	}

	hexStr := hex.EncodeToString(bytes)
	return "HBX-" + strings.ToUpper(hexStr)
}

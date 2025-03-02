package auth

import (
	"crypto/rand"
	"encoding/base32"
	// "fmt"
	// "strings"

	"github.com/pquerna/otp/totp"
)

// GenerateSecret generates a new secret key for TOTP-based 2FA
func (s *AuthService) GenerateSecret() string {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
	return secret
}

// ValidateCode validates a TOTP code against the user's secret
func (s *AuthService) ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateBackupCodes generates a set of backup codes for 2FA recovery
func (s *AuthService) GenerateBackupCodes() []string {
	const (
		numCodes     = 8
		codeLength   = 8
		codeCharset  = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	)

	var codes []string
	for i := 0; i < numCodes; i++ {
		bytes := make([]byte, codeLength)
		if _, err := rand.Read(bytes); err != nil {
			continue
		}

		for j := 0; j < codeLength; j++ {
			bytes[j] = codeCharset[int(bytes[j])%len(codeCharset)]
		}
		codes = append(codes, string(bytes))
	}

	return codes
}
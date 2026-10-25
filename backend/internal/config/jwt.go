package config

import (
	"errors"
	"strings"
)

// JWTSecret must be read after LoadEnv, never during package initialization.
// Preserve the exact key bytes and never include them in an error.
func JWTSecret() ([]byte, error) {
	secret := GetEnv("JWT_SECRET")
	if len(secret) < 32 || strings.TrimSpace(secret) == "" {
		return nil, errors.New("JWT_SECRET must contain at least 32 bytes and must not be blank")
	}
	return []byte(secret), nil
}

package config_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/config"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/middleware"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
)

func TestJWTSecretValidation(t *testing.T) {
	for _, scenario := range []struct {
		name, secret string
		valid        bool
	}{
		{"empty", "", false}, {"short", strings.Repeat("x", 31), false},
		{"blank", strings.Repeat(" ", 32), false}, {"minimum", strings.Repeat("x", 32), true},
		{"unicode-bytes", strings.Repeat("ก", 11), true},
		{"exact-bytes", "  " + strings.Repeat("x", 32) + "  ", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", scenario.secret)
			key, err := config.JWTSecret()
			if scenario.valid {
				if err != nil || !bytes.Equal(key, []byte(scenario.secret)) {
					t.Fatal("valid key changed or rejected")
				}
			} else if err == nil || key != nil {
				t.Fatal("invalid key accepted")
			}
		})
	}
}

func TestJWTSecretAfterDotenv(t *testing.T) {
	// Packages have already initialized; dotenv must still configure both signing
	// and verification. Do not use a real environment file or external auth service.
	t.Chdir(t.TempDir())
	const fileSecret = "dotenv-only-regression-32-byte-secret"
	if err := os.WriteFile(".env", []byte("JWT_SECRET="+fileSecret+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JWT_SECRET", "restore-on-cleanup")
	if err := os.Unsetenv("JWT_SECRET"); err != nil {
		t.Fatal(err)
	}
	config.LoadEnv()
	key, err := config.JWTSecret()
	if err != nil || string(key) != fileSecret {
		t.Fatal("dotenv secret unavailable")
	}
	encoded, err := service.GenerateToken("dotenv-test-user")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := middleware.ParseToken(encoded); err != nil {
		t.Fatal(err)
	}
	const systemSecret = "system-env-regression-32-byte-secret"
	t.Setenv("JWT_SECRET", systemSecret)
	config.LoadEnv()
	key, err = config.JWTSecret()
	if err != nil || string(key) != systemSecret {
		t.Fatal("dotenv overrode process environment")
	}
	if _, err := middleware.ParseToken(encoded); err == nil {
		t.Fatal("verification retained the old key")
	}
	t.Setenv("JWT_SECRET", "")
	config.LoadEnv()
	if _, err := config.JWTSecret(); err == nil {
		t.Fatal("invalid system value silently fell back to dotenv")
	}
}

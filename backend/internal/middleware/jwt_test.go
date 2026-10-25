package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/middleware"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "jwt-regression-only-32-byte-secret"
const testUserID = "753badcb-8950-4ff2-a7b0-14d036052376"

func TestGeneratedTokenPolicy(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	before := time.Now().Unix()
	encoded, err := service.GenerateToken(testUserID)
	if err != nil {
		t.Fatal(err)
	}
	// Independent verification checks that signing really used the runtime secret.
	token, err := jwt.Parse(encoded, func(*jwt.Token) (any, error) { return []byte(testSecret), nil },
		jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		t.Fatal(err)
	}
	claims := token.Claims.(jwt.MapClaims)
	iat, err := claims.GetIssuedAt()
	if err != nil || iat == nil {
		t.Fatal("missing issued-at")
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		t.Fatal("missing expiration")
	}
	if iat.Unix() < before || iat.Unix() > time.Now().Unix() || exp.Sub(iat.Time) != 8*time.Hour {
		t.Fatal("session is not fixed at eight hours")
	}
	if claims["user_id"] != testUserID {
		t.Fatal("user ID changed")
	}
	if _, err := middleware.ParseToken(encoded); err != nil {
		t.Fatalf("generated token rejected: %v", err)
	}
}

func TestParseTokenPolicy(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	for _, scenario := range []string{"valid", "expired", "expiry-boundary", "legacy-no-exp", "invalid-exp", "wrong-signature", "empty-key", "HS384", "HS512", "missing-user", "numeric-user", "empty-user", "future-issued-at", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			claims := jwt.MapClaims{"user_id": testUserID, "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix()}
			method := jwt.SigningMethodHS256
			key := []byte(testSecret)
			switch scenario {
			case "expired":
				claims["exp"] = time.Now().Add(-time.Minute).Unix()
			case "expiry-boundary":
				claims["exp"] = time.Now().Unix()
			case "legacy-no-exp":
				delete(claims, "exp")
			case "invalid-exp":
				claims["exp"] = "tomorrow"
			case "wrong-signature":
				key = []byte(strings.Repeat("x", 32))
			case "empty-key":
				key = nil
			case "HS384":
				method = jwt.SigningMethodHS384
			case "HS512":
				method = jwt.SigningMethodHS512
			case "missing-user":
				delete(claims, "user_id")
			case "numeric-user":
				claims["user_id"] = 123
			case "empty-user":
				claims["user_id"] = ""
			case "future-issued-at":
				claims["iat"] = time.Now().Add(time.Hour).Unix()
			}
			encoded, err := jwt.NewWithClaims(method, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "malformed" {
				encoded = "not-a-jwt"
			}
			_, err = middleware.ParseToken(encoded)
			if scenario == "valid" && err != nil {
				t.Errorf("valid token rejected: %v", err)
			}
			if scenario != "valid" && err == nil {
				t.Error("unsafe token accepted")
			}
		})
	}
}

func TestJWTInvalidConfiguration(t *testing.T) {
	for _, secret := range []string{"", "short", strings.Repeat(" ", 32), strings.Repeat("x", 31)} {
		t.Run("length-"+strconv.Itoa(len(secret)), func(t *testing.T) {
			t.Setenv("JWT_SECRET", secret)
			encoded, err := service.GenerateToken(testUserID)
			if err == nil || encoded != "" {
				t.Error("issued token with invalid secret")
			}
			if _, err := middleware.ParseToken(encoded); err == nil {
				t.Error("accepted token with invalid secret")
			}
		})
	}
}

func TestAuthMiddlewareRejectsInvalidBeforeDB(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	// No DB is configured: rejected credentials must never reach a user lookup.
	for _, header := range []string{"", "Bearer invalid", "Basic invalid"} {
		app := fiber.New()
		app.Use(middleware.AuthMiddleware)
		app.Get("/", func(c fiber.Ctx) error { t.Error("unauthorized handler reached"); return c.SendStatus(200) })
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", header)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 401 {
			t.Errorf("status = %d, want 401", res.StatusCode)
		}
	}
}

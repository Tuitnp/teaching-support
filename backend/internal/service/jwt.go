package service

import (
	"time"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken creates a JWT for a given user ID.
func GenerateToken(userID string) (string, error) {
	secret, err := config.JWTSecret()
	if err != nil {
		return "", err
	}
	now := time.Now()

	claims := jwt.MapClaims{
		"user_id": userID,
		"iat":     now.Unix(),
		"exp":     now.Add(8 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	return token.SignedString(secret)
}

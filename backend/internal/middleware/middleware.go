package middleware

import (
	"errors"
	"strings"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/config"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	apperrors "github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	claims, err := ParseToken(tokenStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", claims["user_id"]).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid user",
		})
	}

	c.Locals("user_role", string(user.Role))
	c.Locals("user_id", claims["user_id"])
	return c.Next()
}

// ParseToken validates and parses a JWT token string.
func ParseToken(tokenStr string) (jwt.MapClaims, error) {
	secret, err := config.JWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt())

	if err != nil {
		return nil, err
	}

	// Extract claims if valid
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok || strings.TrimSpace(userID) == "" {
			return nil, errors.New("invalid user identity")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func CourseMiddleware(c fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return fiber.ErrUnauthorized
	}

	courseID := c.Params("course_id")
	if courseID == "" {
		return fiber.ErrBadRequest
	}

	role, err := repository.ActiveCourseRole(userID.(string), courseID)
	if err != nil {
		if err == apperrors.ErrMembershipConflict {
			return c.Status(409).JSON(fiber.Map{"error": err.Error()})
		}
		return fiber.ErrForbidden
	}

	c.Locals("course_role", role)

	return c.Next()
}

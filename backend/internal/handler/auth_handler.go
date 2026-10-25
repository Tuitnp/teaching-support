package handler

import (
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/gofiber/fiber/v3"
)

func RegisterAuthRoutes(app fiber.Router) {
	app.Post("/login", login)
}

func login(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("login_attempt")
	
	var form models.LoginForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("login_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	user, err := service.AuthenticateUser(&form)
	if err != nil {
		log.Error("login_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	token, err := service.GenerateToken(user.ID)
	if err != nil {
		log.Error("login_failed", "error", err)
		return SendError(c, err)
	}

	response := models.LoginResponse{
		Token: 		token,
		Username: 	user.Username,
		EnName:		user.EnName,
		ThName: 	user.ThName,
		Role: 		string(user.Role),	
	}
	
	log.Info("login_success",
		"username", user.Username,
	)
	
	return c.JSON(response)
}

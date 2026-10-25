package handler

import (
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func RegisterCommentRoutes(app fiber.Router) {
	app.Post("", createOrUpdateComment)
	app.Post("/:comment_id", toggleComment)
}

func createOrUpdateComment(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("comment_create_or_update_attempt")

	courseID := c.Params("course_id")
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)
	submissionID := c.Params("submission_id")

	if !models.HasPermission(role, "assignment:comment") {
		log.Error("comment_create_or_update_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.CommentForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("comment_create_or_update_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	data, err := service.CreateOrUpdateComment(courseID, submissionID, userID, role, &form)
	if err != nil {
		log.Error("comment_create_or_update_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("comment_create_or_update_success",
		"course_id", courseID,
		"submission_id", submissionID,
	)

	return c.JSON(data)
}

func toggleComment(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("comment_toggle_attempt")

	courseID := c.Params("course_id")
	submissionID := c.Params("submission_id")
	commentID := c.Params("comment_id")
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)

	if !models.HasPermission(role, "assignment:comment") {
		log.Error("comment_toggle_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	data, err := service.ToggleComment(courseID, submissionID, commentID, userID, role)
	if err != nil {
		log.Error("comment_toggle_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("comment_toggle_success",
		"course_id", courseID,
		"submission_id", submissionID,
		"comment_id", commentID,
	)

	return c.JSON(data)
}

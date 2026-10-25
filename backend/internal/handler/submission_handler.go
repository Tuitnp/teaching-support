package handler

import (
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func RegisterSubmissionRoutes(app fiber.Router) {
	app.Post("", createSubmission)
	app.Put("/:submission_id", updateSubmission)
	app.Put("/:submission_id/grade", updateGrade)
	app.Put("/:submission_id/grade-with-comment", updateCommentAndGrade)
	app.Get("/:submission_id/read", readSubmission)
}

func createSubmission(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("submission_create_attempt")
	
	courseID := c.Params("course_id")
	assignmentID := c.Params("assignment_id")
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)

	if !models.HasPermission(role, "submission:create") {
		log.Error("submission_create_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	file, _ := c.FormFile("file")

	submission, err := service.CreateSubmission(courseID, assignmentID, userID, file)
	if err != nil {
		log.Error("submission_create_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("submission_create_success",
		"course_id", courseID,
		"assignment_id", assignmentID,
	)
	
	return c.JSON(submission)
}

func updateSubmission(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("submission_update_attempt")
	
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)
	submissionID := c.Params("submission_id")

	if !models.HasPermission(role, "submission:update") {
		log.Error("submission_update_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	file, _ := c.FormFile("file")

	result, err := service.UpdateSubmission(submissionID, userID, file)
	if err != nil {
		log.Error("submission_update_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("submission_update_success",
		"submission_id", submissionID,
	)
	
	return c.JSON(result)
}

func updateGrade(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("grade_update_attempt")
	
	courseID := c.Params("course_id")
	submissionID := c.Params("submission_id")
	role := c.Locals("course_role").(string)

	if !models.HasPermission(role, "assignment:grade") {
		log.Error("grade_update_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.GradeForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("grade_update_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	result, err := service.UpdateGrade(courseID, submissionID, role, &form)
	if err != nil {
		log.Error("grade_update_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("grade_update_success",
		"course_id", courseID,
		"submission_id", submissionID,
	)
	
	return c.JSON(result)
}

func updateCommentAndGrade(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("comment_and_grade_update_attempt")
	
	courseID := c.Params("course_id")
	submissionID := c.Params("submission_id")
	role := c.Locals("course_role").(string)
	userID := c.Locals("user_id").(string)

	if !models.HasPermission(role, "assignment:grade") {
		log.Error("comment_and_grade_update_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}
	
	if !models.HasPermission(role, "assignment:comment") {
		log.Error("comment_and_grade_update_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.GradeAndCommentForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("comment_and_grade_update_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	result, err := service.UpdateGradeAndComment(courseID, submissionID, userID, role, &form)
	if err != nil {
		log.Error("comment_and_grade_update_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("comment_and_grade_update_success",
		"course_id", courseID,
		"submission_id", submissionID,
	)
	
	return c.JSON(result)
}

func readSubmission(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)
	submissionID := c.Params("submission_id")
	
	content, err := service.ReadSubmission(submissionID, userID, role)
	
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(content)
}

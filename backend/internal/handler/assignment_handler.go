package handler

import (
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func RegisterAssignmentRoutes(app fiber.Router) {
	app.Post("", createAssignment)
	app.Get("", getAssignments)
	app.Get("/export", getAssignmentsExport)
	app.Get("/:assignment_id", getAssignment)
	app.Put("/:assignment_id", updateAssignment)
	app.Delete("/:assignment_id", deleteAssignment)
	app.Get("/:assignment_id/summary", getAssignmentSummary)
	app.Post("/:assignment_id/override", createAssignmentOverride)
	app.Get("/:assignment_id/auto-grading", autoGradingAssignment)
	app.Get("/:assignment_id/submissions/download", downloadSubmissions)
	app.Get("/:assignment_id/export", getAssignmentExport)
}

func createAssignment(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("assignment_create_attempt")

	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")

	if !models.HasPermission(role, "assignment:create") {
		log.Error("assignment_create_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.AssignmentForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("assignment_create_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	files, err := uploadedFiles(c, "files")
	if err != nil {
		return SendError(c, err)
	}

	assignment, err := service.CreateAssignment(courseID, userID, &form, files)
	if err != nil {
		log.Error("assignment_create_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("assignment_create_success",
		"course_id", courseID,
	)

	return c.JSON(assignment)
}

func getAssignments(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")

	assignments, err := service.GetAssignments(userID, courseID, role)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(fiber.Map{
		"role":        role,
		"assignments": assignments,
	})
}

func getAssignment(c fiber.Ctx) error {
	courseID := c.Params("course_id")
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)
	assignmentID := c.Params("assignment_id")

	data, err := service.GetAssignment(userID, courseID, role, assignmentID)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(fiber.Map{
		"role":       	role,
		"assignment": 	data.Assignment,
		"submissions":	data.Submissions,
	})
}

func updateAssignment(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("assignment_update_attempt")

	courseID := c.Params("course_id")
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)
	assignmentID := c.Params("assignment_id")

	if !models.HasPermission(role, "assignment:update") {
		log.Error("assignment_update_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	files, err := uploadedFiles(c, "files")
	if err != nil {
		return SendError(c, err)
	}

	var form models.AssignmentForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("assignment_update_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	data, err := service.UpdateAssignment(courseID, assignmentID, userID, &form, files)
	if err != nil {
		log.Error("assignment_update_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("assignment_update_success",
		"course_id", courseID,
		"assignment_id", assignmentID,
	)

	return c.JSON(data)
}

func deleteAssignment(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("assignment_delete_attempt")
	
	courseID := c.Params("course_id")
	role := c.Locals("course_role").(string)
	assignmentID := c.Params("assignment_id")

	if !models.HasPermission(role, "assignment:delete") {
		log.Error("assignment_delete_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	err := service.DeleteAssignment(courseID, assignmentID)
	if err != nil {
		log.Error("assignment_delete_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("assignment_delete_success",
		"course_id", courseID,
		"assignment_id", assignmentID,
	)
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "assignment deleted",
	})
}

func createAssignmentOverride(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("assignment_override_create_attempt")
	
	courseID := c.Params("course_id")
	assignmentID := c.Params("assignment_id")
	role := c.Locals("course_role").(string)

	if !models.HasPermission(role, "assignment:update") {
		log.Error("assignment_override_create_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.AssignmentOverrideForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("assignment_override_create_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	override, err := service.CreateAssignmentOverride(courseID, assignmentID, form)
	if err != nil {
		log.Error("assignment_override_create_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("assignment_override_create_success",
		"course_id", courseID,
		"assignment_id", assignmentID,
	)
	
	return c.JSON(override)
}

func getAssignmentSummary(c fiber.Ctx) error {	
	courseID := c.Params("course_id")
	role := c.Locals("course_role").(string)
	assignmentID := c.Params("assignment_id")

	if !models.HasPermission(role, "assignment:view_all") {
		return SendError(c, errors.ErrForbidden)
	}

	summary, err := service.GetAssignmentSummary(courseID, assignmentID)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(summary)
}

func autoGradingAssignment(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("assignment_auto_grading_attempt")
	
	userID := c.Locals("user_id").(string)
	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")
	assignmentID := c.Params("assignment_id")

	if !models.HasPermission(role, "ai") {
		log.Error("assignment_auto_grading_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	err := service.AutoGradingAssignment(userID, courseID, assignmentID)
	if err != nil {
		log.Error("assignment_auto_grading_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("assignment_auto_grading_success",
		"course_id", courseID,
		"assignment_id", assignmentID,
	)
	
	return c.JSON(fiber.Map{
		"message": "auto grading started",
	})
}

func downloadSubmissions(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("assignment_download_submissions")
	
	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")
	assignmentID := c.Params("assignment_id")

	if !models.HasPermission(role, "file:download_all") {
		log.Error("assignment_download_submissions_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	zipData, err := service.DownloadSubmissions(courseID, assignmentID)
	if err != nil {
		log.Error("assignment_download_submissions_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", "attachment; filename=submissions.zip")
	
	log.Info("assignment_download_submissions_success")
	
	return c.Send(zipData)
}

func getAssignmentsExport(c fiber.Ctx) error {
	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")
	
	if !models.HasPermission(role, "course:export") {
		return SendError(c, errors.ErrForbidden)
	}

	csv, err := service.GetAssignmentsExport(courseID)
	if err != nil {
		return SendError(c, err)
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=assignments_export.csv")

	return c.Send(csv)
}

func getAssignmentExport(c fiber.Ctx) error {
	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")
	assignmentID := c.Params("assignment_id")
	
	if !models.HasPermission(role, "assignment:export") {
		return SendError(c, errors.ErrForbidden)
	}

	csv, err := service.GetAssignmentExport(courseID, assignmentID)
	if err != nil {
		return SendError(c, err)
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=assignment_export.csv")

	return c.Send(csv)
}

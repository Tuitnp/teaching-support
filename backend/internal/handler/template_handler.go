package handler

import (
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func RegisterTemplateRoutes(app fiber.Router) {
	app.Post("", createAssignmentTemplate)
	app.Get("", getAssignmentTemplates)
	app.Get("/short", getShortAssignmentTemplates)
	app.Get("/:template_id", getAssignmentTemplate)
	app.Put("/:template_id", updateAssignmentTemplate)
	app.Delete("/:template_id", deleteAssignmentTemplate)
}

func createAssignmentTemplate(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("assignment_template_create_attempt")

	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)

	if !models.HasPermission(role, "template:create") {
		log.Error("assignment_template_create_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.AssignmentTemplateForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("assignment_template_create_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	files, err := uploadedFiles(c, "files")
	if err != nil {
		return SendError(c, err)
	}

	template, err := service.CreateAssignmentTemplate(userID, &form, files)
	if err != nil {
		log.Error("assignment_template_create_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("assignment_template_create_success")

	return c.JSON(template)
}

func getAssignmentTemplates(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)

	if !models.HasPermission(role, "template:view") {
		return SendError(c, errors.ErrForbidden)
	}

	templates, err := service.GetAssignmentTemplates(userID)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(templates)
}

func getShortAssignmentTemplates(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)

	if !models.HasPermission(role, "template:view") {
		return SendError(c, errors.ErrForbidden)
	}

	templates, err := service.GetShortAssignmentTemplates(userID)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(templates)
}

func getAssignmentTemplate(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)
	templateID := c.Params("template_id")

	if !models.HasPermission(role, "template:view") {
		return SendError(c, errors.ErrForbidden)
	}

	template, err := service.GetAssignmentTemplate(userID, templateID)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(template)
}

func updateAssignmentTemplate(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("assignment_template_update_attempt")

	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)
	templateID := c.Params("template_id")

	if !models.HasPermission(role, "template:update") {
		log.Error("assignment_template_update_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	files, err := uploadedFiles(c, "files")
	if err != nil {
		return SendError(c, err)
	}

	var form models.AssignmentTemplateForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("assignment_template_update_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	template, err := service.UpdateAssignmentTemplate(templateID, userID, &form, files)
	if err != nil {
		log.Error("assignment_template_update_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("assignment_template_update_success",
		"template_id", templateID,
	)

	return c.JSON(template)
}

func deleteAssignmentTemplate(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("assignment_template_delete_attempt")
	
	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)
	templateID := c.Params("template_id")

	if !models.HasPermission(role, "template:delete") {
		log.Error("assignment_template_delete_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	err := service.DeleteAssignmentTemplate(userID, templateID)
	if err != nil {
		log.Error("assignment_template_delete_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("assignment_template_delete_success",
		"template_id", templateID,
	)
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "template deleted",
	})
}

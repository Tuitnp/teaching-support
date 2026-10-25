package handler

import (
	"fmt"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func RegisterAttachmentRoutes(app fiber.Router) {
	app.Post("", createAttachments)
	app.Get("", getAttachments)
	app.Get("/:attachment_id/download", downloadAttachment)
	app.Delete("/:attachment_id", deleteAttachment)
	app.Get("/:attachment_id", getAttachmentDetail)
}

func createAttachments(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("attachments_create_attempt")

	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)

	if !models.HasPermission(role, "template:create") {
		log.Error("attachments_create_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	files, err := uploadedFiles(c, "files")
	if err != nil {
		return SendError(c, err)
	}

	err = service.CreateAttachments(userID, files)
	if err != nil {
		log.Error("attachments_create_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("attachments_create_success")

	return c.JSON(fiber.Map{
		"message": "attachments created",
	})
}

func getAttachments(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)

	if !models.HasPermission(role, "template:view") {
		return SendError(c, errors.ErrForbidden)
	}

	attachments, err := service.GetAttachments(userID)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(attachments)
}

func downloadAttachment(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	userID := c.Locals("user_id").(string)
	attachmentID := c.Params("attachment_id")

	log.Info("attachment_download_attempt",
		"user_id", userID,
		"attachment_id", attachmentID,
	)

	attachment, filename, contentType, err := service.DownloadAttachment(userID, attachmentID)
	if err != nil {
		log.Error("attachment_download_failed",
			"error", err,
			"user_id", userID,
			"attachment_id", attachmentID,
		)
		return SendError(c, err)
	}

	log.Info("attachment_download_success",
		"user_id", userID,
		"attachment_id", attachmentID,
		"filename", filename,
		"content_type", contentType,
		"file_size", len(attachment),
	)

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	return c.Send(attachment)
}

func deleteAttachment(c fiber.Ctx) error {
	log := logger.WithRequest(c)
	
	log.Info("attachment_delete_attempt")
	
	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)
	attachmentID := c.Params("attachment_id")

	if !models.HasPermission(role, "template:delete") {
		log.Error("attachment_delete_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	err := service.DeleteAttachment(userID, attachmentID)
	if err != nil {
		log.Error("attachment_delete_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("attachment_delete_success")
	
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "attachment deleted",
	})
}

func getAttachmentDetail(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)
	attachmentID := c.Params("attachment_id")

	if !models.HasPermission(role, "template:view") {
		return SendError(c, errors.ErrForbidden)
	}

	attachmentDetail, err := service.GetAttachmentDetail(userID, attachmentID)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(attachmentDetail)
}

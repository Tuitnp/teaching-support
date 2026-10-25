package repository

import (
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func CreateAttachment(attachment *models.Attachment) error {
	return database.DB.Create(attachment).Error
}

func ReplaceAssignmentAttachments(id string, attachments []models.Attachment) error {
	return database.DB.
		Model(&models.Assignment{ID: id}).
		Association("Attachments").
		Replace(attachments)
}

func ReplaceAssignmentTemplateAttachments(id string, attachments []models.Attachment) error {
	return database.DB.
		Model(&models.AssignmentTemplate{ID: id}).
		Association("Attachments").
		Replace(attachments)
}

func GetAttachment(userID, attachmentID string) (*models.Attachment, error) {
	if err := models.ValidateUUIDs(attachmentID); err != nil {
		return nil, err
	}

	var attachment models.Attachment

	err := database.DB.
		Where("user_id = ? AND id = ?", userID, attachmentID).
		First(&attachment).
		Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrAttachmentNotFound
	}

	return &attachment, err
}

func GetAttachmentByID(attachmentID string) (*models.Attachment, error) {
	if err := models.ValidateUUIDs(attachmentID); err != nil {
		return nil, err
	}

	var attachment models.Attachment

	err := database.DB.
		Where("id = ?", attachmentID).
		First(&attachment).
		Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrAttachmentNotFound
	}

	return &attachment, err
}

func GetAttachments(userID string) ([]models.Attachment, error) {
	var attachments []models.Attachment

	err := database.DB.
		Where("user_id = ?", userID).
		Find(&attachments).
		Error

	return attachments, err
}

func GetAttachmentsByIDs(userID string, attachmentIDs []string) ([]models.Attachment, error) {
	var existingAttachments []models.Attachment

	err := database.DB.
		Where("id IN ? AND user_id = ?", attachmentIDs, userID).
		Find(&existingAttachments).
		Error

	return existingAttachments, err
}

func DeleteAttachment(attachment *models.Attachment) error {
	return database.DB.Delete(attachment).Error
}

func GetAssignmentsByAttachmentID(attachmentID string) ([]map[string]any, error) {
	var results []map[string]any
	
	err := database.DB.
		Table("assignments").
		Select("assignments.title, assignments.course_id").
		Joins("INNER JOIN assignment_attachments ON assignment_attachments.assignment_id = assignments.id").
		Where("assignment_attachments.attachment_id = ?", attachmentID).
		Find(&results).
		Error
	
	return results, err
}

func GetTemplatesByAttachmentID(attachmentID string) ([]map[string]any, error) {
	var results []map[string]any
	
	err := database.DB.
		Table("assignment_templates").
		Select("assignment_templates.title").
		Joins("INNER JOIN assignment_template_attachments ON assignment_template_attachments.assignment_template_id = assignment_templates.id").
		Where("assignment_template_attachments.attachment_id = ?", attachmentID).
		Find(&results).
		Error
	
	return results, err
}

// GetSubmissionByAttachmentID checks if an attachment is from a submission and returns the submission with its assignment
func GetSubmissionByAttachmentID(attachmentID string) (*models.Submission, error) {
	var submission models.Submission

	err := database.DB.
		Preload("Assignment").
		Where("attachment_id = ?", attachmentID).
		First(&submission).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &submission, err
}

// GetCourseIDsByAttachmentID returns all course IDs from assignments that use this attachment
func GetCourseIDsByAttachmentID(attachmentID string) ([]string, error) {
	var courseIDs []string

	err := database.DB.
		Table("assignments").
		Select("DISTINCT assignments.course_id").
		Joins("INNER JOIN assignment_attachments ON assignment_attachments.assignment_id = assignments.id").
		Where("assignment_attachments.attachment_id = ?", attachmentID).
		Pluck("course_id", &courseIDs).
		Error

	return courseIDs, err
}

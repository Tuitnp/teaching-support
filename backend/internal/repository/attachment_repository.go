package repository

import (
	stderrors "errors"
	"time"

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

type AttachmentAccessReference struct {
	Role      string
	Visible   bool
	StartDate time.Time
	Handout   bool
}

// Keep each reference's assignment state and active course role together. A
// shared file may be authorized through any reference, never through an unrelated
// course's membership or another assignment's release state.
func GetAttachmentAccessReferences(userID, attachmentID string) ([]AttachmentAccessReference, error) {
	var references []AttachmentAccessReference
	handout := `EXISTS (SELECT 1 FROM assignment_attachments aa
		WHERE aa.assignment_id = a.id AND aa.attachment_id = ?)`
	submission := `EXISTS (SELECT 1 FROM submissions s
		WHERE s.assignment_id = a.id AND s.attachment_id = ?)`
	err := database.DB.
		Table("assignments a").
		Select("cm.role, a.visible, a.start_date, "+handout+" AS handout", attachmentID).
		Joins("JOIN course_members cm ON cm.course_id = a.course_id").
		Where("cm.user_id = ? AND cm.status = ?", userID, models.StatusActive).
		Where("("+handout+" OR "+submission+")", attachmentID, attachmentID).
		Scan(&references).Error
	return references, err
}

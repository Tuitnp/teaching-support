package repository

import (
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func CreateAssignmentTemplate(template *models.AssignmentTemplate) (string, error) {
	err := database.DB.Create(template).Error
	if err != nil {
		return "", err
	}
	
	return template.ID, nil
}

func GetAssignmentTemplates(userID string) ([]models.AssignmentTemplate, error) {
	var templates []models.AssignmentTemplate

	err := database.DB.
		Preload("Attachments").
		Preload("Tags").
		Where("created_by = ?", userID).
		Find(&templates).
		Error

	return templates, err
}

func GetAssignmentTemplate(userID, templateID string) (*models.AssignmentTemplate, error) {
	if err := models.ValidateUUIDs(templateID); err != nil {
		return nil, err
	}

	var template models.AssignmentTemplate

	err := database.DB.
		Preload("Attachments").
		Preload("Tags").
		Where("created_by = ?", userID).
		First(&template, "id = ?", templateID).
		Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrTemplateNotFound
	}

	return &template, err
}

func UpdateAssignmentTemplate(id string, updates map[string]any) error {
	return database.DB.
		Model(&models.AssignmentTemplate{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

func DeleteAssignmentTemplate(assignment *models.AssignmentTemplate) error {
	return database.DB.Delete(assignment).Error
}

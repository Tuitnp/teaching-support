package repository

import (
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func CreateAICredential(credential *models.AICredential) error {
	return database.DB.Create(credential).Error
}

func GetAICredential(userID, id string) (*models.AICredential, error) {
	if err := models.ValidateUUIDs(id); err != nil {
		return nil, err
	}

	var credential *models.AICredential

	err := database.DB.
		Where("user_id = ? AND id = ?", userID, id).
		First(&credential).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrAIConfigNotFound
	}

	return credential, err
}

func GetAICredentials(userID string) ([]models.AICredential, error) {
	var credentials []models.AICredential

	err := database.DB.
		Where("user_id = ?", userID).
		Find(&credentials).Error

	return credentials, err
}

func UpdateAICredential(id string, updates map[string]any) error {
	return database.DB.
		Model(&models.AICredential{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

func DeleteAICredential(id string) error {
	return database.DB.
		Delete(&models.AICredential{}, "id = ?", id).
		Error
}

func CreateAIConfig(config *models.AIConfig) error {
	return database.DB.Create(config).Error
}

func GetAIConfig(userID, aiConfigID string) (*models.AIConfig, error) {
	if err := models.ValidateUUIDs(aiConfigID); err != nil {
		return nil, err
	}

	var config *models.AIConfig

	err := database.DB.
		Preload("AICredential").
		Where("user_id = ? AND id = ?", userID, aiConfigID).
		First(&config).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrAIConfigNotFound
	}

	return config, err
}

func GetAIConfigs(userID, aiCredentialID string) ([]models.AIConfig, error) {
	if err := models.ValidateUUIDs(aiCredentialID); err != nil {
		return nil, err
	}

	var configs []models.AIConfig

	err := database.DB.
		Preload("AICredential").
		Where("user_id = ? AND ai_credential_id = ?", userID, aiCredentialID).
		Find(&configs).Error

	return configs, err
}

func GetAIConfigByID(userID, aiConfigID string) (*models.AIConfig, error) {
	if err := models.ValidateUUIDs(aiConfigID); err != nil {
		return nil, err
	}

	var config models.AIConfig

	err := database.DB.
		Preload("AICredential").
		Where("user_id = ? AND id = ?", userID, aiConfigID).
		First(&config).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrAIConfigNotFound
	}

	return &config, err
}

func UpdateAIConfig(userID, aiConfigID string, updates map[string]any) error {
	result := database.DB.
		Model(&models.AIConfig{}).
		Where("user_id = ? AND id = ?", userID, aiConfigID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.ErrAIConfigNotFound
	}
	return nil
}

func DeleteAIConfig(id string) error {
	return database.DB.
		Delete(&models.AIConfig{}, "id = ?", id).
		Error
}

func GetGradingJob(assignmentID, teacherID string) (*models.GradingJob, error) {
	var gradingJob models.GradingJob

	err := database.DB.
		Where("assignment_id = ? AND teacher_id = ?", assignmentID, teacherID).
		First(&gradingJob).Error

	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &gradingJob, nil
}

func GetGradingJobsByTeacher(teacherID string) ([]models.GradingJob, error) {
	var gradingJobs []models.GradingJob

	err := database.DB.
		Preload("Assignment").
		Where("teacher_id = ?", teacherID).
		Order("created_at DESC").
		Find(&gradingJobs).Error

	return gradingJobs, err
}

func GetGradingJobByID(gradingJobID, teacherID string) (*models.GradingJob, error) {
	var gradingJob models.GradingJob

	err := database.DB.
		Preload("Assignment").
		Preload("Teacher").
		Where("id = ? AND teacher_id = ?", gradingJobID, teacherID).
		First(&gradingJob).Error

	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &gradingJob, nil
}

func DeleteGradingJob(gradingJobID, teacherID string) error {
	return database.DB.
		Where("id = ? AND teacher_id = ?", gradingJobID, teacherID).
		Delete(&models.GradingJob{}).Error
}

func CreatePromptTemplate(prompt *models.PromptTemplate) error {
	return database.DB.Create(prompt).Error
}

func GetPromptTemplatesByUser(userID string) ([]models.PromptTemplate, error) {
	var prompts []models.PromptTemplate

	err := database.DB.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&prompts).Error

	return prompts, err
}

func GetPromptTemplateByID(userID, promptID string) (*models.PromptTemplate, error) {
	var prompt models.PromptTemplate

	err := database.DB.
		Where("user_id = ? AND id = ?", userID, promptID).
		First(&prompt).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrPromptNotFound
	}

	return &prompt, err
}

func DeletePromptTemplateByID(userID, promptID string) error {
	return database.DB.
		Where("user_id = ? AND id = ?", userID, promptID).
		Delete(&models.PromptTemplate{}).Error
}

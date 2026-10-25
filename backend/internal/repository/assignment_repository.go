package repository

import (
	"context"
	stderrors "errors"
	"time"

	"gorm.io/gorm"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func CreateAssignment(assignment *models.Assignment) error {
	return database.DB.Create(assignment).Error
}

func GetAssignment(courseID, assignmentID string) (*models.Assignment, error) {
	if err := models.ValidateUUIDs(assignmentID); err != nil {
		return nil, err
	}

	var assignment models.Assignment

	err := database.DB.
		Where("course_id = ? AND id = ?", courseID, assignmentID).
		First(&assignment).
		Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrAssignmentNotFound
	}

	return &assignment, err
}

func GetAssignmentWithRelations(courseID, assignmentID string) (*models.Assignment, error) {
	if err := models.ValidateUUIDs(assignmentID); err != nil {
		return nil, err
	}

	var assignment models.Assignment

	err := database.DB.
		Preload("Tags").
		Preload("Attachments").
		Where("course_id = ?", courseID).
		First(&assignment, "id = ?", assignmentID).
		Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrAssignmentNotFound
	}

	return &assignment, err
}

func GetAssignmentWithAIConfig(ctx context.Context, assignmentID string) (*models.Assignment, error) {
	if err := models.ValidateUUIDs(assignmentID); err != nil {
		return nil, err
	}

	var assignment models.Assignment

	err := database.DB.WithContext(ctx).
		Preload("AIConfig").
		Preload("AIConfig.AICredential").
		First(&assignment, "id = ?", assignmentID).
		Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrAssignmentNotFound
	}

	return &assignment, err
}

// func GetAssignmentMaxPointPromptAndDescription(assignmentID string) (*int16, *string, string, error) {
// 	var assignment models.Assignment

// 	err := database.DB.
// 		Where("id = ?", assignmentID).
// 		First(&assignment).
// 		Error

// 	if stderrors.Is(err, gorm.ErrRecordNotFound) {
// 		return nil, nil, "", errors.ErrAssignmentNotFound
// 	}

// 	return &assignment.Point, assignment.Prompt, assignment.Description, err
// }

func GetVisibleAssignment(courseID, assignmentID string) (*models.Assignment, error) {
	if err := models.ValidateUUIDs(assignmentID); err != nil {
		return nil, err
	}

	var assignment models.Assignment

	err := database.DB.
		Preload("Tags").
		Preload("Attachments").
		Where(`
			course_id = ?
			AND visible = ?
			AND start_date <= ?
		`, courseID, true, time.Now()).
		First(&assignment, "id = ?", assignmentID).
		Error

	return &assignment, err
}

func GetAssignmentsByCourse(courseID string) ([]models.Assignment, error) {
	var assignments []models.Assignment

	err := database.DB.
		Preload("Tags").
		Preload("Attachments").
		Where("course_id = ?", courseID).
		Find(&assignments).
		Error

	return assignments, err
}

func GetVisibleAssignments(courseID string) ([]models.Assignment, error) {
	var assignments []models.Assignment

	err := database.DB.
		Preload("Tags").
		Where(`
			course_id = ?
			AND visible = ?
			AND start_date <= ?
		`, courseID, true, time.Now()).
		Find(&assignments).
		Error

	return assignments, err
}

func UpdateAssignment(id string, updates map[string]any) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		return updateAssignment(tx, id, updates)
	})
}

func updateAssignment(tx *gorm.DB, id string, updates map[string]any) error {
	assignment, err := lockAssignment(tx, id)
	if err != nil {
		return err
	}
	previousMaximum := assignment.Point
	if value, ok := updates["point"].(int16); ok {
		assignment.Point = value
	}
	if value, ok := updates["start_date"].(time.Time); ok {
		assignment.StartDate = value
	}
	if value, ok := updates["due_date"].(time.Time); ok {
		assignment.DueDate = value
	}
	if value, ok := updates["close_date"].(time.Time); ok {
		assignment.CloseDate = value
	}
	if err := models.ValidateAssignmentValues(assignment.Point, assignment.StartDate, assignment.DueDate, assignment.CloseDate); err != nil {
		return err
	}
	if assignment.Point < previousMaximum {
		if err := checkMaximumPoints(tx, id, assignment.Point); err != nil {
			return err
		}
	}
	return tx.Model(&models.Assignment{}).Where("id = ?", id).Updates(updates).Error
}

func DeleteAssignment(assignment *models.Assignment) error {
	return database.DB.Delete(assignment).Error
}

func DeleteAssignmentDependencies(assignmentID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM comments WHERE submission_id IN (SELECT id FROM submissions WHERE assignment_id = ?)`, assignmentID).Error; err != nil {
			return err
		}

		if err := tx.Where("assignment_id = ?", assignmentID).Delete(&models.Submission{}).Error; err != nil {
			return err
		}

		if err := tx.Exec(`DELETE FROM assignment_attachments WHERE assignment_id = ?`, assignmentID).Error; err != nil {
			return err
		}

		if err := tx.Exec(`DELETE FROM assignment_tags WHERE assignment_id = ?`, assignmentID).Error; err != nil {
			return err
		}

		if err := tx.Where("assignment_id = ?", assignmentID).Delete(&models.AssignmentOverride{}).Error; err != nil {
			return err
		}

		if err := tx.Where("assignment_id = ?", assignmentID).Delete(&models.GradingJob{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func CreateAssignmentOverride(
	override *models.AssignmentOverride,
) (*models.AssignmentOverride, error) {

	if err := database.DB.Create(override).Error; err != nil {
		return nil, err
	}

	return override, nil
}

func GetOverridesByStudent(userID string) ([]models.AssignmentOverride, error) {
	var overrides []models.AssignmentOverride

	err := database.DB.
		Where("student_id = ?", userID).
		Find(&overrides).
		Error

	return overrides, err
}

func GetAssignmentOverride(assignmentID, userID string) (*models.AssignmentOverride, error) {
	if err := models.ValidateUUIDs(assignmentID, userID); err != nil {
		return nil, err
	}

	var o models.AssignmentOverride

	err := database.DB.
		Where("assignment_id = ? AND student_id = ?", assignmentID, userID).
		Take(&o).
		Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &o, err
}

func DeleteAssignmentOverride(override *models.AssignmentOverride) error {
	return database.DB.Delete(override).Error
}

func GetOverridesByAssignment(assignmentID string) ([]models.AssignmentOverride, error) {
	var overrides []models.AssignmentOverride
	err := database.DB.Where("assignment_id = ?", assignmentID).Find(&overrides).Error
	return overrides, err
}

package repository

import (
	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// All grade writers and assignment edits lock the assignment before touching a
// submission. A concurrent maximum reduction cannot invalidate a new grade.
func lockAssignment(tx *gorm.DB, assignmentID string) (*models.Assignment, error) {
	var assignment models.Assignment
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&assignment, "id = ?", assignmentID).Error; err != nil {
		return nil, err
	}
	return &assignment, nil
}

func checkMaximumPoints(db *gorm.DB, assignmentID string, point int16) error {
	var exceeds bool
	if err := db.Raw("SELECT EXISTS (SELECT 1 FROM submissions WHERE assignment_id = ? AND point > ?)", assignmentID, point).Scan(&exceeds).Error; err != nil {
		return err
	}
	if exceeds {
		return errors.ErrMaximumBelowGrade
	}
	return nil
}

func CheckMaximumPoints(assignmentID string, point int16) error {
	return checkMaximumPoints(database.DB, assignmentID, point)
}

func UpdateManualGrade(assignmentID, submissionID string, point int16, role string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		assignment, err := lockAssignment(tx, assignmentID)
		if err != nil {
			return err
		}
		if err := models.ValidateGrade(&point, assignment.Point); err != nil {
			return err
		}
		result := tx.Model(&models.Submission{}).Where("id = ? AND assignment_id = ?", submissionID, assignmentID).Updates(map[string]any{"point": point, "graded_by": role})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.ErrSubmissionNotFound
		}
		return nil
	})
}

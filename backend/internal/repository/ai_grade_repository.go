package repository

import (
	stderrors "errors"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"gorm.io/gorm"
)

// SaveAIGradeAndComment checks eligibility at write time. The UPDATE holds the
// submission row lock until feedback commits, serializing with manual grades.
// Normal runs only fill genuinely ungraded rows; forced runs may replace AI work.
func SaveAIGradeAndComment(assignmentID, submissionID, teacherID string, point int16, comment string, force bool) (bool, error) {
	applied := false
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		assignment, err := lockAssignment(tx, assignmentID)
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		eligible := "(graded_by IS NULL AND point IS NULL)"
		if force {
			eligible += " OR graded_by = 'ai'"
		}
		// Preserve R4's no-op for missing, foreign or protected rows, including
		// when a stale provider score exceeds a newly reduced maximum.
		var count int64
		if err := tx.Model(&models.Submission{}).Where("id = ? AND assignment_id = ?", submissionID, assignmentID).Where("(" + eligible + ")").Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		if err := models.ValidateGrade(&point, assignment.Point); err != nil {
			return err
		}

		result := tx.Model(&models.Submission{}).
			Where("id = ? AND assignment_id = ?", submissionID, assignmentID).
			Where("(" + eligible + ")").
			Updates(map[string]any{"point": point, "graded_by": string(models.RoleAI)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := createOrUpdateComment(tx, &models.Comment{
			SubmissionID: submissionID, Comment: comment, CreatedByRole: models.RoleAI,
			CreatedBy: teacherID, Visible: true,
		}); err != nil {
			return err
		}
		applied = true
		return nil
	})
	return applied && err == nil, err
}

package repository

import (
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func CreateComment(comment *models.Comment) (*models.Comment, error) {
	if err := database.DB.Create(comment).Error; err != nil {
		return nil, err
	}

	return comment, nil
}

func CreateOrUpdateComment(comment *models.Comment) error {
	return createOrUpdateComment(database.DB, comment)
}

func createOrUpdateComment(db *gorm.DB, comment *models.Comment) error {
	var existing models.Comment

	err := db.
		Where("submission_id = ? AND created_by_role = ?", comment.SubmissionID, comment.CreatedByRole).
		First(&existing).Error

	if err == nil {
		return db.Model(&existing).Updates(map[string]any{
			"comment":    comment.Comment,
			"visible":    comment.Visible,
			"created_by": comment.CreatedBy,
		}).Error
	}

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(comment).Error
	}

	return err
}

func GetCommentByIDAndCourse(
	commentID string,
	submissionID string,
	courseID string,
) (*models.Comment, error) {

	var comment models.Comment

	err := database.DB.
		Joins("JOIN submissions ON submissions.id = comments.submission_id").
		Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where(`
			comments.id = ?
			AND submissions.id = ?
			AND assignments.course_id = ?
		`, commentID, submissionID, courseID).
		First(&comment).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrCommentNotFound
	}

	return &comment, err
}

func GetCommentBySubmissionAndUserAndRole(submissionID, userID, role string) (*models.Comment, error) {
	var comment models.Comment

	err := database.DB.
		Where("submission_id = ? AND created_by = ? AND created_by_role = ?", submissionID, userID, role).
		First(&comment).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrCommentNotFound
	}

	return &comment, err
}

func UpdateComment(comment *models.Comment) error {
	return database.DB.Save(comment).Error
}

// Serialize authored upserts with grading to prevent duplicate feedback during
// concurrent requests, without requiring a new database constraint.
func SaveAuthoredComment(assignmentID string, comment *models.Comment) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := lockAssignment(tx, assignmentID); err != nil {
			return err
		}
		return upsertAuthoredComment(tx, comment)
	})
}

func upsertAuthoredComment(tx *gorm.DB, comment *models.Comment) error {
	var existing models.Comment
	err := tx.Where("submission_id = ? AND created_by = ? AND created_by_role = ?",
		comment.SubmissionID, comment.CreatedBy, comment.CreatedByRole).First(&existing).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return tx.Create(comment).Error
	}
	if err != nil {
		return err
	}
	if err := tx.Model(&existing).Updates(map[string]any{"comment": comment.Comment, "visible": comment.Visible}).Error; err != nil {
		return err
	}
	comment.ID = existing.ID
	return nil
}

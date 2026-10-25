package repository

import (
	stderrors "errors"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"gorm.io/gorm"
)

const currentSubmissionOrder = "COALESCE(created_at,'0001-01-01 00:00:00+00'::timestamptz) DESC, id DESC"
const currentSubmissionIDs = `SELECT DISTINCT ON (student_id) id FROM submissions
 WHERE assignment_id=? ORDER BY student_id, COALESCE(created_at,'0001-01-01 00:00:00+00'::timestamptz) DESC,id DESC`

func checkCurrentSubmission(tx *gorm.DB, assignmentID, userID, id string) error {
	var current models.Submission
	if err := tx.Select("id").Where("assignment_id=? AND student_id=?", assignmentID, userID).Order(currentSubmissionOrder).Take(&current).Error; err != nil {
		return err
	}
	if current.ID != id {
		return errors.ErrHistoricalSubmission
	}
	return nil
}

func CheckCurrentSubmission(assignmentID, userID, id string) error {
	return checkCurrentSubmission(database.DB, assignmentID, userID, id)
}

// Storage staging is already finished. Assignment locks serialize both first
// creation and replacements, without a new unique index over retained history.
func SaveCurrentSubmission(assignmentID, userID, expectedID string, attachment *models.Attachment) (*models.Submission, error) {
	var current models.Submission
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := lockAssignment(tx, assignmentID)
		if err != nil {
			return err
		}
		err = tx.Where("assignment_id=? AND student_id=?", assignmentID, userID).Order(currentSubmissionOrder).Take(&current).Error
		if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		create := stderrors.Is(err, gorm.ErrRecordNotFound)
		if expectedID != "" && (create || current.ID != expectedID) {
			return errors.ErrHistoricalSubmission
		}
		var attachmentID *string
		if attachment != nil {
			if err := tx.Create(attachment).Error; err != nil {
				return err
			}
			attachmentID = &attachment.ID
		}
		if create {
			current = models.Submission{AssignmentID: assignmentID, StudentID: userID, AttachmentID: attachmentID}
			if err := tx.Create(&current).Error; err != nil {
				return err
			}
		} else {
			result := tx.Model(&models.Submission{}).Where("id=? AND student_id=? AND assignment_id=?", current.ID, userID, assignmentID).Update("attachment_id", attachmentID)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.ErrSubmissionNotFound
			}
		}
		return tx.Preload("Attachment").Preload("Comments", "visible=?", true).Where("id=? AND student_id=? AND assignment_id=?", current.ID, userID, assignmentID).First(&current).Error
	})
	return &current, err
}

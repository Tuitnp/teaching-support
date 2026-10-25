package repository

import (
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func CreateSubmission(submission *models.Submission) error {
	return database.DB.Create(submission).Error
}

func GetSubmission(courseID, submissionID string) (*models.Submission, error) {
	if err := models.ValidateUUIDs(submissionID); err != nil {
		return nil, err
	}

	var submission models.Submission

	err := database.DB.
		Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("submissions.id = ? AND assignments.course_id = ?", submissionID, courseID).
		First(&submission).Error

	if err != nil {
		return nil, err
	}

	return &submission, nil
}

func GetSubmissions(assignmentID string) ([]models.Submission, error) {
	var submissions []models.Submission

	err := database.DB.
		InnerJoins("Attachment").
		Where("assignment_id = ?", assignmentID).
		Find(&submissions).Error

	if err != nil {
		return nil, err
	}

	return submissions, nil
}

func GetSubmissionsWithComments(assignmentID string) ([]models.Submission, error) {
	var submissions []models.Submission

	err := database.DB.
		Where("assignment_id = ?", assignmentID).
		Preload("Attachment").
		Preload("Student").
		Preload("Comments").
		Find(&submissions).
		Error

	return submissions, err
}

func GetSubmissionWithAssignment(submissionID, userID string) (*models.Submission, error) {
	if err := models.ValidateUUIDs(submissionID); err != nil {
		return nil, err
	}

	var submission models.Submission

	err := database.DB.
		Preload("Assignment").
		Preload("Attachment").
		Where("id = ? AND student_id = ?", submissionID, userID).
		First(&submission).Error

	if err != nil {
		return nil, err
	}

	return &submission, nil
}

func GetStudentSubmissions(assignmentID, userID string) ([]models.Submission, error) {
	var submissions []models.Submission

	err := database.DB.
		Where("assignment_id = ? AND student_id = ?", assignmentID, userID).
		Preload("Attachment").
		Preload("Student").
		Preload("Comments", "visible = ?", true).
		Find(&submissions).
		Error

	return submissions, err
}

func CheckSubmissionInCourse(courseID, submissionID string) error {
	var submission models.Submission

	err := database.DB.
		Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("submissions.id = ? AND assignments.course_id = ?", submissionID, courseID).
		First(&submission).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return errors.ErrSubmissionNotFound
	}

	return err
}

func CheckSubmissionInAssignment(courseID, assignmentID, submissionID string) error {
	if err := models.ValidateUUIDs(assignmentID, submissionID); err != nil {
		return err
	}

	var submission models.Submission

	err := database.DB.
		Select("submissions.id").
		Joins("JOIN assignments ON assignments.id = submissions.assignment_id").
		Where("submissions.id = ? AND submissions.assignment_id = ? AND assignments.course_id = ?",
			submissionID, assignmentID, courseID).
		First(&submission).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return errors.ErrSubmissionNotFound
	}

	return err
}

func GetAssignmentSubmissions(assignmentID string) ([]models.Submission, error) {
	var submissions []models.Submission

	err := database.DB.
		Preload("Attachment").
		Where("assignment_id = ?", assignmentID).
		Find(&submissions).Error

	return submissions, err
}

func GetSubmissionsWithAttachments(assignmentID string) ([]models.Submission, error) {
	var submissions []models.Submission

	err := database.DB.
		Preload("Attachment").
		Preload("Student").
		Where("assignment_id = ?", assignmentID).
		Find(&submissions).Error

	return submissions, err
}

func GetSubmissionWithAttachment(id string) (*models.Submission, error) {
	var submission models.Submission

	err := database.DB.
		Preload("Attachment").
		First(&submission, "id = ?", id).Error

	if err != nil {
		return nil, err
	}

	return &submission, nil
}

func UpdateSubmission(id string, updates map[string]any) error {
	return database.DB.Model(&models.Submission{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// ReplaceSubmissionAttachment changes metadata and the reference atomically,
// without saving a stale submission's grades or deleting any storage object.
func ReplaceSubmissionAttachment(id, userID string, attachment *models.Attachment) (*models.Submission, error) {
	var submission models.Submission
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var attachmentID *string
		if attachment != nil {
			if err := tx.Create(attachment).Error; err != nil {
				return err
			}
			attachmentID = &attachment.ID
		}
		result := tx.Model(&models.Submission{}).
			Where("id = ? AND student_id = ?", id, userID).
			Update("attachment_id", attachmentID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.ErrSubmissionNotFound
		}
		return tx.Preload("Attachment").Where("id = ? AND student_id = ?", id, userID).
			First(&submission).Error
	})
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

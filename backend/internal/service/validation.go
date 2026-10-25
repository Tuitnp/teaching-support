package service

import (
	"mime/multipart"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
)

func assignmentValues(form *models.AssignmentForm, existing *models.Assignment) (models.Assignment, error) {
	var result models.Assignment
	if existing != nil {
		result = *existing
	}
	if form.Point != nil {
		result.Point = *form.Point
	}
	if form.StartDate != nil {
		result.StartDate = *form.StartDate
	}
	if form.DueDate != nil {
		result.DueDate = *form.DueDate
	}
	if form.CloseDate != nil {
		result.CloseDate = *form.CloseDate
	}
	return result, models.ValidateAssignmentValues(result.Point, result.StartDate, result.DueDate, result.CloseDate)
}

func validateAIReference(userID string, id *string) error {
	if id == nil {
		return nil
	}
	if err := models.ValidateUUIDs(*id); err != nil {
		return err
	}
	_, err := repository.GetAIConfig(userID, *id)
	return err
}

// Check the entire batch before its first upload or metadata insert.
func validateAttachmentInputs(ids []string, files []*multipart.FileHeader) error {
	if len(ids)+len(files) > 5 {
		return errors.ErrTooManyAttachments
	}
	if err := models.ValidateUUIDs(ids...); err != nil {
		return err
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return errors.ErrBadRequest
		}
		seen[id] = true
	}
	for _, file := range files {
		if file == nil || file.Size < 0 {
			return errors.ErrBadRequest
		}
		if file.Size > maxAttachmentSizeBytes {
			return errors.ErrAttachmentTooLarge
		}
	}
	return nil
}

func validateSubmissionFile(file *multipart.FileHeader, required bool) error {
	if file == nil {
		if required {
			return errors.ErrSubmissionFileRequired
		}
		return nil
	}
	if file.Size <= 0 {
		return errors.ErrSubmissionFileRequired
	}
	if file.Size > maxSubmissionFileSizeBytes {
		return errors.ErrSubmissionFileTooLarge
	}
	return nil
}

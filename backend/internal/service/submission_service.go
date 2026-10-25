package service

import (
	"io"
	"mime/multipart"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
)

const maxSubmissionFileSizeBytes int64 = 5 * 1024 * 1024

func CreateSubmission(
	courseID,
	assignmentID,
	userID string,
	file *multipart.FileHeader,
) (*models.ResponseSubmission, error) {

	assignment, err := repository.GetAssignment(courseID, assignmentID)
	if err != nil {
		return nil, err
	}

	if err := checkSubmissionWindow(assignment, userID, time.Now()); err != nil {
		return nil, err
	}

	if err := validateSubmissionFile(file, true); err != nil {
		return nil, err
	}

	var attachmentID *string

	if file != nil {
		if file.Size > maxSubmissionFileSizeBytes {
			return nil, errors.ErrSubmissionFileTooLarge
		}
		attachment, err := uploadSubmissionFile(userID, file)
		if err != nil {
			return nil, err
		}
		attachmentID = &attachment.ID
	}

	submission := models.Submission{
		AssignmentID: assignmentID,
		StudentID:    userID,
		AttachmentID: attachmentID,
	}

	if err := repository.CreateSubmission(&submission); err != nil {
		return nil, err
	}

	response := models.ConvertSubmissionToResponse(&submission)

	return &response, nil
}

func UpdateSubmission(
	submissionID string,
	userID string,
	file *multipart.FileHeader,
) (*models.ResponseSubmission, error) {

	submission, err := repository.GetSubmissionWithAssignment(submissionID, userID)
	if err != nil {
		return nil, err
	}

	if err := checkSubmissionWindow(&submission.Assignment, userID, time.Now()); err != nil {
		return nil, err
	}

	if err := validateSubmissionFile(file, false); err != nil {
		return nil, err
	}

	attachment, err := handleAttachmentUpdate(file, userID)
	if err != nil {
		return nil, err
	}

	submission, err = repository.ReplaceSubmissionAttachment(submission.ID, userID, attachment)
	if err != nil {
		return nil, err
	}

	resp := models.ConvertSubmissionToResponse(submission)

	return &resp, nil
}

func UpdateGrade(
	courseID string,
	submissionID string,
	role string,
	form *models.GradeForm,
) (map[string]any, error) {

	submission, err := repository.GetSubmission(courseID, submissionID)
	if err != nil {
		return nil, err
	}

	if !canGradeSubmission(role, submission) {
		return nil, errors.ErrForbidden
	}

	if form.Point == nil {
		return nil, errors.ErrInvalidGrade
	}
	if err := repository.UpdateManualGrade(submission.AssignmentID, submission.ID, *form.Point, role); err != nil {
		return nil, err
	}

	data := map[string]any{
		"submission_id": submission.ID,
		"point":         *form.Point,
		"graded_by":     role,
	}

	return data, nil
}

func UpdateGradeAndComment(
	courseID string,
	submissionID string,
	userID string,
	role string,
	form *models.GradeAndCommentForm,
) (map[string]any, error) {

	submission, err := repository.GetSubmission(courseID, submissionID)
	if err != nil {
		return nil, err
	}

	if !canGradeSubmission(role, submission) {
		return nil, errors.ErrForbidden
	}

	if form.Point == nil {
		return nil, errors.ErrInvalidGrade
	}
	if err := repository.UpdateManualGrade(submission.AssignmentID, submission.ID, *form.Point, role); err != nil {
		return nil, err
	}

	commentForm := models.CommentForm{
		Comment: form.Comment,
		Visible: form.Visible,
	}

	responseComment, err := CreateOrUpdateComment(courseID, submissionID, userID, role, &commentForm)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"submission_id": submission.ID,
		"point":         *form.Point,
		"graded_by":     role,
		"comment_id":    responseComment.ID,
		"comment":       responseComment.Comment,
		"visible":       responseComment.Visible,
	}

	return data, nil
}

func ReadSubmission(
	submissionID string,
	userID string,
	role string,
) (string, error) {
	submission, err := repository.GetSubmissionWithAttachment(submissionID)
	if err != nil {
		return "", err
	}

	allowed :=
		models.HasPermission(role, "file:read_all") ||
			(models.HasPermission(role, "file:read_own") &&
				submission.StudentID == userID)

	if !allowed {
		return "", errors.ErrForbidden
	}

	if submission.Attachment == nil {
		return "", errors.ErrAttachmentNotFound
	}

	content, err := storage.ReadFile(submission.Attachment.FileKey)
	if err != nil {
		return "", err
	}

	return content, nil
}

func uploadSubmissionFile(userID string, file *multipart.FileHeader) (*models.Attachment, error) {
	attachment, err := prepareSubmissionFile(userID, file)
	if err != nil {
		return nil, err
	}
	if err := repository.CreateAttachment(attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

// Upload bytes before changing references. Retain staged bytes on DB failure:
// deleting them after an uncertain commit could delete the now-current file.
func prepareSubmissionFile(userID string, file *multipart.FileHeader) (*models.Attachment, error) {

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}

	fileKey, err := storage.UploadFile(data, file.Filename)
	if err != nil {
		return nil, err
	}

	attachment := models.Attachment{
		FileName: file.Filename,
		FileType: file.Header.Get("Content-Type"),
		FileKey:  fileKey,
		Size: 	  file.Size,
		UserID:   userID,
	}

	return &attachment, nil
}

func handleAttachmentUpdate(
	file *multipart.FileHeader,
	userID string,
) (*models.Attachment, error) {
	if file == nil {
		return nil, nil
	}
	return prepareSubmissionFile(userID, file)
}

func canGradeSubmission(role string, s *models.Submission) bool {
	switch role {

	case "teacher":
		return true

	case "ta":
		if s.GradedBy == nil {
			return true
		}
		return *s.GradedBy == "ai"

	default:
		return false
	}
}

func checkSubmissionWindow(assignment *models.Assignment, userID string, now time.Time) error {
	override, err := repository.GetAssignmentOverride(assignment.ID, userID)
	if err != nil {
		return err
	}
	if !models.AcceptsSubmissionAt(assignment, override, now) {
		return errors.ErrAssignmentNotAvailable
	}
	return nil
}

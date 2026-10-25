package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
)

const maxAttachmentSizeBytes int64 = 10 * 1024 * 1024

func stringifyValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprint(val)
	}
}

func CreateAttachments(
	userID string,
	files []*multipart.FileHeader,
) error {
	if err := validateAttachmentInputs(nil, files); err != nil {
		return err
	}

	for _, file := range files {
		if file.Size > maxAttachmentSizeBytes {
			return errors.ErrAttachmentTooLarge
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		data, readErr := io.ReadAll(src)
		src.Close()
		if readErr != nil {
			return readErr
		}

		fileKey, err := storage.UploadFile(data, file.Filename)
		if err != nil {
			return err
		}

		att := models.Attachment{
			FileName: file.Filename,
			FileType: file.Header.Get("Content-Type"),
			FileKey:  fileKey,
			Size:     file.Size,
			UserID:   userID,
		}

		if err := repository.CreateAttachment(&att); err != nil {
			return err
		}
	}

	return nil
}

func GetAttachments(userID string) (*[]models.ResponseAttachment, error) {
	attachments, err := repository.GetAttachments(userID)
	if err != nil {
		return nil, err
	}

	response := models.ConvertAttachmentsToResponse(attachments)

	return &response, nil
}

func DownloadAttachment(userID, attachmentID string) ([]byte, string, string, error) {
	// Get the attachment details
	attachment, err := repository.GetAttachmentByID(attachmentID)
	if err != nil {
		return nil, "", "", err
	}

	if attachment.UserID != userID {
		references, err := repository.GetAttachmentAccessReferences(userID, attachmentID)
		if err != nil {
			return nil, "", "", err
		}
		allowed := false
		now := time.Now()
		for _, reference := range references {
			if canReadAttachmentReference(reference, now) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, "", "", errors.ErrForbidden
		}
	}

	// User has permission, download the file
	fileBytes, err := storage.DownloadFile(attachment.FileKey)
	if err != nil {
		return nil, "", "", err
	}

	return fileBytes, attachment.FileName, attachment.FileType, nil
}

// Membership and assignment state come from the same scoped reference.
func canReadAttachmentReference(reference repository.AttachmentAccessReference, now time.Time) bool {
	if models.HasPermission(reference.Role, "file:read_all") {
		return true
	}
	return reference.Handout && reference.Visible &&
		models.HasPermission(reference.Role, "assignment:view_visible") &&
		!reference.StartDate.IsZero() && !reference.StartDate.After(now)
}

func DeleteAttachment(userID, attachmentID string) error {
	attachment, err := repository.GetAttachment(userID, attachmentID)
	if err != nil {
		return err
	}

	// Delete the file from storage
	if err := storage.DeleteFile(attachment.FileKey); err != nil {
		return err
	}

	if err := repository.DeleteAttachment(attachment); err != nil {
		return err
	}

	return nil
}

func GetAttachmentDetail(userID, attachmentID string) (*models.ResponseAttachmentDetail, error) {
	// Verify the attachment exists and belongs to the user
	_, err := repository.GetAttachment(userID, attachmentID)
	if err != nil {
		return nil, err
	}

	// Get assignments that use this attachment
	assignments, err := repository.GetAssignmentsByAttachmentID(attachmentID)
	if err != nil {
		return nil, err
	}

	// Get templates that use this attachment
	templates, err := repository.GetTemplatesByAttachmentID(attachmentID)
	if err != nil {
		return nil, err
	}

	// Format assignments as "Title (Course ID)"
	relatedAssignments := make([]string, len(assignments))
	for i, a := range assignments {
		title := stringifyValue(a["title"])
		courseID := stringifyValue(a["course_id"])
		relatedAssignments[i] = fmt.Sprintf("%s (%s)", title, courseID)
	}

	// Format templates as "Title"
	relatedTemplates := make([]string, len(templates))
	for i, t := range templates {
		relatedTemplates[i] = stringifyValue(t["title"])
	}

	response := models.ResponseAttachmentDetail{
		RelatedAssignments: relatedAssignments,
		RelatedTemplates:   relatedTemplates,
	}

	return &response, nil
}

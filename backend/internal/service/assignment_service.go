package service

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"mime/multipart"

	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/queue"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
)

func CreateAssignment(
	courseID,
	userID string,
	form *models.AssignmentForm,
	files []*multipart.FileHeader,
) (*models.ResponseAssignment, error) {
	values, err := assignmentValues(form, nil)
	if err != nil {
		return nil, err
	}
	if err := validateAIReference(userID, form.AIConfigID); err != nil {
		return nil, err
	}

	attachments, err := prepareAttachments(userID, form.Attachments, files)
	if err != nil {
		return nil, err
	}

	assignment := models.Assignment{
		CourseID:    courseID,
		Title:       form.Title,
		Description: form.Description,
		Point:       values.Point,
		StartDate:   values.StartDate,
		DueDate:     values.DueDate,
		CloseDate:   values.CloseDate,
		Attachments: attachments,
		AIConfigID:  form.AIConfigID,
		Prompt:      form.Prompt,
		Visible:     form.Visible,
	}

	if err := repository.CreateAssignmentContent(&assignment, form.Tags); err != nil {
		return nil, err
	}

	resp := models.ConvertAssignmentToResponse(&assignment, false, nil)
	return &resp, nil
}

func GetAssignments(userID, courseID, role string) (*[]models.ResponseAssignment, error) {
	var assignments []models.Assignment
	var err error

	if models.HasPermission(role, "assignment:view_all") {
		assignments, err = repository.GetAssignmentsByCourse(courseID)

	} else if models.HasPermission(role, "assignment:view_visible") {
		assignments, err = repository.GetVisibleAssignments(courseID)

	} else {
		return nil, errors.ErrForbidden
	}

	if err != nil {
		return nil, err
	}

	if models.HasPermission(role, "assignment:view_visible") &&
		!models.HasPermission(role, "assignment:view_all") {
		now := time.Now()
		filteredAssignments := make([]models.Assignment, 0, len(assignments))
		for _, assignment := range assignments {
			if assignment.StartDate.IsZero() || !assignment.StartDate.After(now) {
				filteredAssignments = append(filteredAssignments, assignment)
			}
		}
		assignments = filteredAssignments
	}

	overrideMap := map[string]models.AssignmentOverride{}

	if models.HasPermission(role, "submission:view_own") {
		overrides, err := repository.GetOverridesByStudent(userID)
		if err != nil {
			return nil, err
		}

		for _, o := range overrides {
			if previous, ok := overrideMap[o.AssignmentID]; !ok || o.ID > previous.ID {
				overrideMap[o.AssignmentID] = o
			}
		}
	}

	response := models.ConvertAssignmentsToResponse(assignments, overrideMap)

	return &response, nil
}

func GetAssignment(userID, courseID, role, assignmentID string) (*models.ResponseDetailedAssignment, error) {
	var assignment *models.Assignment
	var submissions []models.Submission
	var override *models.AssignmentOverride
	var err error

	if models.HasPermission(role, "assignment:view_all") {
		assignment, err = repository.GetAssignmentWithRelations(courseID, assignmentID)

	} else if models.HasPermission(role, "assignment:view_visible") {
		assignment, err = repository.GetVisibleAssignment(courseID, assignmentID)

	} else {
		return nil, errors.ErrForbidden
	}

	if err != nil {
		return nil, err
	}

	if models.HasPermission(role, "submission:view_all") {
		submissions, err = repository.GetSubmissionsWithComments(assignmentID)

	} else if models.HasPermission(role, "submission:view_own") {
		submissions, err = repository.GetStudentSubmissions(assignmentID, userID)
		if err != nil {
			return nil, err
		}

		override, err = repository.GetAssignmentOverride(assignmentID, userID)
		if err != nil {
			return nil, err
		}

	} else {
		return nil, errors.ErrForbidden
	}

	response := models.ConvertDetailedAssignmentToResponse(assignment, &submissions, override)

	return &response, nil
}

func UpdateAssignment(
	courseID string,
	assignmentID string,
	userID string,
	form *models.AssignmentForm,
	files []*multipart.FileHeader,
) (*models.ResponseAssignment, error) {

	assignment, err := repository.GetAssignmentWithRelations(courseID, assignmentID)
	if err != nil {
		return nil, err
	}

	values, err := assignmentValues(form, assignment)
	if err != nil {
		return nil, err
	}
	if form.Point != nil && values.Point < assignment.Point {
		if err := repository.CheckMaximumPoints(assignmentID, values.Point); err != nil {
			return nil, err
		}
	}
	if err := validateAIReference(userID, form.AIConfigID); err != nil {
		return nil, err
	}
	attachments, err := prepareAttachments(userID, form.Attachments, files)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}

	if form.Title != "" {
		updates["title"] = form.Title
	}

	if form.Description != "" {
		updates["description"] = form.Description
	}

	if form.Point != nil {
		updates["point"] = *form.Point
	}

	if form.StartDate != nil {
		updates["start_date"] = *form.StartDate
	}

	if form.DueDate != nil {
		updates["due_date"] = *form.DueDate
	}

	if form.CloseDate != nil {
		updates["close_date"] = *form.CloseDate
	}

	if form.AIConfigID != nil {
		updates["ai_config_id"] = form.AIConfigID
	}

	if form.Prompt != nil {
		updates["prompt"] = form.Prompt
	}

	updates["visible"] = form.Visible

	assignment, err = repository.UpdateAssignmentContent(courseID, assignmentID, updates, attachments, form.Tags)
	if err != nil {
		return nil, err
	}

	response := models.ConvertAssignmentToResponse(assignment, false, nil)
	return &response, nil
}

func DeleteAssignment(courseID, assignmentID string) error {

	assignment, err := repository.GetAssignment(courseID, assignmentID)
	if err != nil {
		return err
	}

	// Get all submissions for this assignment with their attachments
	submissions, err := repository.GetSubmissionsWithAttachments(assignment.ID)
	if err != nil {
		return err
	}

	// Delete all submission attachment files from S3
	for _, submission := range submissions {
		if submission.Attachment != nil {
			if err := storage.DeleteFile(submission.Attachment.FileKey); err != nil {
				logger.Log.Error("error_delete_file_from_s3",
					"submission_id", submission.ID,
				)
				continue
			}
		}
	}

	if err := repository.DeleteAssignmentDependencies(assignment.ID); err != nil {
		return err
	}

	if err := repository.DeleteAssignment(assignment); err != nil {
		return err
	}

	return nil
}

func CreateAssignmentOverride(
	courseID string,
	assignmentID string,
	form models.AssignmentOverrideForm,
) (*models.AssignmentOverride, error) {

	if err := models.ValidateUUIDs(assignmentID, form.StudentID); err != nil {
		return nil, err
	}

	assignment, err := repository.GetAssignment(courseID, assignmentID)
	if err != nil {
		return nil, err
	}

	if assignment.ID == "" {
		return nil, errors.ErrAssignmentNotFound
	}

	// Check if new due date is more than now - 1 minute (offset for delay)
	if form.ExtendedDueDate.Before(time.Now().Add(-1 * time.Minute)) {
		return nil, errors.ErrInvalidDueDate
	}

	// Check if new due date is more than assignment due date
	if form.ExtendedDueDate.Before(assignment.DueDate) {
		return nil, errors.ErrDueDateBeforeAssignment
	}

	return repository.ReplaceAssignmentOverride(courseID, assignmentID, form.StudentID, form.ExtendedDueDate.UTC())
}

func GetAssignmentSummary(courseID, assignmentID string) (*models.ResponseAssignmentSummary, error) {

	assignment, err := repository.GetAssignment(courseID, assignmentID)
	if err != nil {
		return nil, err
	}

	courseMembers, err := repository.GetActiveStudents(courseID)
	if err != nil {
		return nil, err
	}

	submissions, err := repository.GetAssignmentSubmissions(assignmentID)
	if err != nil {
		return nil, err
	}

	overrides, err := repository.GetOverridesByAssignment(assignmentID)
	if err != nil {
		return nil, err
	}
	return buildAssignmentSummary(assignment, courseMembers, submissions, overrides)
}

func AutoGradingAssignment(userID, courseID, assignmentID string) error {
	assignment, err := repository.GetAssignment(courseID, assignmentID)
	if err != nil {
		return err
	}

	if assignment.AIConfigID == nil {
		return errors.ErrAIConfigNotFound
	}

	job, err := repository.GetGradingJob(assignmentID, userID)
	if err != nil {
		return err
	}
	if job != nil {
		if err := reconcileGradingJob(job); err != nil {
			return err
		}
	}
	return repository.EnqueueGradingRun(assignmentID, userID, func(id string, force bool) error {
		return queue.EnqueueAutoGrading(queue.AutoGradingPayload{JobID: id, AssignmentID: assignmentID, TeacherID: userID, ForceRegade: force})
	})
}

func DownloadSubmissions(courseID, assignmentID string) ([]byte, error) {
	if _, err := repository.GetAssignment(courseID, assignmentID); err != nil {
		return nil, err
	}

	submissions, err := repository.GetSubmissionsWithAttachments(assignmentID)
	if err != nil {
		return nil, err
	}

	zipBuffer := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuffer)

	fileCount := 0

	for _, submission := range submissions {

		if submission.Attachment == nil {
			continue
		}

		fileBytes, err := storage.DownloadFile(submission.Attachment.FileKey)
		if err != nil {
			continue
		}

		safeFileName := sanitizeFileName(submission.Attachment.FileName)
		zipFileName := fmt.Sprintf("%s_%s", submission.Student.Username, safeFileName)

		zipFile, err := zipWriter.Create(zipFileName)
		if err != nil {
			continue
		}

		_, err = zipFile.Write(fileBytes)
		if err != nil {
			continue
		}

		fileCount++
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	if fileCount == 0 {
		return nil, errors.ErrAttachmentNotFound
	}

	return zipBuffer.Bytes(), nil
}

func GetAssignmentsExport(courseID string) ([]byte, error) {
	// Get all active students in the course
	courseMembers, err := repository.GetActiveStudents(courseID)
	if err != nil {
		return nil, err
	}

	// Get all assignments in the course
	assignments, err := repository.GetAssignmentsByCourse(courseID)
	if err != nil {
		return nil, err
	}

	// Create a map of studentID -> courseMember for easy lookup
	studentMap := make(map[string]*models.CourseMember)
	for i := range courseMembers {
		studentMap[courseMembers[i].UserID] = &courseMembers[i]
	}

	// Create a map of assignmentID -> assignment for easy lookup
	assignmentMap := make(map[string]*models.Assignment)
	for i := range assignments {
		assignmentMap[assignments[i].ID] = &assignments[i]
	}

	// Create a map of assignmentID -> submissions
	submissionsMap := make(map[string]map[string]*models.Submission)
	for _, assignment := range assignments {
		submissions, err := repository.GetAssignmentSubmissions(assignment.ID)
		if err != nil {
			return nil, err
		}
		submissionsMap[assignment.ID] = models.LatestFileSubmissions(submissions)
	}

	// Calculate total max points for all assignments
	var totalMaxPoints int16
	for _, assignment := range assignments {
		totalMaxPoints += assignment.Point
	}

	// Create CSV buffer
	csvBuffer := new(bytes.Buffer)
	csvWriter := csv.NewWriter(csvBuffer)

	// Build header row
	header := []string{"username", "EnName", "ThName", "Email"}
	for _, assignment := range assignments {
		header = append(header, fmt.Sprintf("%s (%d)", assignment.Title, assignment.Point))
	}
	header = append(header, fmt.Sprintf("total point (%d)", totalMaxPoints), "percentage")

	// Write header
	if err := csvWriter.Write(header); err != nil {
		return nil, err
	}

	// Write each student's row
	for _, member := range courseMembers {
		row := []string{
			member.User.Username,
			member.User.EnName,
			member.User.ThName,
			member.User.Email,
		}

		var totalPoints int16
		for _, assignment := range assignments {
			var earnedPoint *int16
			if submission := submissionsMap[assignment.ID][member.UserID]; submission != nil {
				earnedPoint = submission.Point
			}

			if earnedPoint != nil {
				row = append(row, fmt.Sprintf("%d", *earnedPoint))
				totalPoints += *earnedPoint
			} else {
				row = append(row, "")
			}
		}

		// Calculate percentage
		percentage := 0.0
		if totalMaxPoints > 0 {
			percentage = float64(totalPoints) / float64(totalMaxPoints) * 100
		}

		row = append(row, fmt.Sprintf("%d", totalPoints), fmt.Sprintf("%.2f", percentage))

		if err := csvWriter.Write(row); err != nil {
			return nil, err
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, err
	}

	return csvBuffer.Bytes(), nil
}

func GetAssignmentExport(courseID, assignmentID string) ([]byte, error) {
	// Get assignment details
	assignment, err := repository.GetAssignment(courseID, assignmentID)
	if err != nil {
		return nil, err
	}

	// Get assignment summary which includes all submission data
	summary, err := GetAssignmentSummary(courseID, assignmentID)
	if err != nil {
		return nil, err
	}

	// Create CSV buffer
	csvBuffer := new(bytes.Buffer)
	csvWriter := csv.NewWriter(csvBuffer)

	// Build header row
	header := []string{
		"username",
		"EnName",
		"ThName",
		"Email",
		fmt.Sprintf("%s (%d)", assignment.Title, assignment.Point),
		"Percentage",
		"Status",
	}

	// Write header
	if err := csvWriter.Write(header); err != nil {
		return nil, err
	}

	// Write each student's row
	for _, submission := range summary.SubmissionList {
		row := []string{
			submission.StudentID,
			submission.EnName,
			submission.ThName,
			submission.Email,
			"", // Assignment title + max point column
			"", // Percentage column will be filled below
			submission.SubmissionStatus,
		}

		// Fill point and percentage if submitted
		if submission.SubmissionStatus != "no submitted" {
			row[4] = fmt.Sprintf("%d", submission.Point)
			row[5] = fmt.Sprintf("%.2f", submission.Percentage)
		}

		if err := csvWriter.Write(row); err != nil {
			return nil, err
		}
	}

	csvWriter.Flush()
	if err := csvWriter.Error(); err != nil {
		return nil, err
	}

	return csvBuffer.Bytes(), nil
}

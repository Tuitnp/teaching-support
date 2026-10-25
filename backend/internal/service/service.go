package service

import (
	"io"
	"mime/multipart"
	"strings"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
)

func buildAssignmentSummary(
	assignment *models.Assignment,
	courseMembers []models.CourseMember,
	submissions []models.Submission,
	overrides []models.AssignmentOverride,
) (*models.ResponseAssignmentSummary, error) {

	totalStudents := int16(len(courseMembers))

	submissionMap := models.LatestFileSubmissions(submissions)
	overrideMap := make(map[string]*models.AssignmentOverride, len(overrides))
	for i := range overrides {
		row := &overrides[i]
		if previous := overrideMap[row.StudentID]; previous == nil || row.ID > previous.ID {
			overrideMap[row.StudentID] = row
		}
	}

	var submitted int16
	var incomplete int16
	var graded int16
	var ungraded int16
	var scores []float32

	submissionList := make([]models.ResponseAssignmentSubmissionList, 0, len(courseMembers))

	for _, cm := range courseMembers {

		var point int16
		var percentage float32
		status := "no submitted"

		submission, ok := submissionMap[cm.UserID]

		if ok && submission.AttachmentID != nil {

			due, _ := models.EffectiveDeadlines(assignment, overrideMap[cm.UserID])
			uploadedAt := models.CurrentFileTime(submission)
			switch {
			case uploadedAt.IsZero():
				status = "timing unavailable"
			case uploadedAt.After(due):
				status = "overdue"
			default:
				status = "submitted"
			}

			if submission.Point != nil {
				point = *submission.Point
				percentage = float32(point) / float32(assignment.Point) * 100
				graded++
				scores = append(scores, float32(point))
			} else {
				ungraded++
			}

			submitted++
		} else {
			incomplete++
		}

		submissionList = append(submissionList, models.ResponseAssignmentSubmissionList{
			UserID:           cm.UserID,
			StudentID:        cm.User.Username,
			EnName:           cm.User.EnName,
			ThName:           cm.User.ThName,
			Email:            cm.User.Email,
			Point:            point,
			Percentage:       percentage,
			SubmissionStatus: status,
		})
	}

	// Calculate statistics
	distribution := createScoreDistribution(scores, float32(assignment.Point))

	var avgScore float32
	var highestScore float32
	var lowestScore float32
	var medianScore float32
	var submissionRate float32

	if graded > 0 {
		sum := float32(0)
		highestScore = scores[0]
		lowestScore = scores[0]

		for _, score := range scores {
			sum += score
			if score > highestScore {
				highestScore = score
			}
			if score < lowestScore {
				lowestScore = score
			}
		}

		avgScore = sum / float32(graded)
		medianScore = calculateMedian(scores)
	}

	if totalStudents > 0 {
		submissionRate = float32(submitted) / float32(totalStudents) * 100
	}

	response := models.ResponseAssignmentSummary{
		Statistic: models.ResponseAssignmentStatistic{
			Student:        totalStudents,
			Submitted:      submitted,
			Incomplete:     incomplete,
			NotStarted:     totalStudents - submitted - incomplete,
			SubmissionRate: submissionRate,
			Graded:         graded,
			Ungraded:       ungraded,
			AverageScore:   avgScore,
			HighestScore:   highestScore,
			LowestScore:    lowestScore,
			MedianScore:    medianScore,
			Distribution:   distribution,
		},
		SubmissionList: submissionList,
	}

	return &response, nil
}

func createScoreDistribution(scores []float32, maxPoints float32) []models.ScoreDistribution {

	if maxPoints <= 0 {
		maxPoints = 100
	}

	binSize := maxPoints / 5

	distribution := make([]models.ScoreDistribution, 5)

	for i := range distribution {
		start := float32(i) * binSize
		end := start + binSize

		if i == len(distribution)-1 {
			end = maxPoints
		}

		distribution[i] = models.ScoreDistribution{
			RangeStart: start,
			RangeEnd:   end,
			Count:      0,
		}
	}

	for _, score := range scores {
		for i := range distribution {
			if i == len(distribution)-1 {

				if score >= distribution[i].RangeStart && score <= distribution[i].RangeEnd {
					distribution[i].Count++
					break
				}
			} else {
				if score >= distribution[i].RangeStart && score < distribution[i].RangeEnd {
					distribution[i].Count++
					break
				}
			}
		}
	}

	return distribution
}

func calculateMedian(scores []float32) float32 {
	if len(scores) == 0 {
		return 0
	}

	// Simple sorting using bubble sort for small slices
	sorted := make([]float32, len(scores))
	copy(sorted, scores)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

func sanitizeFileName(fileName string) string {
	// Replace path separators and other unsafe characters
	unsafeChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := fileName
	for _, char := range unsafeChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}
	return sanitized
}

func resolveTags(names []string) ([]models.Tag, error) {

	if len(names) > 5 {
		names = names[:5]
	}

	var clean []string
	for _, t := range names {
		name := strings.TrimSpace(t)
		if name != "" {
			clean = append(clean, name)
		}
	}

	existing, err := repository.FindTags(clean)
	if err != nil {
		return nil, err
	}

	tagMap := map[string]models.Tag{}
	for _, t := range existing {
		tagMap[t.Name] = t
	}

	var result []models.Tag

	for _, name := range clean {

		if tag, ok := tagMap[name]; ok {
			result = append(result, tag)
			continue
		}

		newTag := models.Tag{Name: name}

		if err := repository.CreateTag(&newTag); err != nil {
			return nil, err
		}

		result = append(result, newTag)
	}

	return result, nil
}

// prepareAttachments stages storage bytes without committing metadata/references.
// Bytes are retained after failure or an uncertain commit, as in R6.
func handleAttachments(userID string, existingIDs []string, files []*multipart.FileHeader) ([]models.Attachment, error) {
	attachments, err := prepareAttachments(userID, existingIDs, files)
	if err != nil {
		return nil, err
	}
	for i := range attachments {
		if attachments[i].ID == "" {
			if err := repository.CreateAttachment(&attachments[i]); err != nil {
				return nil, err
			}
		}
	}
	return attachments, nil
}

func prepareAttachments(
	userID string,
	existingIDs []string,
	files []*multipart.FileHeader,
) ([]models.Attachment, error) {

	if err := validateAttachmentInputs(existingIDs, files); err != nil {
		return nil, err
	}

	var attachments []models.Attachment

	existing, err := repository.GetAttachmentsByIDs(userID, existingIDs)
	if err != nil {
		return nil, err
	}

	if len(existing) != len(existingIDs) {
		return nil, errors.ErrAttachmentNotFound
	}

	attachments = append(attachments, existing...)

	for _, file := range files {
		if file.Size > maxAttachmentSizeBytes {
			return nil, errors.ErrAttachmentTooLarge
		}

		src, err := file.Open()
		if err != nil {
			return nil, err
		}

		data, readErr := io.ReadAll(src)
		src.Close()
		if readErr != nil {
			return nil, readErr
		}

		fileKey, err := storage.UploadFile(data, file.Filename)
		if err != nil {
			return nil, err
		}

		att := models.Attachment{
			FileName: file.Filename,
			FileType: file.Header.Get("Content-Type"),
			FileKey:  fileKey,
			Size:     file.Size,
			UserID:   userID,
		}

		attachments = append(attachments, att)
	}

	return attachments, nil
}

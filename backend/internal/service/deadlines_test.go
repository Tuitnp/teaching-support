package service

import (
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func TestSummaryUsesUploadTimeAndStudentExtension(t *testing.T) {
	due := time.Date(2026, 10, 25, 17, 0, 0, 123456000, time.UTC)
	a := &models.Assignment{DueDate: due, CloseDate: due.Add(time.Hour), Point: 10}
	attachmentID := "file"
	point := int16(7)
	members := []models.CourseMember{{UserID: "student"}}
	for _, tc := range []struct {
		name      string
		uploaded  time.Time
		extension time.Duration
		want      string
	}{
		{"before due", due.Add(-time.Nanosecond), 0, "submitted"},
		{"exact due", due, 0, "submitted"},
		{"after due", due.Add(time.Nanosecond), 0, "overdue"},
		{"own extension", due.Add(time.Hour), time.Hour, "submitted"},
		{"after extension", due.Add(time.Hour + time.Nanosecond), time.Hour, "overdue"},
		{"unknown historical time", time.Time{}, 0, "timing unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			submissions := []models.Submission{{ID: "current", StudentID: "student", AttachmentID: &attachmentID, Attachment: &models.Attachment{CreatedAt: tc.uploaded}, Point: &point}}
			overrides := []models.AssignmentOverride{{StudentID: "other", ExtendedDueDate: due.Add(24 * time.Hour)}}
			if tc.extension != 0 {
				overrides = append(overrides, models.AssignmentOverride{StudentID: "student", ExtendedDueDate: due.Add(tc.extension)})
			}
			for _, grader := range []string{"", "teacher", "ai"} {
				submissions[0].GradedBy = &grader
				submissions[0].UpdatedAt = due.Add(48 * time.Hour)
				got, err := buildAssignmentSummary(a, members, submissions, overrides)
				if err != nil {
					t.Fatal(err)
				}
				if row := got.SubmissionList[0]; row.SubmissionStatus != tc.want || row.Point != point {
					t.Fatalf("grader %s: %+v, want %s and point %d", grader, row, tc.want, point)
				}
			}
		})
	}
}

func TestSummaryUsesCurrentRecordScore(t *testing.T) {
	due := time.Now()
	a := &models.Assignment{DueDate: due, Point: 10}
	fileID := "file"
	oldPoint, newPoint := int16(3), int16(8)
	submissions := []models.Submission{
		{ID: "old", StudentID: "student", CreatedAt: due.Add(-time.Hour), AttachmentID: &fileID, Attachment: &models.Attachment{CreatedAt: due.Add(-time.Hour)}, Point: &oldPoint},
		{ID: "new", StudentID: "student", CreatedAt: due, AttachmentID: &fileID, Attachment: &models.Attachment{CreatedAt: due}, Point: &newPoint},
		{ID: "detached", StudentID: "student", CreatedAt: due.Add(-2 * time.Hour), UpdatedAt: due.Add(time.Hour), Point: &oldPoint},
	}
	got, err := buildAssignmentSummary(a, []models.CourseMember{{UserID: "student"}, {UserID: "empty"}}, submissions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.SubmissionList[1].Point != 8 || got.SubmissionList[1].SubmissionStatus != "submitted" || got.SubmissionList[0].SubmissionStatus != "no submitted" || got.Statistic.Submitted != 1 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

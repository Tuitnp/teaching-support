package service

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func TestCurrentSummaryCategories(t *testing.T) {
	at := time.Now()
	file := "file"
	zero, oldScore := int16(0), int16(9)
	assignment := &models.Assignment{Point: 10, DueDate: at}
	members := []models.CourseMember{{UserID: "a"}, {UserID: "b"}, {UserID: "c"}, {UserID: "d"}, {UserID: "a"}}
	rows := []models.Submission{
		{ID: "a-current", StudentID: "a", CreatedAt: at, AttachmentID: &file, Attachment: &models.Attachment{CreatedAt: at}, Point: &zero},
		{ID: "b-history", StudentID: "b", CreatedAt: at.Add(-time.Hour), AttachmentID: &file, Point: &oldScore},
		{ID: "b-current", StudentID: "b", CreatedAt: at, Point: &oldScore},
		{ID: "d-current", StudentID: "d", CreatedAt: at, AttachmentID: &file},
	}
	var baseline string
	for range 2 {
		result, err := buildAssignmentSummary(assignment, members, rows, nil)
		if err != nil {
			t.Fatal(err)
		}
		stats := result.Statistic
		if stats.Student != 4 || stats.Submitted != 2 || stats.Incomplete != 1 || stats.NotStarted != 1 || stats.Graded != 1 || stats.Ungraded != 1 || stats.SubmissionRate != 50 || stats.AverageScore != 0 || stats.Distribution[0].Count != 1 {
			t.Fatalf("wrong categories/zero score: %+v", stats)
		}
		for _, row := range result.SubmissionList {
			if row.HasGrade != (row.UserID == "a") {
				t.Fatalf("wrong grade presence: %+v", row)
			}
		}
		raw, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		if baseline != "" && baseline != string(raw) {
			t.Fatal("summary depends on query order")
		}
		baseline = string(raw)
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}
}

func TestSummaryStudentCountsDoNotWrap(t *testing.T) {
	members := make([]models.CourseMember, 33000)
	for i := range members {
		members[i].UserID = fmt.Sprintf("student-%05d", i)
	}
	result, err := buildAssignmentSummary(&models.Assignment{Point: 10}, members, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Statistic.Student != 33000 || result.Statistic.NotStarted != 33000 || result.Statistic.SubmissionRate != 0 {
		t.Fatalf("count overflow: %+v", result.Statistic)
	}
}

func TestSummaryLegacyZeroMaximumIsSerializable(t *testing.T) {
	point := int16(0)
	file := "file"
	result, err := buildAssignmentSummary(&models.Assignment{}, []models.CourseMember{{UserID: "student"}}, []models.Submission{{ID: "current", StudentID: "student", AttachmentID: &file, Point: &point}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatal("legacy maximum produced a non-finite statistic", err)
	}
}

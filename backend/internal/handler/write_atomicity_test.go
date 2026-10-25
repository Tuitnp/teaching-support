package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
)

func TestManualGradeCommentAtomicity(t *testing.T) {
	db := submissionTestDB(t)
	for _, name := range []string{"comment-update-failure", "comment-insert-failure", "grade-failure", "retry"} {
		t.Run(name, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			actor := f.teacher
			switch name {
			case "comment-update-failure", "comment-insert-failure":
				mustExec(t, f.db, `ALTER TABLE comments ADD CONSTRAINT reject_feedback CHECK (comment <> 'feedback')`)
				if name == "comment-insert-failure" {
					actor = f.studentAsTeacher
				}
			case "grade-failure":
				mustExec(t, f.db, `ALTER TABLE submissions ADD CONSTRAINT reject_grade CHECK (point IS NULL OR point <> 7)`)
			}
			before := f.snapshot(t)
			path := submissionPath("COURSA", f.a1, f.own, "/grade-with-comment")
			status, body := f.request(t, actor, http.MethodPut, path, "grade-comment")
			if name != "retry" {
				if status != 500 || before != f.snapshot(t) || f.files.calls != 0 {
					t.Fatalf("failed write was not atomic: status %d, body %s", status, body)
				}
				return
			}
			if status != 200 {
				t.Fatalf("status %d: %s", status, body)
			}
			status, body = f.request(t, actor, http.MethodPut, path, "grade-comment")
			if status != 200 {
				t.Fatalf("retry status %d: %s", status, body)
			}
			var comments []models.Comment
			if err := f.db.Where("submission_id=? AND created_by=? AND created_by_role='teacher'", f.own, actor).Find(&comments).Error; err != nil {
				t.Fatal(err)
			}
			if len(comments) != 1 || comments[0].ID != f.comment || comments[0].Comment != "feedback" {
				t.Fatal("retry duplicated or replaced feedback identity")
			}
		})
	}
}

func TestConcurrentAuthoredFeedback(t *testing.T) {
	db := submissionTestDB(t)
	f := newSubmissionFixture(t, db)
	mustExec(t, f.db, `DELETE FROM comments WHERE submission_id=?`, f.own)
	if err := f.db.Commit().Error; err != nil {
		t.Fatal(err)
	}
	database.DB = db
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := service.CreateOrUpdateComment("COURSA", f.own, f.teacher, "teacher", &models.CommentForm{Comment: "feedback", Visible: true})
			done <- err
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent feedback timed out")
		}
	}
	var count int64
	if err := db.Model(&models.Comment{}).Where("submission_id=? AND created_by=? AND created_by_role='teacher'", f.own, f.teacher).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("authored feedback count = %d, error %v", count, err)
	}
}

package handler_test

import (
	"net/http"
	"strings"
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

func TestAssignmentContentAtomicity(t *testing.T) {
	db := submissionTestDB(t)
	for _, create := range []bool{false, true} {
		for _, failure := range []string{"attachment", "tag", "attachment-join", "tag-join", "success"} {
			name := map[bool]string{false: "edit", true: "create"}[create] + "/" + failure
			t.Run(name, func(t *testing.T) {
				f := newSubmissionFixture(t, db)
				validationTables(t, f.db)
				mustExec(t, f.db, `INSERT INTO tags (name) VALUES ('existing')`)
				var old string
				if err := f.db.Raw(`SELECT attachment_id FROM submissions WHERE id=?`, f.own).Scan(&old).Error; err != nil {
					t.Fatal(err)
				}
				mustExec(t, f.db, `INSERT INTO assignment_attachments VALUES (?,?)`, f.a1, old)
				switch failure {
				case "attachment":
					mustExec(t, f.db, `ALTER TABLE attachments ADD CONSTRAINT reject_upload CHECK (file_name <> 'fixture.py')`)
				case "tag":
					mustExec(t, f.db, `ALTER TABLE tags ADD CONSTRAINT reject_tag CHECK (name <> 'new')`)
				case "attachment-join":
					mustExec(t, f.db, `ALTER TABLE assignment_attachments ADD CONSTRAINT reject_link CHECK (false) NOT VALID`)
				case "tag-join":
					mustExec(t, f.db, `ALTER TABLE assignment_tags ADD CONSTRAINT reject_tag_link CHECK (false) NOT VALID`)
				}
				fields := map[string][]string{"title": {"Changed"}, "tags": {"existing", "new"}, "visible": {"true"}}
				path, method := "/api/courses/COURSA/assignments/"+f.a1, http.MethodPut
				if create {
					path, method = "/api/courses/COURSA/assignments", http.MethodPost
					fields["point"] = []string{"10"}
					fields["start"] = []string{time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)}
					fields["due"] = []string{time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
					fields["close"] = []string{time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339Nano)}
				}
				ct, body := validationMultipart(t, fields, "files", []int{1})
				before := validationSnapshot(t, f)
				status, data := validationRequest(t, f, f.teacher, method, path, ct, body)
				if failure != "success" {
					// Staged storage bytes may remain, but no metadata/scalars/joins/tags may commit.
					staged := firstStagedKey(f)
					if staged == "" || f.files.calls != 1 {
						t.Fatal("failed DB write did not retain staged upload")
					}
					delete(f.files.data, staged)
					if status != 500 || validationSnapshot(t, f) != before {
						t.Fatalf("partial %s write: %d %s", name, status, data)
					}
					return
				}
				if status != 200 || !strings.Contains(data, "Changed") || !strings.Contains(data, "fixture.py") || !strings.Contains(data, "existing") || !strings.Contains(data, "new") {
					t.Fatalf("content not saved: %d %s", status, data)
				}
			})
		}
	}
}

// Locate only newly staged bytes, without deleting the pre-existing fixture data.
func firstStagedKey(f *submissionFixture) string {
	for key := range f.files.data {
		if key != f.own && key != f.peer && key != f.foreign {
			return key
		}
	}
	return ""
}

func TestOverrideFailurePreservesExtension(t *testing.T) {
	f := newSubmissionFixture(t, submissionTestDB(t))
	previous := time.Now().Add(3 * time.Hour).UTC()
	mustExec(t, f.db, `INSERT INTO assignment_overrides (assignment_id,student_id,extended_due_date) VALUES (?,?,?)`, f.a1, f.student, previous)
	mustExec(t, f.db, `CREATE FUNCTION reject_extension() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic extension failure'; END $$`)
	mustExec(t, f.db, `CREATE TRIGGER reject_extension BEFORE UPDATE ON assignment_overrides FOR EACH ROW EXECUTE FUNCTION reject_extension()`)
	var before string
	if err := f.db.Raw(`SELECT row_to_json(o)::text FROM assignment_overrides o`).Scan(&before).Error; err != nil {
		t.Fatal(err)
	}
	_, err := service.CreateAssignmentOverride("COURSA", f.a1, models.AssignmentOverrideForm{StudentID: f.student, ExtendedDueDate: previous.Add(time.Hour)})
	if err == nil {
		t.Fatal("expected extension failure")
	}
	var after string
	if err := f.db.Raw(`SELECT row_to_json(o)::text FROM assignment_overrides o`).Scan(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatal("failed replacement destroyed extension")
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

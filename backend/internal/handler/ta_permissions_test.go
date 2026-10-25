package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	apperrors "github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/google/uuid"
)

func TestTACourseReadAccess(t *testing.T) {
	db := submissionTestDB(t)
	for _, name := range []string{"courses", "list", "detail", "summary", "members", "read", "download", "inactive-download", "handout", "global-student-course-ta", "global-ta-course-student", "inactive", "withdraw", "drop", "no-membership", "foreign-course", "wrong-parent"} {
		t.Run(name, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			validationTables(t, f.db)
			mustExec(t, f.db, `UPDATE users SET role = 'ta' WHERE id = ?`, f.ta)
			mustExec(t, f.db, `UPDATE assignments SET visible = false, start_date = ? WHERE id = ?`, time.Now().Add(30*time.Minute), f.a1)
			path := submissionPath("COURSA", f.a1, f.own, "/read")
			want := 200
			switch name {
			case "courses":
				mustExec(t, f.db, `CREATE TABLE courses (id text PRIMARY KEY, name text, course_code text, day_of_week text, start_time text, end_time text, room text, credits smallint, section text, semester text, teacher_id uuid, created_at timestamptz, updated_at timestamptz)`)
				mustExec(t, f.db, `INSERT INTO courses (id,name,teacher_id) VALUES ('COURSA','Course A',?), ('COURSB','Course B',?)`, f.teacher, f.teacher)
				path = "/api/courses"
			case "list":
				path = "/api/courses/COURSA/assignments"
			case "detail":
				path = "/api/courses/COURSA/assignments/" + f.a1
			case "summary":
				path = "/api/courses/COURSA/assignments/" + f.a1 + "/summary"
			case "members":
				path = "/api/courses/COURSA/members"
			case "download", "inactive-download", "handout":
				var id string
				if err := f.db.Raw(`SELECT attachment_id FROM submissions WHERE id = ?`, f.own).Scan(&id).Error; err != nil {
					t.Fatal(err)
				}
				if name == "handout" {
					mustExec(t, f.db, `UPDATE submissions SET attachment_id = NULL WHERE id = ?`, f.own)
					mustExec(t, f.db, `INSERT INTO assignment_attachments VALUES (?, ?)`, f.a1, id)
				}
				path = "/api/attachments/" + id + "/download"
				if name == "inactive-download" {
					mustExec(t, f.db, `UPDATE course_members SET status='inactive' WHERE user_id=?`, f.ta)
					want = 403
				}
			case "global-student-course-ta":
				mustExec(t, f.db, `UPDATE users SET role='student' WHERE id=?`, f.ta)
			case "global-ta-course-student":
				mustExec(t, f.db, `UPDATE course_members SET role='student' WHERE user_id=?`, f.ta)
				want = 403
			case "inactive", "withdraw", "drop":
				mustExec(t, f.db, `UPDATE course_members SET status=? WHERE user_id=?`, name, f.ta)
				want = 403
			case "no-membership":
				mustExec(t, f.db, `DELETE FROM course_members WHERE user_id=?`, f.ta)
				want = 403
			case "foreign-course":
				path, want = submissionPath("COURSB", f.b, f.foreign, "/read"), 403
			case "wrong-parent":
				path, want = submissionPath("COURSA", f.a2, f.own, "/read"), 404
			}
			before := validationSnapshot(t, f)
			status, body := f.request(t, f.ta, http.MethodGet, path, "")
			if status != want {
				t.Fatalf("status = %d, want %d: %s", status, want, body)
			}
			if validationSnapshot(t, f) != before {
				t.Fatal("read changed data")
			}
			if want != 200 && f.files.calls != 0 {
				t.Fatal("denied read accessed storage")
			}
			if name == "list" && !strings.Contains(string(body), f.a1) {
				t.Fatal("TA cannot see hidden upcoming assignment")
			}
			if name == "courses" && (!strings.Contains(string(body), "COURSA") || strings.Contains(string(body), "COURSB")) {
				t.Fatal("global TA course list did not respect active membership")
			}
			if name == "detail" && (!strings.Contains(string(body), f.own) || !strings.Contains(string(body), f.peer) || strings.Contains(string(body), f.foreign)) {
				t.Fatal("wrong submission scope")
			}
		})
	}
}

func TestManualGradeRolePrecedence(t *testing.T) {
	db := submissionTestDB(t)
	for _, role := range []string{"ta", "teacher", "student"} {
		for _, suffix := range []string{"/grade", "/grade-with-comment"} {
			for _, previous := range []string{"ungraded", "ai", "teacher", "ta", "unknown", "unattributed", "empty-grader", "human-without-point"} {
				t.Run(role+suffix+"/"+previous, func(t *testing.T) {
					f := newSubmissionFixture(t, db)
					actor := f.ta
					if role == "teacher" {
						actor = f.teacher
					}
					if role == "student" {
						actor = f.student
					}
					var point any = int16(4)
					var grader any = previous
					switch previous {
					case "ungraded":
						point, grader = nil, nil
					case "unattributed":
						grader = nil
					case "empty-grader":
						grader = ""
					case "human-without-point":
						point, grader = nil, "teacher"
					}
					mustExec(t, f.db, `UPDATE submissions SET point=?, graded_by=? WHERE id=?`, point, grader, f.own)
					before := f.snapshot(t)
					want := 403
					if role == "teacher" || (role == "ta" && (previous == "ungraded" || previous == "ai")) {
						want = 200
					}
					status, body := f.request(t, actor, http.MethodPut, submissionPath("COURSA", f.a1, f.own, suffix), "grade")
					if status != want {
						t.Fatalf("status = %d, want %d: %s", status, want, body)
					}
					if want == 403 && f.snapshot(t) != before {
						t.Fatal("denied grade changed grade or comments")
					}
					if f.files.calls != 0 {
						t.Fatal("grading accessed file storage")
					}
					if want == 200 {
						var saved models.Submission
						if err := f.db.First(&saved, "id=?", f.own).Error; err != nil {
							t.Fatal(err)
						}
						if saved.Point == nil || *saved.Point != 7 || saved.GradedBy == nil || *saved.GradedBy != role {
							t.Fatal("wrong persisted grade")
						}
						if suffix == "/grade-with-comment" {
							var count int64
							if err := f.db.Model(&models.Comment{}).Where("submission_id=? AND created_by=? AND created_by_role=? AND comment='feedback'", f.own, actor, role).Count(&count).Error; err != nil || count != 1 {
								t.Fatal("missing authored feedback")
							}
						}
					}
				})
			}
		}
	}
}

func TestTAFeedbackOwnership(t *testing.T) {
	db := submissionTestDB(t)
	for _, name := range []string{"create", "update-own", "toggle-own", "toggle-other-ta", "toggle-teacher", "toggle-ai", "teacher-toggle-ta"} {
		t.Run(name, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			own, other, ai := uuid.NewString(), uuid.NewString(), uuid.NewString()
			mustExec(t, f.db, `INSERT INTO comments (id,submission_id,comment,created_by_role,created_by,visible) VALUES (?,?,'my feedback','ta',?,true), (?,?,'other feedback','ta',?,true), (?,?,'AI feedback','ai',?,true)`, own, f.own, f.ta, other, f.own, f.other, ai, f.own, f.teacher)
			path := submissionPath("COURSA", f.a1, f.own, "/comment")
			actor, want := f.ta, 200
			switch name {
			case "create":
				mustExec(t, f.db, `DELETE FROM comments WHERE id=?`, own)
			case "toggle-own":
				path += "/" + own
			case "toggle-other-ta":
				path, want = path+"/"+other, 403
			case "toggle-teacher":
				path, want = path+"/"+f.comment, 403
			case "toggle-ai":
				path, want = path+"/"+ai, 403
			case "teacher-toggle-ta":
				actor, path = f.teacher, path+"/"+own
			}
			before := f.snapshot(t)
			status, body := f.request(t, actor, http.MethodPost, path, "comment")
			if status != want {
				t.Fatalf("status = %d, want %d: %s", status, want, body)
			}
			if want == 403 && f.snapshot(t) != before {
				t.Fatal("denied visibility toggle changed feedback")
			}
			if want == 200 {
				var result struct {
					ID      string `json:"id"`
					Visible bool   `json:"visible"`
				}
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatal(err)
				}
				if strings.Contains(name, "toggle") && (result.ID != own || result.Visible) {
					t.Fatal("own feedback not hidden")
				}
				if name == "create" || name == "update-own" {
					var comment models.Comment
					if err := f.db.First(&comment, "id=?", result.ID).Error; err != nil {
						t.Fatal(err)
					}
					if comment.CreatedBy != f.ta || comment.CreatedByRole != models.RoleTeacherAssistance || comment.Comment != "feedback" {
						t.Fatal("incorrect feedback owner/content")
					}
				}
			}
		})
	}
}

func TestTACriticalActionsDenied(t *testing.T) {
	db := submissionTestDB(t)
	for _, action := range []struct {
		name, method, path string
		global             bool
	}{
		{"course-create", "POST", "/api/courses", true},
		{"course-update", "PUT", "/api/courses/COURSA", false},
		{"course-delete", "DELETE", "/api/courses/COURSA", false},
		{"enroll", "POST", "/api/courses/COURSA/enroll", false},
		{"member-update", "PUT", "/api/courses/COURSA/members/MEMBER", false},
		{"member-delete", "DELETE", "/api/courses/COURSA/members/MEMBER", false},
		{"assignment-create", "POST", "/api/courses/COURSA/assignments", false},
		{"assignment-update", "PUT", "/api/courses/COURSA/assignments/ASSIGNMENT", false},
		{"assignment-delete", "DELETE", "/api/courses/COURSA/assignments/ASSIGNMENT", false},
		{"override", "POST", "/api/courses/COURSA/assignments/ASSIGNMENT/override", false},
		{"ai-grading", "GET", "/api/courses/COURSA/assignments/ASSIGNMENT/auto-grading", false},
		{"zip", "GET", "/api/courses/COURSA/assignments/ASSIGNMENT/submissions/download", false},
		{"course-export", "GET", "/api/courses/COURSA/assignments/export", false},
		{"assignment-export", "GET", "/api/courses/COURSA/assignments/ASSIGNMENT/export", false},
		{"submission-create", "POST", "/api/courses/COURSA/assignments/ASSIGNMENT/submissions", false},
		{"submission-update", "PUT", "/api/courses/COURSA/assignments/ASSIGNMENT/submissions/SUBMISSION", false},
		{"template-create", "POST", "/api/templates", true},
		{"template-list", "GET", "/api/templates", true},
		{"template-detail", "GET", "/api/templates/SUBMISSION", true},
		{"template-update", "PUT", "/api/templates/SUBMISSION", true},
		{"template-delete", "DELETE", "/api/templates/SUBMISSION", true},
		{"upload", "POST", "/api/attachments", true},
		{"attachment-list", "GET", "/api/attachments", true},
		{"attachment-delete", "DELETE", "/api/attachments/SUBMISSION", true},
		{"ai-config", "GET", "/api/ai/configs/SUBMISSION", true},
		{"python", "POST", "/api/run/python3", true},
	} {
		t.Run(action.name, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			validationTables(t, f.db)
			// Course gates must restrict a TA even when the global role is teacher.
			// Global operations instead use the independent global role.
			if action.global {
				mustExec(t, f.db, `UPDATE users SET role='ta' WHERE id=?`, f.ta)
			}
			path := strings.NewReplacer("ASSIGNMENT", f.a1, "SUBMISSION", f.own, "MEMBER", f.student).Replace(action.path)
			before := validationSnapshot(t, f)
			status, body := f.request(t, f.ta, action.method, path, action.name)
			if status != 403 {
				t.Fatalf("status = %d, want 403: %s", status, body)
			}
			if validationSnapshot(t, f) != before || f.files.calls != 0 {
				t.Fatal("denied action changed data or accessed storage")
			}
		})
	}
}

func TestTAGradeRechecksConcurrentHumanGrade(t *testing.T) {
	for _, withComment := range []bool{false, true} {
		t.Run(map[bool]string{false: "grade", true: "grade-comment"}[withComment], func(t *testing.T) {
			db := submissionTestDB(t)
			f := newSubmissionFixture(t, db)
			if err := f.db.Commit().Error; err != nil {
				t.Fatal(err)
			}
			f.db, database.DB = db, db
			control := db.Begin()
			t.Cleanup(func() { _ = control.Rollback().Error })
			var pid int
			if err := control.Raw(`SELECT pg_backend_pid()`).Scan(&pid).Error; err != nil {
				t.Fatal(err)
			}
			var locked string
			if err := control.Raw(`SELECT id FROM assignments WHERE id=? FOR UPDATE`, f.a1).Scan(&locked).Error; err != nil {
				t.Fatal(err)
			}
			mustExec(t, control, `UPDATE submissions SET point=9, graded_by='teacher' WHERE id=?`, f.own)
			done := make(chan error, 1)
			go func() {
				p := int16(7)
				var err error
				if withComment {
					_, err = service.UpdateGradeAndComment("COURSA", f.own, f.ta, "ta", &models.GradeAndCommentForm{Point: &p, Comment: "stale TA feedback", Visible: true})
				} else {
					_, err = service.UpdateGrade("COURSA", f.own, "ta", &models.GradeForm{Point: &p})
				}
				done <- err
			}()
			blocked := false
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				if err := db.Raw(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid)))`, pid).Scan(&blocked).Error; err != nil {
					t.Fatal(err)
				}
				if blocked {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !blocked {
				t.Fatal("TA did not wait for teacher's assignment lock")
			}
			if err := control.Commit().Error; err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != apperrors.ErrForbidden {
					t.Fatalf("got %v, want forbidden", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("TA write remained blocked")
			}
			var saved models.Submission
			if err := db.First(&saved, "id=?", f.own).Error; err != nil {
				t.Fatal(err)
			}
			if saved.Point == nil || *saved.Point != 9 || saved.GradedBy == nil || *saved.GradedBy != "teacher" {
				t.Fatal("TA replaced concurrent teacher grade")
			}
			var count int64
			if err := db.Model(&models.Comment{}).Where("created_by_role='ta'").Count(&count).Error; err != nil || count != 0 {
				t.Fatal("rejected TA write left feedback")
			}
		})
	}
}

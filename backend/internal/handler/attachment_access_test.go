package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAttachmentDownloadAccess(t *testing.T) {
	db := submissionTestDB(t)
	for _, name := range []string{
		"released-student", "hidden-student", "upcoming-student", "closed-student", "unknown-start",
		"teacher-hidden", "teacher-upcoming", "teacher-submission", "teacher-hidden-submission",
		"peer-submission", "owner-submission", "owner-hidden-inactive", "owner-detached", "owner-template",
		"nonowner-detached", "nonowner-template", "inactive-student", "inactive-teacher",
		"withdraw-teacher", "drop-teacher", "no-membership", "unknown-role", "admin",
		"global-teacher-course-student", "global-student-course-teacher", "no-auth", "malformed", "missing",
		"hidden-plus-released", "submission-plus-released", "submission-other-course-plus-authorized",
		"cross-course-membership", "cross-course-release", "unrelated-visible-assignment",
	} {
		t.Run(name, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			mustExec(t, f.db, `CREATE TABLE assignment_attachments (assignment_id uuid, attachment_id uuid)`)
			id := uuid.NewString()
			f.files.data[id] = []byte("handout bytes")
			mustExec(t, f.db, `INSERT INTO attachments (id, file_key, file_name, file_type, user_id) VALUES (?, ?, 'handout.txt', 'text/plain', ?)`, id, id, f.teacher)
			link := func(assignment string) {
				mustExec(t, f.db, `INSERT INTO assignment_attachments VALUES (?, ?)`, assignment, id)
			}
			link(f.a1)
			actor, want := f.student, http.StatusOK
			if name == "owner-submission" || name == "peer-submission" || name == "teacher-submission" || name == "teacher-hidden-submission" {
				mustExec(t, f.db, `DELETE FROM assignment_attachments`)
				mustExec(t, f.db, `UPDATE submissions SET attachment_id = ? WHERE id = ?`, id, f.own)
				mustExec(t, f.db, `UPDATE attachments SET user_id = ? WHERE id = ?`, f.student, id)
			}
			switch name {
			case "hidden-student", "global-teacher-course-student":
				mustExec(t, f.db, `UPDATE assignments SET visible = false WHERE id = ?`, f.a1)
				want = 403
				if name == "global-teacher-course-student" {
					actor = f.teacherAsStudent
				}
			case "upcoming-student":
				mustExec(t, f.db, `UPDATE assignments SET start_date = ? WHERE id = ?`, time.Now().Add(time.Hour), f.a1)
				want = 403
			case "closed-student":
				mustExec(t, f.db, `UPDATE assignments SET due_date = ?, close_date = ? WHERE id = ?`, time.Now().Add(-time.Minute), time.Now().Add(-time.Minute), f.a1)
			case "unknown-start":
				mustExec(t, f.db, `UPDATE assignments SET start_date = NULL WHERE id = ?`, f.a1)
				want = 403
			case "teacher-hidden", "teacher-hidden-submission", "global-student-course-teacher":
				mustExec(t, f.db, `UPDATE assignments SET visible = false WHERE id = ?`, f.a1)
				actor = f.studentAsTeacher // Non-owner, course teacher despite global student role.
			case "teacher-upcoming":
				mustExec(t, f.db, `UPDATE assignments SET start_date = ? WHERE id = ?`, time.Now().Add(time.Hour), f.a1)
				actor = f.studentAsTeacher
			case "teacher-submission":
				actor = f.teacher
			case "peer-submission":
				actor, want = f.other, 403
			case "owner-hidden-inactive":
				actor = f.teacher
				mustExec(t, f.db, `UPDATE assignments SET visible = false, start_date = ?`, time.Now().Add(time.Hour))
				mustExec(t, f.db, `UPDATE course_members SET status = 'inactive' WHERE user_id = ?`, actor)
			case "owner-detached", "owner-template", "nonowner-detached", "nonowner-template":
				mustExec(t, f.db, `DELETE FROM assignment_attachments`)
				if name == "owner-template" || name == "nonowner-template" {
					mustExec(t, f.db, `CREATE TABLE assignment_template_attachments (assignment_template_id uuid, attachment_id uuid)`)
					mustExec(t, f.db, `INSERT INTO assignment_template_attachments VALUES (?, ?)`, uuid.NewString(), id)
				}
				if name == "owner-detached" || name == "owner-template" {
					actor = f.teacher
				} else {
					want = 403
				}
			case "inactive-student", "inactive-teacher", "withdraw-teacher", "drop-teacher", "no-membership", "unknown-role", "admin":
				want = 403
				if name != "inactive-student" {
					actor = f.studentAsTeacher
				}
				switch name {
				case "inactive-student", "inactive-teacher":
					mustExec(t, f.db, `UPDATE course_members SET status = 'inactive' WHERE user_id = ?`, actor)
				case "withdraw-teacher":
					mustExec(t, f.db, `UPDATE course_members SET status = 'withdraw' WHERE user_id = ?`, actor)
				case "drop-teacher":
					mustExec(t, f.db, `UPDATE course_members SET status = 'drop' WHERE user_id = ?`, actor)
				case "no-membership":
					mustExec(t, f.db, `DELETE FROM course_members WHERE user_id = ?`, actor)
				case "unknown-role":
					mustExec(t, f.db, `UPDATE course_members SET role = 'unexpected' WHERE user_id = ?`, actor)
				case "admin":
					actor = f.admin
				}
			case "no-auth":
				actor, want = "", 401
			case "malformed":
				id, want = "bad-id", 400
			case "missing":
				id, want = uuid.NewString(), 404
			case "hidden-plus-released":
				mustExec(t, f.db, `UPDATE assignments SET visible = false WHERE id = ?`, f.a1)
				link(f.a2)
			case "submission-plus-released", "submission-other-course-plus-authorized":
				sid := f.peer
				if name == "submission-other-course-plus-authorized" {
					sid = f.foreign
				}
				mustExec(t, f.db, `UPDATE submissions SET attachment_id = ? WHERE id = ?`, id, sid)
			case "cross-course-membership":
				mustExec(t, f.db, `DELETE FROM assignment_attachments`)
				link(f.b) // Membership in A cannot authorize a B reference (B membership inactive).
				want = 403
			case "cross-course-release":
				mustExec(t, f.db, `UPDATE assignments SET visible = false WHERE id = ?`, f.a1)
				link(f.b) // Visible reference in B cannot authorize a hidden reference in active course A.
				want = 403
			case "unrelated-visible-assignment":
				mustExec(t, f.db, `UPDATE assignments SET visible = false WHERE id = ?`, f.a1)
				want = 403 // Visible A2 does not reference this attachment.
			}
			before := f.snapshot(t)
			status, body := f.request(t, actor, http.MethodGet, "/api/attachments/"+id+"/download", "")
			if status != want {
				t.Fatalf("status = %d, want %d: %s", status, want, body)
			}
			calls := 0
			if want == 200 {
				calls = 1
				if string(body) != "handout bytes" {
					t.Errorf("wrong file: %s", body)
				}
			}
			if f.files.calls != calls || f.snapshot(t) != before {
				t.Fatal("unexpected storage access or mutation")
			}
		})
	}
}

func TestAttachmentDownloadFailsClosedOnDatabaseError(t *testing.T) {
	f := newSubmissionFixture(t, submissionTestDB(t))
	var id string
	if err := f.db.Raw(`SELECT attachment_id FROM submissions WHERE id = ?`, f.own).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	// Missing join table deliberately causes authorization lookup failure.
	status, _ := f.request(t, f.teacher, http.MethodGet, "/api/attachments/"+id+"/download", "")
	if status != 500 || f.files.calls != 0 {
		t.Fatalf("status = %d, storage calls = %d", status, f.files.calls)
	}
}

package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/google/uuid"
)

func TestMembershipDuplicatePolicy(t *testing.T) {
	db := submissionTestDB(t)
	for _, name := range []string{"identical", "role-conflict", "status-conflict", "teacher-conflict", "resolved"} {
		t.Run(name, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			validationTables(t, f.db)
			actor, role, status := f.student, "student", "active"
			switch name {
			case "role-conflict", "resolved":
				role = "teacher"
			case "status-conflict":
				status = "inactive"
			case "teacher-conflict":
				actor, role = f.teacher, "student"
			}
			mustExec(t, f.db, `INSERT INTO course_members (user_id,course_id,role,status) VALUES (?,'COURSA',?,?)`, actor, role, status)
			if name == "resolved" {
				code, body := validationRequest(t, f, f.teacher, "PUT", "/api/courses/COURSA/members/"+actor, "application/json", strings.NewReader(`{"new_role":"student","new_status":"active"}`))
				if code != 200 {
					t.Fatalf("resolution failed %d: %s", code, body)
				}
			}
			before := f.snapshot(t)
			want := 409
			if name == "identical" || name == "resolved" {
				want = 200
			}
			code, body := f.request(t, actor, "GET", submissionPath("COURSA", f.a1, f.own, "/read"), "")
			if code != want {
				t.Fatalf("read %d want %d: %s", code, want, body)
			}
			if f.snapshot(t) != before {
				t.Fatal("read changed records")
			}
			if want == 409 && f.files.calls != 0 {
				t.Fatal("ambiguous membership reached storage")
			}
			for _, suffix := range []string{"/summary", "/export"} {
				code, body = f.request(t, f.teacher, "GET", "/api/courses/COURSA/assignments/"+f.a1+suffix, "")
				if code != want {
					t.Fatalf("report %s status %d want %d: %s", suffix, code, want, body)
				}
			}
			code, body = f.request(t, f.teacher, "GET", "/api/courses/COURSA/assignments/export", "")
			if code != want {
				t.Fatalf("course export %d want %d: %s", code, want, body)
			}
			if want == 200 {
				rows, err := repository.GetActiveStudents("COURSA")
				if err != nil || len(rows) != 3 {
					t.Fatalf("student count=%d error=%v", len(rows), err)
				}
			}
		})
	}
}

func TestMembershipAttachmentConflict(t *testing.T) {
	f := newSubmissionFixture(t, submissionTestDB(t))
	validationTables(t, f.db)
	mustExec(t, f.db, `INSERT INTO course_members (user_id,course_id,role,status) VALUES (?,'COURSA','teacher','active')`, f.student)
	var own, peer string
	if err := f.db.Raw(`SELECT attachment_id FROM submissions WHERE id=?`, f.own).Scan(&own).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Raw(`SELECT attachment_id FROM submissions WHERE id=?`, f.peer).Scan(&peer).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id   string
		want int
	}{{own, 200}, {peer, 403}} {
		before := f.files.calls
		code, body := f.request(t, f.student, http.MethodGet, "/api/attachments/"+tc.id+"/download", "")
		if code != tc.want {
			t.Fatalf("download %d want %d: %s", code, tc.want, body)
		}
		if tc.want == 403 && f.files.calls != before {
			t.Fatal("conflicting non-owner reached storage")
		}
	}
}

func TestConcurrentEnrollmentNoDuplicates(t *testing.T) {
	db := submissionTestDB(t)
	f := newSubmissionFixture(t, db)
	mustExec(t, f.db, `CREATE TABLE courses (id text PRIMARY KEY)`)
	mustExec(t, f.db, `INSERT INTO courses VALUES ('COURSA')`)
	actor := uuid.NewString()
	mustExec(t, f.db, `INSERT INTO users (id,username,role) VALUES (?,'new-student','student')`, actor)
	if err := f.db.Commit().Error; err != nil {
		t.Fatal(err)
	}
	database.DB = db
	done := make(chan bool, 2)
	fail := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			created, err := repository.EnrollCourseMember(&models.CourseMember{UserID: actor, CourseID: "COURSA", Role: "student", Status: "active"})
			done <- created
			fail <- err
		}()
	}
	count := 0
	for i := 0; i < 2; i++ {
		select {
		case created := <-done:
			if created {
				count++
			}
		case <-time.After(5 * time.Second):
			t.Fatal("enrollment timed out")
		}
		if err := <-fail; err != nil {
			t.Fatal(err)
		}
	}
	var rows int64
	if err := db.Model(&models.CourseMember{}).Where("course_id='COURSA' AND user_id=?", actor).Count(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 || rows != 1 {
		t.Fatalf("created=%d rows=%d", count, rows)
	}
	// Existing inactive enrollment is retained; this does not silently reactivate it.
	mustExec(t, db, `UPDATE course_members SET status='inactive' WHERE user_id=?`, actor)
	result, err := service.EnrollCourse("COURSA", &models.EnrollmentForms{UserEnroll: []models.EnrollmentForm{{Username: "new-student", CourseRole: "student"}}})
	if err != nil || len(result.EnrollmentResult) != 1 || result.EnrollmentResult[0].Status != "Already In Course" {
		t.Fatal("existing membership changed")
	}
}

func TestDuplicateOverrideUsesStableID(t *testing.T) {
	f := newSubmissionFixture(t, submissionTestDB(t))
	validationTables(t, f.db)
	low, high := "00000000-0000-4000-8000-000000000001", "ffffffff-ffff-4fff-bfff-ffffffffffff"
	first, second := time.Now().UTC().Truncate(time.Microsecond).Add(3*time.Hour), time.Now().UTC().Truncate(time.Microsecond).Add(4*time.Hour)
	mustExec(t, f.db, `INSERT INTO assignment_overrides (id,assignment_id,student_id,extended_due_date) VALUES (?,?,?,?), (?,?,?,?)`, high, f.a1, f.student, first, low, f.a1, f.student, second)
	selected, err := repository.GetAssignmentOverride(f.a1, f.student)
	if err != nil || selected.ID != high {
		t.Fatalf("override=%v err=%v", selected, err)
	}
	var before string
	if err := f.db.Raw(`SELECT row_to_json(o)::text FROM assignment_overrides o WHERE id=?`, low).Scan(&before).Error; err != nil {
		t.Fatal(err)
	}
	updated, err := service.CreateAssignmentOverride("COURSA", f.a1, models.AssignmentOverrideForm{StudentID: f.student, ExtendedDueDate: second.Add(time.Hour)})
	if err != nil || updated.ID != high {
		t.Fatalf("updated=%v error=%v", updated, err)
	}
	var after string
	if err := f.db.Raw(`SELECT row_to_json(o)::text FROM assignment_overrides o WHERE id=?`, low).Scan(&after).Error; err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("non-canonical extension changed")
	}
	data, err := service.GetAssignments(f.student, "COURSA", "student")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(data)
	if !strings.Contains(string(raw), updated.ExtendedDueDate.UTC().Format(time.RFC3339Nano)) {
		t.Fatalf("list used other extension: %s", raw)
	}
}

func TestConcurrentExtensionUsesOneRow(t *testing.T) {
	db := submissionTestDB(t)
	f := newSubmissionFixture(t, db)
	if err := f.db.Commit().Error; err != nil {
		t.Fatal(err)
	}
	database.DB = db
	type result struct {
		id  string
		err error
	}
	done := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func(hours int) {
			row, err := service.CreateAssignmentOverride("COURSA", f.a1, models.AssignmentOverrideForm{StudentID: f.student, ExtendedDueDate: time.Now().Add(time.Duration(hours) * time.Hour)})
			id := ""
			if row != nil {
				id = row.ID
			}
			done <- result{id, err}
		}(i + 3)
	}
	ids := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case value := <-done:
			if value.err != nil {
				t.Fatal(value.err)
			}
			ids[value.id] = true
		case <-time.After(5 * time.Second):
			t.Fatal("extension timed out")
		}
	}
	var count int64
	if err := db.Model(&models.AssignmentOverride{}).Where("assignment_id=? AND student_id=?", f.a1, f.student).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(ids) != 1 {
		t.Fatalf("extension rows=%d identities=%d", count, len(ids))
	}
}

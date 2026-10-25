package handler_test

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/google/uuid"
)

func TestSubmissionScheduleHTTP(t *testing.T) {
	db := submissionTestDB(t)
	for _, action := range []string{"create", "update"} {
		for _, policy := range []struct {
			name                         string
			start, due, close, extension time.Duration
			want                         int
		}{
			{"before start", time.Hour, 2 * time.Hour, 3 * time.Hour, 0, 403},
			{"open", -time.Hour, time.Hour, 2 * time.Hour, 0, 200},
			{"after close", -3 * time.Hour, -2 * time.Hour, -time.Hour, 0, 403},
			{"extension does not shorten close", -3 * time.Hour, -2 * time.Hour, time.Hour, -time.Hour, 200},
			{"extension reopens close", -3 * time.Hour, -2 * time.Hour, -time.Hour, time.Hour, 200},
		} {
			t.Run(action+"/"+policy.name, func(t *testing.T) {
				f := newSubmissionFixture(t, db)
				now := time.Now()
				mustExec(t, f.db, `UPDATE assignments SET start_date=?, due_date=?, close_date=? WHERE id=?`, now.Add(policy.start), now.Add(policy.due), now.Add(policy.close), f.a1)
				if policy.extension != 0 {
					mustExec(t, f.db, `INSERT INTO assignment_overrides (assignment_id,student_id,extended_due_date) VALUES (?,?,?)`, f.a1, f.student, now.Add(policy.extension))
				}
				path := "/api/courses/COURSA/assignments/" + f.a1 + "/submissions"
				method := http.MethodPost
				if action == "update" {
					method = http.MethodPut
					path += "/" + f.own
				}
				before := f.snapshot(t)
				status, body := f.request(t, f.student, method, path, "update")
				if (policy.want == 200 && status != 200 && status != 201) || (policy.want != 200 && status != policy.want) {
					t.Fatalf("status %d, want %d: %s", status, policy.want, body)
				}
				if policy.want != 200 && (f.snapshot(t) != before || f.files.calls != 0) {
					t.Fatal("closed window changed data/storage")
				}
			})
		}
	}
}

func TestSubmissionOverrideLookupFailure(t *testing.T) {
	db := submissionTestDB(t)
	for _, update := range []bool{false, true} {
		t.Run(map[bool]string{false: "create", true: "update"}[update], func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			mustExec(t, f.db, `ALTER TABLE assignment_overrides RENAME TO unavailable_overrides`)
			before := f.snapshot(t)
			// Savepoint keeps the intentional query error from aborting snapshots.
			mustExec(t, f.db, `SAVEPOINT unavailable_override`)
			path := "/api/courses/COURSA/assignments/" + f.a1 + "/submissions"
			method := http.MethodPost
			if update {
				method = http.MethodPut
				path += "/" + f.own
			}
			status, _ := f.request(t, f.student, method, path, "update")
			mustExec(t, f.db, `ROLLBACK TO SAVEPOINT unavailable_override`)
			if status != 500 || f.snapshot(t) != before || f.files.calls != 0 {
				t.Fatalf("lookup failure was not closed: status %d, storage calls %d", status, f.files.calls)
			}
		})
	}
}

func TestSummaryUploadReplacementAndGradingHTTP(t *testing.T) {
	db := submissionTestDB(t)
	f := newSubmissionFixture(t, db)
	due := time.Now().Add(-2 * time.Hour).Truncate(time.Microsecond)
	mustExec(t, f.db, `UPDATE assignments SET start_date=?,due_date=?,close_date=? WHERE id=?`, due.Add(-time.Hour), due, time.Now().Add(time.Hour), f.a1)
	mustExec(t, f.db, `UPDATE attachments SET created_at=? WHERE id=(SELECT attachment_id FROM submissions WHERE id=?)`, due, f.own)
	mustExec(t, f.db, `UPDATE submissions SET point=3,updated_at=? WHERE id=?`, time.Now().Add(time.Hour), f.own)
	check := func(want string, point int16) {
		t.Helper()
		status, body := f.request(t, f.teacher, http.MethodGet, "/api/courses/COURSA/assignments/"+f.a1+"/summary", "")
		if status != 200 {
			t.Fatalf("summary %d: %s", status, body)
		}
		var summary models.ResponseAssignmentSummary
		if err := json.Unmarshal(body, &summary); err != nil {
			t.Fatal(err)
		}
		for _, row := range summary.SubmissionList {
			if row.UserID == f.student {
				if row.SubmissionStatus != want || row.Point != point {
					t.Fatalf("row %+v, want %s, %d", row, want, point)
				}
				return
			}
		}
		t.Fatal("student missing from summary")
	}
	check("submitted", 3) // exact due, despite later UpdatedAt
	status, body := f.request(t, f.teacher, http.MethodPut, submissionPath("COURSA", f.a1, f.own, "/grade"), "grade")
	if status != 200 {
		t.Fatalf("grade %d: %s", status, body)
	}
	check("submitted", 7)
	// Simulate the score/grader/timestamp changes of an AI write independently
	// of the worker tests. No grading write may change current file time.
	mustExec(t, f.db, `UPDATE submissions SET graded_by='ai',point=8,updated_at=? WHERE id=?`, time.Now().Add(2*time.Hour), f.own)
	check("submitted", 8)
	mustExec(t, f.db, `INSERT INTO assignment_overrides (assignment_id,student_id,extended_due_date) VALUES (?,?,?)`, f.a1, f.student, due.Add(time.Hour))
	// Add a later file-bearing attempt, with a deliberately earlier row UpdatedAt.
	attachmentID, attemptID := uuid.NewString(), uuid.NewString()
	mustExec(t, f.db, `INSERT INTO attachments (id,created_at) VALUES (?,?)`, attachmentID, due.Add(time.Hour))
	mustExec(t, f.db, `INSERT INTO submissions (id,assignment_id,student_id,attachment_id,point,updated_at) VALUES (?,?,?,?,9,?)`, attemptID, f.a1, f.student, attachmentID, due.Add(-time.Hour))
	check("submitted", 9) // exact personal extended due
	// Replacing the older attempt makes its current file the latest.
	status, body = f.request(t, f.student, http.MethodPut, submissionPath("COURSA", f.a1, f.own, ""), "update")
	if status != 200 {
		t.Fatalf("replace %d: %s", status, body)
	}
	check("overdue", 8)
	// Summary and single-assignment CSV share the same timing/selection policy.
	csv, err := service.GetAssignmentExport("COURSA", f.a1)
	if err != nil || !strings.Contains(string(csv), "overdue") {
		t.Fatalf("CSV export failed: %v", err)
	}
	loaded, err := repository.GetAssignmentSubmissions(f.a1)
	if err != nil {
		t.Fatal(err)
	}
	selected := models.LatestFileSubmissions(loaded)[f.student]
	if selected.ID != f.own || models.CurrentFileTime(selected).IsZero() {
		t.Fatal("replacement timestamp not loaded")
	}
}

func TestAssignmentScheduleAPIAndEditRoundTrip(t *testing.T) {
	db := submissionTestDB(t)
	f := newSubmissionFixture(t, db)
	for _, ddl := range []string{
		`CREATE TABLE tags (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text)`,
		`CREATE TABLE assignment_tags (assignment_id uuid, tag_id uuid)`,
		`CREATE TABLE assignment_attachments (assignment_id uuid, attachment_id uuid)`,
	} {
		mustExec(t, f.db, ddl)
	}
	start := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	due, close := start.Add(2*time.Hour), start.Add(4*time.Hour)
	mustExec(t, f.db, `UPDATE assignments SET start_date=?,due_date=?,close_date=? WHERE id=?`, start, due, close, f.a1)
	path := "/api/courses/COURSA/assignments/" + f.a1
	check := func(actor string, wantDue, wantClose time.Time) models.ResponseAssignment {
		t.Helper()
		status, body := f.request(t, actor, http.MethodGet, path, "")
		if status != 200 {
			t.Fatalf("detail %d: %s", status, body)
		}
		var response models.ResponseDetailedAssignment
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatal(err)
		}
		want := []time.Time{start, wantDue, wantClose}
		for i, got := range []string{response.Assignment.StartDate, response.Assignment.DueDate, response.Assignment.CloseDate} {
			if got != want[i].Format(time.RFC3339Nano) {
				t.Fatalf("timestamp %q, want %v", got, want[i])
			}
		}
		return response.Assignment
	}
	response := check(f.teacher, due, close)
	parsedStart, _ := time.Parse(time.RFC3339Nano, response.StartDate)
	parsedDue, _ := time.Parse(time.RFC3339Nano, response.DueDate)
	parsedClose, _ := time.Parse(time.RFC3339Nano, response.CloseDate)
	if _, err := service.UpdateAssignment("COURSA", f.a1, f.teacher, &models.AssignmentForm{
		StartDate: &parsedStart, DueDate: &parsedDue, CloseDate: &parsedClose, Visible: true,
	}, nil); err != nil {
		t.Fatal(err)
	}
	check(f.teacher, due, close)
	for _, extended := range []time.Time{due.Add(time.Hour), close.Add(time.Hour)} {
		mustExec(t, f.db, `DELETE FROM assignment_overrides WHERE assignment_id=? AND student_id=?`, f.a1, f.student)
		mustExec(t, f.db, `INSERT INTO assignment_overrides (assignment_id,student_id,extended_due_date) VALUES (?,?,?)`, f.a1, f.student, extended)
		wantClose := close
		if extended.After(close) {
			wantClose = extended
		}
		check(f.student, extended, wantClose)
		status, body := f.request(t, f.student, http.MethodGet, "/api/courses/COURSA/assignments", "")
		if status != 200 {
			t.Fatalf("list %d: %s", status, body)
		}
		var list struct {
			Assignments []models.ResponseAssignment `json:"assignments"`
		}
		if err := json.Unmarshal(body, &list); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, a := range list.Assignments {
			if a.ID == f.a1 {
				found = true
				if a.DueDate != extended.Format(time.RFC3339Nano) || a.CloseDate != wantClose.Format(time.RFC3339Nano) {
					t.Fatalf("list timestamps %+v", a)
				}
			}
		}
		if !found {
			t.Fatal("assignment missing from student list")
		}
	}

	// The course CSV must select the same current file-bearing attempt as the
	// summary, even if another row was graded/updated more recently.
	mustExec(t, f.db, `UPDATE assignments SET title='R8 CSV' WHERE id=?`, f.a1)
	mustExec(t, f.db, `UPDATE attachments SET created_at=? WHERE id=(SELECT attachment_id FROM submissions WHERE id=?)`, due, f.own)
	mustExec(t, f.db, `UPDATE submissions SET point=8 WHERE id=?`, f.own)
	oldFile := uuid.NewString()
	mustExec(t, f.db, `INSERT INTO attachments (id,created_at) VALUES (?,?)`, oldFile, start)
	mustExec(t, f.db, `INSERT INTO submissions (assignment_id,student_id,attachment_id,point,updated_at) VALUES (?,?,?,3,?)`, f.a1, f.student, oldFile, close.Add(time.Hour))
	mustExec(t, f.db, `INSERT INTO submissions (assignment_id,student_id,point,updated_at) VALUES (?,?,10,?)`, f.a1, f.student, close.Add(2*time.Hour))
	data, err := service.GetAssignmentsExport("COURSA")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	column := -1
	for i, header := range rows[0] {
		if header == "R8 CSV (10)" {
			column = i
		}
	}
	if column < 0 {
		t.Fatal("assignment missing from CSV")
	}
	for _, row := range rows[1:] {
		if row[0] == f.student {
			if row[column] != "8" {
				t.Fatalf("CSV score %s, want latest current file score 8", row[column])
			}
			return
		}
	}
	t.Fatal("student missing from course CSV")
}

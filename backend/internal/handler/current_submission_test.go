package handler_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
	"github.com/google/uuid"
)

func historyRecord(t *testing.T, f *submissionFixture, at time.Time) string {
	t.Helper()
	id, attachment := uuid.NewString(), uuid.NewString()
	f.files.data[id] = []byte("historical answer")
	mustExec(t, f.db, `INSERT INTO attachments (id,file_key,file_name,user_id,created_at) VALUES (?,?,'history.py',?,?)`, attachment, id, f.student, time.Now().Add(time.Hour))
	mustExec(t, f.db, `INSERT INTO submissions (id,assignment_id,student_id,attachment_id,point,graded_by,created_at) VALUES (?,?,?,?,9,'ai',?)`, id, f.a1, f.student, attachment, at)
	return id
}

func TestCurrentSubmissionPostPreservesIdentity(t *testing.T) {
	f := newSubmissionFixture(t, submissionTestDB(t))
	mustExec(t, f.db, `UPDATE submissions SET point=7,graded_by='teacher' WHERE id=?`, f.own)
	mustExec(t, f.db, `INSERT INTO comments (submission_id,comment,created_by_role,created_by,visible) VALUES (?,'hidden feedback','ai',?,false)`, f.own, f.teacher)
	for i := 0; i < 2; i++ {
		code, body := f.request(t, f.student, "POST", "/api/courses/COURSA/assignments/"+f.a1+"/submissions", "update")
		if code != 200 {
			t.Fatalf("POST %d: %s", code, body)
		}
		var result models.ResponseSubmission
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatal(err)
		}
		if result.ID != f.own || !result.IsCurrent || result.Point == nil || *result.Point != 7 || len(result.Comments) != 1 || result.Comments[0].ID != f.comment {
			t.Fatalf("lost submission state: %+v", result)
		}
		var count int64
		if err := f.db.Model(&models.Submission{}).Where("assignment_id=? AND student_id=?", f.a1, f.student).Count(&count).Error; err != nil || count != 1 {
			t.Fatal("POST inserted a second attempt")
		}
	}
	if _, ok := f.files.data[f.own]; !ok {
		t.Fatal("replacement removed original bytes")
	}
}

func TestCurrentHistoryReadAndReports(t *testing.T) {
	f := newSubmissionFixture(t, submissionTestDB(t))
	validationTables(t, f.db)
	history := historyRecord(t, f, time.Now().Add(-time.Hour))
	path := "/api/courses/COURSA/assignments/" + f.a1
	code, body := f.request(t, f.student, "GET", path, "")
	if code != 200 {
		t.Fatalf("detail %d: %s", code, body)
	}
	var detail models.ResponseDetailedAssignment
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Submissions) != 2 || detail.Submissions[0].ID != f.own || !detail.Submissions[0].IsCurrent || detail.Submissions[1].IsCurrent {
		t.Fatal("detail default/history flags are wrong")
	}
	before := f.files.calls
	code, body = f.request(t, f.student, "PUT", submissionPath("COURSA", f.a1, history, ""), "update")
	if code != 409 || f.files.calls != before {
		t.Fatalf("history edit reached storage: %d %s", code, body)
	}
	code, body = f.request(t, f.student, "GET", submissionPath("COURSA", f.a1, history, "/read"), "")
	if code != 200 || !strings.Contains(string(body), "historical answer") {
		t.Fatal("history is not readable")
	}
	code, body = f.request(t, f.student, "PUT", submissionPath("COURSA", f.a1, f.own, ""), "")
	if code != 200 {
		t.Fatalf("detach %d: %s", code, body)
	}
	summary, err := service.GetAssignmentSummary("COURSA", f.a1)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Statistic.Submitted != 1 || summary.Statistic.Incomplete != 1 || summary.Statistic.NotStarted != 1 {
		t.Fatalf("wrong categories: %+v", summary.Statistic)
	}
	for _, row := range summary.SubmissionList {
		if row.UserID == f.student && (row.SubmissionStatus != "incomplete" || row.HasGrade) {
			t.Fatal("history revived in summary")
		}
	}
	for _, course := range []bool{false, true} {
		var data []byte
		if course {
			data, err = service.GetAssignmentsExport("COURSA")
		} else {
			data, err = service.GetAssignmentExport("COURSA", f.a1)
		}
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if row[0] == f.student && row[4] != "" {
				t.Fatalf("history grade in export: %v", row)
			}
		}
	}
	archive, err := service.DownloadSubmissions("COURSA", f.a1)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 1 || strings.Contains(reader.File[0].Name, "history.py") {
		t.Fatal("bulk export included a historical file")
	}
	eligible, err := repository.GetSubmissions(context.Background(), f.a1)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range eligible {
		if row.StudentID == f.student {
			t.Fatal("AI selected detached current or historical file")
		}
	}
	prior := f.snapshot(t)
	if applied, err := repository.SaveAIGradeAndComment(f.a1, history, f.teacher, 2, "must not replace history", true); err != nil || applied {
		t.Fatalf("AI changed history: applied=%v err=%v", applied, err)
	}
	if f.snapshot(t) != prior {
		t.Fatal("historical feedback/grade changed")
	}
}

func TestPostUsesLatestCreatedLegacyRecord(t *testing.T) {
	f := newSubmissionFixture(t, submissionTestDB(t))
	canonical := historyRecord(t, f, time.Now().Add(time.Hour))
	before := f.snapshot(t)
	code, body := f.request(t, f.student, "POST", "/api/courses/COURSA/assignments/"+f.a1+"/submissions", "update")
	if code != 200 {
		t.Fatalf("POST %d: %s", code, body)
	}
	var result models.ResponseSubmission
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	if result.ID != canonical || result.Point == nil || *result.Point != 9 {
		t.Fatal("POST did not reuse canonical legacy record")
	}
	if f.snapshot(t) == before {
		t.Fatal("POST did not replace the file")
	}
	var count int64
	if err := f.db.Model(&models.Submission{}).Where("assignment_id=? AND student_id=?", f.a1, f.student).Count(&count).Error; err != nil || count != 2 {
		t.Fatal("history was deleted or new attempt inserted")
	}
}

func TestConcurrentInitialUploadsReuseOneRecord(t *testing.T) {
	db := submissionTestDB(t)
	f := newSubmissionFixture(t, db)
	files := &replacementFiles{data: map[string][]byte{}}
	for key, value := range f.files.data {
		files.data[key] = value
	}
	storage.Provider = files
	if err := f.db.Commit().Error; err != nil {
		t.Fatal(err)
	}
	database.DB = db
	first, second := replacementFile(t, "first"), replacementFile(t, "second")
	type result struct {
		id  string
		err error
	}
	done := make(chan result, 2)
	for _, file := range []*multipart.FileHeader{first, second} {
		go func(file *multipart.FileHeader) {
			row, err := service.CreateSubmission("COURSA", f.a1, f.teacherAsStudent, file)
			id := ""
			if row != nil {
				id = row.ID
			}
			done <- result{id, err}
		}(file)
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
			t.Fatal("concurrent upload timed out")
		}
	}
	var count int64
	if err := db.Model(&models.Submission{}).Where("assignment_id=? AND student_id=?", f.a1, f.teacherAsStudent).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || count != 1 || files.deletes != 0 {
		t.Fatalf("identities=%d rows=%d deletes=%d", len(ids), count, files.deletes)
	}
}

func TestCurrentSelectionSQLMatchesModel(t *testing.T) {
	db := submissionTestDB(t)
	for _, name := range []string{"known-time", "tied-time", "unknown-time", "detached"} {
		t.Run(name, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			id := historyRecord(t, f, time.Now().Add(-time.Hour))
			switch name {
			case "tied-time":
				mustExec(t, f.db, `UPDATE submissions SET created_at=? WHERE assignment_id=? AND student_id=?`, time.Now(), f.a1, f.student)
			case "unknown-time":
				mustExec(t, f.db, `UPDATE submissions SET created_at=NULL WHERE assignment_id=? AND student_id=?`, f.a1, f.student)
			case "detached":
				mustExec(t, f.db, `UPDATE submissions SET created_at=?,attachment_id=NULL WHERE id=?`, time.Now().Add(time.Hour), id)
			}
			rows, err := repository.GetAssignmentSubmissions(f.a1)
			if err != nil {
				t.Fatal(err)
			}
			current := models.CurrentSubmissions(rows)[f.student]
			eligible, err := repository.GetSubmissions(context.Background(), f.a1)
			if err != nil {
				t.Fatal(err)
			}
			found := ""
			for _, row := range eligible {
				if row.StudentID == f.student {
					found = row.ID
				}
			}
			want := ""
			if current.AttachmentID != nil {
				want = current.ID
			}
			if found != want {
				t.Fatalf("SQL selected %s, model selected %s with file %v", found, current.ID, current.AttachmentID)
			}
		})
	}
}

func TestCourseExportDoesNotOverflowMaximums(t *testing.T) {
	f := newSubmissionFixture(t, submissionTestDB(t))
	validationTables(t, f.db)
	mustExec(t, f.db, `UPDATE assignments SET point=32767 WHERE course_id='COURSA'`)
	mustExec(t, f.db, `UPDATE submissions SET point=32767,graded_by='teacher' WHERE id=?`, f.own)
	mustExec(t, f.db, `INSERT INTO submissions (id,assignment_id,student_id,attachment_id,point,graded_by,created_at) SELECT ?,?,student_id,attachment_id,32767,'teacher',? FROM submissions WHERE id=?`, uuid.NewString(), f.a2, time.Now(), f.own)
	data, err := service.GetAssignmentsExport("COURSA")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rows[0][len(rows[0])-2], "65534") {
		t.Fatalf("maximum overflow: %v", rows[0])
	}
	for _, row := range rows[1:] {
		if row[0] == f.student && row[len(row)-2] != "65534" {
			t.Fatalf("score overflow: %v", row)
		}
	}
	mustExec(t, f.db, `UPDATE assignments SET updated_at=? WHERE id=?`, time.Now(), f.a1)
	again, err := service.GetAssignmentsExport("COURSA")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Fatal("CSV order changed after a non-reporting update")
	}
}

func TestSubmissionAdmissionRemainsBeforeUpload(t *testing.T) {
	for _, post := range []bool{false, true} {
		t.Run(map[bool]string{false: "PUT", true: "POST"}[post], func(t *testing.T) {
			f := newSubmissionFixture(t, submissionTestDB(t))
			files := &replacementFiles{data: map[string][]byte{}}
			for key, value := range f.files.data {
				files.data[key] = value
			}
			files.onUpload = func() {
				mustExec(t, f.db, `UPDATE assignments SET due_date=?,close_date=? WHERE id=?`, time.Now().Add(-2*time.Minute), time.Now().Add(-time.Minute), f.a1)
			}
			storage.Provider = files
			var row *models.ResponseSubmission
			var err error
			if post {
				row, err = service.CreateSubmission("COURSA", f.a1, f.student, replacementFile(t, "accepted before upload"))
			} else {
				row, err = service.UpdateSubmission(f.own, f.student, replacementFile(t, "accepted before upload"))
			}
			if err != nil || row == nil || row.ID != f.own {
				t.Fatalf("changed admission boundary: row=%v error=%v", row, err)
			}
		})
	}
}

package handler_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/router"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// This suite uses only an explicitly supplied disposable PostgreSQL database.
// It never calls application startup or AutoMigrate. Do not run in parallel:
// production packages currently expose DB, logger and storage as globals.
func submissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEACHING_SUPPORT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEACHING_SUPPORT_TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") ||
		!strings.HasPrefix(strings.TrimPrefix(u.Path, "/"), "teaching_support_test") {
		t.Fatal("test URL must name a dedicated teaching_support_test* PostgreSQL database")
	}
	open := func(dsn string) *gorm.DB {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
		if err != nil {
			t.Fatal("cannot connect to test database") // Never print credentials.
		}
		conn, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		return db
	}
	admin := open(dsn)
	schema := "r1_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	mustExec(t, admin, `CREATE SCHEMA "`+schema+`"`)
	t.Cleanup(func() { mustExec(t, admin, `DROP SCHEMA "`+schema+`" CASCADE`) })
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db := open(u.String())
	for _, ddl := range []string{
		`CREATE TABLE users (id uuid PRIMARY KEY, username text, email text, th_name text, en_name text, user_type text, role text, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE course_members (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), user_id uuid, course_id text, role text, status text, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE assignments (id uuid PRIMARY KEY, course_id text, title text, description text, point smallint, start_date timestamptz, due_date timestamptz, close_date timestamptz, visible boolean, ai_config_id uuid, prompt text, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE attachments (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), file_key text, file_name text, file_type text, size bigint, user_id uuid, created_at timestamptz)`,
		`CREATE TABLE submissions (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), assignment_id uuid, student_id uuid, point smallint, graded_by text, attachment_id uuid, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE assignment_overrides (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), assignment_id uuid, student_id uuid, extended_due_date timestamptz)`,
		`CREATE TABLE comments (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), submission_id uuid, comment text, created_by_role text, created_by text, visible boolean, created_at timestamptz, updated_at timestamptz)`,
	} {
		mustExec(t, db, ddl)
	}
	oldDB, oldStorage, oldLog := database.DB, storage.Provider, logger.Log
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Cleanup(func() { database.DB, storage.Provider, logger.Log = oldDB, oldStorage, oldLog })
	return db
}

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
}

type submissionFixture struct {
	db                                                                     *gorm.DB
	app                                                                    *fiber.App
	files                                                                  *submissionFiles
	teacher, student, other, teacherAsStudent, studentAsTeacher, ta, admin string
	a1, a2, b, own, peer, foreign, comment, foreignComment                 string
}

func newSubmissionFixture(t *testing.T, db *gorm.DB) *submissionFixture {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	t.Cleanup(func() { _ = tx.Rollback().Error })
	f := &submissionFixture{db: tx, files: &submissionFiles{data: map[string][]byte{}}}
	database.DB, storage.Provider = tx, f.files
	actor := func(globalRole, courseRole string) string {
		id := uuid.NewString()
		mustExec(t, tx, `INSERT INTO users (id, username, role) VALUES (?, ?, ?)`, id, id, globalRole)
		mustExec(t, tx, `INSERT INTO course_members (user_id, course_id, role, status) VALUES (?, 'COURSA', ?, 'active')`, id, courseRole)
		return id
	}
	f.teacher = actor("teacher", "teacher")
	f.student = actor("student", "student")
	f.other = actor("student", "student")
	f.teacherAsStudent = actor("teacher", "student")
	f.studentAsTeacher = actor("student", "teacher")
	f.ta = actor("teacher", "ta")
	f.admin = actor("admin", "admin")
	mustExec(t, tx, `INSERT INTO course_members (user_id, course_id, role, status) VALUES (?, 'COURSB', 'student', 'inactive')`, f.student)
	assignment := func(course string) string {
		id := uuid.NewString()
		mustExec(t, tx, `INSERT INTO assignments (id, course_id, title, point, start_date, due_date, close_date, visible) VALUES (?, ?, 'Exercise', 10, ?, ?, ?, true)`, id, course, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
		return id
	}
	f.a1, f.a2, f.b = assignment("COURSA"), assignment("COURSA"), assignment("COURSB")
	submission := func(assignment, owner string) string {
		id, attachment := uuid.NewString(), uuid.NewString()
		f.files.data[id] = []byte("answer: " + id)
		mustExec(t, tx, `INSERT INTO attachments (id, file_key, file_name, user_id) VALUES (?, ?, 'answer.py', ?)`, attachment, id, owner)
		mustExec(t, tx, `INSERT INTO submissions (id, assignment_id, student_id, attachment_id) VALUES (?, ?, ?, ?)`, id, assignment, owner, attachment)
		return id
	}
	f.own, f.peer, f.foreign = submission(f.a1, f.student), submission(f.a1, f.other), submission(f.b, f.student)
	for sid, cid := range map[string]*string{f.own: &f.comment, f.foreign: &f.foreignComment} {
		*cid = uuid.NewString()
		mustExec(t, tx, `INSERT INTO comments (id, submission_id, comment, created_by_role, created_by, visible) VALUES (?, ?, 'original', 'teacher', ?, true)`, *cid, sid, f.teacher)
	}
	f.app = fiber.New()
	router.SetupRoutes(f.app)
	return f
}

type submissionFiles struct {
	data  map[string][]byte
	calls int
}

func (s *submissionFiles) Upload(_ context.Context, data []byte, _ string) (string, error) {
	s.calls++
	key := uuid.NewString()
	s.data[key] = append([]byte(nil), data...)
	return key, nil
}
func (s *submissionFiles) ReadText(_ context.Context, key string) (string, error) {
	s.calls++
	data, ok := s.data[key]
	if !ok {
		return "", fmt.Errorf("missing fixture file")
	}
	return string(data), nil
}
func (s *submissionFiles) Download(ctx context.Context, key string) ([]byte, error) {
	data, err := s.ReadText(ctx, key)
	return []byte(data), err
}
func (s *submissionFiles) Delete(_ context.Context, key string) error {
	s.calls++
	delete(s.data, key)
	return nil
}

func (f *submissionFixture) request(t *testing.T, actor, method, path, action string) (int, []byte) {
	t.Helper()
	var body bytes.Buffer
	contentType := "application/json"
	if action == "update" {
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "replacement.py")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(part, "replacement answer"); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		contentType = writer.FormDataContentType()
	} else {
		body.WriteString(`{"point":7,"comment":"feedback","visible":true}`)
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", contentType)
	if actor != "" {
		token, err := service.GenerateToken(actor)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := f.app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second, FailOnTimeout: true})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, data
}

func submissionPath(course, assignment, submission, suffix string) string {
	return "/api/courses/" + course + "/assignments/" + assignment + "/submissions/" + submission + suffix
}

// Snapshot all mutable records and stored bytes, including timestamps. A denied
// request must not just fail: it must leave the original answer/grade intact.
func (f *submissionFixture) snapshot(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	for _, table := range []string{"submissions", "attachments", "comments"} {
		var rows string
		err := f.db.Raw("SELECT COALESCE(json_agg(r ORDER BY id)::text, '[]') FROM " + table + " r").Scan(&rows).Error
		if err != nil {
			t.Fatal(err)
		}
		out.WriteString(rows)
	}
	data, err := json.Marshal(f.files.data)
	if err != nil {
		t.Fatal(err)
	}
	out.Write(data)
	return out.String()
}

func TestSubmissionResourceScope(t *testing.T) {
	db := submissionTestDB(t)
	for _, action := range []struct{ name, method, suffix string }{
		{"read", http.MethodGet, "/read"},
		{"update", http.MethodPut, ""},
		{"grade", http.MethodPut, "/grade"},
		{"grade-comment", http.MethodPut, "/grade-with-comment"},
		{"comment", http.MethodPost, "/comment"},
		{"toggle", http.MethodPost, "/comment/"},
	} {
		for _, mismatch := range []string{"same-course-assignment", "foreign-submission", "foreign-assignment", "foreign-hierarchy", "missing-assignment", "missing-submission", "inactive", "withdraw", "drop", "no-membership"} {
			t.Run(action.name+"/"+mismatch, func(t *testing.T) {
				f := newSubmissionFixture(t, db)
				assignment, sid, cid := f.a1, f.own, f.comment
				switch mismatch {
				case "same-course-assignment":
					assignment = f.a2
				case "foreign-submission":
					sid, cid = f.foreign, f.foreignComment
				case "foreign-assignment":
					assignment = f.b
				case "foreign-hierarchy":
					assignment, sid, cid = f.b, f.foreign, f.foreignComment
				case "missing-assignment":
					assignment = uuid.NewString()
				case "missing-submission":
					sid = uuid.NewString()
				}
				suffix := action.suffix
				if action.name == "toggle" {
					suffix += cid
				}
				actor := f.teacher
				if action.name == "update" {
					actor = f.student
				}
				want := 404
				switch mismatch {
				case "inactive", "withdraw", "drop":
					mustExec(t, f.db, `UPDATE course_members SET status = ? WHERE user_id = ?`, mismatch, actor)
					want = 403
				case "no-membership":
					mustExec(t, f.db, `DELETE FROM course_members WHERE user_id = ?`, actor)
					want = 403
				}
				before := f.snapshot(t)
				status, body := f.request(t, actor, action.method, submissionPath("COURSA", assignment, sid, suffix), action.name)
				if status != want {
					t.Errorf("status = %d, want %d: %s", status, want, body)
				}
				if f.snapshot(t) != before || f.files.calls != 0 {
					t.Error("rejected request accessed storage or changed records/bytes")
				}
			})
		}
	}
}

func TestSubmissionAccessControls(t *testing.T) {
	db := submissionTestDB(t)
	for _, scenario := range []string{"teacher-read", "owner-read", "owner-update", "teacher-grade", "teacher-grade-comment", "teacher-comment", "teacher-toggle", "global-teacher-course-student", "global-student-course-teacher", "student-other", "student-grade", "teacher-update", "ta", "admin", "inactive", "withdraw", "drop", "no-membership", "no-auth", "uppercase-ids", "create", "download"} {
		t.Run(scenario, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			actor, assignment, sid, method, suffix, action, want := f.teacher, f.a1, f.own, http.MethodGet, "/read", "read", 200
			switch scenario {
			case "owner-read":
				actor = f.student
			case "owner-update":
				actor, method, suffix, action = f.student, http.MethodPut, "", "update"
			case "teacher-grade":
				method, suffix = http.MethodPut, "/grade"
			case "teacher-grade-comment":
				method, suffix = http.MethodPut, "/grade-with-comment"
			case "teacher-comment":
				method, suffix = http.MethodPost, "/comment"
			case "teacher-toggle":
				method, suffix = http.MethodPost, "/comment/"+f.comment
			case "global-teacher-course-student":
				actor, want = f.teacherAsStudent, 403
			case "global-student-course-teacher":
				actor = f.studentAsTeacher
			case "student-other":
				actor, sid, want = f.student, f.peer, 403
			case "student-grade":
				actor, method, suffix, want = f.student, http.MethodPut, "/grade", 403
			case "teacher-update":
				method, suffix, action, want = http.MethodPut, "", "update", 403
			case "ta":
				actor, want = f.ta, 403
			case "admin":
				actor, want = f.admin, 403
			case "inactive", "withdraw", "drop":
				mustExec(t, f.db, `UPDATE course_members SET status = ? WHERE user_id = ?`, scenario, f.teacher)
				want = 403
			case "no-membership":
				mustExec(t, f.db, `DELETE FROM course_members WHERE user_id = ?`, f.teacher)
				want = 403
			case "no-auth":
				actor, want = "", 401
			case "uppercase-ids":
				assignment, sid = strings.ToUpper(assignment), strings.ToUpper(sid)
			case "create":
				actor, sid, suffix, method = f.student, "", "", http.MethodPost
			case "download":
				sid, suffix = "download", ""
			}
			before := f.snapshot(t)
			status, body := f.request(t, actor, method, submissionPath("COURSA", assignment, sid, suffix), action)
			if status != want {
				t.Fatalf("status = %d, want %d: %s", status, want, body)
			}
			if want != 200 {
				if f.snapshot(t) != before || f.files.calls != 0 {
					t.Error("denied access changed records/bytes or accessed storage")
				}
				return
			}
			switch scenario {
			case "teacher-read", "owner-read", "global-student-course-teacher", "uppercase-ids":
				var content string
				if err := json.Unmarshal(body, &content); err != nil || content != "answer: "+f.own {
					t.Errorf("wrong answer: %s", body)
				}
			case "owner-update":
				var submission models.Submission
				if err := f.db.Preload("Attachment").First(&submission, "id = ?", f.own).Error; err != nil {
					t.Fatal(err)
				}
				if submission.Attachment == nil || string(f.files.data[submission.Attachment.FileKey]) != "replacement answer" {
					t.Error("replacement not stored")
				}
			case "teacher-grade", "teacher-grade-comment":
				var submission models.Submission
				if err := f.db.First(&submission, "id = ?", f.own).Error; err != nil {
					t.Fatal(err)
				}
				if submission.Point == nil || *submission.Point != 7 || submission.GradedBy == nil || *submission.GradedBy != "teacher" {
					t.Error("grade not stored")
				}
			case "teacher-comment", "teacher-toggle":
				var comment models.Comment
				if err := f.db.First(&comment, "id = ?", f.comment).Error; err != nil {
					t.Fatal(err)
				}
				if scenario == "teacher-comment" && comment.Comment != "feedback" {
					t.Error("comment not updated")
				}
				if scenario == "teacher-toggle" && comment.Visible {
					t.Error("comment not toggled")
				}
			case "create":
				var count int64
				if err := f.db.Model(&models.Submission{}).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 4 {
					t.Errorf("submission count = %d, want 4", count)
				}
			case "download":
				archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
				if err != nil || len(archive.File) != 2 {
					t.Error("assignment download did not return both files")
				}
			}
		})
	}
}

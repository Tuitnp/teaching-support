package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	apperrors "github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func validationTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	aiConfigTables(t, db)
	for _, sql := range []string{
		`ALTER TABLE assignments ALTER COLUMN id SET DEFAULT gen_random_uuid()`,
		`CREATE TABLE tags (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), name text UNIQUE, created_at timestamptz)`,
		`CREATE TABLE assignment_tags (assignment_id uuid, tag_id uuid)`,
		`CREATE TABLE assignment_attachments (assignment_id uuid, attachment_id uuid)`,
		`CREATE TABLE assignment_templates (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), title text, description text, point smallint, ai_config_id uuid, prompt text, created_by uuid, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE assignment_template_tags (assignment_template_id uuid, tag_id uuid)`,
		`CREATE TABLE assignment_template_attachments (assignment_template_id uuid, attachment_id uuid)`,
	} {
		mustExec(t, db, sql)
	}
}

func validationSnapshot(t *testing.T, f *submissionFixture) string {
	t.Helper()
	result := f.snapshot(t)
	for _, table := range []string{"assignments", "tags", "assignment_tags", "assignment_attachments", "assignment_templates", "assignment_template_tags", "assignment_template_attachments", "ai_configs", "ai_credentials"} {
		var value string
		if err := f.db.Raw("SELECT COALESCE(jsonb_agg(to_jsonb(r) ORDER BY to_jsonb(r)::text)::text, '[]') FROM " + table + " r").Scan(&value).Error; err != nil {
			t.Fatal(err)
		}
		result += value
	}
	return result
}

func validationRequest(t *testing.T, f *submissionFixture, actor, method, path, contentType string, body io.Reader) (int, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", contentType)
	token, err := service.GenerateToken(actor)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := f.app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second, FailOnTimeout: true})
	if err != nil {
		return 0, err.Error() // fasthttp can reject malformed multipart before routing
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(data)
}

func validationMultipart(t *testing.T, fields map[string][]string, fileField string, sizes []int) (string, *bytes.Buffer) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for key, values := range fields {
		for _, value := range values {
			if err := w.WriteField(key, value); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, size := range sizes {
		part, err := w.CreateFormFile(fileField, "fixture.py")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(bytes.Repeat([]byte("x"), size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return w.FormDataContentType(), &body
}

func TestAssignmentAndTemplateInputValidation(t *testing.T) {
	db := submissionTestDB(t)
	validationTables(t, db)
	for _, kind := range []string{"assignment", "template"} {
		for _, edit := range []bool{false, true} {
			for _, scenario := range []string{"valid", "omitted-point", "zero", "negative", "fraction", "overflow", "missing-start", "equal-start-due", "close-before-due", "bad-date", "close-equals-due", "max-points", "too-many", "mixed-five", "mixed-too-many", "owned-ai", "late-oversized", "invalid-reference", "missing-reference", "foreign-reference", "duplicate-reference", "invalid-ai", "foreign-ai", "below-grade", "equal-grade", "same-max-legacy-grade"} {
				if kind == "template" && (strings.Contains(scenario, "date") || strings.Contains(scenario, "due") || scenario == "missing-start" || strings.Contains(scenario, "grade")) {
					continue
				}
				if !edit && strings.Contains(scenario, "grade") {
					continue
				}
				t.Run(kind+"/"+map[bool]string{false: "create", true: "edit"}[edit]+"/"+scenario, func(t *testing.T) {
					f := newAIConfigFixture(t, db)
					templateID := uuid.NewString()
					mustExec(t, f.db, `INSERT INTO assignment_templates (id,title,description,point,created_by) VALUES (?,'original','original',10,?)`, templateID, f.teacher)
					fields := map[string][]string{"title": {"edited"}, "description": {"description"}, "point": {"10"}, "visible": {"true"}, "tags": {"new-tag"}}
					now := time.Now().UTC()
					fields["start"] = []string{now.Add(-time.Hour).Format(time.RFC3339Nano)}
					fields["due"] = []string{now.Add(time.Hour).Format(time.RFC3339Nano)}
					fields["close"] = []string{now.Add(2 * time.Hour).Format(time.RFC3339Nano)}
					want := 400
					var sizes []int
					switch scenario {
					case "valid":
						want = 200
					case "omitted-point":
						delete(fields, "point")
						if edit {
							want = 200
						}
					case "zero":
						fields["point"] = []string{"0"}
					case "negative":
						fields["point"] = []string{"-1"}
					case "fraction":
						fields["point"] = []string{"1.5"}
					case "overflow":
						fields["point"] = []string{"32768"}
					case "max-points":
						fields["point"] = []string{"32767"}
						want = 200
					case "missing-start":
						delete(fields, "start")
						if edit {
							want = 200
						}
					case "equal-start-due":
						fields["start"] = fields["due"]
					case "close-before-due":
						fields["close"] = fields["start"]
					case "bad-date":
						fields["due"] = []string{"invalid"}
					case "close-equals-due":
						fields["close"] = fields["due"]
						want = 200
					case "too-many":
						sizes = []int{1, 1, 1, 1, 1, 1}
					case "mixed-five", "mixed-too-many":
						for range 4 {
							id := uuid.NewString()
							mustExec(t, f.db, `INSERT INTO attachments (id,user_id,file_key,file_name) VALUES (?,?,'fixture','fixture')`, id, f.teacher)
							fields["attachments"] = append(fields["attachments"], id)
						}
						sizes = []int{1}
						if scenario == "mixed-five" {
							want = 200
						} else {
							sizes = append(sizes, 1)
						}
					case "owned-ai":
						fields["ai_config_id"] = []string{f.target}
						want = 200
					case "late-oversized":
						sizes = []int{1, 10*1024*1024 + 1}
					case "invalid-reference":
						fields["attachments"] = []string{"invalid"}
					case "missing-reference":
						fields["attachments"] = []string{uuid.NewString()}
						want = 404
					case "foreign-reference":
						var id string
						f.db.Raw(`SELECT attachment_id::text FROM submissions WHERE id=?`, f.peer).Scan(&id)
						fields["attachments"] = []string{id}
						want = 404
					case "duplicate-reference":
						id := uuid.NewString()
						fields["attachments"] = []string{id, id}
					case "invalid-ai":
						fields["ai_config_id"] = []string{"invalid"}
					case "foreign-ai":
						fields["ai_config_id"] = []string{f.foreignConfig}
						want = 404
					case "below-grade":
						mustExec(t, f.db, `UPDATE submissions SET point=8 WHERE id=?`, f.own)
						fields["point"] = []string{"7"}
					case "same-max-legacy-grade":
						mustExec(t, f.db, `UPDATE submissions SET point=11 WHERE id=?`, f.own)
						want = 200 // Only reductions are constrained; no historical rescaling.
					case "equal-grade":
						mustExec(t, f.db, `UPDATE submissions SET point=8 WHERE id=?`, f.own)
						fields["point"] = []string{"8"}
						want = 200
					}
					method, path := "POST", "/api/courses/COURSA/assignments"
					if kind == "template" {
						path = "/api/templates"
					}
					if edit {
						method = "PUT"
						if kind == "template" {
							path += "/" + templateID
						} else {
							path += "/" + f.a1
						}
					}
					ct, body := validationMultipart(t, fields, "files", sizes)
					before := validationSnapshot(t, f.submissionFixture)
					status, data := validationRequest(t, f.submissionFixture, f.teacher, method, path, ct, body)
					if status != want {
						t.Fatalf("status %d want %d: %s", status, want, data)
					}
					if want != 200 && (validationSnapshot(t, f.submissionFixture) != before || f.files.calls != 0) {
						t.Fatal("invalid request changed records or storage")
					}
					if want == 200 {
						var response struct {
							Point int16 `json:"point"`
						}
						if err := json.Unmarshal([]byte(data), &response); err != nil {
							t.Fatal(err)
						}
						expected := int16(10)
						if values := fields["point"]; len(values) > 0 {
							number, err := strconv.Atoi(values[0])
							if err != nil {
								t.Fatal(err)
							}
							expected = int16(number)
						}
						if response.Point != expected {
							t.Fatalf("maximum %d, want %d", response.Point, expected)
						}
						if scenario == "same-max-legacy-grade" {
							var row models.Submission
							if err := f.db.First(&row, "id=?", f.own).Error; err != nil || row.Point == nil || *row.Point != 11 {
								t.Fatal("unrelated edit changed historical grade")
							}
						}
					}

				})
			}
		}
	}
}

func TestManualGradeInputValidation(t *testing.T) {
	db := submissionTestDB(t)
	for _, suffix := range []string{"/grade", "/grade-with-comment"} {
		for _, value := range []string{"missing", "null", "-1", "11", "1.5", "32768", "0", "10"} {
			t.Run(suffix+"/"+value, func(t *testing.T) {
				f := newSubmissionFixture(t, db)
				body := `{"comment":"must not leak","visible":true`
				if value != "missing" {
					body += `,"point":` + value
				}
				body += "}"
				before := f.snapshot(t)
				status, data := validationRequest(t, f, f.teacher, "PUT", submissionPath("COURSA", f.a1, f.own, suffix), "application/json", strings.NewReader(body))
				valid := value == "0" || value == "10"
				if (!valid && status != 400) || (valid && status != 200) {
					t.Fatalf("status %d: %s", status, data)
				}
				if !valid && f.snapshot(t) != before {
					t.Fatal("invalid grade changed score or comment")
				}
				if valid {
					var sub models.Submission
					if err := f.db.First(&sub, "id=?", f.own).Error; err != nil || sub.Point == nil {
						t.Fatal("valid grade not saved")
					}
					expected := int16(0)
					if value == "10" {
						expected = 10
					}
					if *sub.Point != expected {
						t.Fatalf("grade %d, want %d", *sub.Point, expected)
					}

				}
			})
		}
	}
}

func TestSubmissionFileInputValidation(t *testing.T) {
	db := submissionTestDB(t)
	for _, edit := range []bool{false, true} {
		for _, scenario := range []string{"missing", "empty", "multiple", "valid", "limit", "over-limit", "malformed"} {
			t.Run(map[bool]string{false: "create", true: "edit"}[edit]+"/"+scenario, func(t *testing.T) {
				f := newSubmissionFixture(t, db)
				sizes, want := []int{1}, 400
				switch scenario {
				case "missing":
					sizes = nil
					if edit {
						want = 200
					}
				case "empty":
					sizes = []int{0}
				case "multiple":
					sizes = []int{1, 1}
				case "valid":
					want = 200
				case "limit":
					sizes = []int{5 * 1024 * 1024}
					want = 200
				case "over-limit":
					sizes = []int{5*1024*1024 + 1}
				}
				ct, body := validationMultipart(t, nil, "file", sizes)
				if scenario == "malformed" {
					want = 0 // transport rejects this body before a handler can produce a response
					ct = "multipart/form-data; boundary=missing"
					body = bytes.NewBufferString("broken")
				}
				path, method := "/api/courses/COURSA/assignments/"+f.a1+"/submissions", "POST"
				if edit {
					path += "/" + f.own
					method = "PUT"
				}
				before := f.snapshot(t)
				status, data := validationRequest(t, f, f.student, method, path, ct, body)
				if status != want {
					t.Fatalf("status %d want %d: %s", status, want, data)
				}
				if want != 200 && (f.snapshot(t) != before || f.files.calls != 0) {
					t.Fatal("invalid file changed submission or storage")
				}
			})
		}
	}
}

func TestAttachmentBatchInputValidation(t *testing.T) {
	db := submissionTestDB(t)
	for _, scenario := range []string{"five", "limit", "late-oversized", "six"} {
		t.Run(scenario, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			sizes, want := []int{1, 1, 1, 1, 1}, 200
			switch scenario {
			case "limit":
				sizes = []int{10 * 1024 * 1024}
			case "late-oversized":
				sizes = []int{1, 10*1024*1024 + 1}
				want = 400
			case "six":
				sizes = append(sizes, 1)
				want = 400
			}
			ct, body := validationMultipart(t, nil, "files", sizes)
			before := f.snapshot(t)
			status, data := validationRequest(t, f, f.teacher, "POST", "/api/attachments", ct, body)
			if status != want {
				t.Fatalf("status %d want %d: %s", status, want, data)
			}
			if want != 200 && (f.snapshot(t) != before || f.files.calls != 0) {
				t.Fatal("invalid batch uploaded its earlier valid file")
			}
		})
	}
}

func TestMalformedInputUUIDs(t *testing.T) {
	db := submissionTestDB(t)
	aiConfigTables(t, db)
	for _, scenario := range []string{"assignment", "submission", "template", "attachment", "config", "credential", "missing-credential", "urn-credential"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAIConfigFixture(t, db)
			path, method, body := "/api/courses/COURSA/assignments/invalid", "PUT", `{}`
			switch scenario {
			case "submission":
				path = submissionPath("COURSA", f.a1, "invalid", "/grade")
			case "template":
				path = "/api/templates/invalid"
			case "attachment":
				path = "/api/attachments/invalid"
				method = "DELETE"
			case "config":
				path = "/api/ai/configs/invalid"
				method = "PATCH"
				body = `{"credential_id":"` + f.credential + `"}`
			case "credential", "missing-credential", "urn-credential":
				path = "/api/ai/configs/" + f.target
				method = "PATCH"
				credential := "invalid"
				if scenario == "missing-credential" {
					credential = ""
				}
				if scenario == "urn-credential" {
					credential = "urn:uuid:" + f.credential
				}
				body = `{"credential_id":"` + credential + `"}`
			}
			before := f.submissionFixture.snapshot(t) + f.snapshot(t, "")
			status, data := validationRequest(t, f.submissionFixture, f.teacher, method, path, "application/json", strings.NewReader(body))
			if status != 400 {
				t.Fatalf("status %d: %s", status, data)
			}
			if f.submissionFixture.snapshot(t)+f.snapshot(t, "") != before {
				t.Fatal("malformed ID changed records")
			}
		})
	}
}

func TestInputBoundsAtConcurrentWrite(t *testing.T) {
	db := submissionTestDB(t)
	for _, scenario := range []string{"maximum-before-manual", "maximum-before-ai", "grade-before-maximum", "partial-date-edit"} {
		t.Run(scenario, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			if err := f.db.Commit().Error; err != nil {
				t.Fatal(err)
			}
			f.db = db
			database.DB = db
			before := f.snapshot(t)
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
			want := apperrors.ErrInvalidGrade
			operation := func() error {
				p := int16(9)
				_, err := service.UpdateGrade("COURSA", f.own, "teacher", &models.GradeForm{Point: &p})
				return err
			}
			switch scenario {
			case "maximum-before-manual", "maximum-before-ai":
				mustExec(t, control, `UPDATE assignments SET point=5 WHERE id=?`, f.a1)
				if scenario == "maximum-before-ai" {
					operation = func() error {
						applied, err := repository.SaveAIGradeAndComment(f.a1, f.own, f.teacher, 9, "stale AI feedback", false)
						if applied {
							return io.ErrUnexpectedEOF
						}
						return err
					}
				}
			case "grade-before-maximum":
				mustExec(t, control, `UPDATE submissions SET point=9,graded_by='teacher' WHERE id=?`, f.own)
				want = apperrors.ErrMaximumBelowGrade
				operation = func() error { return repository.UpdateAssignment(f.a1, map[string]any{"point": int16(5)}) }
			case "partial-date-edit":
				now := time.Now().UTC()
				mustExec(t, control, `UPDATE assignments SET due_date=? WHERE id=?`, now.Add(90*time.Minute), f.a1)
				want = apperrors.ErrInvalidSchedule
				operation = func() error {
					return repository.UpdateAssignment(f.a1, map[string]any{"close_date": now.Add(80 * time.Minute)})
				}
			}
			done := make(chan error, 1)
			go func() { done <- operation() }()
			deadline := time.Now().Add(5 * time.Second)
			blocked := false
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
				t.Fatal("write did not contend for assignment lock")
			}
			if err := control.Commit().Error; err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if err != want {
					t.Fatalf("got %v, want %v", err, want)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("write remained blocked")
			}
			if scenario != "grade-before-maximum" && f.snapshot(t) != before {
				t.Fatal("rejected concurrent write changed submission/comment/file")
			}
			var assignment models.Assignment
			if err := db.First(&assignment, "id=?", f.a1).Error; err != nil {
				t.Fatal(err)
			}
			if scenario == "grade-before-maximum" && assignment.Point != 10 {
				t.Fatal("maximum reduction changed assignment despite existing grade")
			}

		})
	}
}

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/queue"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/security"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// No parallel tests: the current app exposes DB, storage, and logger as globals.
// Every fixture owns a random schema in an explicitly named disposable database.
type gradingFixture struct {
	db                  *gorm.DB
	assignment, teacher string
	calls               atomic.Int32
	beforeResponse      func() error
	providerFailure     bool
	responseOverride    func([]aiGradingResult) string
}

func newGradingFixture(t *testing.T) *gradingFixture {
	t.Helper()
	dsn := os.Getenv("TEACHING_SUPPORT_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEACHING_SUPPORT_TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || !strings.HasPrefix(strings.TrimPrefix(u.Path, "/"), "teaching_support_test") {
		t.Fatal("test URL must name a teaching_support_test* database")
	}
	open := func(dsn string) *gorm.DB {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
		if err != nil {
			t.Fatal("cannot connect to test database")
		}
		conn, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = conn.Close() })
		return db
	}
	admin := open(dsn)
	schema := "r4_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	execGrading(t, admin, `CREATE SCHEMA "`+schema+`"`)
	t.Cleanup(func() { execGrading(t, admin, `DROP SCHEMA "`+schema+`" CASCADE`) })
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	db := open(u.String())
	for _, ddl := range []string{
		`CREATE TABLE assignments (id uuid PRIMARY KEY, course_id text, title text, description text, point smallint, ai_config_id uuid, prompt text, visible boolean, start_date timestamptz, due_date timestamptz, close_date timestamptz, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE attachments (id uuid PRIMARY KEY, file_key text, file_name text, file_type text, size bigint, user_id uuid, created_at timestamptz)`,
		`CREATE TABLE submissions (id uuid PRIMARY KEY, assignment_id uuid, student_id uuid, point smallint, graded_by text, attachment_id uuid, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE comments (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), submission_id uuid REFERENCES submissions(id), comment text, created_by_role text, created_by text, visible boolean, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE ai_credentials (id uuid PRIMARY KEY, user_id uuid, provider text, base_url text, encrypted_api_key text, name text, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE ai_configs (id uuid PRIMARY KEY, user_id uuid, ai_credential_id uuid, name text, model text, temperature real, created_at timestamptz, updated_at timestamptz)`,
		`CREATE TABLE grading_jobs (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), assignment_id uuid, teacher_id uuid, status text, progress smallint, total_submissions smallint, processed_submissions smallint, error text, started_at timestamptz, completed_at timestamptz, created_at timestamptz, updated_at timestamptz)`,
	} {
		execGrading(t, db, ddl)
	}
	f := &gradingFixture{db: db, assignment: uuid.NewString(), teacher: uuid.NewString()}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "unexpected route", 400)
			return
		}
		if f.providerFailure {
			http.Error(w, "synthetic failure", 400)
			return
		}
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Messages) == 0 {
			http.Error(w, "invalid request", 400)
			return
		}
		matches := regexp.MustCompile(`"index":(\d+),"id":"([^"]+)"`).FindAllStringSubmatch(request.Messages[len(request.Messages)-1].Content, -1)
		results := []aiGradingResult{}
		for _, match := range matches {
			index, _ := strconv.Atoi(match[1])
			results = append(results, aiGradingResult{Index: index, ID: match[2], Point: 3, Comment: "new AI feedback"})
		}
		if f.beforeResponse != nil {
			if err := f.beforeResponse(); err != nil {
				http.Error(w, "manual fixture failed", 400)
				return
			}
		}
		content, _ := json.Marshal(results)
		if f.responseOverride != nil {
			content = []byte(f.responseOverride(results))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "fixture", "object": "chat.completion", "created": 1, "model": "fixture", "choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": string(content)}, "finish_reason": "stop"}}})
	}))
	t.Cleanup(server.Close)
	t.Setenv("AI_SECRET_KEY", strings.Repeat("k", 32))
	encrypted, err := security.Encrypt("synthetic-provider-key", []byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	credential, config := uuid.NewString(), uuid.NewString()
	execGrading(t, db, `INSERT INTO ai_credentials (id, provider, base_url, encrypted_api_key) VALUES (?, 'custom', ?, ?)`, credential, server.URL+"/v1/", encrypted)
	execGrading(t, db, `INSERT INTO ai_configs (id, ai_credential_id, model) VALUES (?, ?, 'fixture')`, config, credential)
	execGrading(t, db, `INSERT INTO assignments (id, course_id, title, description, point, ai_config_id) VALUES (?, 'COURSA', 'Exercise', '', 10, ?)`, f.assignment, config)
	execGrading(t, db, `INSERT INTO grading_jobs (assignment_id, teacher_id, status) VALUES (?, ?, 'pending')`, f.assignment, f.teacher)
	oldDB, oldStorage, oldLog := database.DB, storage.Provider, logger.Log
	database.DB, storage.Provider = db, gradingFiles{}
	logger.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Cleanup(func() { database.DB, storage.Provider, logger.Log = oldDB, oldStorage, oldLog })
	return f
}

func execGrading(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatal(err)
	}
}

type gradingFiles struct{}

func (gradingFiles) Download(context.Context, string) ([]byte, error) { return []byte("print(1)"), nil }
func (gradingFiles) Upload(context.Context, []byte, string) (string, error) {
	return "", fmt.Errorf("unexpected upload")
}
func (gradingFiles) Delete(context.Context, string) error { return fmt.Errorf("unexpected deletion") }
func (gradingFiles) ReadText(context.Context, string) (string, error) {
	return "", fmt.Errorf("unexpected text read")
}

func (f *gradingFixture) submission(t *testing.T, role any, point any) string {
	t.Helper()
	id, attachment := uuid.NewString(), uuid.NewString()
	execGrading(t, f.db, `INSERT INTO attachments (id, file_key, file_name) VALUES (?, ?, 'answer.py')`, attachment, id)
	execGrading(t, f.db, `INSERT INTO submissions (id, assignment_id, student_id, attachment_id, graded_by, point) VALUES (?, ?, ?, ?, ?, ?)`, id, f.assignment, uuid.NewString(), attachment, role, point)
	return id
}

func (f *gradingFixture) comment(t *testing.T, sid, role, content string) {
	t.Helper()
	execGrading(t, f.db, `INSERT INTO comments (submission_id, created_by_role, created_by, comment, visible) VALUES (?, ?, ?, ?, false)`, sid, role, f.teacher, content)
}

func (f *gradingFixture) snapshot(t *testing.T, sid string) string {
	t.Helper()
	var result string
	if err := f.db.Raw(`SELECT row_to_json(s)::text || COALESCE((SELECT json_agg(c ORDER BY id)::text FROM comments c WHERE submission_id = s.id),'[]') FROM submissions s WHERE s.id = ?`, sid).Scan(&result).Error; err != nil {
		t.Fatal(err)
	}
	return result
}

func (f *gradingFixture) run(force bool) error {
	var jobID string
	_ = f.db.Raw("SELECT id FROM grading_jobs WHERE assignment_id=? AND teacher_id=?", f.assignment, f.teacher).Scan(&jobID).Error
	payload, _ := json.Marshal(queue.AutoGradingPayload{JobID: jobID, AssignmentID: f.assignment, TeacherID: f.teacher, ForceRegade: force})
	return HandleAutoGrading(context.Background(), asynq.NewTask(queue.TypeAutoGrading, payload))
}

func (f *gradingFixture) assertAI(t *testing.T, sid string) {
	t.Helper()
	var s models.Submission
	if err := f.db.Preload("Comments").First(&s, "id = ?", sid).Error; err != nil {
		t.Fatal(err)
	}
	if s.Point == nil || *s.Point != 3 || s.GradedBy == nil || *s.GradedBy != "ai" {
		t.Fatal("AI result was not saved")
	}
	aiComments := 0
	for _, c := range s.Comments {
		if c.CreatedByRole == models.RoleAI {
			aiComments++
			if c.Comment != "new AI feedback" || !c.Visible || c.CreatedBy != f.teacher {
				t.Fatal("AI feedback mismatch")
			}
		}
	}
	if aiComments != 1 {
		t.Fatalf("AI comment count = %d", aiComments)
	}
	var job models.GradingJob
	if err := f.db.First(&job, "assignment_id = ?", f.assignment).Error; err != nil {
		t.Fatal(err)
	}
	if job.Status != models.JobCompleted || job.Progress != 100 || job.ProcessedSubmissions != job.TotalSubmissions {
		t.Fatal("job did not complete")
	}
}

func TestAutoGradingProtectsHumanGrades(t *testing.T) {
	for _, role := range []string{"teacher", "ta", "unknown"} {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/force=%t", role, force), func(t *testing.T) {
				f := newGradingFixture(t)
				human := f.submission(t, role, 0)
				f.comment(t, human, role, "human feedback")
				f.comment(t, human, "ai", "previous AI feedback")
				pending := f.submission(t, nil, nil)
				before := f.snapshot(t, human)
				if err := f.run(force); err != nil {
					t.Fatal(err)
				}
				if f.snapshot(t, human) != before {
					t.Fatal("protected grade or feedback changed")
				}
				f.assertAI(t, pending)
			})
		}
	}
}

func TestAutoGradingManualDuringProvider(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force=%t", force), func(t *testing.T) {
			f := newGradingFixture(t)
			sid := f.submission(t, nil, nil)
			f.comment(t, sid, "ai", "previous AI feedback")
			var afterManual string
			f.beforeResponse = func() error {
				_, err := service.UpdateGrade("COURSA", sid, "teacher", &models.GradeForm{Point: gradePoint(9)})
				if err == nil {
					afterManual = f.snapshot(t, sid)
				}
				return err
			}
			if err := f.run(force); err != nil {
				t.Fatal(err)
			}
			if afterManual == "" || f.snapshot(t, sid) != afterManual {
				t.Fatal("AI overwrote a human grade or its existing feedback")
			}
			// Simulate delivery retry with the same payload and no manual callback.
			f.beforeResponse = nil
			if err := f.run(force); err != nil {
				t.Fatal(err)
			}
			if f.snapshot(t, sid) != afterManual {
				t.Fatal("retry changed protected work")
			}
		})
	}
}

func TestAutoGradingRegradePolicy(t *testing.T) {
	for _, scenario := range []string{"normal-ai", "forced-ai", "forced-failure", "normal-ungraded", "unattributed-score"} {
		t.Run(scenario, func(t *testing.T) {
			f := newGradingFixture(t)
			var role, point any = "ai", int16(2)
			if scenario == "normal-ungraded" {
				role, point = nil, nil
			}
			if scenario == "unattributed-score" {
				role, point = nil, int16(8)
			}
			sid := f.submission(t, role, point)
			if role != nil {
				f.comment(t, sid, "ai", "previous AI feedback")
			}
			before := f.snapshot(t, sid)
			force := strings.HasPrefix(scenario, "forced")
			f.providerFailure = scenario == "forced-failure"
			err := f.run(force)
			if f.providerFailure {
				if err == nil || f.snapshot(t, sid) != before {
					t.Fatal("failed regrade destroyed previous result")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "normal-ai" || scenario == "unattributed-score" {
				if f.calls.Load() != 0 || f.snapshot(t, sid) != before {
					t.Fatal("normal run changed existing score")
				}
			} else {
				f.assertAI(t, sid)
				stable := f.snapshot(t, sid)
				if err := f.run(false); err != nil {
					t.Fatal(err)
				}
				if f.calls.Load() != 1 || f.snapshot(t, sid) != stable {
					t.Fatal("normal retry duplicated or changed AI output")
				}
			}
		})
	}
}

type aiSaveResult struct {
	applied bool
	err     error
}

func startAISave(f *gradingFixture, sid string, force bool) <-chan aiSaveResult {
	done := make(chan aiSaveResult, 1)
	go func() {
		applied, err := repository.SaveAIGradeAndComment(f.assignment, sid, f.teacher, 3, "new AI feedback", force)
		done <- aiSaveResult{applied, err}
	}()
	return done
}

func awaitAISave(t *testing.T, done <-chan aiSaveResult) aiSaveResult {
	t.Helper()
	select {
	case result := <-done:
		return result
	case <-time.After(10 * time.Second):
		t.Fatal("AI save did not finish")
		return aiSaveResult{}
	}
}

// Observe actual PostgreSQL blocking rather than relying on guessed sleep timing.
func awaitGradingBlock(t *testing.T, db *gorm.DB, blocker int) int {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		var blocked int
		if err := db.Raw(`SELECT COALESCE((SELECT pid FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid)) LIMIT 1),0)`, blocker).Scan(&blocked).Error; err != nil {
			t.Fatal(err)
		}
		if blocked != 0 {
			return blocked
		}
		select {
		case <-deadline.C:
			t.Fatal("expected database lock contention was not observed")
			return 0
		case <-tick.C:
		}
	}
}

func TestAIGradeWriteSerialization(t *testing.T) {
	for _, order := range []string{"human-first", "ai-first"} {
		t.Run(order, func(t *testing.T) {
			f := newGradingFixture(t)
			sid := f.submission(t, nil, nil)
			control := f.db.Begin()
			if control.Error != nil {
				t.Fatal(control.Error)
			}
			t.Cleanup(func() { _ = control.Rollback().Error })
			var blocker int
			if err := control.Raw(`SELECT pg_backend_pid()`).Scan(&blocker).Error; err != nil {
				t.Fatal(err)
			}
			if order == "human-first" {
				execGrading(t, control, `UPDATE submissions SET point=9, graded_by='teacher' WHERE id=?`, sid)
				done := startAISave(f, sid, false)
				awaitGradingBlock(t, f.db, blocker)
				if err := control.Commit().Error; err != nil {
					t.Fatal(err)
				}
				result := awaitAISave(t, done)
				if result.err != nil || result.applied {
					t.Fatal("AI did not skip a committed human grade")
				}
			} else {
				// Block AI feedback while its transaction holds assignment/submission
				// locks, then make a real manual grade contend for those locks.
				execGrading(t, f.db, `CREATE FUNCTION hold_ai_feedback() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(42641025); RETURN NEW; END $$`)
				execGrading(t, f.db, `CREATE TRIGGER hold_ai_feedback BEFORE INSERT ON comments FOR EACH ROW EXECUTE FUNCTION hold_ai_feedback()`)
				execGrading(t, control, `SELECT pg_advisory_xact_lock(42641025)`)
				done := startAISave(f, sid, false)
				aiPID := awaitGradingBlock(t, f.db, blocker)
				manualDone := make(chan error, 1)
				go func() {
					_, err := service.UpdateGrade("COURSA", sid, "teacher", &models.GradeForm{Point: gradePoint(9)})
					manualDone <- err
				}()
				awaitGradingBlock(t, f.db, aiPID)
				if err := control.Commit().Error; err != nil {
					t.Fatal(err)
				}
				result := awaitAISave(t, done)
				if result.err != nil || !result.applied {
					t.Fatal("first AI transaction failed")
				}
				select {
				case err := <-manualDone:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(10 * time.Second):
					t.Fatal("manual grade remained blocked")
				}
			}
			var submission models.Submission
			if err := f.db.Preload("Comments").First(&submission, "id=?", sid).Error; err != nil {
				t.Fatal(err)
			}
			if submission.Point == nil || *submission.Point != 9 || submission.GradedBy == nil || *submission.GradedBy != "teacher" {
				t.Fatal("human grade did not win")
			}
			if order == "human-first" && len(submission.Comments) != 0 {
				t.Fatal("AI wrote feedback after the human grade")
			}
		})
	}
}

func TestAIGradeAtomicityAndRetries(t *testing.T) {
	for _, scenario := range []string{"comment-failure", "concurrent-normal", "concurrent-forced", "forced-repeat", "wrong-assignment"} {
		t.Run(scenario, func(t *testing.T) {
			f := newGradingFixture(t)
			sid := f.submission(t, nil, nil)
			before := f.snapshot(t, sid)
			switch scenario {
			case "comment-failure":
				execGrading(t, f.db, `ALTER TABLE comments ADD CONSTRAINT reject_ai CHECK (created_by_role <> 'ai')`)
				result := awaitAISave(t, startAISave(f, sid, false))
				if result.err == nil || result.applied || f.snapshot(t, sid) != before {
					t.Fatal("feedback failure did not roll back AI score")
				}
			case "concurrent-normal", "concurrent-forced":
				force := scenario == "concurrent-forced"
				first, second := startAISave(f, sid, force), startAISave(f, sid, force)
				a, b := awaitAISave(t, first), awaitAISave(t, second)
				if a.err != nil || b.err != nil || (!force && a.applied == b.applied) || (force && (!a.applied || !b.applied)) {
					t.Fatal("concurrent AI saves violated normal/force policy")
				}
				var count int64
				if err := f.db.Model(&models.Comment{}).Where("submission_id=?", sid).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 1 {
					t.Fatal("concurrent saves duplicated feedback")
				}
			case "forced-repeat":
				for range 2 {
					if result := awaitAISave(t, startAISave(f, sid, true)); result.err != nil || !result.applied {
						t.Fatal("forced AI replacement failed")
					}
				}
				var count int64
				if err := f.db.Model(&models.Comment{}).Where("submission_id=?", sid).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 1 {
					t.Fatal("forced repeat duplicated feedback")
				}
			case "wrong-assignment":
				applied, err := repository.SaveAIGradeAndComment(uuid.NewString(), sid, f.teacher, 3, "wrong", true)
				if err != nil || applied || f.snapshot(t, sid) != before {
					t.Fatal("AI save crossed assignment boundary")
				}
			}
		})
	}
}

func gradePoint(point int16) *int16 { return &point }

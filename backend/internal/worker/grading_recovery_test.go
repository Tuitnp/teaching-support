package worker

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/queue"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/storage"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type panicGradingFiles struct{ gradingFiles }

func (panicGradingFiles) Download(context.Context, string) ([]byte, error) {
	panic("synthetic storage panic")
}

func (f *gradingFixture) payload(t *testing.T, force bool) queue.AutoGradingPayload {
	t.Helper()
	var id string
	if err := f.db.Raw(`SELECT id FROM grading_jobs WHERE assignment_id=? AND teacher_id=?`, f.assignment, f.teacher).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	return queue.AutoGradingPayload{JobID: id, AssignmentID: f.assignment, TeacherID: f.teacher, ForceRegade: force}
}
func (f *gradingFixture) job(t *testing.T) models.GradingJob {
	t.Helper()
	var job models.GradingJob
	if err := f.db.Where("assignment_id=? AND teacher_id=?", f.assignment, f.teacher).First(&job).Error; err != nil {
		t.Fatal(err)
	}
	return job
}

func TestGradingRetryLifecycle(t *testing.T) {
	for _, failure := range []string{"provider", "submissions", "config", "storage-panic", "progress", "completion"} {
		t.Run(failure, func(t *testing.T) {
			f := newGradingFixture(t)
			f.submission(t, nil, nil)
			payload := f.payload(t, false)
			switch failure {
			case "provider":
				f.providerFailure = true
			case "submissions":
				execGrading(t, f.db, `ALTER TABLE submissions RENAME TO unavailable_submissions`)
			case "config":
				execGrading(t, f.db, `UPDATE assignments SET ai_config_id=NULL`)
			case "storage-panic":
				storage.Provider = panicGradingFiles{}
			case "progress":
				execGrading(t, f.db, `ALTER TABLE grading_jobs ADD CONSTRAINT reject_progress CHECK (total_submissions IS NULL OR total_submissions=0)`)
			case "completion":
				execGrading(t, f.db, `CREATE FUNCTION reject_completion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status='completed' THEN RAISE EXCEPTION 'synthetic completion failure'; END IF; RETURN NEW; END $$`)
				execGrading(t, f.db, `CREATE TRIGGER reject_completion BEFORE UPDATE ON grading_jobs FOR EACH ROW EXECUTE FUNCTION reject_completion()`)
			}
			for retry := 0; retry <= 3; retry++ {
				if err := handleGradingAttempt(context.Background(), payload, retry, 3); err == nil {
					t.Fatal("expected failed attempt")
				}
				job := f.job(t)
				want := models.JobPending
				if retry == 3 {
					want = models.JobFailed
				}
				if job.Status != want || job.Error == "" || (retry == 3 && job.CompletedAt == nil) || (retry < 3 && job.CompletedAt != nil) {
					t.Fatalf("retry=%d status=%s completed=%v error=%q", retry, job.Status, job.CompletedAt, job.Error)
				}
			}
		})
	}
}

func TestGradingNoWorkAndCompletedDelivery(t *testing.T) {
	f := newGradingFixture(t)
	payload := f.payload(t, true)
	if err := handleGradingAttempt(context.Background(), payload, 0, 3); err != nil {
		t.Fatal(err)
	}
	job := f.job(t)
	if job.Status != models.JobCompleted || job.Progress != 100 || job.TotalSubmissions != 0 || job.ProcessedSubmissions != 0 || job.CompletedAt == nil || f.calls.Load() != 0 {
		t.Fatal("empty job did not complete without provider work")
	}
	f.submission(t, nil, nil)
	if err := handleGradingAttempt(context.Background(), payload, 1, 3); err != nil {
		t.Fatal(err)
	}
	if f.calls.Load() != 0 {
		t.Fatal("completed duplicate delivery ran provider again")
	}
}

func TestGradingPartialRetryPreservesResults(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprint(force), func(t *testing.T) {
			f := newGradingFixture(t)
			var ids []string
			for i := 0; i < 3; i++ {
				var role, point any
				if force {
					role, point = "ai", int16(2)
				}
				id := f.submission(t, role, point)
				ids = append(ids, id)
				if force {
					f.comment(t, id, "ai", "old feedback")
				}
			}
			execGrading(t, f.db, `CREATE SEQUENCE save_attempt`)
			execGrading(t, f.db, `CREATE FUNCTION fail_second_save() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.graded_by='ai' AND nextval('save_attempt')=2 THEN RAISE EXCEPTION 'synthetic second save failure'; END IF; RETURN NEW; END $$`)
			execGrading(t, f.db, `CREATE TRIGGER fail_second_save BEFORE UPDATE ON submissions FOR EACH ROW EXECUTE FUNCTION fail_second_save()`)
			payload := f.payload(t, force)
			if err := handleGradingAttempt(context.Background(), payload, 0, 3); err == nil {
				t.Fatal("expected partial failure")
			}
			var saved models.Submission
			if err := f.db.Preload("Comments").Where("point=3").First(&saved).Error; err != nil {
				t.Fatal(err)
			}
			commentID := saved.Comments[0].ID
			if f.job(t).Status != models.JobPending {
				t.Fatal("partial failure marked terminal")
			}
			f.responseOverride = func(results []aiGradingResult) string {
				for i := range results {
					results[i].Point = 4
				}
				data, _ := json.Marshal(results)
				return string(data)
			}
			if err := handleGradingAttempt(context.Background(), payload, 1, 3); err != nil {
				t.Fatal(err)
			}
			for _, id := range ids {
				var current models.Submission
				if err := f.db.Preload("Comments").First(&current, "id=?", id).Error; err != nil {
					t.Fatal(err)
				}
				want := int16(4)
				if !force && id == saved.ID {
					want = 3
				}
				if current.Point == nil || *current.Point != want || len(current.Comments) != 1 {
					t.Fatal("retry lost/duplicated a result")
				}
				if id == saved.ID && current.Comments[0].ID != commentID {
					t.Fatal("retry changed AI feedback identity")
				}
			}
			if f.job(t).Status != models.JobCompleted || f.calls.Load() != 2 {
				t.Fatal("retry did not complete remaining work")
			}
		})
	}
}

func TestGradingCancellationAndAttemptFence(t *testing.T) {
	for _, name := range []string{"cancelled-provider", "superseded-attempt", "obsolete-job", "legacy-payload", "timeout-cleanup", "late-timeout"} {
		t.Run(name, func(t *testing.T) {
			f := newGradingFixture(t)
			sid := f.submission(t, nil, nil)
			payload := f.payload(t, false)
			before := f.snapshot(t, sid)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch name {
			case "cancelled-provider":
				f.beforeResponse = func() error { cancel(); return nil }
				if err := handleGradingAttempt(ctx, payload, 0, 3); err == nil {
					t.Fatal("cancelled provider committed")
				}
				if f.job(t).Status != models.JobPending {
					t.Fatal("cancellation did not prepare retry")
				}
			case "superseded-attempt":
				f.beforeResponse = func() error {
					_, err := repository.BeginGradingAttempt(context.Background(), payload.JobID, f.assignment, f.teacher)
					return err
				}
				if err := handleGradingAttempt(ctx, payload, 0, 3); !stderrors.Is(err, repository.ErrObsoleteGradingAttempt) {
					t.Fatalf("old attempt returned %v", err)
				}
				if f.job(t).Status != models.JobProcessing {
					t.Fatal("old cleanup changed newer attempt")
				}
			case "obsolete-job":
				execGrading(t, f.db, `UPDATE grading_jobs SET id=?`, uuid.NewString())
				if err := handleGradingAttempt(ctx, payload, 0, 3); err == nil || f.calls.Load() != 0 {
					t.Fatal("obsolete job ran")
				}
			case "legacy-payload":
				payload.JobID = ""
				data, _ := json.Marshal(payload)
				if err := HandleAutoGrading(ctx, asynq.NewTask(queue.TypeAutoGrading, data)); !stderrors.Is(err, asynq.SkipRetry) {
					t.Fatal("legacy identity was accepted")
				}
			case "timeout-cleanup", "late-timeout":
				started := time.Now().Add(-31 * time.Minute)
				if name == "late-timeout" {
					started = time.Now()
				}
				execGrading(t, f.db, `UPDATE grading_jobs SET status='processing',started_at=?`, started)
				data, _ := json.Marshal(payload)
				cancel()
				HandleGradingError(ctx, asynq.NewTask(queue.TypeAutoGrading, data), context.DeadlineExceeded)
				want := models.JobFailed
				if name == "late-timeout" {
					want = models.JobProcessing
				}
				if f.job(t).Status != want {
					t.Fatal("timed out final attempt remained processing")
				}
			}
			if f.snapshot(t, sid) != before {
				t.Fatal("cancelled/obsolete attempt changed grade or feedback")
			}
		})
	}
}

func TestGradingDatabaseCancellation(t *testing.T) {
	f := newGradingFixture(t)
	sid := f.submission(t, nil, nil)
	payload := f.payload(t, false)
	attempt, err := repository.BeginGradingAttempt(context.Background(), payload.JobID, f.assignment, f.teacher)
	if err != nil {
		t.Fatal(err)
	}
	before := f.snapshot(t, sid)
	control := f.db.Begin()
	t.Cleanup(func() { _ = control.Rollback().Error })
	var pid int
	if err := control.Raw(`SELECT pg_backend_pid()`).Scan(&pid).Error; err != nil {
		t.Fatal(err)
	}
	var id string
	if err := control.Raw(`SELECT id FROM assignments WHERE id=? FOR UPDATE`, f.assignment).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan aiSaveResult, 1)
	go func() {
		applied, err := repository.SaveAIGradeForAttempt(ctx, attempt, sid, 3, "cancelled feedback", false)
		done <- aiSaveResult{applied, err}
	}()
	awaitGradingBlock(t, f.db, pid)
	cancel()
	result := awaitAISave(t, done)
	if result.applied || result.err == nil {
		t.Fatal("cancelled blocked write succeeded")
	}
	if err := control.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	if f.snapshot(t, sid) != before {
		t.Fatal("cancelled SQL changed a grade")
	}
}

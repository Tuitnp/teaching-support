package handler_test

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	apperrors "github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/queue"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type gradingQueueFake struct {
	mu                     sync.Mutex
	payloads               []queue.AutoGradingPayload
	options                []asynq.Option
	tasks                  map[string]*asynq.TaskInfo
	enqueueErr, inspectErr error
}

func (q *gradingQueueFake) EnqueueContext(ctx context.Context, t *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := ctx.Deadline(); !ok {
		return nil, fmt.Errorf("enqueue needs a deadline")
	}
	var payload queue.AutoGradingPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return nil, err
	}
	q.payloads = append(q.payloads, payload)
	q.options = opts
	info := &asynq.TaskInfo{ID: payload.JobID, State: asynq.TaskStatePending}
	q.tasks[payload.JobID] = info // Also models acceptance with a lost acknowledgement.
	return info, q.enqueueErr
}
func (q *gradingQueueFake) GetTaskInfo(_ string, id string) (*asynq.TaskInfo, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.inspectErr != nil {
		return nil, q.inspectErr
	}
	if info, ok := q.tasks[id]; ok {
		return info, nil
	}
	return nil, asynq.ErrTaskNotFound
}
func jobFixture(t *testing.T, f *submissionFixture) *gradingQueueFake {
	t.Helper()
	validationTables(t, f.db)
	mustExec(t, f.db, `CREATE TABLE grading_jobs (id uuid PRIMARY KEY DEFAULT gen_random_uuid(), assignment_id uuid, teacher_id uuid, status text, progress smallint DEFAULT 0, total_submissions smallint DEFAULT 0, processed_submissions smallint DEFAULT 0, error text, started_at timestamptz, completed_at timestamptz, created_at timestamptz, updated_at timestamptz, UNIQUE(assignment_id,teacher_id))`)
	config := uuid.NewString()
	mustExec(t, f.db, `INSERT INTO ai_configs (id,user_id) VALUES (?,?)`, config, f.teacher)
	mustExec(t, f.db, `UPDATE assignments SET ai_config_id=? WHERE id=?`, config, f.a1)
	oldClient, oldInspector := queue.Client, queue.Inspector
	t.Cleanup(func() { queue.Client, queue.Inspector = oldClient, oldInspector })
	q := &gradingQueueFake{tasks: map[string]*asynq.TaskInfo{}}
	queue.Client, queue.Inspector = q, q
	return q
}

func TestGradingEnqueueRecovery(t *testing.T) {
	for _, scenario := range []string{"new-failure", "completed-failure", "accepted-commit-failure", "success", "concurrent"} {
		t.Run(scenario, func(t *testing.T) {
			db := submissionTestDB(t)
			f := newSubmissionFixture(t, db)
			q := jobFixture(t, f)
			oldID := uuid.NewString()
			if scenario == "completed-failure" || scenario == "accepted-commit-failure" {
				mustExec(t, f.db, `INSERT INTO grading_jobs (id,assignment_id,teacher_id,status,progress) VALUES (?,?,?,'completed',100)`, oldID, f.a1, f.teacher)
			}
			if scenario == "new-failure" || scenario == "completed-failure" {
				q.enqueueErr = fmt.Errorf("synthetic lost queue acknowledgement")
			}
			if scenario == "accepted-commit-failure" || scenario == "concurrent" {
				if err := f.db.Commit().Error; err != nil {
					t.Fatal(err)
				}
				f.db, database.DB = db, db
			}
			if scenario == "accepted-commit-failure" {
				mustExec(t, db, `CREATE FUNCTION reject_job_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic commit failure'; END $$`)
				mustExec(t, db, `CREATE CONSTRAINT TRIGGER reject_job_commit AFTER INSERT ON grading_jobs DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_job_commit()`)
			}
			if scenario == "concurrent" {
				done := make(chan error, 2)
				for i := 0; i < 2; i++ {
					go func() { done <- service.AutoGradingAssignment(f.teacher, "COURSA", f.a1) }()
				}
				ok, limited := 0, 0
				for i := 0; i < 2; i++ {
					select {
					case err := <-done:
						if err == nil {
							ok++
						} else if stderrors.Is(err, apperrors.ErrAIGradingLimit) {
							limited++
						} else {
							t.Fatal(err)
						}
					case <-time.After(10 * time.Second):
						t.Fatal("start timed out")
					}
				}
				if ok != 1 || limited != 1 || len(q.payloads) != 1 {
					t.Fatalf("starts=%d, limited=%d, enqueued=%d", ok, limited, len(q.payloads))
				}
				return
			}
			err := service.AutoGradingAssignment(f.teacher, "COURSA", f.a1)
			if scenario == "success" {
				if err != nil {
					t.Fatal(err)
				}
				settings := map[asynq.OptionType]any{}
				for _, option := range q.options {
					settings[option.Type()] = option.Value()
				}
				if settings[asynq.TimeoutOpt] != 30*time.Minute || settings[asynq.MaxRetryOpt] != 3 || settings[asynq.TaskIDOpt] != q.payloads[0].JobID || settings[asynq.RetentionOpt] != 24*time.Hour {
					t.Fatalf("unexpected queue contract: %v", settings)
				}
				return
			}
			if err == nil || len(q.payloads) != 1 {
				t.Fatal("expected enqueue/commit failure")
			}
			orphan := q.payloads[0]
			if attempt, err := repository.BeginGradingAttempt(context.Background(), orphan.JobID, f.a1, f.teacher); err == nil || attempt != nil {
				t.Fatal("uncommitted queue task acquired a job")
			}
			var jobs []models.GradingJob
			if err := f.db.Find(&jobs).Error; err != nil {
				t.Fatal(err)
			}
			if scenario == "new-failure" {
				if len(jobs) != 0 {
					t.Fatal("failed enqueue left pending reservation")
				}
			} else if len(jobs) != 1 || jobs[0].ID != oldID || jobs[0].Status != "completed" {
				t.Fatal("failed start destroyed previous completed job")
			}
			q.enqueueErr = nil
			if scenario == "accepted-commit-failure" {
				mustExec(t, db, `DROP TRIGGER reject_job_commit ON grading_jobs`)
			}
			if err := service.AutoGradingAssignment(f.teacher, "COURSA", f.a1); err != nil {
				t.Fatal(err)
			}
			if len(q.payloads) != 2 || q.payloads[1].JobID == orphan.JobID || q.payloads[1].ForceRegade != (scenario != "new-failure") {
				t.Fatal("recovery lost run identity or force policy")
			}
		})
	}
}

func TestGradingStateReconciliation(t *testing.T) {
	db := submissionTestDB(t)
	for _, name := range []string{"active", "retry", "archived", "completed", "missing", "redis-error"} {
		t.Run(name, func(t *testing.T) {
			f := newSubmissionFixture(t, db)
			q := jobFixture(t, f)
			id := uuid.NewString()
			mustExec(t, f.db, `INSERT INTO grading_jobs (id,assignment_id,teacher_id,status,started_at) VALUES (?,?,?,'processing',?)`, id, f.a1, f.teacher, time.Now().Add(-time.Minute))
			state := asynq.TaskStateActive
			want := "processing"
			switch name {
			case "retry":
				state, want = asynq.TaskStateRetry, "pending"
			case "archived":
				state, want = asynq.TaskStateArchived, "failed"
			case "completed":
				state, want = asynq.TaskStateCompleted, "completed"
			case "missing":
				want = "failed"
			case "redis-error":
				q.inspectErr = fmt.Errorf("synthetic queue outage")
			}
			if name != "missing" {
				q.tasks[id] = &asynq.TaskInfo{ID: id, State: state}
			}
			status, body := f.request(t, f.teacher, http.MethodGet, "/api/ai/jobs", "")
			if name == "redis-error" {
				if status != 500 {
					t.Fatalf("status %d: %s", status, body)
				}
			} else if status != 200 {
				t.Fatalf("status %d: %s", status, body)
			}
			var job models.GradingJob
			if err := f.db.First(&job, "id=?", id).Error; err != nil {
				t.Fatal(err)
			}
			if job.Status != want {
				t.Fatalf("status=%s, want %s", job.Status, want)
			}
			if (want == "failed" || want == "completed") && job.CompletedAt == nil {
				t.Fatal("terminal state lacks timestamp")
			}
		})
	}
}

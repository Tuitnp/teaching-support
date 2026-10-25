package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/queue"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/hibiken/asynq"
)

func HandleAutoGrading(ctx context.Context, t *asynq.Task) error {
	var payload queue.AutoGradingPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("invalid grading payload: %w", asynq.SkipRetry)
	}
	// Legacy tasks cannot be safely bound to a current run. Job reads identify
	// their missing deterministic queue ID and make the old reservation restartable.
	if err := models.ValidateUUIDs(payload.JobID, payload.AssignmentID, payload.TeacherID); err != nil {
		return fmt.Errorf("invalid or legacy grading identity: %w", asynq.SkipRetry)
	}
	retried, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	return handleGradingAttempt(ctx, payload, retried, maxRetry)
}

func handleGradingAttempt(parent context.Context, payload queue.AutoGradingPayload, retried, maxRetry int) (runErr error) {
	ctx, cancel := context.WithTimeout(parent, queue.AutoGradingTimeout)
	defer cancel()
	attempt, err := repository.BeginGradingAttempt(ctx, payload.JobID, payload.AssignmentID, payload.TeacherID)
	if err != nil {
		return err
	}
	if attempt == nil {
		return nil
	} // Completed/failed duplicate delivery.
	defer func() {
		if recover() != nil {
			runErr = fmt.Errorf("grading attempt panicked")
		}
		if runErr == nil {
			runErr = ctx.Err()
		}
		status, problem := models.JobCompleted, ""
		if runErr != nil {
			status, problem = models.JobPending, "Grading attempt failed; waiting for an automatic retry."
			if retried >= maxRetry {
				status, problem = models.JobFailed, "Grading attempt failed; automatic retries exhausted."
			}
		}
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := repository.FinishGradingAttempt(cleanup, attempt, status, problem); err != nil {
			if runErr == nil {
				runErr = err
				status, problem = models.JobPending, "Could not finalize grading; waiting for an automatic retry."
				if retried >= maxRetry {
					status, problem = models.JobFailed, "Could not finalize grading; automatic retries exhausted."
				}
				_ = repository.FinishGradingAttempt(cleanup, attempt, status, problem)
			}
			logger.Log.Error("grading_outcome_persistence_failed", "job_id", payload.JobID, "error", err)
		}
	}()
	return runAutoGrading(ctx, attempt, payload.ForceRegade)
}

// Asynq may observe a timeout before a provider returns to our deferred cleanup.
// Fence only the expired attempt; a newer attempt has a different start token.
func HandleGradingError(ctx context.Context, t *asynq.Task, err error) {
	if ctx.Err() == nil {
		return
	}
	var payload queue.AutoGradingPayload
	if json.Unmarshal(t.Payload(), &payload) != nil || models.ValidateUUIDs(payload.JobID, payload.AssignmentID, payload.TeacherID) != nil {
		return
	}
	cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	job, readErr := repository.GetGradingRun(cleanup, payload.JobID, payload.TeacherID)
	if readErr != nil || job == nil || job.Status != models.JobProcessing || job.StartedAt == nil || time.Since(*job.StartedAt) < queue.AutoGradingTimeout {
		return
	}
	retried, _ := asynq.GetRetryCount(ctx)
	maxRetry, _ := asynq.GetMaxRetry(ctx)
	status, problem := models.JobPending, "Grading attempt timed out; waiting for an automatic retry."
	if retried >= maxRetry {
		status, problem = models.JobFailed, "Grading attempt timed out; automatic retries exhausted."
	}
	if updateErr := repository.ReconcileGradingJob(job, status, problem); updateErr != nil {
		logger.Log.Error("grading_timeout_persistence_failed", "job_id", job.ID, "error", updateErr)
	}
}

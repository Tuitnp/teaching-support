package service

import (
	stderrors "errors"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/queue"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/hibiken/asynq"
)

// Redis can recover/archive a killed worker without calling its error handler.
// Reconcile on job reads/restarts rather than guessing from a wall-clock age.
func reconcileGradingJob(job *models.GradingJob) error {
	if job.Status != models.JobPending && job.Status != models.JobProcessing {
		return nil
	}
	info, err := queue.InspectAutoGrading(job.ID)
	status, problem := "", ""
	if stderrors.Is(err, asynq.ErrTaskNotFound) {
		status, problem = models.JobFailed, "Queue task is missing or belongs to a legacy run; restart grading."
	} else if err != nil {
		return err
	} else {
		switch info.State {
		case asynq.TaskStateActive:
			return nil
		case asynq.TaskStatePending, asynq.TaskStateScheduled, asynq.TaskStateRetry:
			status = models.JobPending
		case asynq.TaskStateCompleted:
			status = models.JobCompleted
		case asynq.TaskStateArchived:
			status, problem = models.JobFailed, "Queue task exhausted retries or was archived."
		default:
			return nil
		}
	}
	return repository.ReconcileGradingJob(job, status, problem)
}

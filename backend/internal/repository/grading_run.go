package repository

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrObsoleteGradingAttempt = stderrors.New("grading attempt is no longer current")

type GradingAttempt struct {
	JobID, AssignmentID, TeacherID string
	StartedAt                      time.Time
}

func GetGradingRun(ctx context.Context, id, teacherID string) (*models.GradingJob, error) {
	var job models.GradingJob
	err := database.DB.WithContext(ctx).Where("id=? AND teacher_id=?", id, teacherID).First(&job).Error
	return &job, err
}

// Enqueue is bounded by the caller. Holding the assignment lock serializes starts.
// A rejected enqueue rolls back the reservation. An accepted-but-unacknowledged
// task cannot write unless its exact new job ID also commits in PostgreSQL.
func EnqueueGradingRun(assignmentID, teacherID string, enqueue func(string, bool) error) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		assignment, err := lockAssignment(tx, assignmentID)
		if err != nil {
			return err
		}
		if assignment.AIConfigID == nil {
			return errors.ErrAIConfigNotFound
		}
		var previous models.GradingJob
		err = tx.Where("assignment_id=? AND teacher_id=?", assignmentID, teacherID).First(&previous).Error
		if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		force := previous.Status == models.JobCompleted
		if err == nil {
			if previous.Status == models.JobPending || previous.Status == models.JobProcessing {
				return errors.ErrAIGradingLimit
			}
			if err := tx.Delete(&previous).Error; err != nil {
				return err
			}
		}
		job := models.GradingJob{AssignmentID: assignmentID, TeacherID: teacherID, Status: models.JobPending}
		if err := tx.Create(&job).Error; err != nil {
			return err
		}
		return enqueue(job.ID, force)
	})
}

func BeginGradingAttempt(ctx context.Context, id, assignmentID, teacherID string) (*GradingAttempt, error) {
	var attempt *GradingAttempt
	err := database.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job models.GradingJob
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND assignment_id=? AND teacher_id=?", id, assignmentID, teacherID).First(&job).Error; err != nil {
			return err
		}
		if job.Status == models.JobCompleted || job.Status == models.JobFailed {
			return nil
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		if job.StartedAt != nil && !now.After(*job.StartedAt) {
			now = job.StartedAt.Add(time.Microsecond)
		}
		if err := tx.Model(&job).Updates(map[string]any{"status": models.JobProcessing, "started_at": now, "completed_at": nil, "error": "", "progress": 0, "processed_submissions": 0, "total_submissions": 0}).Error; err != nil {
			return err
		}
		attempt = &GradingAttempt{JobID: id, AssignmentID: assignmentID, TeacherID: teacherID, StartedAt: now}
		return nil
	})
	return attempt, err
}

func attemptQuery(db *gorm.DB, a *GradingAttempt) *gorm.DB {
	return db.Model(&models.GradingJob{}).Where("id=? AND assignment_id=? AND teacher_id=? AND started_at=? AND status=?", a.JobID, a.AssignmentID, a.TeacherID, a.StartedAt, models.JobProcessing)
}

func checkGradingAttempt(tx *gorm.DB, a *GradingAttempt) error {
	var job models.GradingJob
	err := attemptQuery(tx, a).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&job).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return ErrObsoleteGradingAttempt
	}
	return err
}

func UpdateGradingAttempt(ctx context.Context, a *GradingAttempt, total, processed int) error {
	progress := 100
	if total > 0 {
		progress = processed * 100 / total
	}
	result := attemptQuery(database.DB.WithContext(ctx), a).Updates(map[string]any{"total_submissions": total, "processed_submissions": processed, "progress": progress})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrObsoleteGradingAttempt
	}
	return nil
}

func FinishGradingAttempt(ctx context.Context, a *GradingAttempt, status, problem string) error {
	var completed any
	if status != models.JobPending {
		completed = time.Now().UTC()
	}
	values := map[string]any{"status": status, "error": problem, "completed_at": completed}
	if status == models.JobCompleted {
		values["progress"] = 100
	}
	result := attemptQuery(database.DB.WithContext(ctx), a).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrObsoleteGradingAttempt
	}
	return nil
}

// Compare the observed state/token so reconciliation cannot overwrite a newer
// attempt that started while Redis was being inspected.
func ReconcileGradingJob(job *models.GradingJob, status, problem string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := database.DB.WithContext(ctx).Model(&models.GradingJob{}).Where("id=? AND status=?", job.ID, job.Status)
	if job.StartedAt == nil {
		query = query.Where("started_at IS NULL")
	} else {
		query = query.Where("started_at=?", job.StartedAt)
	}
	values := map[string]any{"status": status, "error": problem, "completed_at": nil}
	if status == models.JobCompleted || status == models.JobFailed {
		values["completed_at"] = time.Now().UTC()
	}
	if status == models.JobCompleted {
		values["progress"] = 100
	}
	return query.Updates(values).Error
}

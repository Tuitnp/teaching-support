package repository

import (
	stderrors "errors"
	"strings"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func assignmentTags(tx *gorm.DB, names []string) ([]models.Tag, error) {
	if len(names) > 5 {
		names = names[:5]
	}
	var tags []models.Tag
	seen := map[string]bool{}
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		tag := models.Tag{Name: name}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoNothing: true}).Create(&tag).Error; err != nil {
			return nil, err
		}
		tag = models.Tag{}
		if err := tx.Where("name=?", name).First(&tag).Error; err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func CreateAssignmentContent(assignment *models.Assignment, names []string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		tags, err := assignmentTags(tx, names)
		if err != nil {
			return err
		}
		assignment.Tags = tags
		return tx.Create(assignment).Error
	})
}

// Uploads have already finished. Metadata, scalars, both joins, and the response
// query share this short DB transaction; no storage call holds these locks.
func UpdateAssignmentContent(courseID, id string, updates map[string]any, attachments []models.Attachment, names []string) (*models.Assignment, error) {
	var assignment models.Assignment
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := updateAssignment(tx, id, updates); err != nil {
			return err
		}
		if err := tx.Model(&models.Assignment{ID: id}).Association("Attachments").Replace(attachments); err != nil {
			return err
		}
		tags, err := assignmentTags(tx, names)
		if err != nil {
			return err
		}
		if err := tx.Model(&models.Assignment{ID: id}).Association("Tags").Replace(tags); err != nil {
			return err
		}
		return tx.Preload("Tags").Preload("Attachments").Where("id=? AND course_id=?", id, courseID).First(&assignment).Error
	})
	return &assignment, err
}

func ReplaceAssignmentOverride(courseID, assignmentID, studentID string, due time.Time) (*models.AssignmentOverride, error) {
	var override models.AssignmentOverride
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		assignment, err := lockAssignment(tx, assignmentID)
		if err != nil {
			return err
		}
		if assignment.CourseID != courseID {
			return errors.ErrAssignmentNotFound
		}
		if due.Before(time.Now().Add(-time.Minute)) {
			return errors.ErrInvalidDueDate
		}
		if due.Before(assignment.DueDate) {
			return errors.ErrDueDateBeforeAssignment
		}
		err = tx.Where("assignment_id=? AND student_id=?", assignmentID, studentID).Take(&override).Error
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			override = models.AssignmentOverride{AssignmentID: assignmentID, StudentID: studentID, ExtendedDueDate: due}
			return tx.Create(&override).Error
		}
		if err != nil {
			return err
		}
		if err := tx.Model(&override).Update("extended_due_date", due).Error; err != nil {
			return err
		}
		override.ExtendedDueDate = due
		return nil
	})
	return &override, err
}

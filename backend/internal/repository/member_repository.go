package repository

import (
	stderrors "errors"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

// SQL aliases in this fixed predicate are internal, never supplied by requests.
const consistentMembershipSQL = `NOT EXISTS (SELECT 1 FROM course_members conflict
 WHERE conflict.user_id=cm.user_id AND conflict.course_id=cm.course_id
 AND (conflict.role IS DISTINCT FROM cm.role OR conflict.status IS DISTINCT FROM cm.status))`

func EnrollCourseMember(member *models.CourseMember) (bool, error) {
	created := false
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var course models.Course
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id=?", member.CourseID).First(&course).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&models.CourseMember{}).Where("course_id=? AND user_id=?", member.CourseID, member.UserID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

func ActiveCourseRole(userID, courseID string) (string, error) {
	var rows []models.CourseMember
	if err := database.DB.Where("user_id=? AND course_id=?", userID, courseID).Find(&rows).Error; err != nil {
		return "", err
	}
	members, err := models.ConsistentMemberships(rows)
	if err != nil {
		return "", err
	}
	if len(members) == 0 || members[0].Status != "active" {
		return "", errors.ErrForbidden
	}
	return members[0].Role, nil
}

func GetActiveStudents(courseID string) ([]models.CourseMember, error) {
	rows, err := GetCourseMembers(courseID)
	if err != nil {
		return nil, err
	}
	rows, err = models.ConsistentMemberships(rows)
	if err != nil {
		return nil, err
	}
	students := make([]models.CourseMember, 0, len(rows))
	for _, row := range rows {
		if row.Role == "student" && row.Status == "active" {
			students = append(students, row)
		}
	}
	return students, nil
}

func GetCourseMembers(courseID string) ([]models.CourseMember, error) {
	var members []models.CourseMember

	err := database.DB.
		Where("course_id = ?", courseID).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, username, en_name, th_name, email")
		}).
		Find(&members).
		Error

	return models.DistinctMemberships(members), err
}

func GetCourseMember(courseID, memberID string) (*models.CourseMember, error) {

	var member models.CourseMember

	err := database.DB.
		Where("course_id = ? AND user_id = ?", courseID, memberID).
		First(&member).Error

	if err != nil {
		return nil, err
	}

	return &member, nil
}

func IsUserInCourse(userID, courseID string) (bool, error) {
	var member models.CourseMember

	err := database.DB.
		Where("user_id = ? AND course_id = ?", userID, courseID).
		First(&member).Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func UpdateCourseMember(courseID, memberID string, updates map[string]any) (*models.CourseMember, error) {
	var member models.CourseMember

	err := database.DB.
		Model(&models.CourseMember{}).
		Where("course_id = ? AND user_id = ?", courseID, memberID).
		Updates(updates).Error

	if err != nil {
		return nil, err
	}

	err = database.DB.
		Where("course_id = ? AND user_id = ?", courseID, memberID).
		Preload("User").
		First(&member).Error

	if err != nil {
		return nil, err
	}

	return &member, nil
}

func DeleteCourseMember(courseID, memberID string) error {
	return database.DB.
		Where("course_id = ? AND user_id = ?", courseID, memberID).
		Delete(&models.CourseMember{}).Error
}

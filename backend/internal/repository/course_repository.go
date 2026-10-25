package repository

import (
	stderrors "errors"

	"gorm.io/gorm"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/database"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

func CreateCourseWithMember(
	course *models.Course,
	member *models.CourseMember,
) (*models.Course, error) {

	var result models.Course

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Create course
		if err := tx.Create(course).Error; err != nil {
			return err
		}

		// Create member
		if err := tx.Create(member).Error; err != nil {
			return err
		}

		if err := tx.
			Preload("Teacher").
			First(&result, "id = ?", course.ID).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func GetAllCourses() ([]models.Course, error) {

	var courses []models.Course

	err := database.DB.Preload("Teacher").Find(&courses).Error
	return courses, err
}

func GetCoursesByUser(userID string) ([]models.Course, error) {

	var courses []models.Course

	err := database.DB.
		Preload("Teacher").
		Joins("JOIN course_members cm ON cm.course_id = courses.id").
		Where("cm.user_id=? AND cm.status=?", userID, "active").Where(consistentMembershipSQL).Distinct("courses.*").
		Find(&courses).Error

	return courses, err
}

func GetCourseByID(courseID string) (*models.Course, error) {
	var course models.Course

	err := database.DB.
		Preload("Teacher").
		First(&course, "id = ?", courseID).
		Error

	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.ErrCourseNotFound
	}
	return &course, nil
}

func CourseIDExists(courseID string) (bool, error) {
	var count int64
	err := database.DB.
		Model(&models.Course{}).
		Where("id = ?", courseID).
		Limit(1).
		Count(&count).
		Error
	return count > 0, err
}

func UpdateCourse(courseID string, updates map[string]any) error {
	return database.DB.
		Model(&models.Course{}).
		Where("id = ?", courseID).
		Updates(updates).
		Error
}

func DeleteCourse(courseID string) error {
	return database.DB.
		Delete(&models.Course{}, "id = ?", courseID).
		Error
}

func DeleteCourseWithDependencies(courseID string) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("course_id = ?", courseID).Delete(&models.CourseMember{}).Error; err != nil {
			return err
		}

		if err := tx.Where("course_id = ?", courseID).Delete(&models.Assignment{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&models.Course{}, "id = ?", courseID).Error; err != nil {
			return err
		}

		return nil
	})
}

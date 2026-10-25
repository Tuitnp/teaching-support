package service

import (
	stderrors "errors"
	"fmt"
	"math/rand"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
)

func generateUniqueCourseID() (string, error) {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const maxAttempts = 50

	for range maxAttempts {
		id := make([]byte, 6)

		for i := range id {
			id[i] = letters[rand.Intn(len(letters))]
		}

		candidate := string(id)

		exists, err := repository.CourseIDExists(candidate)
		if err != nil {
			return "", fmt.Errorf("failed to check course id uniqueness: %w", err)
		}

		if !exists {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique course id after %d attempts", maxAttempts)
}

func CreateCourse(userID, role string, form *models.CourseForm) (*models.ResponseCourse, error) {
	courseID, err := generateUniqueCourseID()
	if err != nil {
		return nil, err
	}

	course := &models.Course{
		ID:         courseID,
		Name:       form.Name,
		CourseCode: form.CourseCode,
		DayOfWeek:  form.DayOfWeek,
		StartTime:  form.StartTime,
		EndTime:    form.EndTime,
		Room:       form.Room,
		Credits:    form.Credits,
		Section:    form.Section,
		Semester:   form.Semester,
		TeacherID:  userID,
	}

	fmt.Println(form)

	member := &models.CourseMember{
		UserID:   userID,
		CourseID: courseID,
		Role:     role,
		Status:   "active",
	}

	newCourse, err := repository.CreateCourseWithMember(course, member)
	if err != nil {
		return nil, err
	}

	response := models.ConvertCourseToResponse(newCourse)

	return &response, nil
}

func GetCourses(userID, userRole string) ([]models.ResponseCourse, error) {
	var courses []models.Course
	var err error

	if models.HasPermission(userRole, "course:view_all") {
		courses, err = repository.GetAllCourses()

	} else if models.HasPermission(userRole, "course:view_own") {
		courses, err = repository.GetCoursesByUser(userID)

	} else {
		return nil, errors.ErrForbidden
	}

	if err != nil {
		return nil, err
	}

	response := models.ConvertCoursesToResponse(courses)

	return response, nil
}

func GetCourse(courseID string) (*models.ResponseCourse, error) {
	course, err := repository.GetCourseByID(courseID)
	if err != nil {
		return nil, err
	}

	response := models.ConvertCourseToResponse(course)

	return &response, nil
}

func UpdateCourse(
	courseID string,
	form *models.CourseForm,
) (*models.ResponseCourse, error) {

	course, err := repository.GetCourseByID(courseID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}

	if form.Name != "" {
		updates["name"] = form.Name
	}

	if form.CourseCode != "" {
		updates["course_code"] = form.CourseCode
	}

	if form.DayOfWeek != "" {
		updates["day_of_week"] = form.DayOfWeek
	}

	if form.StartTime != "" {
		updates["start_time"] = form.StartTime
	}

	if form.EndTime != "" {
		updates["end_time"] = form.EndTime
	}

	if form.Room != "" {
		updates["room"] = form.Room
	}

	if form.Credits != 0 {
		updates["credits"] = form.Credits
	}

	if form.Section != "" {
		updates["section"] = form.Section
	}

	if form.Semester != "" {
		updates["semester"] = form.Semester
	}

	if len(updates) > 0 {
		if err := repository.UpdateCourse(course.ID, updates); err != nil {
			return nil, err
		}

		course, err = repository.GetCourseByID(course.ID)
		if err != nil {
			return nil, err
		}
	}

	response := models.ConvertCourseToResponse(course)
	return &response, nil
}

func DeleteCourse(courseID string) error {
	course, err := repository.GetCourseByID(courseID)
	if err != nil {
		return err
	}

	// Get all assignments for the course
	assignments, err := repository.GetAssignmentsByCourse(courseID)
	if err != nil {
		return err
	}

	for _, assignment := range assignments {
		if err := DeleteAssignment(courseID, assignment.ID); err != nil {
			return err
		}
	}

	return repository.DeleteCourseWithDependencies(course.ID)
}

// o(n) querry (Absolutely 3N)
func EnrollCourse(
	courseID string,
	form *models.EnrollmentForms,
) (*models.ResponseEnrollmentResults, error) {

	results := &models.ResponseEnrollmentResults{
		EnrollmentResult: make([]models.ResponseEnrollmentResult, 0, len(form.UserEnroll)),
	}

	for _, enrollment := range form.UserEnroll {

		result := models.ResponseEnrollmentResult{
			Username: enrollment.Username,
			Role:     enrollment.CourseRole,
			Status:   "Success",
		}

		user, err := repository.GetUserByUsername(enrollment.Username)
		if err != nil {

			if stderrors.Is(err, errors.ErrUserNotFound) {
				result.Status = "Need Register"
			} else {
				result.Status = "Error"
			}

			results.EnrollmentResult = append(results.EnrollmentResult, result)
			continue
		}

		created, err := repository.EnrollCourseMember(&models.CourseMember{UserID: user.ID, CourseID: courseID, Role: enrollment.CourseRole, Status: "active"})
		if err != nil {
			result.Status = "Error"
		}

		if err == nil && !created {
			result.Status = "Already In Course"
		}
		results.EnrollmentResult = append(results.EnrollmentResult, result)
	}

	return results, nil
}

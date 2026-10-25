package handler

import (
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/logger"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/service"
	"github.com/gofiber/fiber/v3"
)

func RegisterCourseWithoutIDRoutes(app fiber.Router) {
	app.Post("/", createCourse)
	app.Get("/", getCourses)
}

func RegisterCourseRoutes(app fiber.Router) {
	app.Get("/", getCourse)
	app.Put("/", updateCourse)
	app.Delete("/", deleteCourse)
	app.Post("/enroll", enrollCourse)
}

func validateCourseForm(form *models.CourseForm, requireAll bool) bool {
	validDays := map[string]struct{}{
		"Monday":    {},
		"Tuesday":   {},
		"Wednesday": {},
		"Thursday":  {},
		"Friday":    {},
		"Saturday":  {},
		"Sunday":    {},
	}

	if requireAll {
		if form.Name == "" || form.CourseCode == "" || form.DayOfWeek == "" || form.StartTime == "" || form.EndTime == "" || form.Credits == 0 || form.Section == "" || form.Semester == "" {
			return false
		}
	}

	if form.DayOfWeek != "" {
		if _, ok := validDays[form.DayOfWeek]; !ok {
			return false
		}
	}

	if form.StartTime != "" {
		if _, err := time.Parse("15:04", form.StartTime); err != nil {
			return false
		}
	}

	if form.EndTime != "" {
		if _, err := time.Parse("15:04", form.EndTime); err != nil {
			return false
		}
	}

	if form.StartTime != "" && form.EndTime != "" {
		startTime, err := time.Parse("15:04", form.StartTime)
		if err != nil {
			return false
		}

		endTime, err := time.Parse("15:04", form.EndTime)
		if err != nil {
			return false
		}

		if !startTime.Before(endTime) {
			return false
		}
	}

	return true
}

func createCourse(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("course_create_attempt")

	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)

	if !models.HasPermission(role, "course:create") {
		log.Error("course_create_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.CourseForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("course_create_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	if !validateCourseForm(&form, true) {
		log.Error("course_create_failed",
			"error", errors.ErrBadRequest,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	data, err := service.CreateCourse(userID, role, &form)
	if err != nil {
		log.Error("course_create_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("course_create_success")

	return c.JSON(data)
}

func getCourses(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	role := c.Locals("user_role").(string)

	courses, err := service.GetCourses(userID, role)
	if err != nil {
		return SendError(c, err)
	}

	return c.JSON(courses)
}

func getCourse(c fiber.Ctx) error {
	courseID := c.Params("course_id")

	course, err := service.GetCourse(courseID)
	if err != nil {
		SendError(c, err)
	}

	return c.JSON(course)
}

func updateCourse(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("course_update_attempt")

	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")

	if !models.HasPermission(role, "course:update") {
		log.Error("course_update_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.CourseForm
	if err := c.Bind().Body(&form); err != nil {
		log.Error("course_update_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	if !validateCourseForm(&form, false) {
		log.Error("course_update_failed",
			"error", errors.ErrBadRequest,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	data, err := service.UpdateCourse(courseID, &form)
	if err != nil {
		log.Error("course_update_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("course_update_success",
		"course_id", courseID,
	)

	return c.JSON(data)
}

func deleteCourse(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("course_delete_attempt")

	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")

	if !models.HasPermission(role, "course:delete") {
		log.Error("course_delete_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	err := service.DeleteCourse(courseID)
	if err != nil {
		log.Error("course_delete_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("course_delete_success",
		"course_id", courseID,
	)

	return c.JSON(fiber.Map{
		"message": "course deleted",
	})
}

func enrollCourse(c fiber.Ctx) error {
	log := logger.WithRequest(c)

	log.Info("course_enroll_attempt")

	role := c.Locals("course_role").(string)
	courseID := c.Params("course_id")

	if !models.HasPermission(role, "course:enroll") {
		log.Error("course_enroll_failed",
			"error", errors.ErrForbidden,
		)
		return SendError(c, errors.ErrForbidden)
	}

	var form models.EnrollmentForms
	if err := c.Bind().Body(&form); err != nil {
		log.Error("course_enroll_failed",
			"error", err,
		)
		return SendError(c, errors.ErrBadRequest)
	}

	data, err := service.EnrollCourse(courseID, &form)
	if err != nil {
		log.Error("course_enroll_failed",
			"error", err,
		)
		return SendError(c, err)
	}

	log.Info("course_enroll_success",
		"course_id", courseID,
	)

	return c.JSON(data)
}

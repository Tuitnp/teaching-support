package middleware

import (
	stderrors "errors"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
	"github.com/gofiber/fiber/v3"
)

// SubmissionMiddleware binds every nested submission action to both URL parents.
// CourseMiddleware must run first to establish active membership and course role.
func SubmissionMiddleware(c fiber.Ctx) error {
	err := repository.CheckSubmissionInAssignment(
		c.Params("course_id"), c.Params("assignment_id"), c.Params("submission_id"),
	)
	if stderrors.Is(err, errors.ErrSubmissionNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errors.ErrSubmissionNotFound.Message})
	}
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return c.Next()
}

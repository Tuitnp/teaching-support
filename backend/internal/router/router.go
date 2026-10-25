package router

import (
	"github.com/6510615294/Tech-Support-CN101/backend/internal/handler"
	"github.com/6510615294/Tech-Support-CN101/backend/internal/middleware"
	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api")

	authGroup := api.Group("/auth")
	handler.RegisterAuthRoutes(authGroup)

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware)

	handler.RegisterMeRoutes(protected)
	handler.RegisterRunRoutes(protected)

	attachments := protected.Group("/attachments")
	handler.RegisterAttachmentRoutes(attachments)

	templates := protected.Group("/templates")
	handler.RegisterTemplateRoutes(templates)

	ai := protected.Group("/ai")
	handler.RegisterAIRoutes(ai)

	courseWithoutID := protected.Group("/courses")
	handler.RegisterCourseWithoutIDRoutes(courseWithoutID)
	courses := courseWithoutID.Group("/:course_id", middleware.CourseMiddleware)
	handler.RegisterCourseRoutes(courses)

	members := courses.Group("/members")
	handler.RegisterMemberRoutes(members)

	assignments := courses.Group("/assignments")
	handler.RegisterAssignmentRoutes(assignments)
	
	submissions := assignments.Group("/:assignment_id/submissions")
	submissions.Use("/:submission_id", middleware.SubmissionMiddleware)
	handler.RegisterSubmissionRoutes(submissions)

	comment := submissions.Group("/:submission_id/comment")
	handler.RegisterCommentRoutes(comment)
}

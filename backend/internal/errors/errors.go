package errors

type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

var (
	ErrMembershipConflict      = &AppError{409, "conflicting course membership roles/statuses; a teacher must resolve all matching rows"}
	ErrInvalidPoints           = &AppError{400, "maximum points must be a positive integer"}
	ErrInvalidSchedule         = &AppError{400, "assignment dates must satisfy start < due <= close"}
	ErrInvalidGrade            = &AppError{400, "grade must be an integer between zero and the assignment maximum"}
	ErrMaximumBelowGrade       = &AppError{400, "maximum points cannot be lower than an existing grade"}
	ErrSubmissionFileRequired  = &AppError{400, "a nonempty submission file is required"}
	ErrAIError                 = &AppError{429, "ai error"}
	ErrAIGradingLimit          = &AppError{429, "ai grading reach limit(1)"}
	ErrUserNotFound            = &AppError{404, "user not found"}
	ErrAIConfigNotFound        = &AppError{404, "ai config not found"}
	ErrAIConfigAlreadyExists   = &AppError{409, "ai config already exists"}
	ErrAICredentialNotFound    = &AppError{409, "ai credential not exists"}
	ErrAttachmentNotFound      = &AppError{404, "attachment not found"}
	ErrCommentNotFound         = &AppError{404, "comment not found"}
	ErrSubmissionNotFound      = &AppError{404, "submission not found"}
	ErrTemplateNotFound        = &AppError{404, "template not found"}
	ErrAssignmentNotFound      = &AppError{404, "assignment not found"}
	ErrCourseNotFound          = &AppError{404, "course not found"}
	ErrAssignmentNotAvailable  = &AppError{403, "assignment not available"}
	ErrForbidden               = &AppError{403, "forbidden"}
	ErrUnauthorized            = &AppError{401, "unauthorized"}
	ErrBadRequest              = &AppError{400, "bad request"}
	ErrTooManyAttachments      = &AppError{400, "too many attachment"}
	ErrAttachmentTooLarge      = &AppError{400, "attachment file size must be <= 10MB"}
	ErrSubmissionFileTooLarge  = &AppError{400, "submission file size must be <= 5MB"}
	ErrInvalidCredentials      = &AppError{400, "invalid username or password"}
	ErrTUAPI                   = &AppError{500, "internal server error (TU API)"}
	ErrCodeExecutor            = &AppError{500, "internal server error (judge0)"}
	ErrInvalidDueDate          = &AppError{400, "extended due date must be in the future"}
	ErrDueDateBeforeAssignment = &AppError{400, "extended due date must be after the assignment due date"}
	ErrCannotDeleteJob         = &AppError{400, "can only delete completed or failed jobs"}
	ErrGradingJobNotFound      = &AppError{404, "grading job not found"}
	ErrPromptNotFound          = &AppError{404, "prompt not found"}
)

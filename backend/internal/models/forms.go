package models

import (
	"time"
	// "gorm.io/gorm"
)

type LoginForm struct {
	Username string `json:"username" example:"6512345678"`
	Password string `json:"password" example:"mypassword"`
}

type CourseForm struct {
	Name       string `json:"name" form:"name"`
	CourseCode string `json:"course_code" form:"course_code"`
	DayOfWeek  string `json:"day_of_week" form:"day_of_week"`
	StartTime  string `json:"start_time" form:"start_time"`
	EndTime    string `json:"end_time" form:"end_time"`
	Room       string `json:"room" form:"room"`
	Credits    int16  `json:"credits" form:"credits"`
	Section    string `json:"section" form:"section"`
	Semester   string `json:"semester" form:"semester"`
}

type AssignmentForm struct {
	Title       string     `form:"title"`
	Description string     `form:"description"`
	Point       *int16     `form:"point"`
	StartDate   *time.Time `form:"start"`
	DueDate     *time.Time `form:"due"`
	CloseDate   *time.Time `form:"close"`
	Tags        []string   `form:"tags"`
	Attachments []string   `form:"attachments"`
	AIConfigID  *string    `form:"ai_config_id"`
	Prompt      *string    `form:"prompt"`
	Visible     bool       `form:"visible"`
}

type SubmissionForm struct {
	Answer string `json:"answer" form:"answer"`
}

type CommentForm struct {
	Comment string `json:"comment" form:"comment"`
	Visible bool   `json:"visible" form:"visible"`
}

type GradeForm struct {
	Point *int16 `json:"point" form:"point"`
}

type GradeAndCommentForm struct {
	Point   *int16 `json:"point" form:"point"`
	Comment string `json:"comment" form:"comment"`
	Visible bool   `json:"visible" form:"visible"`
}

type PythonCodeForm struct {
	SourceCode string `json:"source_code" form:"source_code"`
	Input      string `json:"stdin" form:"stdin"`
}

type EnrollmentForm struct {
	Username   string `json:"username" form:"username"`
	CourseRole string `json:"course_role" form:"course_role"`
}

type EnrollmentForms struct {
	UserEnroll []EnrollmentForm `json:"user_enroll" form:"user_enroll"`
}

type UpdateMemberForm struct {
	NewRole   string `json:"new_role" form:"new_role"`
	NewStatus string `json:"new_status" form:"new_status"`
}

type AssignmentOverrideForm struct {
	StudentID       string    `json:"student_id" form:"student_id"`
	ExtendedDueDate time.Time `json:"extended_due_date" form:"extended_due_date"`
}

type AssignmentTemplateForm struct {
	Title       string   `json:"title" form:"title"`
	Description string   `json:"description" form:"description"`
	Point       *int16   `json:"point" form:"point"`
	Tags        []string `json:"tags" form:"tags"`
	Attachments []string `json:"attachments" form:"attachments"`
	AIConfigID  *string  `form:"ai_config_id"`
	Prompt      *string  `form:"prompt"`
}

type AICredentialForm struct {
	Name        string  `json:"name" form:"name"`
	Provider    string  `json:"provider" form:"provider"`
	APIKey      string  `json:"api_key" form:"api_key"`
	BaseURL     string  `json:"base_url" form:"base_url"`
}

type AIConfigForm struct {
	Name		    string 	`json:"name" form:"name"`
	AICredentialID	string  `json:"credential_id" form:"credential_id"`
	Model       	string  `json:"model" form:"model"`
	Temperature 	float32 `json:"temperature" form:"temperature"`
}

type PromptTemplateForm struct {
	Name		    string 	`json:"name" form:"name"`
	Prompt 			string 	`json:"prompt" form:"prompt"`
}

type AISubmissionForm struct {
	SubmissionID string `json:"id"`
	Answer       string `json:"answer"`
}

type AIForm struct {
	AIConfig         ResponseAIConfig   `json:"config"`
	MaxPoint         int16              `json:"max_point"`
	AssignmentPrompt string             `json:"prompt"`
	Submissions      []AISubmissionForm `json:"submissions"`
}

type AIGradingForm struct {
	SubmissionID string `json:"id"`
	Comment      string `json:"comment"`
	Point        int16  `json:"point"`
}

type AIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

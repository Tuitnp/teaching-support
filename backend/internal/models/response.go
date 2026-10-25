package models

import "time"

type ResponseUserData struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Faculty  string `json:"faculty"`
	Role     string `json:"role"`
}

type ResponseCourse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CourseCode string `json:"course_code"`
	DayOfWeek  string `json:"day_of_week"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Room       string `json:"room"`
	Credits    int16  `json:"credits"`
	Section    string `json:"section"`
	Semester   string `json:"semester"`
	Teacher    string `json:"teacher"`
}

type ResponseAttachment struct {
	ID        string `json:"id"`
	FileName  string `json:"file_name"`
	FileType  string `json:"file_type"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

type ResponseAssignment struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Point       int16                `json:"point"`
	StartDate   string               `json:"start_date"`
	DueDate     string               `json:"due_date"`
	CloseDate   string               `json:"close_date"`
	Attachments []ResponseAttachment `json:"attachments"`
	Tags        []string             `json:"tags"`
	AIConfigID  string               `json:"ai_config_id"`
	Prompt      string               `json:"prompt,omitempty"`
	Visible     bool                 `json:"visible"`
}

type ResponseComment struct {
	ID        string `json:"id"`
	Comment   string `json:"comment"`
	CreatedBy string `json:"commentator"`
	Visible   bool   `json:"visible"`
}

type ResponseSubmission struct {
	IsCurrent    bool              `json:"is_current"`
	ID           string            `json:"id"`
	Submitter    string            `json:"submitter"`
	Answer       string            `json:"answer"`
	Point        *int16            `json:"point"`
	GradedBy     *string           `json:"graded_by"`
	AttachmentID *string           `json:"attachment_id"`
	FileName     *string           `json:"file_name"`
	Comments     []ResponseComment `json:"comments"`
}

type ResponseDetailedAssignment struct {
	Assignment  ResponseAssignment   `json:"assignment"`
	Submissions []ResponseSubmission `json:"submissions"`
}

type ResponsePythonCode struct {
	Stdout string  `json:"stdout"`
	Stderr *string `json:"stderr"`
	Status string  `json:"status"`
	Time   string  `json:"time"`
	Memory int     `json:"memory"`
}

type ScoreDistribution struct {
	RangeStart float32 `json:"range_start"`
	RangeEnd   float32 `json:"range_end"`
	Count      int     `json:"count"`
}

type ResponseAssignmentStatistic struct {
	Student        int                 `json:"students"`
	Submitted      int                 `json:"submitted"`
	Incomplete     int                 `json:"incomplete"`
	NotStarted     int                 `json:"not_started"`
	SubmissionRate float32             `json:"submission_rate"`
	Graded         int                 `json:"graded"`
	Ungraded       int                 `json:"ungrade"`
	AverageScore   float32             `json:"avg_score"`
	HighestScore   float32             `json:"highest_score"`
	LowestScore    float32             `json:"lowest_score"`
	MedianScore    float32             `json:"median_score"`
	Distribution   []ScoreDistribution `json:"distribution"`
}

type ResponseAssignmentSubmissionList struct {
	HasGrade         bool    `json:"has_grade"`
	UserID           string  `json:"user_id"`
	StudentID        string  `json:"student_id"`
	EnName           string  `json:"en_name"`
	ThName           string  `json:"th_name"`
	Email            string  `json:"email"`
	Point            int16   `json:"point"`
	Percentage       float32 `json:"percentage"`
	SubmissionStatus string  `json:"submission_status"`
}

type ResponseAssignmentSummary struct {
	Statistic      ResponseAssignmentStatistic        `json:"statistic"`
	SubmissionList []ResponseAssignmentSubmissionList `json:"submission_list"`
}

type ResponseEnrollmentResult struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type ResponseEnrollmentResults struct {
	EnrollmentResult []ResponseEnrollmentResult `json:"enrollment_result"`
}

type ResponseStudentPoint struct {
	UserID     string  `json:"user_id"`
	StudentID  string  `json:"student_id"`
	EnName     string  `json:"en_name"`
	ThName     string  `json:"th_name"`
	Email      string  `json:"email"`
	PointList  []int16 `json:"point_list"`
	Total      int16   `json:"total"`
	Percentage float32 `json:"percentage"`
	Status     string  `json:"status"`
}

type ResponseMember struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	EnName   string `json:"en_name"`
	ThName   string `json:"th_name"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	Role     string `json:"role"`
}

type ResponseAssignmentOverride struct {
	AssignmentID    string `json:"assignment_id"`
	StudentID       string `json:"student_id"`
	ExtendedDueDate string `json:"extended_due_date"`
}

type ResponseAssignmentTemplates struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Point       int16                `json:"point"`
	Attachments []ResponseAttachment `json:"attachments"`
	Tags        []string             `json:"tags"`
	AIConfigID  string               `json:"ai_config_id"`
	Prompt      string               `json:"prompt,omitempty"`
}

type ResponseShortAssignmentTemplates struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

type ResponseAssignmentTemplate struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Point       int16                `json:"point"`
	Attachments []ResponseAttachment `json:"attachments"`
	Tags        []string             `json:"tags"`
	AIConfigID  string               `json:"ai_config_id"`
	Prompt      string               `json:"prompt,omitempty"`
	Visible     bool                 `json:"visible"`
}

type ResponseAICredential struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	BaseURL   string `json:"base_url"`
	CreatedAt string `json:"created_at"`
}

type ResponseAIConfig struct {
	ID             string  `json:"id"`
	CredentialID   string  `json:"credential_id"`
	ConfigName     string  `json:"config_name"`
	CredentialName string  `json:"credential_name"`
	Model          string  `json:"model"`
	Temperature    float32 `json:"temperature"`
	CreatedAt      string  `json:"created_at"`
}

type ResponseModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ResponsePrompt struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	EnName   string `json:"en_name"`
	ThName   string `json:"th_name"`
	Role     string `json:"role"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"invalid_credentials"`
}

type ResponseAttachmentDetail struct {
	RelatedAssignments []string `json:"related_assignments"`
	RelatedTemplates   []string `json:"related_templates"`
}

type GradingJobResponse struct {
	ID                   string     `json:"id"`
	CourseID             string     `json:"course_id"`
	AssignmentID         string     `json:"assignment_id"`
	AssignmentTitle      string     `json:"assignment_title"`
	Status               string     `json:"status"`
	Progress             int16      `json:"progress"`
	TotalSubmissions     int16      `json:"total_submissions"`
	ProcessedSubmissions int16      `json:"processed_submissions"`
	Error                string     `json:"error,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type GradingJobsResponse struct {
	Jobs []GradingJobResponse `json:"jobs"`
}

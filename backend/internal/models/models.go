package models

import (
	"slices"
	"time"
	// "gorm.io/gorm"
)

type Role string

const (
	RoleTeacher           Role = "teacher"
	RoleStudent           Role = "student"
	RoleTeacherAssistance Role = "ta"
	RoleAdmin             Role = "admin"
	RoleAI                Role = "ai"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusWithdraw Status = "withdraw"
	StatusDrop     Status = "drop"
)

const (
	JobPending    = "pending"
	JobProcessing = "processing"
	JobCompleted  = "completed"
	JobFailed     = "failed"
)

var RolePermissions = map[string][]string{
	"ai": {
		"assignment:grade",
	},
	"teacher": {
		"course:create",
		"course:view_own",
		"course:update",
		"course:delete",
		"course:enroll",
		"course:export",
		"member:view_all",
		"member:update",
		"member:delete",
		"assignment:create",
		"assignment:view_all",
		"assignment:update",
		"assignment:delete",
		"assignment:comment",
		"assignment:comment_any",
		"assignment:grade",
		"assignment:export",
		"submission:view_all",
		"template:create",
		"template:view",
		"template:update",
		"template:delete",
		"file:read_all",
		"file:download_all",
		"run:python",
		"ai",
	},
	"student": {
		"course:view_own",
		"member:view_all",
		"assignment:view_visible",
		"submission:create",
		"submission:view_own",
		"submission:update",
		"submission:delete",
		"file:read_own",
	},
	"ta": {
		"course:view_own",
		"member:view_all",
		"assignment:view_all",
		"assignment:comment",
		"assignment:comment_own",
		"assignment:grade",
		"submission:view_all",
		"file:read_all",
	},
}

func HasPermission(role string, permission string) bool {
	return slices.Contains(RolePermissions[role], permission)
}

// Teachers retain final authority. TAs may supply an initial grade or review an
// AI grade, but cannot replace any human, unknown, or unattributed existing grade.
func CanManuallyGradeSubmission(role string, submission *Submission) bool {
	if !HasPermission(role, "assignment:grade") {
		return false
	}
	switch Role(role) {
	case RoleTeacher:
		return true
	case RoleTeacherAssistance:
		return (submission.Point == nil && submission.GradedBy == nil) ||
			(submission.GradedBy != nil && *submission.GradedBy == string(RoleAI))
	default:
		return false
	}
}

type User struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Username  string    `gorm:"unique;not null;index" json:"username"`
	ThName    string    `gorm:"default:ชื่อ สกุล" json:"th_name"`
	EnName    string    `gorm:"default:Name Surname" json:"en_name"`
	UserType  string    `json:"user_type"`
	Role      Role      `gorm:"type:VARCHAR(20);not null" json:"role"`
	Email     string    `gorm:"unique;not null" json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Course struct {
	ID         string    `gorm:"primaryKey;size:6" json:"id"`
	Name       string    `gorm:"not null" json:"name"`
	CourseCode string    `gorm:"type:varchar(20);not null" json:"course_code"`
	DayOfWeek  string    `gorm:"type:varchar(20);not null" json:"day_of_week"`
	StartTime  string    `gorm:"type:varchar(10);not null" json:"start_time"`
	EndTime    string    `gorm:"type:varchar(10);not null" json:"end_time"`
	Room       string    `gorm:"type:varchar(50)" json:"room"`
	Credits    int16     `gorm:"not null" json:"credits"`
	Section    string    `gorm:"not null" json:"section"`
	Semester   string    `gorm:"not null" json:"semester"`
	TeacherID  string    `gorm:"not null;index" json:"teacher_id"`
	Teacher    User      `gorm:"foreignKey:TeacherID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CourseMember struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"not null;index" json:"user_id"`
	CourseID  string    `gorm:"not null;index" json:"course_id"`
	Role      string    `gorm:"type:varchar(20);not null" json:"role"`
	Status    string    `gorm:"not null" json:"status"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"-"`
	Course    Course    `gorm:"foreignKey:CourseID;constraint:OnDelete:CASCADE" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Attachment struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	FileKey   string    `gorm:"not null" json:"-"`
	FileName  string    `json:"file_name"`
	FileType  string    `json:"file_type"`
	Size      int64     `json:"size"`
	UserID    string    `gorm:"not null;index" json:"user_id"`
	Uploader  User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"unique;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Assignment struct {
	ID          string       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	CourseID    string       `gorm:"not null;index" json:"course_id"`
	Course      Course       `gorm:"foreignKey:CourseID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Title       string       `gorm:"not null" json:"title"`
	Description string       `json:"description"`
	Point       int16        `json:"point"`
	StartDate   time.Time    `json:"start_date"`
	DueDate     time.Time    `json:"due_date"`
	CloseDate   time.Time    `json:"close_date"`
	Attachments []Attachment `gorm:"many2many:assignment_attachments;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"attachments"`
	Tags        []Tag        `gorm:"many2many:assignment_tags;constraint:OnDelete:CASCADE;" json:"tags"`
	AIConfigID  *string      `gorm:"index" json:"ai_config_id,omitempty"`
	AIConfig    *AIConfig    `gorm:"foreignKey:AIConfigID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	Prompt      *string      `json:"prompt,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Visible     bool         `gorm:"default:true" json:"visible"`
}

type AssignmentOverride struct {
	ID              string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	AssignmentID    string     `gorm:"not null" json:"assignment_id"`
	Assignment      Assignment `gorm:"foreignKey:AssignmentID;references:ID;constraint:OnDelete:CASCADE;" json:"-"`
	StudentID       string     `gorm:"not null;index" json:"student_id"`
	Student         User       `gorm:"foreignKey:StudentID;references:ID;constraint:OnDelete:CASCADE;" json:"-"`
	ExtendedDueDate time.Time  `json:"extended_due_date"`
}

type Comment struct {
	ID            string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	SubmissionID  string    `gorm:"not null;index" json:"submission_id"`
	Comment       string    `gorm:"not null" json:"comment"`
	CreatedByRole Role      `gorm:"type:VARCHAR(20);not null" json:"created_by_role"`
	CreatedBy     string    `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Visible       bool      `gorm:"default:true" json:"visible"`
}

type Submission struct {
	ID           string      `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	AssignmentID string      `gorm:"not null;index" json:"assignment_id"`
	Assignment   Assignment  `gorm:"foreignKey:AssignmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Point        *int16      `gorm:"null" json:"point,omitempty"`
	GradedBy     *string     `gorm:"type:VARCHAR(20);null" json:"graded_by,omitempty"`
	AttachmentID *string     `gorm:"null" json:"attachment_id,omitempty"`
	Attachment   *Attachment `gorm:"foreignKey:AttachmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	Comments     []Comment   `gorm:"foreignKey:SubmissionID;constraint:OnDelete:CASCADE;" json:"comments"`
	StudentID    string      `gorm:"not null;index" json:"student_id"`
	Student      User        `gorm:"foreignKey:StudentID;references:ID" json:"-"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type AssignmentTemplate struct {
	ID          string       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Title       string       `gorm:"not null" json:"title"`
	Description string       `json:"description"`
	Point       int16        `json:"point"`
	Attachments []Attachment `gorm:"many2many:assignment_template_attachments;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"attachments"`
	Tags        []Tag        `gorm:"many2many:assignment_template_tags;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"tags"`
	AIConfigID  *string      `gorm:"index" json:"ai_config_id,omitempty"`
	AIConfig    *AIConfig    `gorm:"foreignKey:AIConfigID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	Prompt      *string      `json:"prompt,omitempty"`
	CreatedBy   string       `gorm:"type:uuid" json:"created_by"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type AIConfig struct {
	ID             string       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID         string       `gorm:"not null;index" json:"user_id"`
	User           User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Name           string       `gorm:"not null" json:"name"`
	AICredentialID string       `gorm:"not null;index" json:"ai_credential_id"`
	AICredential   AICredential `gorm:"foreignKey:AICredentialID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Model          string       `gorm:"not null" json:"model"`
	Temperature    float32      `gorm:"default:0.2" json:"temperature"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type AICredential struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID          string    `gorm:"not null;index" json:"user_id"`
	User            User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Name            string    `gorm:"not null" json:"name"`
	Provider        string    `gorm:"not null" json:"provider"`
	BaseURL         string    `gorm:"not null" json:"base_url"`
	EncryptedAPIKey string    `gorm:"not null" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PromptTemplate struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"not null;index" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Name      string    `gorm:"not null" json:"name"`
	Prompt    string    `gorm:"not null" json:"prompt"`
	CreatedAt time.Time `json:"created_at"`
}

type GradingJob struct {
	ID                   string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	AssignmentID         string     `gorm:"not null;index;uniqueIndex:uniq_assignment_teacher" json:"assignment_id"`
	Assignment           Assignment `gorm:"foreignKey:AssignmentID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	TeacherID            string     `gorm:"not null;index;uniqueIndex:uniq_assignment_teacher" json:"teacher_id"`
	Teacher              User       `gorm:"foreignKey:TeacherID;references:ID;constraint:OnDelete:CASCADE;" json:"-"`
	Status               string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"`
	Progress             int16      `gorm:"default:0" json:"progress"`
	TotalSubmissions     int16      `gorm:"default:0" json:"total_submissions"`
	ProcessedSubmissions int16      `gorm:"default:0" json:"processed_submissions"`
	Error                string     `gorm:"type:text" json:"error,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

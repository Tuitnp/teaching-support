package queue

const TypeAutoGrading = "assignment:auto_grading"

type AutoGradingPayload struct {
	JobID        string `json:"job_id"`
	AssignmentID string `json:"assignment_id"`
	TeacherID    string `json:"teacher_id"`
	ForceRegade  bool   `json:"force_regrade"`
}

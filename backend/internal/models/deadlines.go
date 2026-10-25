package models

import "time"

// EffectiveDeadlines applies a student's extension without shortening access.
func EffectiveDeadlines(a *Assignment, override *AssignmentOverride) (due, close time.Time) {
	due, close = a.DueDate, a.CloseDate
	if override != nil {
		due = override.ExtendedDueDate
		if due.After(close) {
			close = due
		}
	}
	return due, close
}

// AcceptsSubmissionAt includes the exact start and close instants.
func AcceptsSubmissionAt(a *Assignment, override *AssignmentOverride, now time.Time) bool {
	_, close := EffectiveDeadlines(a, override)
	return a.Visible && !now.Before(a.StartDate) && !now.After(close)
}

// CurrentFileTime is independent of grading and comment updates. A missing
// historical timestamp stays unknown; UpdatedAt cannot reconstruct upload time.
func CurrentFileTime(s *Submission) time.Time {
	if s.AttachmentID == nil || s.Attachment == nil {
		return time.Time{}
	}
	return s.Attachment.CreatedAt
}

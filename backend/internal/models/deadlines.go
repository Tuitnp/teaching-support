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

// LatestFileSubmissions selects the latest current file per student, independent
// of query order. Detached attempts are excluded. Known timestamps precede
// unknown ones; equal or unknown timestamps use ID solely as a stable tie-break.
func LatestFileSubmissions(submissions []Submission) map[string]*Submission {
	selected := make(map[string]*Submission)
	for i := range submissions {
		s := &submissions[i]
		if s.AttachmentID == nil {
			continue
		}
		previous := selected[s.StudentID]
		if previous == nil || CurrentFileTime(s).After(CurrentFileTime(previous)) ||
			(CurrentFileTime(s).Equal(CurrentFileTime(previous)) && s.ID > previous.ID) {
			selected[s.StudentID] = s
		}
	}
	return selected
}

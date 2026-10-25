package models

import "sort"

// One assignment per slice. Creation time determines record identity, not upload
// or grading time. Unknown/tied creation times use UUID only for stability.
func CurrentSubmissions(rows []Submission) map[string]*Submission {
	selected := make(map[string]*Submission)
	for i := range rows {
		row := &rows[i]
		previous := selected[row.StudentID]
		if previous == nil || NewerSubmission(*row, *previous) {
			selected[row.StudentID] = row
		}
	}
	return selected
}

func NewerSubmission(a, b Submission) bool {
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.After(b.CreatedAt)
	}
	return a.ID > b.ID
}

func OrderSubmissionHistory(rows []Submission) {
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Student.Username != b.Student.Username {
			return a.Student.Username < b.Student.Username
		}
		if a.StudentID != b.StudentID {
			return a.StudentID < b.StudentID
		}
		return NewerSubmission(a, b)
	})
}

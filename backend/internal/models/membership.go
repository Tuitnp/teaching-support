package models

import (
	"sort"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
)

// Keep conflicting rows visible to staff, but collapse identical legacy rows.
func DistinctMemberships(rows []CourseMember) []CourseMember {
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.User.Username != b.User.Username {
			return a.User.Username < b.User.Username
		}
		if a.UserID != b.UserID {
			return a.UserID < b.UserID
		}
		if a.Role != b.Role {
			return a.Role < b.Role
		}
		if a.Status != b.Status {
			return a.Status < b.Status
		}
		return a.ID > b.ID
	})
	result := make([]CourseMember, 0, len(rows))
	for _, row := range rows {
		if len(result) > 0 {
			previous := result[len(result)-1]
			if previous.UserID == row.UserID && previous.Role == row.Role && previous.Status == row.Status {
				continue
			}
		}
		result = append(result, row)
	}
	return result
}

func ConsistentMemberships(rows []CourseMember) ([]CourseMember, error) {
	rows = DistinctMemberships(rows)
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.UserID] {
			return nil, errors.ErrMembershipConflict
		}
		seen[row.UserID] = true
	}
	return rows, nil
}

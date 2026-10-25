package models

import (
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/errors"
	"github.com/google/uuid"
)

// Only accept the canonical UUID shape understood by PostgreSQL. Parse alone
// also accepts URNs, which must not be passed through to a uuid SQL parameter.
func ValidateUUIDs(ids ...string) error {
	for _, id := range ids {
		if len(id) != 36 {
			return errors.ErrBadRequest
		}
		if _, err := uuid.Parse(id); err != nil {
			return errors.ErrBadRequest
		}
	}
	return nil
}

func ValidateAssignmentValues(point int16, start, due, close time.Time) error {
	if point <= 0 {
		return errors.ErrInvalidPoints
	}
	if start.IsZero() || due.IsZero() || close.IsZero() || !start.Before(due) || close.Before(due) {
		return errors.ErrInvalidSchedule
	}
	return nil
}

func ValidateGrade(point *int16, maximum int16) error {
	if point == nil || *point < 0 || *point > maximum {
		return errors.ErrInvalidGrade
	}
	return nil
}

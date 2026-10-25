package service

import (
	"testing"
	"time"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/repository"
)

func TestAttachmentReleaseBoundary(t *testing.T) {
	start := time.Date(2026, 10, 25, 10, 0, 0, 123456000, time.UTC)
	ref := repository.AttachmentAccessReference{Role: "student", Visible: true, StartDate: start, Handout: true}
	for _, tc := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{"before-start", start.Add(-time.Nanosecond), false},
		{"exact-start", start, true},
		{"after-start", start.Add(time.Nanosecond), true},
		{"after-close", start.Add(365 * 24 * time.Hour), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := canReadAttachmentReference(ref, tc.now); got != tc.want {
				t.Fatalf("allowed = %v, want %v", got, tc.want)
			}
		})
	}
}

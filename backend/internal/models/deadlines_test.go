package models

import (
	"testing"
	"time"
)

func TestAssignmentDeadlineRoundTrip(t *testing.T) {
	start, err := time.Parse(time.RFC3339Nano, "2026-10-25T23:59:59.123456789+07:00")
	if err != nil {
		t.Fatal(err)
	}
	a := Assignment{StartDate: start, DueDate: start.Add(time.Hour), CloseDate: start.Add(3 * time.Hour)}
	for _, extension := range []time.Duration{0, 2 * time.Hour, 4 * time.Hour} {
		var override *AssignmentOverride
		if extension != 0 {
			override = &AssignmentOverride{ExtendedDueDate: start.Add(extension)}
		}
		due, close := EffectiveDeadlines(&a, override)
		empty := []Submission{}
		responses := []ResponseAssignment{
			ConvertAssignmentToResponse(&a, override != nil, override),
			ConvertDetailedAssignmentToResponse(&a, &empty, override).Assignment,
		}
		for _, response := range responses {
			for field, pair := range map[string]struct {
				got  string
				want time.Time
			}{"start": {response.StartDate, start}, "due": {response.DueDate, due}, "close": {response.CloseDate, close}} {
				parsed, err := time.Parse(time.RFC3339Nano, pair.got)
				if err != nil || !parsed.Equal(pair.want) || pair.got[len(pair.got)-1] != 'Z' {
					t.Errorf("extension %v: %s = %q, want UTC instant %v", extension, field, pair.got, pair.want)
				}
			}
		}
	}
}

func TestSubmissionWindowBoundaries(t *testing.T) {
	start := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	a := Assignment{Visible: true, StartDate: start, DueDate: start.Add(time.Hour), CloseDate: start.Add(3 * time.Hour)}
	for _, extension := range []time.Duration{0, 2 * time.Hour, 4 * time.Hour} {
		var override *AssignmentOverride
		if extension != 0 {
			override = &AssignmentOverride{ExtendedDueDate: start.Add(extension)}
		}
		due, close := EffectiveDeadlines(&a, override)
		for _, tc := range []struct {
			at   time.Time
			want bool
		}{
			{start.Add(-time.Nanosecond), false}, {start, true}, {due, true},
			{close.Add(-time.Nanosecond), true}, {close, true}, {close.Add(time.Nanosecond), false},
		} {
			if got := AcceptsSubmissionAt(&a, override, tc.at); got != tc.want {
				t.Errorf("extension %v, time %v: got %v, want %v", extension, tc.at, got, tc.want)
			}
		}
	}
	a.Visible = false
	if AcceptsSubmissionAt(&a, nil, start) {
		t.Fatal("hidden assignment accepted a submission")
	}
}

func TestCurrentSubmissionSelection(t *testing.T) {
	attachmentID := "file"
	at := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	submissions := []Submission{
		{ID: "a", StudentID: "student", CreatedAt: at, AttachmentID: &attachmentID, Attachment: &Attachment{CreatedAt: at.Add(10 * time.Hour)}},
		{ID: "b", StudentID: "student", CreatedAt: at.Add(time.Hour), AttachmentID: &attachmentID, Attachment: &Attachment{CreatedAt: at.Add(time.Hour)}},
		{ID: "z", StudentID: "student", CreatedAt: at.Add(2 * time.Hour), UpdatedAt: at.Add(10 * time.Hour)}, // detached
		{ID: "y", StudentID: "student", AttachmentID: &attachmentID, UpdatedAt: at.Add(20 * time.Hour)},      // unknown
	}
	for range 2 {
		if got := CurrentSubmissions(submissions)["student"].ID; got != "z" {
			t.Fatalf("selected %s", got)
		}
		for i, j := 0, len(submissions)-1; i < j; i, j = i+1, j-1 {
			submissions[i], submissions[j] = submissions[j], submissions[i]
		}
	}
	unknown := []Submission{{ID: "a", StudentID: "student", AttachmentID: &attachmentID}, {ID: "b", StudentID: "student", AttachmentID: &attachmentID}}
	if got := CurrentSubmissions(unknown)["student"]; got.ID != "b" || !CurrentFileTime(got).IsZero() {
		t.Fatal("unknown time was invented or unstable")
	}
}

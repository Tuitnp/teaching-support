package worker

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/6510615294/Tech-Support-CN101/backend/internal/models"
)

var invalidGradingResponses = []struct{ name, raw string }{
	{"duplicate-index", `[{"index":0,"point":1},{"index":0,"point":9}]`},
	{"missing-item", `[{"index":0,"point":1}]`},
	{"extra-item", `[{"index":0},{"index":1},{"index":0}]`},
	{"negative-index", `[{"index":0},{"index":-1}]`},
	{"out-of-range", `[{"index":0},{"index":2}]`},
	{"all-invalid", `[{"index":8},{"index":9}]`},
	{"missing-index", `[{"point":9},{"index":1}]`},
	{"null-index", `[{"index":null},{"index":1}]`},
	{"null-item", `[null,{"index":1}]`},
	{"string-index", `[{"index":"0"},{"index":1}]`},
	{"fractional-index", `[{"index":0.5},{"index":1}]`},
	{"duplicate-index-field", `[{"index":1,"index":0},{"index":1}]`},
	{"case-duplicate-index", `[{"INDEX":1,"index":0},{"index":1}]`},
	{"conflicting-id", `[{"index":0,"id":"wrong"},{"index":1}]`},
	{"empty-id", `[{"index":0,"id":""},{"index":1}]`},
	{"null-id", `[{"index":0,"id":null},{"index":1}]`},
	{"numeric-id", `[{"index":0,"id":1},{"index":1}]`},
	{"duplicate-id-field", `[{"index":0,"id":"wrong","id":"a"},{"index":1}]`},
	{"empty-array", `[]`},
	{"null-array", `null`},
}

func TestGradingResultAttribution(t *testing.T) {
	batch := []models.Submission{{ID: "a"}, {ID: "b"}}
	for _, scenario := range invalidGradingResponses {
		t.Run(scenario.name, func(t *testing.T) {
			var results []aiGradingResult
			err := json.Unmarshal([]byte(scenario.raw), &results)
			if err == nil {
				_, err = normalizeAIGradingResults(results, batch, 10)
			}
			if err == nil {
				t.Fatal("ambiguous response accepted")
			}
		})
	}
	t.Run("shuffled-exact-mapping", func(t *testing.T) {
		var results []aiGradingResult
		if err := json.Unmarshal([]byte(`[{"index":1,"id":"b","point":7,"comment":" second "},{"index":0,"point":3,"comment":"first"}]`), &results); err != nil {
			t.Fatal(err)
		}
		mapped, err := normalizeAIGradingResults(results, batch, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(mapped) != 2 || mapped[0].SubmissionID != "a" || mapped[0].Point != 3 || mapped[0].Comment != "first" || mapped[1].SubmissionID != "b" || mapped[1].Point != 7 || mapped[1].Comment != "second" {
			t.Fatalf("incorrect mapping: %+v", mapped)
		}
	})
	t.Run("existing-score-comment-normalization", func(t *testing.T) {
		mapped, err := normalizeAIGradingResults([]aiGradingResult{{Index: 0, Point: -1}, {Index: 1, Point: 99, Comment: "  "}}, batch, 10)
		if err != nil || mapped[0].Point != 0 || mapped[1].Point != 10 || mapped[1].Comment != "No detailed feedback provided by AI." {
			t.Fatal("normalization behavior changed")
		}
	})
	t.Run("empty-batch", func(t *testing.T) {
		if _, err := normalizeAIGradingResults(nil, nil, 10); err == nil {
			t.Fatal("empty batch accepted")
		}
	})
}

func TestInvalidAIResponseDoesNotWriteGrades(t *testing.T) {
	for _, scenario := range invalidGradingResponses {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/force=%t", scenario.name, force), func(t *testing.T) {
				f := newGradingFixture(t)
				var role, point any
				if force {
					role, point = "ai", int16(2)
				}
				ids := []string{f.submission(t, role, point), f.submission(t, role, point)}
				before := map[string]string{}
				for _, id := range ids {
					f.comment(t, id, "ai", "previous feedback")
					before[id] = f.snapshot(t, id)
				}
				f.responseOverride = func([]aiGradingResult) string { return scenario.raw }
				if err := f.run(force); err == nil {
					t.Fatal("invalid provider batch succeeded")
				}
				for _, id := range ids {
					if f.snapshot(t, id) != before[id] {
						t.Fatal("rejected batch changed a grade or comment")
					}
				}
				var job models.GradingJob
				if err := f.db.First(&job, "assignment_id=?", f.assignment).Error; err != nil {
					t.Fatal(err)
				}
				if job.Status != models.JobFailed {
					t.Fatal("rejected batch was not marked failed")
				}
			})
		}
	}
}

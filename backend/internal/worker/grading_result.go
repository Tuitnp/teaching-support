package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Preserve attribution presence at the JSON boundary: zero is a valid index,
// while omitted/null indexes and repeated fields are not valid identities.
func (r *aiGradingResult) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return fmt.Errorf("grading result must be an object")
	}
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		name := strings.ToLower(token.(string))
		if seen[name] {
			return fmt.Errorf("duplicate grading field %q", name)
		}
		seen[name] = true
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
	}
	var value struct {
		Index   *int    `json:"index"`
		ID      *string `json:"id"`
		Point   int16   `json:"point"`
		Comment string  `json:"comment"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.Index == nil {
		return fmt.Errorf("grading index is required")
	}
	if seen["id"] && (value.ID == nil || *value.ID == "") {
		return fmt.Errorf("grading ID must be a nonempty string when supplied")
	}
	*r = aiGradingResult{Index: *value.Index, Point: value.Point, Comment: value.Comment}
	if value.ID != nil {
		r.ID = *value.ID
	}
	return nil
}

package storage

import (
	"fmt"

	"github.com/google/uuid"
)

// newObjectKey gives each upload a random identity independent of its display
// filename, avoiding both same-name collisions and filesystem name aliases.
func newObjectKey() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate storage key: %w", err)
	}
	return id.String(), nil
}

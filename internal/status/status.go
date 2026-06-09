// Package status reads and writes the JSON status file that records the
// outcome of the last bootconf run. The status directory lives at
// <bootconf.directory>/.bootconf/ and contains status.json.
package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/offline-lab/bootconf/internal/module"
)

// SectionStatus is an alias for module.Result so status files share the same
// type without a field-for-field copy at write time.
type SectionStatus = module.Result

// RunStatus records the overall outcome of a bootconf run, including
// per-section results and a timestamp.
type RunStatus struct {
	Timestamp time.Time       `json:"timestamp"`
	Overall   bool            `json:"overall"`
	Sections  []SectionStatus `json:"sections"`
}

// Write persists a RunStatus as JSON to <dir>/status.json.
func Write(dir string, s *RunStatus) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create status dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}
	path := filepath.Join(dir, "status.json")
	if err := os.WriteFile(path, data, 0640); err != nil {
		return fmt.Errorf("failed to write status: %w", err)
	}
	return nil
}

// Read loads the most recent RunStatus from <dir>/status.json.
func Read(dir string) (*RunStatus, error) {
	path := filepath.Join(dir, "status.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	var s RunStatus
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &s, nil
}

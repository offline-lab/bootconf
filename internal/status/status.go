package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type SectionStatus struct {
	Section  string `json:"section"`
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration,omitempty"`
}

type RunStatus struct {
	Timestamp time.Time       `json:"timestamp"`
	Overall   bool            `json:"overall"`
	Sections  []SectionStatus `json:"sections"`
}

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

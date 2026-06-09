package status

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStatusWriteRead(t *testing.T) {
	dir := t.TempDir()
	timestamp := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	original := &RunStatus{
		Timestamp: timestamp,
		Overall:   true,
		Sections: []SectionStatus{
			{Section: "system", Success: true, Message: "ok"},
			{Section: "wifi", Success: false, Error: "bad ssid"},
		},
	}

	if err := Write(dir, original); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	readBack, err := Read(dir)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if readBack.Overall != original.Overall {
		t.Errorf("overall: got %v, want %v", readBack.Overall, original.Overall)
	}
	if !readBack.Timestamp.Equal(original.Timestamp) {
		t.Errorf("timestamp: got %v, want %v", readBack.Timestamp, original.Timestamp)
	}
	if len(readBack.Sections) != len(original.Sections) {
		t.Fatalf("sections: got %d, want %d", len(readBack.Sections), len(original.Sections))
	}
	for idx, section := range readBack.Sections {
		if section.Section != original.Sections[idx].Section {
			t.Errorf("section[%d]: got %q, want %q", idx, section.Section, original.Sections[idx].Section)
		}
		if section.Success != original.Sections[idx].Success {
			t.Errorf("section[%d] success: got %v, want %v", idx, section.Success, original.Sections[idx].Success)
		}
	}
}

func TestStatusReadMissing(t *testing.T) {
	_, err := Read(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestStatusWriteCreatesDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "subdir")

	original := &RunStatus{
		Timestamp: time.Now().UTC(),
		Overall:   true,
		Sections:  []SectionStatus{},
	}

	if err := Write(dir, original); err != nil {
		t.Fatalf("write to new dir failed: %v", err)
	}

	path := filepath.Join(dir, "status.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("status.json was not created")
	}
}

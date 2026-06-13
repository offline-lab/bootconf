package files

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

// TestFilesCopyFile verifies that an enabled module copies source content
// to the destination and applies the requested permissions.
func TestFilesCopyFile(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "test.conf")
	sourceContent := []byte("key=value\n")
	if err := os.WriteFile(sourceFile, sourceContent, 0644); err != nil {
		t.Fatal(err)
	}

	destFile := filepath.Join(destDir, "etc", "test.conf")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: sourceFile, Destination: destFile, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Section != "files" {
		t.Errorf("expected section 'files', got %q", result.Section)
	}
	if result.Message != "wrote 1 file(s)" {
		t.Errorf("expected message 'copied 1 file(s)', got %q", result.Message)
	}

	copiedContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}
	if string(copiedContent) != string(sourceContent) {
		t.Errorf("expected content %q, got %q", sourceContent, copiedContent)
	}

	fileInfo, statErr := os.Stat(destFile)
	if statErr != nil {
		t.Fatalf("failed to stat dest file: %v", statErr)
	}
	if fileInfo.Mode().Perm() != 0640 {
		t.Errorf("expected perm 0640, got %04o", fileInfo.Mode().Perm())
	}
}

// TestFilesExistingFileGetsNewSuffix verifies that when the destination
// already exists, the new content is written to dest + ".new" and the
// original file is left untouched.
func TestFilesExistingFileGetsNewSuffix(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "app.conf")
	newContent := []byte("new content\n")
	if err := os.WriteFile(sourceFile, newContent, 0644); err != nil {
		t.Fatal(err)
	}

	destSubdir := filepath.Join(destDir, "etc")
	if err := os.MkdirAll(destSubdir, 0750); err != nil {
		t.Fatal(err)
	}

	existingFile := filepath.Join(destSubdir, "app.conf")
	oldContent := []byte("old content\n")
	if err := os.WriteFile(existingFile, oldContent, 0644); err != nil {
		t.Fatal(err)
	}

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: sourceFile, Destination: existingFile, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	redirectedFile := existingFile + ".new"
	redirectedContent, err := os.ReadFile(redirectedFile)
	if err != nil {
		t.Fatalf("failed to read .new file: %v", err)
	}
	if string(redirectedContent) != string(newContent) {
		t.Errorf("expected new content in .new file, got %q", redirectedContent)
	}

	originalContent, _ := os.ReadFile(existingFile)
	if string(originalContent) != string(oldContent) {
		t.Error("existing file should not be modified")
	}
}

// TestFilesChmodApplied verifies that chmod="755" is correctly applied
// to the destination file.
func TestFilesChmodApplied(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "script.sh")
	if err := os.WriteFile(sourceFile, []byte("#!/bin/sh\n"), 0644); err != nil {
		t.Fatal(err)
	}

	destFile := filepath.Join(destDir, "usr", "local", "bin", "script.sh")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: sourceFile, Destination: destFile, Chmod: "755"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	fileInfo, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if fileInfo.Mode().Perm() != 0755 {
		t.Errorf("expected perm 0755, got %04o", fileInfo.Mode().Perm())
	}
}

// TestFilesMissingSource verifies that a missing source file produces
// an error and no destination file is created.
func TestFilesMissingSource(t *testing.T) {
	destDir := t.TempDir()

	destFile := filepath.Join(destDir, "etc", "file.conf")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: "/nonexistent/path/file.conf", Destination: destFile, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if result.Success {
		t.Error("expected failure for missing source")
	}
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}

	if _, err := os.Stat(destFile); err == nil {
		t.Error("dest file should not exist when source is missing")
	}
}

// TestFilesDryRunNoWrites verifies that dry-run mode returns success
// without writing any files.
func TestFilesDryRunNoWrites(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "data.txt")
	if err := os.WriteFile(sourceFile, []byte("data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	destFile := filepath.Join(destDir, "etc", "data.txt")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: sourceFile, Destination: destFile, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), true, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Message != "would write 1 file(s) (dry-run)" {
		t.Errorf("unexpected message: %q", result.Message)
	}

	if _, err := os.Stat(destFile); err == nil {
		t.Error("no files should be written during dry run")
	}
}

// TestFilesDisabled verifies that a disabled module returns immediately
// with "files disabled" and no processing occurs.
func TestFilesDisabled(t *testing.T) {
	module := New(config.FilesConfig{Enabled: false})

	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Message != "files disabled" {
		t.Errorf("unexpected message: %q", result.Message)
	}
}

// TestFilesEmptyList verifies that an enabled module with no file entries
// succeeds without errors.
func TestFilesEmptyList(t *testing.T) {
	module := New(config.FilesConfig{Enabled: true})

	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Section != "files" {
		t.Errorf("expected section 'files', got %q", result.Section)
	}
}

// TestFilesMultipleFiles verifies that three file entries are all copied
// correctly with their respective content.
func TestFilesMultipleFiles(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	firstSource := filepath.Join(sourceDir, "first.conf")
	secondSource := filepath.Join(sourceDir, "second.conf")
	thirdSource := filepath.Join(sourceDir, "third.conf")

	if err := os.WriteFile(firstSource, []byte("first\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secondSource, []byte("second\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(thirdSource, []byte("third\n"), 0644); err != nil {
		t.Fatal(err)
	}

	firstDest := filepath.Join(destDir, "etc", "first.conf")
	secondDest := filepath.Join(destDir, "opt", "second.conf")
	thirdDest := filepath.Join(destDir, "var", "third.conf")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: firstSource, Destination: firstDest, Chmod: "640"},
			{Source: secondSource, Destination: secondDest, Chmod: "640"},
			{Source: thirdSource, Destination: thirdDest, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Message != "wrote 3 file(s)" {
		t.Errorf("expected message 'copied 3 file(s)', got %q", result.Message)
	}

	firstContent, err := os.ReadFile(firstDest)
	if err != nil {
		t.Fatalf("failed to read first dest: %v", err)
	}
	if string(firstContent) != "first\n" {
		t.Errorf("expected 'first\\n', got %q", firstContent)
	}

	secondContent, err := os.ReadFile(secondDest)
	if err != nil {
		t.Fatalf("failed to read second dest: %v", err)
	}
	if string(secondContent) != "second\n" {
		t.Errorf("expected 'second\\n', got %q", secondContent)
	}

	thirdContent, err := os.ReadFile(thirdDest)
	if err != nil {
		t.Fatalf("failed to read third dest: %v", err)
	}
	if string(thirdContent) != "third\n" {
		t.Errorf("expected 'third\\n', got %q", thirdContent)
	}
}

// TestFilesDefaultChmod640 verifies that chmod "640" (the default applied
// by config.SetDefaults) produces the expected 0640 permissions.
func TestFilesDefaultChmod640(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "default.conf")
	if err := os.WriteFile(sourceFile, []byte("defaults\n"), 0644); err != nil {
		t.Fatal(err)
	}

	destFile := filepath.Join(destDir, "etc", "default.conf")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: sourceFile, Destination: destFile, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	fileInfo, err := os.Stat(destFile)
	if err != nil {
		t.Fatalf("failed to stat dest: %v", err)
	}
	if fileInfo.Mode().Perm() != 0640 {
		t.Errorf("expected default perm 0640, got %04o", fileInfo.Mode().Perm())
	}
}

// TestFilesDestinationDirCreated verifies that nested non-existent
// destination directories are created and the file is copied.
func TestFilesDestinationDirCreated(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	sourceFile := filepath.Join(sourceDir, "deep.conf")
	sourceContent := []byte("deep content\n")
	if err := os.WriteFile(sourceFile, sourceContent, 0644); err != nil {
		t.Fatal(err)
	}

	nestedDest := filepath.Join(destDir, "a", "b", "c", "d", "deep.conf")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: sourceFile, Destination: nestedDest, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	copiedContent, err := os.ReadFile(nestedDest)
	if err != nil {
		t.Fatalf("failed to read nested dest: %v", err)
	}
	if string(copiedContent) != string(sourceContent) {
		t.Errorf("expected %q, got %q", sourceContent, copiedContent)
	}

	parentDir := filepath.Dir(nestedDest)
	dirInfo, statErr := os.Stat(parentDir)
	if statErr != nil {
		t.Fatalf("parent directory not created: %v", statErr)
	}
	if !dirInfo.IsDir() {
		t.Error("parent path is not a directory")
	}
}

// TestFilesMultipleErrors verifies that two entries with missing sources
// both produce errors and the result reports the correct count.
func TestFilesMultipleErrors(t *testing.T) {
	destDir := t.TempDir()

	firstDest := filepath.Join(destDir, "etc", "first.conf")
	secondDest := filepath.Join(destDir, "etc", "second.conf")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: "/nonexistent/first.conf", Destination: firstDest, Chmod: "640"},
			{Source: "/nonexistent/second.conf", Destination: secondDest, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if result.Success {
		t.Error("expected failure when all sources are missing")
	}
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}
	if result.Message != "completed with 2 errors" {
		t.Errorf("expected message 'completed with 2 errors', got %q", result.Message)
	}

	if _, err := os.Stat(firstDest); err == nil {
		t.Error("first dest file should not exist")
	}
	if _, err := os.Stat(secondDest); err == nil {
		t.Error("second dest file should not exist")
	}
}

// TestFilesPartialSuccess verifies that with 3 entries where 1 has a
// missing source, the 2 valid files are still copied and errors are
// reported for the failed entry.
func TestFilesPartialSuccess(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	validSourceFirst := filepath.Join(sourceDir, "valid1.conf")
	validSourceSecond := filepath.Join(sourceDir, "valid2.conf")

	if err := os.WriteFile(validSourceFirst, []byte("valid1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validSourceSecond, []byte("valid2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	validDestFirst := filepath.Join(destDir, "etc", "valid1.conf")
	validDestSecond := filepath.Join(destDir, "etc", "valid2.conf")
	invalidDest := filepath.Join(destDir, "etc", "missing.conf")

	module := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Source: validSourceFirst, Destination: validDestFirst, Chmod: "640"},
			{Source: "/nonexistent/missing.conf", Destination: invalidDest, Chmod: "640"},
			{Source: validSourceSecond, Destination: validDestSecond, Chmod: "640"},
		},
	})

	result := module.Run(context.Background(), false, false)

	if result.Success {
		t.Error("expected failure due to missing source")
	}
	if result.Message != "completed with 1 errors" {
		t.Errorf("expected message 'completed with 1 errors', got %q", result.Message)
	}
	if result.Error == "" {
		t.Error("expected non-empty error string")
	}

	firstContent, err := os.ReadFile(validDestFirst)
	if err != nil {
		t.Fatalf("valid first file should exist: %v", err)
	}
	if string(firstContent) != "valid1\n" {
		t.Errorf("expected 'valid1\\n', got %q", firstContent)
	}

	secondContent, err := os.ReadFile(validDestSecond)
	if err != nil {
		t.Fatalf("valid second file should exist: %v", err)
	}
	if string(secondContent) != "valid2\n" {
		t.Errorf("expected 'valid2\\n', got %q", secondContent)
	}

	if _, statErr := os.Stat(invalidDest); statErr == nil {
		t.Error("dest for missing source should not exist")
	}
}

// TestFilesContentWrite verifies that a content entry writes the inline string
// to the destination with correct permissions.
func TestFilesContentWrite(t *testing.T) {
	destDir := t.TempDir()
	destFile := filepath.Join(destDir, "etc", "motd")

	result := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Content: "Welcome to bootconf\n", Destination: destFile, Chmod: "644"},
		},
	}).Run(context.Background(), false, false)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if result.Message != "wrote 1 file(s)" {
		t.Errorf("unexpected message: %q", result.Message)
	}

	got, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("destination not written: %v", err)
	}
	if string(got) != "Welcome to bootconf\n" {
		t.Errorf("unexpected content: %q", string(got))
	}
}

// TestFilesContentDryRun verifies no file is written in dry-run mode.
func TestFilesContentDryRun(t *testing.T) {
	destDir := t.TempDir()
	destFile := filepath.Join(destDir, "etc", "motd")

	result := New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Content: "Hello\n", Destination: destFile, Chmod: "640"},
		},
	}).Run(context.Background(), true, false)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if _, err := os.Stat(destFile); err == nil {
		t.Error("dry-run must not write destination file")
	}
}

// TestFilesContentExistingGetsNewSuffix verifies that a content entry does
// not overwrite an existing destination.
func TestFilesContentExistingGetsNewSuffix(t *testing.T) {
	destDir := t.TempDir()
	destFile := filepath.Join(destDir, "motd")

	if err := os.WriteFile(destFile, []byte("old\n"), 0640); err != nil {
		t.Fatal(err)
	}

	New(config.FilesConfig{
		Enabled: true,
		Files: []config.FileEntry{
			{Content: "new\n", Destination: destFile, Chmod: "640"},
		},
	}).Run(context.Background(), false, false)

	original, _ := os.ReadFile(destFile)
	if string(original) != "old\n" {
		t.Error("original file must not be overwritten")
	}
	newContent, err := os.ReadFile(destFile + ".new")
	if err != nil {
		t.Fatalf("dest.new not written: %v", err)
	}
	if string(newContent) != "new\n" {
		t.Errorf("unexpected content in dest.new: %q", string(newContent))
	}
}

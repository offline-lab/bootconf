package unitrun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

func TestUnitRunDisabled(t *testing.T) {
	result := New(config.UnitRunConfig{Enabled: false}).Run(context.Background(), false)
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	if result.Message != "unitrun disabled" {
		t.Errorf("unexpected message: %q", result.Message)
	}
}

func TestUnitRunDryRunNoWrites(t *testing.T) {
	scriptDir := t.TempDir()
	unitsDir := t.TempDir()

	module := New(config.UnitRunConfig{
		Enabled:   true,
		Directory: scriptDir,
		Units: []config.UnitEntry{
			{Name: "test-task", Enabled: true, Command: "echo hello\n"},
		},
	})
	module.unitsDir = unitsDir

	result := module.Run(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}

	entries, _ := os.ReadDir(scriptDir)
	if len(entries) != 0 {
		t.Errorf("dry-run must not write to script directory, found: %v", entries)
	}
	entries, _ = os.ReadDir(unitsDir)
	if len(entries) != 0 {
		t.Errorf("dry-run must not write to units directory, found: %v", entries)
	}
}

func TestUnitRunProvisionWritesFiles(t *testing.T) {
	scriptDir := t.TempDir()
	unitsDir := t.TempDir()

	module := New(config.UnitRunConfig{
		Enabled:   true,
		Directory: scriptDir,
		Units: []config.UnitEntry{
			{
				Name:    "my-task",
				Enabled: true,
				Dependencies: []string{
					"After=multi-user.target",
					"Before=shutdown.target",
				},
				Command: "echo hello\nexit 0\n",
			},
		},
	})
	module.unitsDir = unitsDir

	// systemctl enable will fail in CI but file writing happens before that.
	// We verify file content and accept the systemctl error.
	module.Run(context.Background(), false)

	scriptPath := filepath.Join(scriptDir, "my-task.sh")
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("script file not written: %v", err)
	}
	if !strings.HasPrefix(string(scriptContent), "#!/bin/bash\n") {
		t.Errorf("script missing shebang, got: %q", string(scriptContent[:20]))
	}
	if !strings.Contains(string(scriptContent), "echo hello") {
		t.Errorf("script missing command body")
	}

	serviceFilePath := filepath.Join(unitsDir, "bootconf-my-task.service")
	serviceContent, err := os.ReadFile(serviceFilePath)
	if err != nil {
		t.Fatalf("unit file not written: %v", err)
	}
	serviceText := string(serviceContent)
	for _, expected := range []string{
		"Description=Bootconf Unit Task my-task",
		"DefaultDependencies=no",
		"After=multi-user.target",
		"Before=shutdown.target",
		"Type=oneshot",
		"ExecStart=" + scriptPath,
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(serviceText, expected) {
			t.Errorf("unit file missing %q\ngot:\n%s", expected, serviceText)
		}
	}
}

func TestUnitRunRemoveDeletesFiles(t *testing.T) {
	scriptDir := t.TempDir()
	unitsDir := t.TempDir()

	scriptPath := filepath.Join(scriptDir, "old-task.sh")
	serviceFilePath := filepath.Join(unitsDir, "bootconf-old-task.service")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\n"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(serviceFilePath, []byte("[Unit]\n"), 0644); err != nil {
		t.Fatal(err)
	}

	module := New(config.UnitRunConfig{
		Enabled:   true,
		Directory: scriptDir,
		Units: []config.UnitEntry{
			{Name: "old-task", Enabled: false},
		},
	})
	module.unitsDir = unitsDir

	// systemctl disable may fail but file removal still runs.
	module.Run(context.Background(), false)

	if _, err := os.Stat(serviceFilePath); err == nil {
		t.Error("unit file should have been removed")
	}
	if _, err := os.Stat(scriptPath); err == nil {
		t.Error("script file should have been removed")
	}
}

func TestUnitRunFirstBootInjectsCondition(t *testing.T) {
	scriptDir := t.TempDir()
	unitsDir := t.TempDir()

	module := New(config.UnitRunConfig{
		Enabled:   true,
		Directory: scriptDir,
		Units: []config.UnitEntry{
			{Name: "first-boot-task", Enabled: true, FirstBoot: true, Command: "echo setup\n"},
		},
	})
	module.unitsDir = unitsDir

	module.Run(context.Background(), false)

	serviceFilePath := filepath.Join(unitsDir, "bootconf-first-boot-task.service")
	content, err := os.ReadFile(serviceFilePath)
	if err != nil {
		t.Fatalf("unit file not written: %v", err)
	}
	if !strings.Contains(string(content), "ConditionFirstBoot=yes") {
		t.Errorf("unit file missing ConditionFirstBoot=yes:\n%s", string(content))
	}
}

func TestRenderServiceFile(t *testing.T) {
	got := renderServiceFile("my-task", []string{"After=network.target", "Before=shutdown.target"}, "/data/scripts/my-task.sh", "")

	for _, expected := range []string{
		"[Unit]\n",
		"Description=Bootconf Unit Task my-task\n",
		"DefaultDependencies=no\n",
		"After=network.target\n",
		"Before=shutdown.target\n",
		"[Service]\n",
		"Type=oneshot\n",
		"ExecStart=/data/scripts/my-task.sh\n",
		"[Install]\n",
		"WantedBy=multi-user.target\n",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("missing %q in rendered unit:\n%s", expected, got)
		}
	}

	if strings.Contains(got, "Environment=PATH") {
		t.Errorf("empty path should not write Environment=PATH line, got:\n%s", got)
	}
}

func TestRenderServiceFileWithPath(t *testing.T) {
	got := renderServiceFile("my-task", nil, "/scripts/my-task.sh", "/usr/lib/framework/bin")

	expected := "Environment=PATH=/usr/lib/framework/bin:/usr/sbin:/usr/bin:/sbin:/bin\n"
	if !strings.Contains(got, expected) {
		t.Errorf("missing %q in rendered unit:\n%s", expected, got)
	}
}

package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

func TestServicesEnableCreatesSentinel(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")

	svcConfig := config.ServicesConfig{
		Directory: servicesDir,
		Services: []config.ServiceEntry{
			{Name: "sshd", Enabled: true, Sentinel: true},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	sentinel := filepath.Join(servicesDir, "sshd")
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("sentinel file not created: %v", err)
	}
}

func TestServicesDisableRemovesSentinel(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0750); err != nil {
		t.Fatal(err)
	}

	sentinel := filepath.Join(servicesDir, "telnetd")
	if err := os.WriteFile(sentinel, nil, 0640); err != nil {
		t.Fatal(err)
	}

	svcConfig := config.ServicesConfig{
		Directory: servicesDir,
		Services: []config.ServiceEntry{
			{Name: "telnetd", Enabled: false, Sentinel: true},
		},
	}
	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatal("sentinel file should have been removed")
	}
}

func TestServicesSentinelFalseNoFile(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")

	svcConfig := config.ServicesConfig{
		Directory: servicesDir,
		Services: []config.ServiceEntry{
			{Name: "cron", Enabled: true, Sentinel: false},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	sentinel := filepath.Join(servicesDir, "cron")
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Error("no sentinel should be created when sentinel=false")
	}
}

func TestServicesDisabledSentinelFalseNoRemove(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0750); err != nil {
		t.Fatal(err)
	}

	existingFile := filepath.Join(servicesDir, "legacy")
	if err := os.WriteFile(existingFile, []byte("should remain"), 0640); err != nil {
		t.Fatal(err)
	}

	svcConfig := config.ServicesConfig{
		Directory: servicesDir,
		Services: []config.ServiceEntry{
			{Name: "legacy", Enabled: false, Sentinel: false},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatalf("file should not have been removed: %v", err)
	}
	if string(data) != "should remain" {
		t.Errorf("file was modified: got %q", string(data))
	}
}

func TestServicesCopyDefaultConfig(t *testing.T) {
	dir := t.TempDir()

	sourceFile := filepath.Join(dir, "source", "sshd.cfg")
	if err := os.MkdirAll(filepath.Dir(sourceFile), 0750); err != nil {
		t.Fatal(err)
	}
	sourceContent := []byte("Port 22\nPermitRootLogin no\n")
	if err := os.WriteFile(sourceFile, sourceContent, 0644); err != nil {
		t.Fatal(err)
	}

	destFile := filepath.Join(dir, "etc", "ssh", "sshd_config")
	svcConfig := config.ServicesConfig{
		Directory: filepath.Join(dir, "services"),
		Services: []config.ServiceEntry{
			{
				Name:     "sshd",
				Enabled:  true,
				Sentinel: true,
				DefaultConfig: config.DefaultConfig{
					Copy:        true,
					Source:      sourceFile,
					Destination: destFile,
				},
			},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("destination file not created: %v", err)
	}
	if string(data) != string(sourceContent) {
		t.Fatalf("content mismatch: got %q, want %q", string(data), string(sourceContent))
	}
}

func TestServicesCopyExistingFileGetsNewSuffix(t *testing.T) {
	dir := t.TempDir()

	sourceFile := filepath.Join(dir, "source", "app.cfg")
	if err := os.MkdirAll(filepath.Dir(sourceFile), 0750); err != nil {
		t.Fatal(err)
	}
	newContent := []byte("new config content\n")
	if err := os.WriteFile(sourceFile, newContent, 0644); err != nil {
		t.Fatal(err)
	}

	destDir := filepath.Join(dir, "etc", "app")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		t.Fatal(err)
	}
	oldContent := []byte("old config content\n")
	destFile := filepath.Join(destDir, "config")
	if err := os.WriteFile(destFile, oldContent, 0644); err != nil {
		t.Fatal(err)
	}

	svcConfig := config.ServicesConfig{
		Directory: filepath.Join(dir, "services"),
		Services: []config.ServiceEntry{
			{
				Name:     "app",
				Enabled:  true,
				Sentinel: true,
				DefaultConfig: config.DefaultConfig{
					Copy:        true,
					Source:      sourceFile,
					Destination: destFile,
				},
			},
		},
	}
	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	newFile := destFile + ".new"
	data, err := os.ReadFile(newFile)
	if err != nil {
		t.Fatalf(".new file not created: %v", err)
	}
	if string(data) != string(newContent) {
		t.Fatalf("content mismatch: got %q, want %q", string(data), string(newContent))
	}

	existing, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("original file missing: %v", err)
	}
	if string(existing) != string(oldContent) {
		t.Fatal("original file was modified")
	}
}

func TestServicesCopyMissingSource(t *testing.T) {
	dir := t.TempDir()

	missingSource := filepath.Join(dir, "nonexistent", "source.cfg")
	destFile := filepath.Join(dir, "etc", "app", "config")

	svcConfig := config.ServicesConfig{
		Directory: filepath.Join(dir, "services"),
		Services: []config.ServiceEntry{
			{
				Name:    "app",
				Enabled: true,
				DefaultConfig: config.DefaultConfig{
					Copy:        true,
					Source:      missingSource,
					Destination: destFile,
				},
			},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if result.Success {
		t.Fatal("expected failure when source file is missing")
	}

	if !strings.Contains(result.Error, "failed to open source") {
		t.Errorf("error should mention source failure, got: %s", result.Error)
	}
}

func TestServicesDryRunNoWrites(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	sourceFile := filepath.Join(dir, "src", "data.cfg")
	if err := os.MkdirAll(filepath.Dir(sourceFile), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	svcConfig := config.ServicesConfig{
		Directory: servicesDir,
		Services: []config.ServiceEntry{
			{Name: "sshd", Enabled: true, Sentinel: true},
			{Name: "telnetd", Enabled: false, Sentinel: true},
			{
				Name:    "app",
				Enabled: true,
				DefaultConfig: config.DefaultConfig{
					Copy:        true,
					Source:      sourceFile,
					Destination: filepath.Join(dir, "dest", "data.cfg"),
				},
			},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	sentinel := filepath.Join(servicesDir, "sshd")
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatal("dry-run should not create sentinel files")
	}

	destFile := filepath.Join(dir, "dest", "data.cfg")
	if _, err := os.Stat(destFile); !os.IsNotExist(err) {
		t.Fatal("dry-run should not copy files")
	}
}

func TestServicesEmptyList(t *testing.T) {
	dir := t.TempDir()
	svcConfig := config.ServicesConfig{
		Directory: filepath.Join(dir, "services"),
	}
	svc := NewServicesModule(svcConfig)

	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Message != "processed 0 service(s)" {
		t.Fatalf("unexpected message: %s", result.Message)
	}
}

func TestServicesMultipleServices(t *testing.T) {
	dir := t.TempDir()
	servicesDir := filepath.Join(dir, "services")
	if err := os.MkdirAll(servicesDir, 0750); err != nil {
		t.Fatal(err)
	}

	removeSentinel := filepath.Join(servicesDir, "telnetd")
	if err := os.WriteFile(removeSentinel, nil, 0640); err != nil {
		t.Fatal(err)
	}

	svcConfig := config.ServicesConfig{
		Directory: servicesDir,
		Services: []config.ServiceEntry{
			{Name: "sshd", Enabled: true, Sentinel: true},
			{Name: "cron", Enabled: true, Sentinel: false},
			{Name: "telnetd", Enabled: false, Sentinel: true},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	sshSentinel := filepath.Join(servicesDir, "sshd")
	if _, err := os.Stat(sshSentinel); os.IsNotExist(err) {
		t.Error("sshd sentinel should have been created")
	}

	cronSentinel := filepath.Join(servicesDir, "cron")
	if _, err := os.Stat(cronSentinel); !os.IsNotExist(err) {
		t.Error("cron sentinel should not exist (sentinel=false)")
	}

	if _, err := os.Stat(removeSentinel); !os.IsNotExist(err) {
		t.Error("telnetd sentinel should have been removed")
	}
}

func TestServicesMultipleEnabledWithCopy(t *testing.T) {
	dir := t.TempDir()

	sshSource := filepath.Join(dir, "src", "sshd.cfg")
	if err := os.MkdirAll(filepath.Dir(sshSource), 0750); err != nil {
		t.Fatal(err)
	}
	sshContent := []byte("Port 22\n")
	if err := os.WriteFile(sshSource, sshContent, 0644); err != nil {
		t.Fatal(err)
	}

	appSource := filepath.Join(dir, "src", "app.cfg")
	appContent := []byte("mode=production\n")
	if err := os.WriteFile(appSource, appContent, 0644); err != nil {
		t.Fatal(err)
	}

	sshDest := filepath.Join(dir, "etc", "ssh", "sshd_config")
	appDest := filepath.Join(dir, "etc", "app", "config")

	svcConfig := config.ServicesConfig{
		Directory: filepath.Join(dir, "services"),
		Services: []config.ServiceEntry{
			{
				Name:     "sshd",
				Enabled:  true,
				Sentinel: true,
				DefaultConfig: config.DefaultConfig{
					Copy:        true,
					Source:      sshSource,
					Destination: sshDest,
				},
			},
			{
				Name:     "app",
				Enabled:  true,
				Sentinel: true,
				DefaultConfig: config.DefaultConfig{
					Copy:        true,
					Source:      appSource,
					Destination: appDest,
				},
			},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	sshData, err := os.ReadFile(sshDest)
	if err != nil {
		t.Fatalf("ssh destination file not created: %v", err)
	}
	if string(sshData) != string(sshContent) {
		t.Errorf("ssh content mismatch: got %q, want %q", string(sshData), string(sshContent))
	}

	appData, err := os.ReadFile(appDest)
	if err != nil {
		t.Fatalf("app destination file not created: %v", err)
	}
	if string(appData) != string(appContent) {
		t.Errorf("app content mismatch: got %q, want %q", string(appData), string(appContent))
	}

	for _, name := range []string{"sshd", "app"} {
		sentinel := filepath.Join(dir, "services", name)
		if _, err := os.Stat(sentinel); os.IsNotExist(err) {
			t.Errorf("%s sentinel not created", name)
		}
	}
}

func TestServicesCopyCreatesDestinationDir(t *testing.T) {
	dir := t.TempDir()

	sourceFile := filepath.Join(dir, "src", "data.cfg")
	if err := os.MkdirAll(filepath.Dir(sourceFile), 0750); err != nil {
		t.Fatal(err)
	}
	sourceContent := []byte("key=value\n")
	if err := os.WriteFile(sourceFile, sourceContent, 0644); err != nil {
		t.Fatal(err)
	}

	deepDestDir := filepath.Join(dir, "deep", "nested", "config", "path")
	destFile := filepath.Join(deepDestDir, "app.cfg")

	svcConfig := config.ServicesConfig{
		Directory: filepath.Join(dir, "services"),
		Services: []config.ServiceEntry{
			{
				Name:    "app",
				Enabled: true,
				DefaultConfig: config.DefaultConfig{
					Copy:        true,
					Source:      sourceFile,
					Destination: destFile,
				},
			},
		},
	}

	svc := NewServicesModule(svcConfig)
	result := svc.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("destination file not created: %v", err)
	}
	if string(data) != string(sourceContent) {
		t.Errorf("content mismatch: got %q, want %q", string(data), string(sourceContent))
	}

	dirInfo, err := os.Stat(deepDestDir)
	if err != nil {
		t.Fatalf("destination directory not created: %v", err)
	}
	if !dirInfo.IsDir() {
		t.Fatal("destination path is not a directory")
	}
}

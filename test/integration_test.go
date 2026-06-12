package test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
	"github.com/offline-lab/bootconf/internal/module/files"
	"github.com/offline-lab/bootconf/internal/module/services"
	"github.com/offline-lab/bootconf/internal/module/ssh"
	"github.com/offline-lab/bootconf/internal/module/system"
	"github.com/offline-lab/bootconf/internal/module/users"
	"github.com/offline-lab/bootconf/internal/module/wifi"
	"github.com/offline-lab/bootconf/internal/status"
)

const (
	testSSID         = "testnet"
	testPasswordHash = "a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3"
	testCountry      = "NL"
	testUserName     = "testadmin"
	testSSHPublicKey = "ssh-ed25519 AAAA testadmin@testhost"
	testFileContent  = "integration test content\n"
	testServiceName  = "testservice"
	uidStart         = 2000
)

type testEnvironment struct {
	baseDir    string
	homeDir    string
	sourceFile string
	cfg        *config.Config
}

func newTestEnvironment(t *testing.T) *testEnvironment {
	t.Helper()

	baseDir := t.TempDir()
	homeDir := filepath.Join(t.TempDir(), "home", testUserName)

	sourceDir := t.TempDir()
	sourceFile := filepath.Join(sourceDir, "testfile.txt")
	if err := os.WriteFile(sourceFile, []byte(testFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Bootconf: config.BootconfConfig{
			Enabled:   true,
			Directory: baseDir,
		},
		System: config.SystemConfig{
			Enabled:  true,
			Timezone: "UTC",
			Hostname: "testhost",
		},
		SSH: config.SSHConfig{
			Enabled:          true,
			Directory:        filepath.Join(baseDir, "ssh"),
			Daemon:           "dropbear",
			Keytype:          "ed25519",
			GenerateHostKeys: true,
		},
		Wifi: config.WifiConfig{
			Enabled:      true,
			Directory:    filepath.Join(baseDir, "wifi"),
			SSID:         testSSID,
			PasswordHash: testPasswordHash,
			Country:      testCountry,
		},
		Services: config.ServicesConfig{
			Enabled:   true,
			Directory: filepath.Join(baseDir, "services"),
			Services: []config.ServiceEntry{
				{
					Name:     testServiceName,
					Enabled:  true,
					Sentinel: true,
				},
			},
		},
		Users: config.UsersConfig{
			Enabled:     true,
			Directory:   filepath.Join(baseDir, "users"),
			TmpfilesDir: filepath.Join(baseDir, "tmpfiles"),
			Users: []config.UserEntry{
				{
					Name:           testUserName,
					Enabled:        true,
					Sudo:           true,
					Home:           homeDir,
					AuthorizedKeys: []string{testSSHPublicKey},
				},
			},
		},
		Files: config.FilesConfig{
			Enabled: true,
			Files: []config.FileEntry{
				{
					Source:      sourceFile,
					Destination: filepath.Join(baseDir, "test", "testfile.txt"),
					Chmod:       "640",
				},
			},
		},
	}

	cfg.SetDefaults()

	return &testEnvironment{
		baseDir:    baseDir,
		homeDir:    homeDir,
		sourceFile: sourceFile,
		cfg:        cfg,
	}
}

func (env *testEnvironment) allModules() []module.Module {
	return []module.Module{
		system.New(env.cfg.System, env.cfg.Bootconf.Directory),
		ssh.New(env.cfg.SSH, env.cfg.Services.Directory),
		wifi.New(env.cfg.Wifi, env.cfg.Services.Directory),
		services.New(env.cfg.Services),
		users.New(env.cfg.Users, uidStart),
		files.New(env.cfg.Files),
	}
}

func (env *testEnvironment) filesystemModules() []module.Module {
	return []module.Module{
		wifi.New(env.cfg.Wifi, env.cfg.Services.Directory),
		services.New(env.cfg.Services),
		users.New(env.cfg.Users, uidStart),
		files.New(env.cfg.Files),
	}
}

func runModules(t *testing.T, modules []module.Module, dryRun bool) []module.Result {
	t.Helper()

	runner := module.NewRunner(modules)
	results := runner.Run(context.Background(), dryRun, "")

	for _, result := range results {
		if !result.Success {
			t.Errorf("section %s failed: %s", result.Section, result.Error)
		}
	}

	return results
}

func assertAllSectionsSucceeded(t *testing.T, results []module.Result) {
	t.Helper()

	for _, result := range results {
		if !result.Success {
			t.Fatalf("section %s failed: %s", result.Section, result.Error)
		}
	}
}

func assertStatusRoundTrip(t *testing.T, statusDir string, results []module.Result, expectedSectionCount int) {
	t.Helper()

	runStatus := &status.RunStatus{
		Timestamp: time.Now().UTC(),
		Overall:   true,
		Sections:  results,
	}

	if err := status.Write(statusDir, runStatus); err != nil {
		t.Fatalf("status.Write failed: %v", err)
	}

	readBack, err := status.Read(statusDir)
	if err != nil {
		t.Fatalf("status.Read failed: %v", err)
	}

	if !readBack.Overall {
		t.Error("status overall should be true")
	}

	if len(readBack.Sections) != expectedSectionCount {
		t.Errorf("status sections: got %d, want %d", len(readBack.Sections), expectedSectionCount)
	}

	for _, section := range readBack.Sections {
		if !section.Success {
			t.Errorf("status section %s should be successful", section.Section)
		}
	}
}

func assertFileContains(t *testing.T, path, substring string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	if !strings.Contains(string(data), substring) {
		t.Errorf("%s: expected to contain %q, got %q", path, substring, string(data))
	}
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}

	if string(data) != expected {
		t.Errorf("%s: got %q, want %q", path, string(data), expected)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected dir %s to exist: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory, got file", path)
	}
}

func TestDryRunPipeline(t *testing.T) {
	env := newTestEnvironment(t)

	results := runModules(t, env.allModules(), true)

	if len(results) != 6 {
		t.Fatalf("expected 6 results, got %d", len(results))
	}

	assertAllSectionsSucceeded(t, results)

	statusDir := filepath.Join(env.baseDir, ".bootconf")
	assertStatusRoundTrip(t, statusDir, results, 6)
}

func TestRealRunCreatesAllArtifacts(t *testing.T) {
	env := newTestEnvironment(t)

	results := runModules(t, env.filesystemModules(), false)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	assertAllSectionsSucceeded(t, results)

	wifiConfPath := filepath.Join(env.baseDir, "wifi", "wpa_supplicant.conf")
	assertFileExists(t, wifiConfPath)
	assertFileContains(t, wifiConfPath, "country="+testCountry)
	assertFileContains(t, wifiConfPath, `ssid="`+testSSID+`"`)
	assertFileContains(t, wifiConfPath, "psk="+testPasswordHash)

	sentinelPath := filepath.Join(env.baseDir, "services", testServiceName)
	assertFileExists(t, sentinelPath)

	expectedSysusers := "u " + testUserName + " 2000 \"testadmin\" " + env.homeDir + " /bin/bash\n" + "m testadmin sudo\n"
	sysusersPath := filepath.Join(env.baseDir, "users", testUserName+".conf")
	assertFileContent(t, sysusersPath, expectedSysusers)

	expectedTmpfiles := "C " + env.homeDir + " - - - - /etc/skel\n"
	tmpfilesPath := filepath.Join(env.baseDir, "tmpfiles", testUserName+".conf")
	assertFileContent(t, tmpfilesPath, expectedTmpfiles)

	assertDirExists(t, env.homeDir)
	assertDirExists(t, filepath.Join(env.homeDir, ".ssh"))

	keysPath := filepath.Join(env.homeDir, ".ssh", "authorized_keys")
	assertFileContent(t, keysPath, testSSHPublicKey+"\n")

	copiedFilePath := filepath.Join(env.baseDir, "test", "testfile.txt")
	assertFileContent(t, copiedFilePath, testFileContent)

	statusDir := filepath.Join(env.baseDir, ".bootconf")
	assertStatusRoundTrip(t, statusDir, results, 4)
}

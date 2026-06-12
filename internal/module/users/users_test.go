package users

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
)

func TestUsersCreateHomeAndKeys(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "alice")

	entries := []config.UserEntry{
		{
			Name:           "alice",
			Enabled:        true,
			Sudo:           true,
			Home:           homedir,
			AuthorizedKeys: []string{"ssh-ed25519 AAAA alice@host", "ssh-rsa BBBB alice@other"},
		},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify sysusers conf.
	sysusersPath := filepath.Join(usersDir, "alice.conf")
	data, err := os.ReadFile(sysusersPath)
	if err != nil {
		t.Fatalf("sysusers config not found: %v", err)
	}
	want := "u alice 1000 \"alice\" " + homedir + " /bin/bash\n" + "m alice sudo\n"
	if string(data) != want {
		t.Errorf("sysusers content = %q, want %q", string(data), want)
	}

	// Verify tmpfiles conf.
	tmpfilesPath := filepath.Join(tmpfilesDir, "alice.conf")
	tmpfilesData, err := os.ReadFile(tmpfilesPath)
	if err != nil {
		t.Fatalf("tmpfiles config not found: %v", err)
	}
	wantTmpfiles := "C " + homedir + " - - - - /etc/skel\n"
	if string(tmpfilesData) != wantTmpfiles {
		t.Errorf("tmpfiles content = %q, want %q", string(tmpfilesData), wantTmpfiles)
	}

	// Verify .ssh directory.
	sshDir := filepath.Join(homedir, ".ssh")
	sshInfo, err := os.Stat(sshDir)
	if err != nil {
		t.Fatalf(".ssh dir not found: %v", err)
	}
	if !sshInfo.IsDir() {
		t.Error(".ssh is not a directory")
	}

	// Verify authorized_keys.
	keysPath := filepath.Join(sshDir, "authorized_keys")
	keysData, err := os.ReadFile(keysPath)
	if err != nil {
		t.Fatalf("authorized_keys not found: %v", err)
	}
	wantKeys := "ssh-ed25519 AAAA alice@host\nssh-rsa BBBB alice@other\n"
	if string(keysData) != wantKeys {
		t.Errorf("authorized_keys = %q, want %q", string(keysData), wantKeys)
	}
}

func TestUsersDisabledRemovesConfig(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Pre-create the sysusers and tmpfiles confs so we can verify removal.
	sysusersFile := filepath.Join(usersDir, "bob.conf")
	if err := os.WriteFile(sysusersFile, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	tmpfilesFile := filepath.Join(tmpfilesDir, "bob.conf")
	if err := os.WriteFile(tmpfilesFile, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	entries := []config.UserEntry{
		{Name: "bob", Enabled: false},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if _, err := os.Stat(sysusersFile); !os.IsNotExist(err) {
		t.Error("sysusers config should be removed")
	}
	if _, err := os.Stat(tmpfilesFile); !os.IsNotExist(err) {
		t.Error("tmpfiles config should be removed")
	}
}

func TestUsersDryRunNoWrites(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "carol")

	entries := []config.UserEntry{
		{
			Name:           "carol",
			Enabled:        true,
			Sudo:           true,
			Home:           homedir,
			AuthorizedKeys: []string{"ssh-ed25519 AAAA carol@host"},
		},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), true)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if _, err := os.Stat(filepath.Join(usersDir, "carol.conf")); !os.IsNotExist(err) {
		t.Error("sysusers config should not exist in dry-run")
	}
	if _, err := os.Stat(filepath.Join(tmpfilesDir, "carol.conf")); !os.IsNotExist(err) {
		t.Error("tmpfiles config should not exist in dry-run")
	}
	if _, err := os.Stat(homedir); !os.IsNotExist(err) {
		t.Error("home dir should not exist in dry-run")
	}
}

func TestUsersMultipleUsers(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homeAlice := filepath.Join(t.TempDir(), "home", "alice")
	homeBob := filepath.Join(t.TempDir(), "home", "bob")

	entries := []config.UserEntry{
		{Name: "alice", Enabled: true, Home: homeAlice},
		{Name: "bob", Enabled: true, Home: homeBob},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 2000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify alice gets uidStart (2000).
	aliceData, err := os.ReadFile(filepath.Join(usersDir, "alice.conf"))
	if err != nil {
		t.Fatal(err)
	}
	wantAlice := "u alice 2000 \"alice\" " + homeAlice + " /bin/bash\n"
	if string(aliceData) != wantAlice {
		t.Errorf("alice sysusers = %q, want %q", string(aliceData), wantAlice)
	}

	bobData, err := os.ReadFile(filepath.Join(usersDir, "bob.conf"))
	if err != nil {
		t.Fatal(err)
	}
	wantBob := "u bob 2001 \"bob\" " + homeBob + " /bin/bash\n"
	if string(bobData) != wantBob {
		t.Errorf("bob sysusers = %q, want %q", string(bobData), wantBob)
	}

	// Verify tmpfiles confs for both users.
	aliceTmpfiles, err := os.ReadFile(filepath.Join(tmpfilesDir, "alice.conf"))
	if err != nil {
		t.Fatalf("alice tmpfiles config not found: %v", err)
	}
	if string(aliceTmpfiles) != "C "+homeAlice+" - - - - /etc/skel\n" {
		t.Errorf("alice tmpfiles = %q", string(aliceTmpfiles))
	}

	bobTmpfiles, err := os.ReadFile(filepath.Join(tmpfilesDir, "bob.conf"))
	if err != nil {
		t.Fatalf("bob tmpfiles config not found: %v", err)
	}
	if string(bobTmpfiles) != "C "+homeBob+" - - - - /etc/skel\n" {
		t.Errorf("bob tmpfiles = %q", string(bobTmpfiles))
	}
}

func TestUsersNoKeys(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "eve")

	entries := []config.UserEntry{
		{Name: "eve", Enabled: true, Home: homedir},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// .ssh should exist, but authorized_keys must not.
	sshInfo, err := os.Stat(filepath.Join(homedir, ".ssh"))
	if err != nil {
		t.Fatalf(".ssh dir not found: %v", err)
	}
	if !sshInfo.IsDir() {
		t.Error(".ssh is not a directory")
	}

	keysPath := filepath.Join(homedir, ".ssh", "authorized_keys")
	if _, err := os.Stat(keysPath); !os.IsNotExist(err) {
		t.Error("authorized_keys should not exist when no keys provided")
	}
}

func TestUsersSingleKey(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "dave")

	singleKey := "ssh-ed25519 AAAA dave@host"
	entries := []config.UserEntry{
		{
			Name:           "dave",
			Enabled:        true,
			Home:           homedir,
			AuthorizedKeys: []string{singleKey},
		},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	keysPath := filepath.Join(homedir, ".ssh", "authorized_keys")
	keysData, err := os.ReadFile(keysPath)
	if err != nil {
		t.Fatalf("authorized_keys not found: %v", err)
	}

	want := singleKey + "\n"
	if string(keysData) != want {
		t.Errorf("authorized_keys = %q, want %q", string(keysData), want)
	}
}

func TestUsersMultipleKeys(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "frank")

	keys := []string{
		"ssh-ed25519 AAAA frank@host1",
		"ssh-rsa BBBB frank@host2",
		"ssh-ed25519 CCCC frank@host3",
	}
	entries := []config.UserEntry{
		{
			Name:           "frank",
			Enabled:        true,
			Home:           homedir,
			AuthorizedKeys: keys,
		},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	keysPath := filepath.Join(homedir, ".ssh", "authorized_keys")
	keysData, err := os.ReadFile(keysPath)
	if err != nil {
		t.Fatalf("authorized_keys not found: %v", err)
	}

	want := "ssh-ed25519 AAAA frank@host1\nssh-rsa BBBB frank@host2\nssh-ed25519 CCCC frank@host3\n"
	if string(keysData) != want {
		t.Errorf("authorized_keys = %q, want %q", string(keysData), want)
	}
}

func TestUsersDisabledSkipsCreation(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "mallory")

	entries := []config.UserEntry{
		{
			Name:    "mallory",
			Enabled: false,
			Home:    homedir,
		},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if _, err := os.Stat(filepath.Join(usersDir, "mallory.conf")); !os.IsNotExist(err) {
		t.Error("sysusers config should not exist for disabled user")
	}
	if _, err := os.Stat(filepath.Join(tmpfilesDir, "mallory.conf")); !os.IsNotExist(err) {
		t.Error("tmpfiles config should not exist for disabled user")
	}
	if _, err := os.Stat(homedir); !os.IsNotExist(err) {
		t.Error("home dir should not exist for disabled user")
	}
}

func TestUsersModuleDisabled(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "alice")

	entries := []config.UserEntry{
		{
			Name:           "alice",
			Enabled:        true,
			Home:           homedir,
			AuthorizedKeys: []string{"ssh-ed25519 AAAA alice@host"},
		},
	}

	cfg := config.UsersConfig{
		Enabled:     false,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Message != "users disabled" {
		t.Errorf("message = %q, want %q", result.Message, "users disabled")
	}

	if _, err := os.Stat(filepath.Join(usersDir, "alice.conf")); !os.IsNotExist(err) {
		t.Error("no files should be created when module is disabled")
	}
}

func TestUsersSshDirPerms(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "grace")

	entries := []config.UserEntry{
		{
			Name:           "grace",
			Enabled:        true,
			Home:           homedir,
			AuthorizedKeys: []string{"ssh-ed25519 AAAA grace@host"},
		},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	sshDir := filepath.Join(homedir, ".ssh")
	info, err := os.Stat(sshDir)
	if err != nil {
		t.Fatalf(".ssh dir not found: %v", err)
	}

	if info.Mode().Perm() != 0700 {
		t.Errorf(".ssh perms = %04o, want 0700", info.Mode().Perm())
	}
}

func TestUsersAuthorizedKeysPerms(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "heidi")

	entries := []config.UserEntry{
		{
			Name:           "heidi",
			Enabled:        true,
			Home:           homedir,
			AuthorizedKeys: []string{"ssh-ed25519 AAAA heidi@host"},
		},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	keysPath := filepath.Join(homedir, ".ssh", "authorized_keys")
	info, err := os.Stat(keysPath)
	if err != nil {
		t.Fatalf("authorized_keys not found: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("authorized_keys perms = %04o, want 0600", info.Mode().Perm())
	}
}

func TestUsersSudoTrueAddsGroupLine(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "admin")

	entries := []config.UserEntry{
		{Name: "admin", Enabled: true, Sudo: true, Home: homedir},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data, err := os.ReadFile(filepath.Join(usersDir, "admin.conf"))
	if err != nil {
		t.Fatalf("sysusers config not found: %v", err)
	}

	want := "u admin 1000 \"admin\" " + homedir + " /bin/bash\n" + "m admin sudo\n"
	if string(data) != want {
		t.Errorf("sysusers content = %q, want %q", string(data), want)
	}
}

func TestUsersSudoFalseNoGroupLine(t *testing.T) {
	usersDir := t.TempDir()
	tmpfilesDir := t.TempDir()
	homedir := filepath.Join(t.TempDir(), "home", "operator")

	entries := []config.UserEntry{
		{Name: "operator", Enabled: true, Sudo: false, Home: homedir},
	}

	cfg := config.UsersConfig{
		Enabled:     true,
		Directory:   usersDir,
		TmpfilesDir: tmpfilesDir,
		Users:       entries,
	}
	users := New(cfg, 1000)
	result := users.Run(context.Background(), false)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data, err := os.ReadFile(filepath.Join(usersDir, "operator.conf"))
	if err != nil {
		t.Fatalf("sysusers config not found: %v", err)
	}

	want := "u operator 1000 \"operator\" " + homedir + " /bin/bash\n"
	if string(data) != want {
		t.Errorf("sysusers content = %q, want %q", string(data), want)
	}
	if strings.Contains(string(data), "m operator sudo") {
		t.Error("sysusers content should not contain sudo group membership when sudo=false")
	}
}

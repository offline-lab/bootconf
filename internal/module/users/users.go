package users

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
)

type UsersModule struct {
	enabled  bool
	entries  []config.UserEntry
	uidStart int
	usersDir string
}

func NewUsersModule(cfg config.UsersConfig, uidStart int) *UsersModule {
	return &UsersModule{
		enabled:  cfg.Enabled,
		entries:  cfg.Users,
		uidStart: uidStart,
		usersDir: cfg.Directory,
	}
}

func (m *UsersModule) Name() string {
	return "users"
}

func (m *UsersModule) Run(_ context.Context, dryRun bool) module.Result {
	if !m.enabled {
		return module.Result{
			Section: "users",
			Success: true,
			Message: "users disabled",
		}
	}

	for entryIdx, entry := range m.entries {
		if !entry.Enabled {
			if !dryRun {
				m.removeUser(entry.Name)
			}
			continue
		}

		uid := m.uidStart + entryIdx
		if err := m.createUser(entry, uid, dryRun); err != nil {
			return module.Result{
				Section: "users",
				Success: false,
				Error:   fmt.Errorf("user %q: %w", entry.Name, err).Error(),
			}
		}
	}

	return module.Result{
		Section: "users",
		Success: true,
		Message: fmt.Sprintf("processed %d users", len(m.entries)),
	}
}

func (m *UsersModule) removeUser(name string) {
	_ = os.Remove(filepath.Join(m.usersDir, name+".conf"))

	if isValidUsername(name) {
		_ = exec.Command("gpasswd", "-d", name, "sudo").Run()
		_ = exec.Command("userdel", name).Run()
	}
}

func (m *UsersModule) createUser(entry config.UserEntry, uid int, dryRun bool) error {
	if dryRun {
		return nil
	}

	if err := os.MkdirAll(m.usersDir, 0755); err != nil {
		return fmt.Errorf("create users dir: %w", err)
	}

	sysusersLine := fmt.Sprintf("u %s %d \"%s\" %s /bin/bash\n", entry.Name, uid, entry.Name, entry.Home)
	if entry.Sudo {
		sysusersLine += fmt.Sprintf("m %s sudo\n", entry.Name)
	}

	if err := os.WriteFile(filepath.Join(m.usersDir, entry.Name+".conf"), []byte(sysusersLine), 0644); err != nil {
		return fmt.Errorf("write sysusers config: %w", err)
	}

	if err := os.MkdirAll(entry.Home, 0755); err != nil {
		return fmt.Errorf("create home: %w", err)
	}
	_ = os.Chown(entry.Home, uid, uid)

	sshDir := filepath.Join(entry.Home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("create .ssh: %w", err)
	}
	_ = os.Chown(sshDir, uid, uid)

	if len(entry.AuthorizedKeys) > 0 {
		keysPath := filepath.Join(sshDir, "authorized_keys")
		keysContent := strings.Join(entry.AuthorizedKeys, "\n") + "\n"
		if err := os.WriteFile(keysPath, []byte(keysContent), 0600); err != nil {
			return fmt.Errorf("write authorized_keys: %w", err)
		}
		_ = os.Chown(keysPath, uid, uid)
	}

	return nil
}

func isValidUsername(name string) bool {
	if len(name) == 0 || name[0] == '-' {
		return false
	}
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' || char == '-') {
			return false
		}
	}
	return true
}

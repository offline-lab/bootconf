// Package users creates system user accounts via systemd-sysusers convention.
// For each enabled user, a .conf file is written containing the sysusers
// directive. Home directories and .ssh/authorized_keys are provisioned
// directly. Disabled users have their config removed and their account
// deleted via userdel.
package users

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/logging"
	"github.com/offline-lab/bootconf/internal/module"
	"github.com/offline-lab/bootconf/internal/run"
)

// UsersModule provisions user accounts from config entries. UIDs are assigned
// sequentially starting from uidStart to avoid collisions with system accounts.
type UsersModule struct {
	enabled  bool
	entries  []config.UserEntry
	uidStart int
	usersDir string
}

// New creates a UsersModule from the given users config and UID start value.
func New(cfg config.UsersConfig, uidStart int) *UsersModule {
	return &UsersModule{
		enabled:  cfg.Enabled,
		entries:  cfg.Users,
		uidStart: uidStart,
		usersDir: cfg.Directory,
	}
}

// Name returns the module identifier "users".
func (usersModule *UsersModule) Name() string { return "users" }

// Run provisions or removes user accounts based on config entries.
func (usersModule *UsersModule) Run(ctx context.Context, dryRun bool) module.Result {
	if !usersModule.enabled {
		return module.Result{Section: usersModule.Name(), Success: true, Message: "users disabled"}
	}

	for index, entry := range usersModule.entries {
		if !entry.Enabled {
			if dryRun {
				logging.Info(usersModule.Name(), "would remove user %q and sysusers config (dry-run)", entry.Name)
			} else {
				logging.Info(usersModule.Name(), "removing user %q", entry.Name)
				usersModule.teardownUser(ctx, entry.Name)
			}
			continue
		}

		uid := usersModule.uidStart + index

		if err := usersModule.provisionUser(ctx, entry, uid, dryRun); err != nil {
			errMsg := fmt.Sprintf("user %q: %v", entry.Name, err)
			logging.Error(usersModule.Name(), "%s", errMsg)
			return module.Result{Section: usersModule.Name(), Success: false, Error: errMsg}
		}
	}

	return module.Result{Section: usersModule.Name(), Success: true, Message: fmt.Sprintf("processed %d users", len(usersModule.entries))}
}

func (usersModule *UsersModule) teardownUser(ctx context.Context, name string) {
	sysusersConf := filepath.Join(usersModule.usersDir, name+".conf")
	if err := os.Remove(sysusersConf); err != nil && !os.IsNotExist(err) {
		logging.Warn(usersModule.Name(), "failed to remove sysusers config %s: %v", sysusersConf, err)
	}

	if !config.IsValidUsername(name) {
		logging.Warn(usersModule.Name(), "skipping userdel for unsafe username %q", name)
		return
	}

	if err := run.Command(ctx, "gpasswd", "-d", name, "sudo"); err != nil {
		logging.Warn(usersModule.Name(), "failed to remove %q from sudo group: %v", name, err)
	}
	if err := run.Command(ctx, "userdel", name); err != nil {
		logging.Warn(usersModule.Name(), "failed to delete user %q: %v", name, err)
	}
}

func (usersModule *UsersModule) provisionUser(ctx context.Context, entry config.UserEntry, uid int, dryRun bool) error {
	sysusersConf := filepath.Join(usersModule.usersDir, entry.Name+".conf")
	sysusersLine := fmt.Sprintf("u %s %d \"%s\" %s /bin/bash\n", entry.Name, uid, entry.Name, entry.Home)
	if entry.Sudo {
		sysusersLine += fmt.Sprintf("m %s sudo\n", entry.Name)
	}

	sshDir := filepath.Join(entry.Home, ".ssh")
	keysPath := filepath.Join(sshDir, "authorized_keys")

	if dryRun {
		logging.Info(usersModule.Name(), "would create users dir %s (dry-run)", usersModule.usersDir)
		logging.Info(usersModule.Name(), "would write sysusers config %s (dry-run)", sysusersConf)
		logging.Info(usersModule.Name(), "would create home %s and .ssh dir (dry-run)", entry.Home)
		if len(entry.AuthorizedKeys) > 0 {
			logging.Info(usersModule.Name(), "would write %d authorized key(s) to %s (dry-run)", len(entry.AuthorizedKeys), keysPath)
		}
		return nil
	}

	logging.Info(usersModule.Name(), "provisioning user %q (uid %d)", entry.Name, uid)

	if err := os.MkdirAll(usersModule.usersDir, 0750); err != nil {
		return fmt.Errorf("create users dir %s: %w", usersModule.usersDir, err)
	}

	if err := os.WriteFile(sysusersConf, []byte(sysusersLine), 0640); err != nil {
		return fmt.Errorf("write sysusers config %s: %w", sysusersConf, err)
	}

	if err := os.MkdirAll(entry.Home, 0750); err != nil {
		return fmt.Errorf("create home %s: %w", entry.Home, err)
	}
	if err := os.Chown(entry.Home, uid, uid); err != nil {
		logging.Warn(usersModule.Name(), "failed to chown %s to uid %d: %v", entry.Home, uid, err)
	}

	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("create .ssh dir %s: %w", sshDir, err)
	}
	if err := os.Chown(sshDir, uid, uid); err != nil {
		logging.Warn(usersModule.Name(), "failed to chown %s to uid %d: %v", sshDir, uid, err)
	}

	if len(entry.AuthorizedKeys) > 0 {
		keysContent := strings.Join(entry.AuthorizedKeys, "\n") + "\n"
		if err := os.WriteFile(keysPath, []byte(keysContent), 0600); err != nil {
			return fmt.Errorf("write authorized_keys %s: %w", keysPath, err)
		}
		if err := os.Chown(keysPath, uid, uid); err != nil {
			logging.Warn(usersModule.Name(), "failed to chown %s to uid %d: %v", keysPath, uid, err)
		}
	}

	return nil
}

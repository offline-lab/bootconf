// Package registry is the single source of truth for bootconf modules.
// To add a new module, append one Entry to Modules — no other file needs editing.
package registry

import (
	"fmt"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/module"
	"github.com/offline-lab/bootconf/internal/module/files"
	"github.com/offline-lab/bootconf/internal/module/services"
	"github.com/offline-lab/bootconf/internal/module/shell"
	"github.com/offline-lab/bootconf/internal/module/ssh"
	"github.com/offline-lab/bootconf/internal/module/system"
	"github.com/offline-lab/bootconf/internal/module/templates"
	"github.com/offline-lab/bootconf/internal/module/unitrun"
	"github.com/offline-lab/bootconf/internal/module/users"
	"github.com/offline-lab/bootconf/internal/module/wifi"
)

// Entry pairs a module name with its constructor.
type Entry struct {
	Name string
	New  func(*config.Config) module.Module
}

// Modules is the canonical ordered list of all bootconf modules.
var Modules = []Entry{
	{"system", func(cfg *config.Config) module.Module { return system.New(cfg.System, cfg.Bootconf.Directory) }},
	{"users", func(cfg *config.Config) module.Module { return users.New(cfg.Users, 2000) }},
	{"wifi", func(cfg *config.Config) module.Module { return wifi.New(cfg.Wifi, cfg.Services.Directory) }},
	{"ssh", func(cfg *config.Config) module.Module { return ssh.New(cfg.SSH, cfg.Services.Directory) }},
	{"services", func(cfg *config.Config) module.Module { return services.New(cfg.Services) }},
	{"files", func(cfg *config.Config) module.Module { return files.New(cfg.Files) }},
	{"templates", func(cfg *config.Config) module.Module { return templates.New(cfg.Templates) }},
	{"unitrun", func(cfg *config.Config) module.Module { return unitrun.New(cfg.UnitRun) }},
	{"shell", func(cfg *config.Config) module.Module { return shell.New(cfg.Shell) }},
}

// Names returns the default module execution order.
func Names() []string {
	names := make([]string, len(Modules))
	for i, e := range Modules {
		names[i] = e.Name
	}
	return names
}

// KnownNames returns the set of all registered module names.
func KnownNames() map[string]bool {
	m := make(map[string]bool, len(Modules))
	for _, e := range Modules {
		m[e.Name] = true
	}
	return m
}

// ApplyDefaults sets cfg.Bootconf.Order to the default module order if it is empty.
func ApplyDefaults(cfg *config.Config) {
	if len(cfg.Bootconf.Order) == 0 {
		cfg.Bootconf.Order = Names()
	}
}

// Validate checks that every name in cfg.Bootconf.Order is a known module
// and that no name appears more than once.
func Validate(cfg *config.Config) error {
	known := KnownNames()
	seen := make(map[string]bool, len(cfg.Bootconf.Order))
	for i, name := range cfg.Bootconf.Order {
		if !known[name] {
			return fmt.Errorf("bootconf.order[%d]: unknown module %q", i, name)
		}
		if seen[name] {
			return fmt.Errorf("bootconf.order: duplicate module %q", name)
		}
		seen[name] = true
	}
	return nil
}

// Build returns modules in the order specified by cfg.Bootconf.Order.
func Build(cfg *config.Config) []module.Module {
	byName := make(map[string]Entry, len(Modules))
	for _, e := range Modules {
		byName[e.Name] = e
	}
	mods := make([]module.Module, 0, len(cfg.Bootconf.Order))
	for _, name := range cfg.Bootconf.Order {
		mods = append(mods, byName[name].New(cfg))
	}
	return mods
}

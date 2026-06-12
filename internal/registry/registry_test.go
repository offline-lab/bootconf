package registry_test

import (
	"testing"

	"github.com/offline-lab/bootconf/internal/config"
	"github.com/offline-lab/bootconf/internal/registry"
)

func TestNames(t *testing.T) {
	names := registry.Names()
	if len(names) != len(registry.Modules) {
		t.Fatalf("Names() length = %d, want %d", len(names), len(registry.Modules))
	}
	if names[0] != "system" {
		t.Errorf("first module = %q, want %q", names[0], "system")
	}
	if names[len(names)-1] != "shell" {
		t.Errorf("last module = %q, want %q", names[len(names)-1], "shell")
	}
}

func TestKnownNames(t *testing.T) {
	known := registry.KnownNames()
	for _, e := range registry.Modules {
		if !known[e.Name] {
			t.Errorf("KnownNames() missing %q", e.Name)
		}
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &config.Config{}
	registry.ApplyDefaults(cfg)
	if len(cfg.Bootconf.Order) != len(registry.Modules) {
		t.Errorf("Order length = %d, want %d", len(cfg.Bootconf.Order), len(registry.Modules))
	}
	// Should not overwrite an explicit order
	cfg.Bootconf.Order = []string{"shell", "system"}
	registry.ApplyDefaults(cfg)
	if cfg.Bootconf.Order[0] != "shell" {
		t.Error("ApplyDefaults overwrote an explicit order")
	}
}

func TestValidate(t *testing.T) {
	cfg := &config.Config{
		Bootconf: config.BootconfConfig{Order: registry.Names()},
	}
	if err := registry.Validate(cfg); err != nil {
		t.Fatalf("valid order failed: %v", err)
	}
}

func TestValidateUnknown(t *testing.T) {
	cfg := &config.Config{
		Bootconf: config.BootconfConfig{Order: []string{"system", "unknown"}},
	}
	if err := registry.Validate(cfg); err == nil {
		t.Error("expected error for unknown module name")
	}
}

func TestValidateDuplicate(t *testing.T) {
	cfg := &config.Config{
		Bootconf: config.BootconfConfig{Order: []string{"system", "system"}},
	}
	if err := registry.Validate(cfg); err == nil {
		t.Error("expected error for duplicate module name")
	}
}

func TestBuild(t *testing.T) {
	cfg := &config.Config{}
	registry.ApplyDefaults(cfg)
	mods := registry.Build(cfg)
	if len(mods) != len(registry.Modules) {
		t.Fatalf("Build() length = %d, want %d", len(mods), len(registry.Modules))
	}
	for i, mod := range mods {
		if mod == nil {
			t.Errorf("Build()[%d] is nil", i)
		}
	}
}

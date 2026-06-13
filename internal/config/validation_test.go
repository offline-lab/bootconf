package config

import "testing"

func validConfig() *Config {
	return &Config{
		Bootconf: BootconfConfig{
			Enabled:   true,
			Directory: "/data/config/bootconf",
		},
		System: SystemConfig{
			Enabled:  true,
			Timezone: "UTC",
			Hostname: "testhost",
		},
		SSH: SSHConfig{
			Enabled:          true,
			Directory:        "/data/config/ssh",
			Keytype:          "ed25519",
			GenerateHostKeys: true,
			Daemon:           "dropbear",
		},
		Wifi: WifiConfig{
			Enabled:      true,
			Directory:    "/data/config/wifi",
			SSID:         "testnet",
			PasswordHash: "a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3",
			Country:      "US",
		},
		Services: ServicesConfig{
			Enabled:   true,
			Directory: "/data/config/services",
			Services: []ServiceEntry{
				{
					Name:    "disco",
					Enabled: true,
					DefaultConfig: DefaultConfig{
						Copy:        true,
						Source:      "/etc/disco/disco.conf",
						Destination: "/data/config/disco/disco.conf",
					},
				},
			},
		},
		Users: UsersConfig{
			Enabled:     true,
			Directory:   "/data/config/users",
			TmpfilesDir: "/data/config/tmpfiles",
			Users: []UserEntry{
				{
					Name:    "admin",
					Enabled: true,
					Sudo:    true,
					Home:    "/data/home/admin",
				},
			},
		},
		Files: FilesConfig{
			Enabled: true,
			Files: []FileEntry{
				{
					Source:      "/boot/firmware/test.conf",
					Destination: "/data/config/test/test.conf",
					Chmod:       "640",
				},
			},
		},
	}
}

func TestValidateValid(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config should pass validation: %v", err)
	}
}

func TestValidateInvalidSSHDaemon(t *testing.T) {
	cfg := validConfig()
	cfg.SSH.Daemon = "badvalue"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid SSH daemon")
	}
}

func TestValidateInvalidSSHKeytype(t *testing.T) {
	cfg := validConfig()
	cfg.SSH.Keytype = "badvalue"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid SSH keytype")
	}
}

func TestValidateWifiNoSSID(t *testing.T) {
	cfg := validConfig()
	cfg.Wifi.SSID = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for wifi enabled without ssid")
	}
}

func TestValidateWifiDisabledOK(t *testing.T) {
	cfg := validConfig()
	cfg.Wifi = WifiConfig{Enabled: false}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled wifi with empty fields should pass: %v", err)
	}
}

func TestValidateUserNoName(t *testing.T) {
	cfg := validConfig()
	cfg.Users.Users = []UserEntry{{Enabled: true, Home: "/data/home/x"}}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for enabled user without name")
	}
}

func TestValidateServiceNoName(t *testing.T) {
	cfg := validConfig()
	cfg.Services.Services = []ServiceEntry{{Enabled: true}}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for enabled service without name")
	}
}

func TestValidateFileNoSource(t *testing.T) {
	cfg := validConfig()
	cfg.Files = FilesConfig{
		Enabled: true,
		Files:   []FileEntry{{Destination: "test/test.conf"}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for file without source")
	}
}

func TestValidateBootconfDirectoryRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Bootconf.Directory = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty bootconf.directory")
	}
}

func TestValidateSSHDirectoryRequired(t *testing.T) {
	cfg := validConfig()
	cfg.SSH.Directory = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty ssh.directory")
	}
}

func TestValidateWifiDirectoryRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Wifi.Directory = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty wifi.directory")
	}
}

func TestValidateServicesDirectoryRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Services.Directory = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty services.directory")
	}
}

func TestValidateUsersDirectoryRequired(t *testing.T) {
	cfg := validConfig()
	cfg.Users.Directory = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty users.directory")
	}
}

func TestValidateMinimalConfigWifiOnly(t *testing.T) {
	cfg := &Config{
		Bootconf: BootconfConfig{Enabled: true, Directory: "/data/config/bootconf"},
		Wifi: WifiConfig{
			Enabled:      true,
			Directory:    "/data/config/wifi",
			SSID:         "TestNet",
			PasswordHash: "a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3c4d5e6f7a2b3",
			Country:      "NL",
		},
	}
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("minimal wifi-only config should pass validation: %v", err)
	}
	if cfg.System.Enabled {
		t.Error("System.Enabled should be false when omitted")
	}
	if cfg.SSH.Enabled {
		t.Error("SSH.Enabled should be false when omitted")
	}
	if cfg.Services.Enabled {
		t.Error("Services.Enabled should be false when omitted")
	}
	if cfg.Users.Enabled {
		t.Error("Users.Enabled should be false when omitted")
	}
	if cfg.Files.Enabled {
		t.Error("Files.Enabled should be false when omitted")
	}
}

func TestValidateEmptyConfigPasses(t *testing.T) {
	cfg := &Config{}
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("completely empty config should pass validation: %v", err)
	}
}

func TestValidateSSHDisabledSkipsDaemonKeytypeValidation(t *testing.T) {
	cfg := &Config{
		Bootconf: BootconfConfig{Enabled: true, Directory: "/data/config/bootconf"},
		SSH:      SSHConfig{Enabled: false, Daemon: "nonsense", Keytype: "garbage"},
	}
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled SSH with invalid daemon/keytype should still pass: %v", err)
	}
}

func TestValidateDisabledSectionsWithGarbageFields(t *testing.T) {
	cfg := &Config{
		Bootconf: BootconfConfig{Enabled: true, Directory: "/data/config/bootconf"},
		Wifi:     WifiConfig{Enabled: false, SSID: "", PasswordHash: "bad", Country: "XX"},
		Services: ServicesConfig{Enabled: false, Directory: ""},
		Users:    UsersConfig{Enabled: false, Directory: ""},
		Files: FilesConfig{
			Enabled: false,
			Files:   []FileEntry{{Source: "", Destination: ""}},
		},
	}
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("all-disabled config with garbage fields should pass: %v", err)
	}
}

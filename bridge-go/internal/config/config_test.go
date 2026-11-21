package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Test_req_BRG_07_MissingConfigUsesDefaults(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Speed != 224 || got.Volume != 2 || got.SessionTimeoutMin != 10 ||
		got.LogLevel != "info" || got.SerialPort != nil || got.WSLDistros != nil {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}

func Test_req_AGT_12_EmptyWslDistroListDisablesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.toml")
	if err := os.WriteFile(path, []byte("wsl_distros = []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.WSLDistros == nil || len(got.WSLDistros) != 0 {
		t.Fatalf("wsl_distros = %#v, want an explicit empty list", got.WSLDistros)
	}
}

func Test_req_BRG_07_ParsesAndValidatesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bridge.toml")
	content := "serial_port = \"/dev/ttyACM3\"\nspeed = 90\nvolume = 0\n" +
		"session_timeout_min = 2.5\nlog_level = \"debug\"\nwsl_distros = [\"Ubuntu\", \"Debian\"]\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.SerialPort == nil || *got.SerialPort != "/dev/ttyACM3" ||
		got.Speed != 90 || got.Volume != 0 || got.SessionTimeoutMin != 2.5 ||
		got.LogLevel != "debug" || len(got.WSLDistros) != 2 ||
		got.WSLDistros[0] != "Ubuntu" || got.WSLDistros[1] != "Debian" {
		t.Fatalf("unexpected config: %+v", got)
	}
}

func Test_req_BRG_22_SaveConfigPreservesDefaultAndExplicitDistroLists(t *testing.T) {
	t.Run("default distro", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "cody", "bridge.toml")
		config := Defaults()
		if err := Save(path, config); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.WSLDistros != nil {
			t.Fatalf("wsl_distros = %#v, want nil default", got.WSLDistros)
		}
	})

	t.Run("explicitly disabled", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bridge.toml")
		config := Defaults()
		config.WSLDistros = []string{}
		if err := Save(path, config); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.WSLDistros == nil || len(got.WSLDistros) != 0 {
			t.Fatalf("wsl_distros = %#v, want explicit empty list", got.WSLDistros)
		}
	})
}

func Test_req_BRG_07_RejectsInvalidConfig(t *testing.T) {
	for _, content := range []string{
		"speed = 256\n",
		"speed = true\n",
		"volume = 4\n",
		"session_timeout_min = 0\n",
		"session_timeout_min = nan\n",
		"serial_port = \"\"\n",
		"log_level = \"verbose\"\n",
		"colour = \"red\"\n",
		"speed = \n",
		"wsl_distros = [\"\"]\n",
		"wsl_distros = [\"Ubuntu\", \"Ubuntu\"]\n",
	} {
		path := filepath.Join(t.TempDir(), "invalid.toml")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("Load accepted %q", content)
		}
	}
}

func Test_req_BRG_16_DefaultConfigAndLogPathsUseUserDataRoots(t *testing.T) {
	root := t.TempDir()
	configRoot := filepath.Join(root, "roaming")
	cacheRoot := filepath.Join(root, "local")
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", configRoot)
		t.Setenv("LOCALAPPDATA", cacheRoot)
	} else {
		t.Setenv("XDG_CONFIG_HOME", configRoot)
		t.Setenv("XDG_CACHE_HOME", cacheRoot)
	}

	configPath, err := DefaultConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(configRoot, "cody", "bridge.toml"); configPath != want {
		t.Fatalf("DefaultConfigPath() = %q, want %q", configPath, want)
	}

	logPath, err := DefaultLogPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cacheRoot, "cody", "bridge.log"); logPath != want {
		t.Fatalf("DefaultLogPath() = %q, want %q", logPath, want)
	}
}

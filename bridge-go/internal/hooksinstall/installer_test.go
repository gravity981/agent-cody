package hooksinstall

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_req_AGT_10_InstallPreservesSettingsAndInstallsBothAgents(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	copyManualAssets(t, assets)
	claudePath := filepath.Join(root, ".claude", "settings.json")
	original := `{"theme":"dark","hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"echo keep"}]},{"hooks":[{"type":"command","command":"old /hooks/claude-code"}]}]}}`
	if err := os.MkdirAll(filepath.Dir(claudePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claudePath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	copilotPath := filepath.Join(root, ".copilot", "hooks")
	installer, err := New(Paths{
		ClaudeSettings: claudePath,
		CopilotHooks:   copilotPath,
		Assets:         assets,
	})
	if err != nil {
		t.Fatal(err)
	}

	status, err := installer.Install()
	if err != nil {
		t.Fatal(err)
	}
	if !status.ClaudeCodeInstalled || !status.CopilotInstalled {
		t.Fatalf("installed status = %+v", status)
	}
	backup, err := os.ReadFile(claudePath + ".bak")
	if err != nil || string(backup) != original {
		t.Fatalf("settings backup = %q, %v", backup, err)
	}

	settingsData, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if err := json.Unmarshal(settingsData, &settings); err != nil {
		t.Fatal(err)
	}
	if settings["theme"] != "dark" {
		t.Fatalf("unrelated setting was not preserved: %+v", settings)
	}
	hooks := settings["hooks"].(map[string]any)
	for _, event := range claudeEvents {
		groups := hooks[event].([]any)
		codyCount := 0
		for _, groupValue := range groups {
			group := groupValue.(map[string]any)
			for _, hookValue := range group["hooks"].([]any) {
				hook := hookValue.(map[string]any)
				if strings.Contains(hook["command"].(string), claudeMarker) {
					codyCount++
				}
			}
		}
		if codyCount != 1 {
			t.Errorf("%s has %d Cody hook groups, want exactly one", event, codyCount)
		}
	}
	if got := len(hooks["SessionStart"].([]any)); got != 2 {
		t.Fatalf("SessionStart groups = %d, want custom + Cody", got)
	}

	for _, name := range copilotFiles {
		source, err := os.ReadFile(filepath.Join(assets, name))
		if err != nil {
			t.Fatal(err)
		}
		target := name
		if name == "copilot-hooks.json" {
			target = "cody.json"
		}
		installed, err := os.ReadFile(filepath.Join(copilotPath, target))
		if err != nil || string(installed) != string(source) {
			t.Errorf("installed Copilot asset %s differs from source: %v", target, err)
		}
	}

	if _, err := installer.Install(); err != nil {
		t.Fatal(err)
	}
	status, err = installer.Status()
	if err != nil || !status.ClaudeCodeInstalled || !status.CopilotInstalled {
		t.Fatalf("repeat install status = %+v, %v", status, err)
	}
}

func Test_req_AGT_10_InvalidClaudeSettingsAreReportedWithoutOverwrite(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets")
	copyManualAssets(t, assets)
	claudePath := filepath.Join(root, "settings.json")
	if err := os.WriteFile(claudePath, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	installer, err := New(Paths{
		ClaudeSettings: claudePath,
		CopilotHooks:   filepath.Join(root, "copilot", "hooks"),
		Assets:         assets,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(); err == nil {
		t.Fatal("Install() succeeded with invalid settings")
	}
	got, err := os.ReadFile(claudePath)
	if err != nil || string(got) != "{invalid" {
		t.Fatalf("invalid settings changed: %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(root, "copilot", "hooks")); !os.IsNotExist(err) {
		t.Fatalf("Copilot hooks were installed after Claude settings failed: %v", err)
	}
}

func Test_req_AGT_10_MissingPackagedAssetsDoNotPartiallyInstall(t *testing.T) {
	root := t.TempDir()
	claudePath := filepath.Join(root, ".claude", "settings.json")
	installer, err := New(Paths{
		ClaudeSettings: claudePath,
		CopilotHooks:   filepath.Join(root, ".copilot", "hooks"),
		Assets:         filepath.Join(root, "missing-assets"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installer.Install(); err == nil {
		t.Fatal("Install() succeeded with missing hook assets")
	}
	if _, err := os.Stat(claudePath); !os.IsNotExist(err) {
		t.Fatalf("Claude settings were changed before assets were validated: %v", err)
	}
}

func copyManualAssets(t *testing.T, destination string) {
	t.Helper()
	sourceDir := filepath.Join("..", "..", "..", "hooks")
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range copilotFiles {
		data, err := os.ReadFile(filepath.Join(sourceDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destination, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

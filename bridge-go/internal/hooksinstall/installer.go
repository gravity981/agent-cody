package hooksinstall

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	claudeMarker = "/hooks/claude-code"
	hookTimeout  = 2
	curlTimeout  = 0.1
)

var claudeEvents = []string{
	"SessionStart",
	"UserPromptSubmit",
	"PreToolUse",
	"PostToolUse",
	"PostToolUseFailure",
	"PermissionRequest",
	"Notification",
	"Stop",
	"StopFailure",
	"SessionEnd",
}

var copilotFiles = []string{
	"copilot-hooks.json",
	"cody-forward.sh",
	"cody-forward.ps1",
}

type Paths struct {
	ClaudeSettings string
	CopilotHooks   string
	Assets         string
}

type Status struct {
	ClaudeCodeInstalled bool   `json:"claude_code_installed"`
	ClaudeSettingsPath  string `json:"claude_settings_path"`
	CopilotInstalled    bool   `json:"copilot_installed"`
	CopilotHooksPath    string `json:"copilot_hooks_path"`
	CurlAvailable       bool   `json:"curl_available"`
}

type Installer struct {
	paths Paths
}

func New(paths Paths) (*Installer, error) {
	if paths.ClaudeSettings == "" || paths.CopilotHooks == "" || paths.Assets == "" {
		return nil, errors.New("Claude settings, Copilot hooks and assets paths are required")
	}
	return &Installer{paths: paths}, nil
}

func Default(assets string) (*Installer, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find user home directory: %w", err)
	}
	claudeDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if claudeDir == "" {
		claudeDir = filepath.Join(home, ".claude")
	}
	copilotDir := os.Getenv("COPILOT_HOME")
	if copilotDir == "" {
		copilotDir = filepath.Join(home, ".copilot")
	}
	return New(Paths{
		ClaudeSettings: filepath.Join(claudeDir, "settings.json"),
		CopilotHooks:   filepath.Join(copilotDir, "hooks"),
		Assets:         assets,
	})
}

func (i *Installer) Status() (Status, error) {
	claudeInstalled, err := claudeHooksInstalled(i.paths.ClaudeSettings)
	if err != nil {
		return Status{}, fmt.Errorf("check Claude Code hooks: %w", err)
	}
	copilotInstalled, err := i.copilotHooksInstalled()
	if err != nil {
		return Status{}, fmt.Errorf("check Copilot hooks: %w", err)
	}
	curlName := "curl"
	if runtime.GOOS == "windows" {
		curlName = "curl.exe"
	}
	_, curlErr := exec.LookPath(curlName)
	return Status{
		ClaudeCodeInstalled: claudeInstalled,
		ClaudeSettingsPath:  i.paths.ClaudeSettings,
		CopilotInstalled:    copilotInstalled,
		CopilotHooksPath:    i.paths.CopilotHooks,
		CurlAvailable:       curlErr == nil,
	}, nil
}

func (i *Installer) Install() (Status, error) {
	if err := i.validateAssets(); err != nil {
		return Status{}, fmt.Errorf("validate hook assets: %w", err)
	}
	if err := i.installClaudeHooks(); err != nil {
		return Status{}, fmt.Errorf("install Claude Code hooks: %w", err)
	}
	if err := i.installCopilotHooks(); err != nil {
		return Status{}, fmt.Errorf("install Copilot hooks: %w", err)
	}
	return i.Status()
}

func (i *Installer) validateAssets() error {
	for _, name := range copilotFiles {
		if _, err := os.ReadFile(filepath.Join(i.paths.Assets, name)); err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
	}
	return nil
}

func (i *Installer) installClaudeHooks() error {
	settings, mode, exists, err := readSettings(i.paths.ClaudeSettings)
	if err != nil {
		return err
	}
	if err := mergeClaudeHooks(settings); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(i.paths.ClaudeSettings), 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	if exists {
		original, err := os.ReadFile(i.paths.ClaudeSettings)
		if err != nil {
			return fmt.Errorf("read settings backup source: %w", err)
		}
		if err := os.WriteFile(i.paths.ClaudeSettings+".bak", original, mode); err != nil {
			return fmt.Errorf("save settings backup: %w", err)
		}
	}
	if err := os.WriteFile(i.paths.ClaudeSettings, append(data, '\n'), mode); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

func (i *Installer) installCopilotHooks() error {
	if err := os.MkdirAll(i.paths.CopilotHooks, 0o700); err != nil {
		return fmt.Errorf("create hooks directory: %w", err)
	}
	for _, name := range copilotFiles {
		data, err := os.ReadFile(filepath.Join(i.paths.Assets, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		target := name
		if name == "copilot-hooks.json" {
			target = "cody.json"
		}
		if err := os.WriteFile(filepath.Join(i.paths.CopilotHooks, target), data, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
	}
	return nil
}

func (i *Installer) copilotHooksInstalled() (bool, error) {
	for _, name := range copilotFiles {
		asset, err := os.ReadFile(filepath.Join(i.paths.Assets, name))
		if err != nil {
			return false, fmt.Errorf("read %s: %w", name, err)
		}
		target := name
		if name == "copilot-hooks.json" {
			target = "cody.json"
		}
		installed, err := os.ReadFile(filepath.Join(i.paths.CopilotHooks, target))
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("read installed %s: %w", target, err)
		}
		if !bytes.Equal(installed, asset) {
			return false, nil
		}
	}
	return true, nil
}

func readSettings(path string) (map[string]any, os.FileMode, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, 0o600, false, nil
	}
	if err != nil {
		return nil, 0, false, fmt.Errorf("read settings: %w", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, 0, false, fmt.Errorf("parse settings JSON: %w", err)
	}
	if settings == nil {
		return nil, 0, false, errors.New("settings must be a JSON object")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, false, fmt.Errorf("stat settings: %w", err)
	}
	return settings, info.Mode().Perm(), true, nil
}

func mergeClaudeHooks(settings map[string]any) error {
	hooksValue, exists := settings["hooks"]
	hooks := map[string]any{}
	if exists {
		var ok bool
		hooks, ok = hooksValue.(map[string]any)
		if !ok {
			return errors.New("settings hooks must be an object")
		}
	}
	for _, event := range claudeEvents {
		var groups []any
		if value, found := hooks[event]; found {
			var ok bool
			groups, ok = value.([]any)
			if !ok {
				return fmt.Errorf("settings hooks.%s must be an array", event)
			}
		}
		merged := make([]any, 0, len(groups)+1)
		for _, group := range groups {
			isCody, err := isCodyGroup(group)
			if err != nil {
				return fmt.Errorf("settings hooks.%s: %w", event, err)
			}
			if !isCody {
				merged = append(merged, group)
			}
		}
		merged = append(merged, claudeHookGroup())
		hooks[event] = merged
	}
	settings["hooks"] = hooks
	return nil
}

func claudeHooksInstalled(path string) (bool, error) {
	settings, _, exists, err := readSettings(path)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return false, nil
	}
	for _, event := range claudeEvents {
		groups, ok := hooks[event].([]any)
		if !ok {
			return false, nil
		}
		found := false
		for _, group := range groups {
			isCody, err := isCodyGroup(group)
			if err != nil {
				return false, fmt.Errorf("settings hooks.%s: %w", event, err)
			}
			found = found || isCody
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

func isCodyGroup(value any) (bool, error) {
	group, ok := value.(map[string]any)
	if !ok {
		return false, errors.New("hook group must be an object")
	}
	hooksValue, found := group["hooks"]
	if !found {
		return false, nil
	}
	hooks, ok := hooksValue.([]any)
	if !ok {
		return false, errors.New("hook group hooks must be an array")
	}
	for _, value := range hooks {
		hook, ok := value.(map[string]any)
		if !ok {
			return false, errors.New("hook must be an object")
		}
		command, _ := hook["command"].(string)
		if strings.Contains(command, claudeMarker) {
			return true, nil
		}
	}
	return false, nil
}

func claudeHookGroup() map[string]any {
	targets := []string{
		"http://127.0.0.1:8765" + claudeMarker,
		"http://host.docker.internal:8765" + claudeMarker,
		"http://172.17.0.1:8765" + claudeMarker,
	}
	command := "p=$(cat); for u in " + strings.Join(targets, " ") +
		"; do r=$(printf '%s' \"$p\" | curl -fsS --connect-timeout 0.1 -m 0.1 " +
		"-H 'Expect:' --data-binary @- \"$u\" 2>/dev/null) " +
		"&& [ \"$r\" = ok ] && break; done; exit 0"
	return map[string]any{
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": command,
				"timeout": hookTimeout,
			},
		},
	}
}

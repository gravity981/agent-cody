package mapping

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

func payloadLines(t *testing.T, name string) []map[string]any {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate mapping test source")
	}
	path := filepath.Join(filepath.Dir(source), "testdata", name)
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var result []map[string]any
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 4_000_000)
	for scanner.Scan() {
		var payload map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		result = append(result, payload)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func Test_req_AGT_01_RecordedPayloadsMapToClaudeStates(t *testing.T) {
	expected := map[string]sessions.State{
		"SessionStart":       sessions.IDLE,
		"UserPromptSubmit":   sessions.THINKING,
		"PreToolUse":         sessions.WORKING,
		"PostToolUse":        sessions.THINKING,
		"PostToolUseFailure": sessions.THINKING,
		"PermissionRequest":  sessions.APPROVAL,
		"Stop":               sessions.COMPLETED,
		"StopFailure":        sessions.ERROR,
	}
	payloads := append(
		payloadLines(t, "claude-code-session.jsonl"),
		payloadLines(t, "claude-code-from-docs.jsonl")...,
	)
	seen := make(map[sessions.State]bool)
	for _, payload := range payloads {
		event, err := ClaudeCode(payload)
		if err != nil {
			t.Fatalf("ClaudeCode(%v): %v", payload, err)
		}
		if event.Agent != "claude-code" || event.Session != "session-1" {
			t.Fatalf("unexpected event identity: %+v", event)
		}
		name := payload["hook_event_name"].(string)
		switch name {
		case "Notification":
			kind := payload["notification_type"].(string)
			if kind == "permission_prompt" && event.State != sessions.APPROVAL {
				t.Fatalf("permission notification mapped to %q", event.State)
			}
			if kind == "idle_prompt" && !event.Ignored {
				t.Fatal("idle notification was not ignored")
			}
		case "SessionEnd":
			if !event.End {
				t.Fatal("SessionEnd did not end its session")
			}
		default:
			if state, ok := expected[name]; ok {
				if event.State != state {
					t.Fatalf("%s mapped to %s, want %s", name, event.State, state)
				}
				seen[state] = true
			}
		}
	}
	for _, state := range []sessions.State{
		sessions.IDLE, sessions.THINKING, sessions.WORKING,
		sessions.APPROVAL, sessions.COMPLETED, sessions.ERROR,
	} {
		if !seen[state] {
			t.Errorf("no recorded payload mapped to %s", state)
		}
	}
}

func Test_req_AGT_01_ClaudePayloadRequiresSessionAndEvent(t *testing.T) {
	for _, payload := range []map[string]any{
		{"hook_event_name": "Stop"},
		{"session_id": "session-1"},
		{"session_id": "", "hook_event_name": "Stop"},
	} {
		if _, err := ClaudeCode(payload); err == nil {
			t.Errorf("ClaudeCode(%v) unexpectedly succeeded", payload)
		}
	}
}

func Test_req_AGT_01_ClaudeNotificationMatcherFallback(t *testing.T) {
	event, err := ClaudeCode(map[string]any{
		"session_id": "session-1", "hook_event_name": "Notification",
		"matcher": "permission_prompt",
	})
	if err != nil || event.State != sessions.APPROVAL {
		t.Fatalf("matcher fallback = %+v, %v", event, err)
	}

	event, err = ClaudeCode(map[string]any{
		"session_id": "session-1", "hook_event_name": "Notification",
		"notification_type": "unknown", "matcher": "permission_prompt",
	})
	if err != nil || !event.Ignored {
		t.Fatalf("non-empty notification type unexpectedly used matcher: %+v, %v", event, err)
	}
}

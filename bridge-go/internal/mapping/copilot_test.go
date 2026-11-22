package mapping

import (
	"testing"

	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

func copilotPayload(name string) map[string]any {
	return map[string]any{
		"session_id":      "copilot-session",
		"timestamp":       "2026-11-22T16:00:00.000Z",
		"cwd":             "/workspace/project",
		"hook_event_name": name,
	}
}

func Test_req_AGT_04_CopilotMapsCoreStates(t *testing.T) {
	for _, test := range []struct {
		name  string
		state sessions.State
	}{
		{"SessionStart", sessions.IDLE},
		{"UserPromptSubmit", sessions.THINKING},
		{"PreToolUse", sessions.WORKING},
		{"PostToolUse", sessions.THINKING},
		{"Stop", sessions.COMPLETED},
	} {
		event, err := Copilot(copilotPayload(test.name))
		if err != nil {
			t.Fatalf("Copilot(%s): %v", test.name, err)
		}
		if event.Agent != "copilot" || event.Session != "/workspace/project" ||
			event.State != test.state || event.End || event.Ignored {
			t.Errorf("%s mapped to %+v, want state %s", test.name, event, test.state)
		}
	}
}

func Test_req_AGT_05_CopilotErrorOccurredMapsError(t *testing.T) {
	event, err := Copilot(copilotPayload("ErrorOccurred"))
	if err != nil {
		t.Fatal(err)
	}
	if event.State != sessions.ERROR || event.Session != "/workspace/project" {
		t.Fatalf("ErrorOccurred mapped to %+v, want ERROR for workspace", event)
	}
}

func Test_req_AGT_06_CopilotUsesWorkspaceAsSessionKey(t *testing.T) {
	first, err := Copilot(copilotPayload("SessionStart"))
	if err != nil {
		t.Fatal(err)
	}
	secondPayload := copilotPayload("SessionStart")
	secondPayload["cwd"] = "/workspace/other"
	second, err := Copilot(secondPayload)
	if err != nil {
		t.Fatal(err)
	}
	if first.Session == second.Session {
		t.Fatalf("different workspaces share a Copilot session key: %+v, %+v", first, second)
	}
}

func Test_req_AGT_08_CopilotAndClaudeCompatibleSourcesShareDedupKey(t *testing.T) {
	payload := copilotPayload("PreToolUse")
	copilotEvent, err := Copilot(payload)
	if err != nil {
		t.Fatal(err)
	}
	claudeRouteEvent, err := ClaudeCode(payload)
	if err != nil {
		t.Fatal(err)
	}
	if copilotEvent != claudeRouteEvent {
		t.Fatalf("same Copilot event mapped differently by its hook sources: %+v != %+v",
			copilotEvent, claudeRouteEvent)
	}
}

func Test_req_AGT_04_CopilotRequiresDeduplicationIdentity(t *testing.T) {
	for _, payload := range []map[string]any{
		{"cwd": "/workspace/project", "hook_event_name": "Stop"},
		{"session_id": "s1", "hook_event_name": "Stop"},
		{"session_id": "s1", "cwd": "/workspace/project", "hook_event_name": "Stop"},
	} {
		if _, err := Copilot(payload); err == nil {
			t.Errorf("Copilot(%v) unexpectedly succeeded", payload)
		}
	}
}

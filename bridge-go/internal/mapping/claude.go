package mapping

import (
	"fmt"

	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

const End = "END"

type Event struct {
	Agent    string
	Session  string
	Name     string
	State    sessions.State
	End      bool
	Ignored  bool
	DedupKey string
}

type PayloadError struct {
	Message string
}

func (e PayloadError) Error() string {
	return e.Message
}

func requiredText(payload map[string]any, key string) (string, error) {
	value, ok := payload[key].(string)
	if !ok || value == "" {
		return "", PayloadError{Message: fmt.Sprintf("missing '%s'", key)}
	}
	return value, nil
}

func ClaudeCode(payload map[string]any) (Event, error) {
	if isCopilotCompatiblePayload(payload) {
		return Copilot(payload)
	}
	session, err := requiredText(payload, "session_id")
	if err != nil {
		return Event{}, err
	}
	name, err := requiredText(payload, "hook_event_name")
	if err != nil {
		return Event{}, err
	}
	event := Event{Agent: "claude-code", Session: session, Name: name}
	switch name {
	case "SessionStart":
		event.State = sessions.IDLE
	case "UserPromptSubmit", "PostToolUse", "PostToolUseFailure":
		event.State = sessions.THINKING
	case "PreToolUse":
		event.State = sessions.WORKING
	case "PermissionRequest":
		event.State = sessions.APPROVAL
	case "Stop":
		event.State = sessions.COMPLETED
	case "StopFailure":
		event.State = sessions.ERROR
	case "SessionEnd":
		event.End = true
	case "Notification":
		kind := notificationKind(payload)
		if kind == "permission_prompt" || kind == "elicitation_dialog" {
			event.State = sessions.APPROVAL
		} else {
			event.Ignored = true
		}
	default:
		event.Ignored = true
	}
	return event, nil
}

func isCopilotCompatiblePayload(payload map[string]any) bool {
	if _, ok := payload["cwd"].(string); !ok {
		return false
	}
	if _, ok := payload["timestamp"]; !ok {
		return false
	}
	name, ok := payload["hook_event_name"].(string)
	if !ok {
		return false
	}
	switch name {
	case "SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse",
		"Stop", "SessionEnd", "ErrorOccurred":
		return true
	default:
		return false
	}
}

func notificationKind(payload map[string]any) string {
	value, exists := payload["notification_type"]
	if !exists || !truthy(value) {
		value = payload["matcher"]
	}
	kind, _ := value.(string)
	return kind
}

func truthy(value any) bool {
	switch value := value.(type) {
	case nil:
		return false
	case bool:
		return value
	case string:
		return value != ""
	case float64:
		return value != 0
	default:
		return true
	}
}

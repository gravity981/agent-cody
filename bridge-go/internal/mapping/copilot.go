package mapping

import "github.com/gravity981/agent-cody/bridge-go/internal/sessions"

func Copilot(payload map[string]any) (Event, error) {
	sessionID, err := requiredText(payload, "session_id")
	if err != nil {
		return Event{}, err
	}
	cwd, err := requiredText(payload, "cwd")
	if err != nil {
		return Event{}, err
	}
	name, err := requiredText(payload, "hook_event_name")
	if err != nil {
		return Event{}, err
	}
	dedupKey, err := hookDedupKey(sessionID, cwd, name, payload["timestamp"])
	if err != nil {
		return Event{}, err
	}

	event := Event{
		Agent: "copilot", Session: cwd, Name: name, DedupKey: dedupKey,
	}
	switch name {
	case "SessionStart", "sessionStart":
		event.State = sessions.IDLE
	case "UserPromptSubmit", "userPromptSubmitted", "PostToolUse", "postToolUse":
		event.State = sessions.THINKING
	case "PreToolUse", "preToolUse":
		event.State = sessions.WORKING
	case "Stop", "agentStop":
		event.State = sessions.COMPLETED
	case "SessionEnd", "sessionEnd":
		event.End = true
	case "ErrorOccurred", "errorOccurred":
		event.State = sessions.ERROR
	default:
		event.Ignored = true
	}
	return event, nil
}

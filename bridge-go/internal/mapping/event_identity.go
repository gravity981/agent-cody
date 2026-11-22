package mapping

import (
	"math"
	"strconv"
	"strings"
	"time"
)

func hookTimestampMillis(value any) (int64, bool) {
	switch value := value.(type) {
	case string:
		timestamp, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return 0, false
		}
		return timestamp.UnixMilli(), true
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value ||
			value < 0 || value >= float64(math.MaxInt64) {
			return 0, false
		}
		return int64(value), true
	case int:
		if value < 0 {
			return 0, false
		}
		return int64(value), true
	case int64:
		if value < 0 {
			return 0, false
		}
		return value, true
	default:
		return 0, false
	}
}

func hookDedupKey(sessionID, cwd, name string, timestamp any) (string, error) {
	millis, ok := hookTimestampMillis(timestamp)
	if !ok {
		return "", PayloadError{Message: "missing or invalid 'timestamp'"}
	}
	return strings.Join([]string{
		sessionID,
		cwd,
		canonicalHookName(name),
		strconv.FormatInt(millis, 10),
	}, "\x00"), nil
}

func canonicalHookName(name string) string {
	switch name {
	case "SessionStart", "sessionStart":
		return "sessionStart"
	case "UserPromptSubmit", "userPromptSubmitted":
		return "userPromptSubmitted"
	case "PreToolUse", "preToolUse":
		return "preToolUse"
	case "PostToolUse", "postToolUse":
		return "postToolUse"
	case "Stop", "agentStop":
		return "agentStop"
	case "SessionEnd", "sessionEnd":
		return "sessionEnd"
	case "ErrorOccurred", "errorOccurred":
		return "errorOccurred"
	default:
		return name
	}
}

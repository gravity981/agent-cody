# Hook payloads (test fixtures)

- `claude-code-session.jsonl`: a real Claude Code session recorded on 2026-11-06 with `hooks/install-claude-code --record` (SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest, Stop). Sanitised: prompts, paths, tool inputs and outputs, answers and ids are replaced; only the structure and the fields the mapping uses are kept.
- `claude-code-from-docs.jsonl`: events not in the recording, written from the Claude Code hooks documentation (Notification, StopFailure, SessionEnd). Replace them with recorded payloads when they occur.

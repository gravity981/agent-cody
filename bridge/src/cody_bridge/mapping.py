"""Agent hook payloads to session events (spec/architecture.md 4.1).

The hooks forward the agent's own payload; all mapping lives here.
"""

from __future__ import annotations

from dataclasses import dataclass

from .sessions import State

END = "END"  # the session ends


@dataclass(frozen=True)
class Event:
    agent: str
    session: str
    name: str
    state: State | str | None  # a State, END, or None (event ignored)


CLAUDE_CODE = {
    "SessionStart": State.IDLE,
    "UserPromptSubmit": State.THINKING,
    "PostToolUse": State.THINKING,
    "PostToolUseFailure": State.THINKING,
    "PreToolUse": State.WORKING,
    "PermissionRequest": State.APPROVAL,
    "Stop": State.COMPLETED,
    "StopFailure": State.ERROR,
    "SessionEnd": END,
}
CLAUDE_CODE_APPROVAL_NOTIFICATIONS = {"permission_prompt", "elicitation_dialog"}


class PayloadError(ValueError):
    """The payload lacks what the agent's hooks always send."""


def _text(payload: dict, key: str) -> str:
    value = payload.get(key)
    if not isinstance(value, str) or not value:
        raise PayloadError(f"missing '{key}'")
    return value


def claude_code(payload: dict) -> Event:
    session = _text(payload, "session_id")
    name = _text(payload, "hook_event_name")
    state = CLAUDE_CODE.get(name)
    if name == "Notification":
        # field name checked against recorded payloads in step 2c (AGT-01)
        kind = payload.get("notification_type") or payload.get("matcher")
        state = State.APPROVAL if kind in CLAUDE_CODE_APPROVAL_NOTIFICATIONS else None
    return Event("claude-code", session, name, state)


# Copilot follows in stage 3 (AGT-04)
AGENTS = {"claude-code": claude_code}

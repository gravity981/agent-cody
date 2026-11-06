"""Claude Code drives all six states (AGT-01), with recorded payloads.

Fixtures: tests/payloads/README.md.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest
from cody_bridge.bridge import Bridge
from cody_bridge.mapping import END, claude_code
from cody_bridge.sessions import Sessions, State
from fakes import FakePort, FakeSystem, run
from test_hooks import bridge_on_default_port, run_hook  # noqa: F401 (fixture)

PAYLOADS = Path(__file__).parent / "payloads"
RECORDED = [
    json.loads(line)
    for line in (PAYLOADS / "claude-code-session.jsonl").read_text().splitlines()
    if line
]
FROM_DOCS = {
    (p["hook_event_name"], p.get("notification_type")): p
    for p in (
        json.loads(line)
        for line in (PAYLOADS / "claude-code-from-docs.jsonl").read_text().splitlines()
        if line
    )
}

EXPECTED = {
    "SessionStart": State.IDLE,
    "UserPromptSubmit": State.THINKING,
    "PreToolUse": State.WORKING,
    "PostToolUse": State.THINKING,
    "PostToolUseFailure": State.THINKING,
    "PermissionRequest": State.APPROVAL,
    "Stop": State.COMPLETED,
}


@pytest.mark.parametrize("index", range(len(RECORDED)))
def test_req_AGT_01_recorded_events_map_to_their_state(index: int) -> None:
    payload = RECORDED[index]
    event = claude_code(payload)
    assert event.agent == "claude-code"
    assert event.session == payload["session_id"]
    assert event.state == EXPECTED[payload["hook_event_name"]]


def test_req_AGT_01_recorded_session_drives_the_device() -> None:
    system = FakeSystem()
    port = system.plug("/dev/ttyACM0", FakePort())
    link = system.link()
    run(link, 0, 0.5)
    bridge = Bridge(Sessions(600), link)
    for t, payload in enumerate(RECORDED, start=1):
        bridge.event(claude_code(payload), t)
    sent = [c.removeprefix("STATE:") for c in port.commands if c.startswith("STATE:")][1:]
    # first prompt: read a file; then a write with permission; a failing command;
    # a command with permission (each ends with Stop)
    assert sent == [
        "THINKING", "WORKING", "THINKING", "COMPLETED",
        "THINKING", "WORKING", "APPROVAL", "THINKING", "COMPLETED",
        "THINKING", "WORKING", "THINKING", "COMPLETED",
        "THINKING", "WORKING", "APPROVAL", "THINKING", "COMPLETED",
    ]  # fmt: skip


def test_req_AGT_01_documented_events_cover_approval_error_and_end() -> None:
    assert claude_code(FROM_DOCS[("Notification", "permission_prompt")]).state == State.APPROVAL
    assert claude_code(FROM_DOCS[("Notification", "idle_prompt")]).state is None  # ignored
    assert claude_code(FROM_DOCS[("StopFailure", None)]).state == State.ERROR
    assert claude_code(FROM_DOCS[("SessionEnd", None)]).state == END


def test_req_AGT_01_all_six_states_are_reachable() -> None:
    states = {claude_code(p).state for p in RECORDED + list(FROM_DOCS.values())}
    assert set(State) <= states


def test_req_AGT_01_session_end_removes_the_session() -> None:
    bridge = Bridge(Sessions(600), FakeSystem().link())
    bridge.event(claude_code(RECORDED[-1]), 1)  # Stop: COMPLETED
    assert bridge.sessions.merged() == State.COMPLETED
    bridge.event(claude_code(FROM_DOCS[("SessionEnd", None)]), 2)
    assert bridge.sessions.all() == []


def test_req_AGT_01_recorded_session_through_the_real_hook(
    bridge_on_default_port: Bridge,  # noqa: F811
) -> None:
    for payload in RECORDED[:4]:  # SessionStart, prompt, tool, tool done
        result, _ = run_hook(payload)
        assert result.returncode == 0 and result.stdout == ""
    assert bridge_on_default_port.sessions.merged() == State.THINKING

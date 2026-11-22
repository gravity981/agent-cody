"""Hook delivery behavior (AGT-07)."""

from __future__ import annotations

import importlib.machinery
import importlib.util
import json
import socket
import subprocess
import time
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "hooks" / "install-claude-code"
COPILOT_FORWARD = ROOT / "hooks" / "cody-forward.sh"
COPILOT_CONFIG = ROOT / "hooks" / "copilot-hooks.json"
loader = importlib.machinery.SourceFileLoader("install_claude_code", str(SCRIPT))
spec = importlib.util.spec_from_loader(loader.name, loader)
installer = importlib.util.module_from_spec(spec)
loader.exec_module(installer)


def _port_free(port: int) -> bool:
    with socket.socket() as connection:
        return connection.connect_ex(("127.0.0.1", port)) != 0


def test_req_AGT_07_without_bridge_returns_fast_silently_and_successfully() -> None:
    if not _port_free(installer.PORT):
        pytest.skip("something listens on the bridge port here")
    start = time.monotonic()
    result = subprocess.run(
        ["sh", "-c", installer.hook_command()],
        input=json.dumps({"session_id": "s1", "hook_event_name": "PreToolUse"}),
        capture_output=True,
        text=True,
        timeout=10,
    )
    elapsed = time.monotonic() - start
    assert result.returncode == 0
    assert result.stdout == ""
    assert result.stderr == ""
    assert elapsed < 1.0


def test_req_AGT_07_copilot_forwarder_without_bridge_returns_fast_silently() -> None:
    if not _port_free(installer.PORT):
        pytest.skip("something listens on the bridge port here")
    start = time.monotonic()
    result = subprocess.run(
        ["sh", str(COPILOT_FORWARD)],
        input=json.dumps(
            {
                "session_id": "s1",
                "timestamp": "2026-11-22T16:00:00Z",
                "cwd": "/workspace/project",
                "hook_event_name": "PreToolUse",
            }
        ),
        capture_output=True,
        text=True,
        timeout=10,
    )
    elapsed = time.monotonic() - start
    assert result.returncode == 0
    assert result.stdout == ""
    assert result.stderr == ""
    assert elapsed < 1.0


def test_req_AGT_09_copilot_user_hook_config_is_valid_json() -> None:
    config = json.loads(COPILOT_CONFIG.read_text(encoding="utf-8"))
    assert config["version"] == 1
    assert set(config["hooks"]) == {
        "SessionStart",
        "UserPromptSubmit",
        "PreToolUse",
        "PostToolUse",
        "Stop",
        "SessionEnd",
        "ErrorOccurred",
    }

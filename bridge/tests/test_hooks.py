"""Claude Code hooks: install script and the hook command (AGT-03, AGT-07, AGT-11)."""

from __future__ import annotations

import importlib.machinery
import importlib.util
import json
import socket
import subprocess
import time
from collections.abc import Iterator
from pathlib import Path

import pytest
from cody_bridge.bridge import Bridge
from cody_bridge.server import serve
from cody_bridge.sessions import Sessions, State
from fakes import FakeSystem

ROOT = Path(__file__).resolve().parents[2]
SCRIPT = ROOT / "hooks" / "install-claude-code"


def load_installer():
    loader = importlib.machinery.SourceFileLoader("install_claude_code", str(SCRIPT))
    spec = importlib.util.spec_from_loader(loader.name, loader)
    module = importlib.util.module_from_spec(spec)
    loader.exec_module(module)
    return module


installer = load_installer()


def install(settings: Path, *args: str) -> dict:
    assert installer.main(["--settings", str(settings), *args]) == 0
    return json.loads(settings.read_text())


def run_hook(payload: dict) -> tuple[subprocess.CompletedProcess, float]:
    start = time.monotonic()
    result = subprocess.run(
        ["sh", "-c", installer.hook_command()],
        input=json.dumps(payload),
        capture_output=True,
        text=True,
        timeout=10,
    )
    return result, time.monotonic() - start


def port_free(port: int) -> bool:
    with socket.socket() as s:
        return s.connect_ex(("127.0.0.1", port)) != 0


# --- AGT-03: set up once at user level, other settings kept ---


def test_req_AGT_03_installs_all_events_at_user_level(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setenv("CLAUDE_CONFIG_DIR", str(tmp_path))
    assert installer.main([]) == 0
    hooks = json.loads((tmp_path / "settings.json").read_text())["hooks"]
    assert set(hooks) == set(installer.EVENTS)
    for groups in hooks.values():
        assert groups == [
            {"hooks": [{"type": "command", "command": installer.hook_command(), "timeout": 2}]}
        ]


def test_req_AGT_03_keeps_other_settings_and_is_idempotent(tmp_path: Path) -> None:
    settings = tmp_path / "settings.json"
    other = {"type": "command", "command": "echo mine"}
    settings.write_text(
        json.dumps({"model": "opus", "hooks": {"Stop": [{"hooks": [other]}]}}, indent=2)
    )
    install(settings)
    data = install(settings)  # second run replaces, doesn't duplicate
    assert data["model"] == "opus"
    assert data["hooks"]["Stop"][0] == {"hooks": [other]}
    assert len(data["hooks"]["Stop"]) == 2
    assert (tmp_path / "settings.json.bak").exists()


def test_req_AGT_03_uninstall_removes_only_cody_hooks(tmp_path: Path) -> None:
    settings = tmp_path / "settings.json"
    other = {"type": "command", "command": "echo mine"}
    settings.write_text(json.dumps({"hooks": {"Stop": [{"hooks": [other]}]}}))
    install(settings)
    data = install(settings, "--uninstall")
    assert data == {"hooks": {"Stop": [{"hooks": [other]}]}}


def test_req_AGT_03_record_option_appends_payloads(tmp_path: Path) -> None:
    record = tmp_path / "payloads.jsonl"
    command = installer.hook_command(str(record))
    subprocess.run(["sh", "-c", command], input='{"a": 1}', text=True, timeout=10, check=True)
    assert record.read_text() == '{"a": 1}\n'


# --- AGT-07: never blocks, never fails the agent ---


def test_req_AGT_07_without_bridge_returns_fast_silently_and_successfully() -> None:
    if not port_free(installer.PORT):
        pytest.skip("something listens on the bridge port here")
    result, seconds = run_hook({"session_id": "s1", "hook_event_name": "PreToolUse"})
    assert result.returncode == 0
    assert result.stdout == ""
    assert result.stderr == ""
    assert seconds < 1.0


# --- AGT-11: finds the bridge; same command everywhere ---


@pytest.fixture
def bridge_on_default_port() -> Iterator[Bridge]:
    if not port_free(installer.PORT):
        pytest.skip("something listens on the bridge port here")
    bridge = Bridge(Sessions(600), FakeSystem().link())
    servers = serve(bridge, ["127.0.0.1"], installer.PORT)
    yield bridge
    for s in servers:
        s.shutdown()
        s.server_close()


def test_req_AGT_11_hook_delivers_events_to_the_bridge(bridge_on_default_port: Bridge) -> None:
    result, seconds = run_hook({"session_id": "s1", "hook_event_name": "PreToolUse"})
    assert result.returncode == 0 and result.stdout == ""
    assert bridge_on_default_port.sessions.merged() == State.WORKING
    assert seconds < 1.0


def test_req_AGT_11_large_payload_is_delivered(bridge_on_default_port: Bridge) -> None:
    payload = {
        "session_id": "s1",
        "hook_event_name": "PostToolUse",
        "tool_response": "x" * 3_000_000,  # larger than a command-line argument may be
    }
    result, _ = run_hook(payload)
    assert result.returncode == 0
    assert bridge_on_default_port.sessions.merged() == State.THINKING


def test_req_AGT_11_command_tries_host_then_container_addresses() -> None:
    command = installer.hook_command()
    assert "for h in 127.0.0.1 host.docker.internal 172.17.0.1; do" in command
    assert "/home/" not in command and "~" not in command  # no file paths

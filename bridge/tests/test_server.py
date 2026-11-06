"""HTTP endpoints and the host command `cody` (BRG-01, BRG-08 – BRG-12, BRG-15)."""

from __future__ import annotations

import json
import os
import subprocess
import urllib.error
import urllib.request
from collections.abc import Iterator
from pathlib import Path

import pytest
from cody_bridge.bridge import Bridge
from cody_bridge.server import serve
from cody_bridge.sessions import Sessions, State
from fakes import FakePort, FakeSystem, run

ROOT = Path(__file__).resolve().parents[2]
CODY = ROOT / "cli" / "cody"


class Running:
    def __init__(self) -> None:
        self.system = FakeSystem()
        self.port = self.system.plug("/dev/ttyACM0", FakePort(version="1.2.3 abc"))
        self.link = self.system.link()
        self.bridge = Bridge(Sessions(600), self.link)
        run(self.link, 0, 0.5)
        self.servers = serve(self.bridge, ["127.0.0.1"], 0)
        host, port = self.servers[0].server_address[:2]
        self.url = f"http://{host}:{port}"

    def request(self, method: str, path: str, body: object = None) -> tuple[int, str]:
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(self.url + path, data=data, method=method)
        req.add_header("Content-Type", "application/json")
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                return resp.status, resp.read().decode()
        except urllib.error.HTTPError as err:
            return err.code, err.read().decode()

    def cody(self, *args: str) -> subprocess.CompletedProcess:
        env = {**os.environ, "CODY_BRIDGE_URL": self.url}
        return subprocess.run([str(CODY), *args], env=env, capture_output=True, text=True)


@pytest.fixture
def running(monkeypatch: pytest.MonkeyPatch) -> Iterator[Running]:
    monkeypatch.setenv("CODY_VERSION", "2.0.0")
    monkeypatch.setenv("CODY_COMMIT", "feed123")
    r = Running()
    yield r
    for server in r.servers:
        server.shutdown()
        server.server_close()


def test_req_BRG_01_receives_agent_events_over_http(running: Running) -> None:
    status, _ = running.request(
        "POST", "/hooks/claude-code", {"session_id": "s1", "hook_event_name": "PreToolUse"}
    )
    assert status == 200
    sessions = json.loads(running.request("GET", "/status")[1])["sessions"]
    assert [(s["agent"], s["session"], s["state"]) for s in sessions] == [
        ("claude-code", "s1", "WORKING")
    ]
    assert "STATE:WORKING" in running.port.commands


@pytest.mark.parametrize(
    "path,body,code",
    [
        ("/hooks/unknown-agent", {"session_id": "s"}, 404),
        ("/hooks/claude-code", {"hook_event_name": "Stop"}, 400),  # no session id
        ("/hooks/claude-code", ["not", "an", "object"], 400),
    ],
)
def test_req_BRG_01_bad_requests_are_refused_without_effect(
    running: Running, path: str, body: object, code: int
) -> None:
    assert running.request("POST", path, body)[0] == code
    assert json.loads(running.request("GET", "/status")[1])["sessions"] == []


def test_req_BRG_08_listens_only_on_the_given_addresses(running: Running) -> None:
    assert [s.server_address[0] for s in running.servers] == ["127.0.0.1"]


def test_req_BRG_08_unavailable_address_is_skipped(running: Running) -> None:
    servers = serve(running.bridge, ["127.0.0.1", "172.31.255.254"], 0)
    try:
        assert [s.server_address[0] for s in servers] == ["127.0.0.1"]
    finally:
        for s in servers:
            s.shutdown()
            s.server_close()


def test_req_BRG_09_compose_restarts_the_bridge_with_docker() -> None:
    compose = (ROOT / "bridge" / "compose.yaml").read_text()
    assert "restart: unless-stopped" in compose
    assert "network_mode: host" in compose


def test_req_BRG_10_status_reports_bridge_version_and_commit(running: Running) -> None:
    status = json.loads(running.request("GET", "/status")[1])
    assert status["bridge"] == {"version": "2.0.0", "commit": "feed123"}


def test_req_BRG_11_cody_status_shows_device_sessions_and_state(running: Running) -> None:
    running.request("POST", "/hooks/claude-code", {"session_id": "s1", "hook_event_name": "Stop"})
    result = running.cody("status")
    assert result.returncode == 0, result.stderr
    assert "device:   connected on /dev/ttyACM0, firmware 1.2.3 abc" in result.stdout
    assert "state:    COMPLETED" in result.stdout
    assert "claude-code  COMPLETED  s1" in result.stdout


def test_req_BRG_11_cody_status_without_device(running: Running) -> None:
    running.system.unplug("/dev/ttyACM0")
    running.bridge.tick(1)
    assert "device:   not connected" in running.cody("status").stdout


def test_req_BRG_11_cody_reports_unreachable_bridge(tmp_path: Path) -> None:
    env = {**os.environ, "CODY_BRIDGE_URL": "http://127.0.0.1:9"}
    result = subprocess.run([str(CODY), "status"], env=env, capture_output=True, text=True)
    assert result.returncode == 1
    assert "doesn't answer" in result.stderr


def test_req_BRG_12_cody_state_sets_and_clears_a_manual_state(running: Running) -> None:
    assert running.cody("state", "WORKING").returncode == 0
    assert running.bridge.sessions.merged() == State.WORKING
    assert "STATE:WORKING" in running.port.commands
    assert running.cody("state", "clear").returncode == 0
    assert running.bridge.sessions.merged() == State.IDLE


def test_req_BRG_12_cody_state_refuses_unknown_states(running: Running) -> None:
    result = running.cody("state", "SLEEPING")
    assert result.returncode == 2
    assert running.bridge.sessions.all() == []
    assert running.request("POST", "/state", {"state": "SLEEPING"})[0] == 400


def test_req_BRG_15_cody_session_clear_removes_all_sessions(running: Running) -> None:
    claude = {"session_id": "s1", "hook_event_name": "UserPromptSubmit"}
    assert running.request("POST", "/hooks/claude-code", claude)[0] == 200
    assert running.cody("state", "APPROVAL").returncode == 0
    assert running.bridge.sessions.merged() == State.APPROVAL
    assert running.cody("session", "clear").returncode == 0
    assert running.bridge.sessions.all() == []
    assert running.bridge.sessions.merged() == State.IDLE
    assert running.port.commands[-1] == "STATE:IDLE"


def test_req_BRG_15_cody_session_needs_clear(running: Running) -> None:
    running.cody("state", "WORKING")
    assert running.cody("session").returncode == 2
    assert running.cody("session", "drop").returncode == 2
    assert running.bridge.sessions.merged() == State.WORKING


def test_req_BRG_10_cody_version() -> None:
    result = subprocess.run([str(CODY), "--version"], capture_output=True, text=True)
    assert result.stdout == "cody 0.0.0+unknown\n"

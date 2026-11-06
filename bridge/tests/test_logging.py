"""The bridge's log (BRG-13) and its size limit (BRG-14)."""

from __future__ import annotations

import io
import logging
import re
from pathlib import Path

import pytest
from cody_bridge import __main__ as cli
from cody_bridge.bridge import Bridge
from cody_bridge.config import ConfigError, parse
from cody_bridge.mapping import Event
from cody_bridge.sessions import Sessions, State
from fakes import FakePort, FakeSystem, run

ROOT = Path(__file__).resolve().parents[2]


def messages(caplog: pytest.LogCaptureFixture, level: int = logging.INFO) -> list[str]:
    return [r.getMessage() for r in caplog.records if r.levelno == level]


def event(name: str, state: State) -> Event:
    return Event("claude-code", "s1", name, state)


def test_req_BRG_13_state_change_logged_without_device(caplog: pytest.LogCaptureFixture) -> None:
    caplog.set_level(logging.INFO)
    bridge = Bridge(Sessions(600), FakeSystem().link())
    bridge.event(event("PreToolUse", State.WORKING), 1)
    assert messages(caplog) == ["state WORKING (device not connected) <- claude-code s1 PreToolUse"]


def test_req_BRG_13_state_change_logged_as_sent_with_device(
    caplog: pytest.LogCaptureFixture,
) -> None:
    system = FakeSystem()
    system.plug("/dev/ttyACM0", FakePort())
    link = system.link()
    run(link, 0, 0.5)
    caplog.clear()
    caplog.set_level(logging.INFO)
    bridge = Bridge(Sessions(600), link)
    bridge.event(event("Stop", State.COMPLETED), 1)
    bridge.event(event("Stop", State.COMPLETED), 2)  # no change: no line
    assert messages(caplog) == ["state COMPLETED (sent) <- claude-code s1 Stop"]


def test_req_BRG_13_timeout_and_manual_changes_name_their_cause(
    caplog: pytest.LogCaptureFixture,
) -> None:
    caplog.set_level(logging.INFO)
    bridge = Bridge(Sessions(60), FakeSystem().link())
    bridge.set_manual(State.THINKING, 0)
    bridge.tick(61)
    assert messages(caplog) == [
        "state THINKING (device not connected) <- manual THINKING",
        "state IDLE (device not connected) <- session timeout",
    ]


def test_req_BRG_13_device_connect_and_loss_are_logged(caplog: pytest.LogCaptureFixture) -> None:
    caplog.set_level(logging.INFO)
    system = FakeSystem()
    system.plug("/dev/ttyACM0", FakePort(version="1.2.3 abc"))
    link = system.link()
    run(link, 0, 0.5)
    system.unplug("/dev/ttyACM0")
    run(link, 0.6, 1)
    lines = messages(caplog)
    assert "connected to /dev/ttyACM0 (firmware 1.2.3 abc)" in lines
    assert any(line.startswith("lost /dev/ttyACM0") for line in lines)


def test_req_BRG_13_events_and_serial_traffic_at_debug(caplog: pytest.LogCaptureFixture) -> None:
    caplog.set_level(logging.DEBUG)
    system = FakeSystem()
    system.plug("/dev/ttyACM0", FakePort())
    link = system.link()
    bridge = Bridge(Sessions(600), link)
    run(link, 0, 6)
    bridge.event(event("PreToolUse", State.WORKING), 6)
    debug = messages(caplog, logging.DEBUG)
    assert "-> VERSION?" in debug
    assert "<- VERSION 1.2.3 abc" in debug
    assert "-> PING" in debug
    assert "<- OK" in debug
    assert "event claude-code s1 PreToolUse -> WORKING" in debug
    assert "-> STATE:WORKING" in debug


def test_req_BRG_13_log_lines_are_formatted(monkeypatch: pytest.MonkeyPatch) -> None:
    out = io.StringIO()
    monkeypatch.setattr(cli.sys, "stdout", out)
    root = logging.getLogger()
    saved = (root.handlers[:], root.level)
    try:
        cli.setup_logging("info")
        logging.getLogger("cody_bridge.bridge").info("state WORKING (sent) <- x")
        logging.getLogger("cody_bridge.device").debug("-> PING")  # below info: not written
    finally:
        root.handlers[:], _ = saved
        root.setLevel(saved[1])
    assert re.fullmatch(
        r"\d{4}-\d\d-\d\d \d\d:\d\d:\d\d INFO  state WORKING \(sent\) <- x\n", out.getvalue()
    )


def test_req_BRG_13_log_level_is_configurable() -> None:
    assert parse({}).log_level == "info"
    assert parse({"log_level": "debug"}).log_level == "debug"
    with pytest.raises(ConfigError, match="log_level"):
        parse({"log_level": "verbose"})


def test_req_BRG_14_log_limited_to_two_files_of_10_mb() -> None:
    compose = (ROOT / "bridge" / "compose.yaml").read_text()
    assert re.search(
        r'logging:\n\s+driver: json-file\n\s+options:\n\s+max-size: "10m"\n\s+max-file: "2"',
        compose,
    )

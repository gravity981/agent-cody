"""The serial connection, with a fake device (BRG-03, BRG-05, BRG-06, BRG-07)."""

from __future__ import annotations

from cody_bridge.sessions import State
from fakes import FakePort, FakeSystem, run


def connected(system: FakeSystem, device: str = "/dev/ttyACM0", **link_args):
    port = system.plug(device, FakePort())
    link = system.link(**link_args)
    run(link, 0, 0.5)
    assert link.connected
    return link, port


def test_req_BRG_03_state_sent_only_when_it_changes() -> None:
    link, port = connected(FakeSystem())
    port.commands.clear()
    link.set_state(State.WORKING, 1)
    link.set_state(State.WORKING, 2)
    link.set_state(State.WORKING, 3)
    link.set_state(State.THINKING, 4)
    assert [c for c in port.commands if c.startswith("STATE")] == [
        "STATE:WORKING",
        "STATE:THINKING",
    ]


def test_req_BRG_05_heartbeat_at_least_every_10_seconds() -> None:
    link, port = connected(FakeSystem())
    port.commands.clear()
    run(link, 0.6, 60.6)
    assert port.commands.count("PING") >= 6
    # no gap of 10 s without anything sent: PING every 5 s at most apart
    assert port.commands.count("PING") >= 60 / 5 - 1


def test_req_BRG_06_finds_device_by_espressif_vid_and_handshake() -> None:
    system = FakeSystem()
    system.plug("/dev/ttyUSB0", FakePort(), vid=0x1A86)  # some other USB serial device
    system.plug("/dev/ttyACM0", FakePort(version="0.1.0 71d95bc"))
    link = system.link()
    run(link, 0, 0.5)
    assert link.connected
    assert link.device == "/dev/ttyACM0"
    assert link.firmware == "0.1.0 71d95bc"
    assert system.opened == ["/dev/ttyACM0"]


def test_req_BRG_06_runs_without_device_and_connects_within_10_s_when_plugged() -> None:
    system = FakeSystem()
    link = system.link()
    t = run(link, 0, 30)
    assert not link.connected
    system.plug("/dev/ttyACM0", FakePort())
    run(link, t, t + 10)
    assert link.connected


def test_req_BRG_06_reconnects_after_replug_and_resends_settings_and_state() -> None:
    system = FakeSystem()
    link, first = connected(system, speed=90, volume=1)
    link.set_state(State.APPROVAL, 1)
    system.unplug("/dev/ttyACM0")
    run(link, 1, 2)
    assert not link.connected
    second = system.plug("/dev/ttyACM1", FakePort())  # comes back under another name
    run(link, 2, 12)
    assert link.connected
    assert second.commands[:4] == ["VERSION?", "SPEED:90", "VOLUME:1", "STATE:APPROVAL"]


def test_req_BRG_06_resends_after_device_reset() -> None:
    link, port = connected(FakeSystem(), speed=70)
    link.set_state(State.ERROR, 1)
    port.commands.clear()
    port.restart()
    run(link, 1, 1.5)
    assert port.commands[:3] == ["SPEED:70", "VOLUME:2", "STATE:ERROR"]


def test_req_BRG_06_skips_espressif_devices_that_are_no_cody() -> None:
    system = FakeSystem()
    system.plug("/dev/ttyACM0", FakePort(cody=False))  # e.g. another ESP32 project
    system.plug("/dev/ttyACM1", FakePort())
    link = system.link()
    run(link, 0, 10)
    assert link.connected
    assert link.device == "/dev/ttyACM1"


def test_req_BRG_06_configured_port_overrides_discovery() -> None:
    system = FakeSystem()
    system.plug("/dev/ttyACM0", FakePort())
    system.plug("/dev/ttyACM5", FakePort())
    link = system.link(serial_port="/dev/ttyACM5")
    run(link, 0, 1)
    assert link.device == "/dev/ttyACM5"


def test_req_BRG_07_device_settings_sent_on_every_connect() -> None:
    system = FakeSystem()
    _, port = connected(system, speed=200, volume=0)
    assert port.commands[:4] == ["VERSION?", "SPEED:200", "VOLUME:0", "STATE:IDLE"]

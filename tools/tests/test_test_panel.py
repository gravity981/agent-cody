"""Tests for the developer hardware test panel."""

from __future__ import annotations

import http.client
import json
import threading
from collections import deque
from http.server import ThreadingHTTPServer
from types import SimpleNamespace
from typing import Any

import pytest
import serial
from cody_tools.test_panel import (
    BAUDRATE,
    PanelController,
    PanelError,
    _make_handler,
)


class FakeSerial:
    def __init__(self, port: str, **kwargs: Any) -> None:
        self.port = port
        self.kwargs = kwargs
        self.commands: list[str] = []
        self.responses: deque[bytes] = deque()
        self.is_open = True
        self.respond = True

    def write(self, data: bytes) -> int:
        command = data.decode("ascii").strip()
        self.commands.append(command)
        if not self.respond:
            return len(data)
        if command == "VERSION?":
            self.responses.extend((b"VERSION 1.2.3 abc\n", b"OK\n"))
        else:
            self.responses.append(b"OK\n")
        return len(data)

    def readline(self) -> bytes:
        return self.responses.popleft() if self.responses else b""

    def close(self) -> None:
        self.is_open = False


def make_controller(
    factory: Any = FakeSerial,
) -> tuple[PanelController, list[FakeSerial]]:
    devices: list[FakeSerial] = []

    def serial_factory(port: str, **kwargs: Any) -> FakeSerial:
        device = factory(port, **kwargs)
        devices.append(device)
        return device

    controller = PanelController(
        serial_factory=serial_factory,
        ports_provider=lambda: [SimpleNamespace(device="/dev/ttyACM0", description="Cody")],
    )
    return controller, devices


def test_req_TST_06_panel_connects_directly_and_owns_serial() -> None:
    controller, devices = make_controller()
    controller.connect("/dev/ttyACM0")
    controller.start_test()
    controller.dispatch("/api/test/speed", {"value": 112})
    controller.dispatch("/api/test/frequency", {"value": 30000})
    controller.dispatch("/api/test/kick", {"value": 0})
    controller.dispatch("/api/test/decay", {"value": "SLOW"})
    controller.dispatch("/api/test/motor", {"value": "START"})
    controller.dispatch("/api/test/state", {"value": "WORKING"})
    controller.dispatch("/api/test/volume", {"value": 3})
    controller.dispatch("/api/test/led", {"r": 1, "g": 2, "b": 3})
    controller.dispatch("/api/test/beep", {"value": "SUCCESS"})
    controller.heartbeat()
    status = controller.status()

    assert devices[0].kwargs["baudrate"] == BAUDRATE
    assert devices[0].kwargs["exclusive"] is True
    assert devices[0].commands == [
        "VERSION?",
        "TEST:START",
        "TEST:SPEED:112",
        "TEST:FREQ:30000",
        "TEST:KICK:0",
        "TEST:DECAY:SLOW",
        "TEST:MOTOR:START",
        "TEST:STATE:WORKING",
        "TEST:VOLUME:3",
        "TEST:LED:1,2,3",
        "TEST:BEEP:SUCCESS",
        "PING",
    ]
    assert status["connected"] is True
    assert status["testMode"] is True
    assert status["firmware"] == "1.2.3 abc"
    controller.close()


def test_req_TST_06_port_open_errors_are_reported() -> None:
    def busy_port(_port: str, **_kwargs: Any) -> None:
        raise serial.SerialException("port busy")

    controller, _ = make_controller(busy_port)
    with pytest.raises(PanelError, match="port busy"):
        controller.connect("/dev/ttyACM0")
    assert controller.status()["error"] == "port busy"


def test_req_TST_03_panel_rejects_out_of_range_tuning() -> None:
    controller, devices = make_controller()
    controller.connect("/dev/ttyACM0")
    controller.start_test()
    command_count = len(devices[0].commands)

    with pytest.raises(ValueError, match="Frequency"):
        controller.dispatch("/api/test/frequency", {"value": 30001})
    with pytest.raises(ValueError, match="Kick"):
        controller.dispatch("/api/test/kick", {"value": 101})
    assert len(devices[0].commands) == command_count
    controller.close()


def test_req_TST_07_browser_heartbeat_loss_stops_test_and_closes_serial() -> None:
    controller, devices = make_controller()
    controller.connect("/dev/ttyACM0")
    controller.start_test()
    controller.watchdog_once(now=controller._last_ui_heartbeat + 11)

    assert devices[0].commands[-1] == "TEST:STOP"
    assert devices[0].is_open is False
    assert controller.status()["connected"] is False
    assert controller.status()["testMode"] is False


def test_req_TST_07_browser_heartbeat_loss_closes_idle_serial() -> None:
    controller, devices = make_controller()
    controller.connect("/dev/ttyACM0")
    controller.watchdog_once(now=controller._last_ui_heartbeat + 11)

    assert devices[0].commands == ["VERSION?"]
    assert devices[0].is_open is False
    assert controller.status()["connected"] is False


def test_req_TST_07_missing_serial_reply_releases_port(monkeypatch: pytest.MonkeyPatch) -> None:
    controller, devices = make_controller()
    controller.connect("/dev/ttyACM0")
    controller.start_test()
    devices[0].respond = False
    monkeypatch.setattr("cody_tools.test_panel.SERIAL_TIMEOUT_S", 0)

    with pytest.raises(PanelError, match="No reply"):
        controller.dispatch("/api/test/motor", {"value": "START"})

    assert devices[0].is_open is False
    assert controller.status()["connected"] is False
    assert controller.status()["testMode"] is False


def test_req_TST_06_http_panel_serves_controls_and_rejects_cross_origin() -> None:
    controller, devices = make_controller()
    server = ThreadingHTTPServer(("127.0.0.1", 0), _make_handler(controller))
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    host, port = server.server_address
    origin = f"http://{host}:{port}"
    connection = http.client.HTTPConnection(host, port, timeout=2)
    try:
        connection.request("GET", "/")
        page = connection.getresponse()
        assert page.status == 200
        assert "State presets" in page.read().decode()

        connection.request(
            "POST",
            "/api/connect",
            body=json.dumps({"port": "/dev/ttyACM0"}),
            headers={"Content-Type": "application/json"},
        )
        response = connection.getresponse()
        assert response.status == 403
        response.read()

        connection.request(
            "POST",
            "/api/connect",
            body=json.dumps({"port": "/dev/ttyACM0"}),
            headers={"Content-Type": "application/json", "Origin": origin},
        )
        response = connection.getresponse()
        assert response.status == 200
        response.read()
        connection.request(
            "POST",
            "/api/test/start",
            body="{}",
            headers={"Content-Type": "application/json", "Origin": origin},
        )
        response = connection.getresponse()
        assert response.status == 200
        response.read()
        assert devices[0].commands[-1] == "TEST:START"
    finally:
        connection.close()
        controller.close()
        server.shutdown()
        server.server_close()
        thread.join(timeout=2)

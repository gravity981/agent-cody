"""Developer-only browser panel for testing Cody firmware over USB serial."""

from __future__ import annotations

import argparse
import json
import logging
import threading
import time
from collections import deque
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any
from urllib.parse import urlsplit

import serial
from serial.tools import list_ports

PORT = 8766
BAUDRATE = 115200
HEARTBEAT_TIMEOUT_S = 10
SERIAL_TIMEOUT_S = 3
MAX_REQUEST_BYTES = 4096
STATES = ("IDLE", "WORKING", "THINKING", "APPROVAL", "COMPLETED", "ERROR")
SOUNDS = ("APPROVAL", "SUCCESS", "WARNING")

HTML_PAGE = """<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Cody hardware test panel</title>
  <style>
    :root { color-scheme: light dark; font: 16px system-ui, sans-serif; }
    body { max-width: 900px; margin: 2rem auto; padding: 0 1rem; }
    h1 { margin-bottom: .25rem; }
    .hint { opacity: .8; }
    section { border: 1px solid #8888; border-radius: .6rem; padding: 1rem; margin: 1rem 0; }
    .row { display: flex; flex-wrap: wrap; align-items: end; gap: .75rem; }
    label { display: grid; gap: .25rem; }
    input, select, button { font: inherit; padding: .45rem .6rem; }
    input[type=number] { width: 7rem; }
    button { cursor: pointer; }
    button:disabled, input:disabled, select:disabled { cursor: not-allowed; opacity: .55; }
    #status { font-weight: 650; }
    #error { color: #d33; min-height: 1.5em; }
    pre { max-height: 12rem; overflow: auto; white-space: pre-wrap; }
  </style>
</head>
<body>
  <h1>Cody hardware test panel</h1>
  <p class="hint">The panel owns USB serial while connected.</p>
  <p class="hint">Stop the Cody bridge before connecting.</p>
  <p id="status">Loading device status...</p>
  <p id="error" role="alert"></p>

  <section>
    <h2>Device</h2>
    <div class="row">
      <label>Serial device
        <select id="port"></select>
      </label>
      <button id="refresh-ports" type="button">Refresh</button>
      <button id="connect" type="button">Connect</button>
      <button id="disconnect" type="button">Disconnect</button>
      <button id="test-start" type="button">Enter test mode</button>
      <button id="test-stop" type="button">Exit test mode</button>
    </div>
  </section>

  <section>
    <h2>State presets</h2>
    <p>These run the normal state effects while test mode remains active.</p>
    <div id="states" class="row"></div>
  </section>

  <section>
    <h2>Motor</h2>
    <div class="row">
      <label>Speed (0-255)
        <input id="speed" type="number" min="0" max="255" value="224">
      </label>
      <label>PWM frequency (Hz)
        <input id="frequency" type="number" min="1000" max="30000" step="100" value="10000">
      </label>
      <label>Startup kick (ms)
        <input id="kick" type="number" min="0" max="100" value="1">
      </label>
      <label>Decay mode
        <select id="decay"><option>FAST</option><option>SLOW</option></select>
      </label>
      <button id="apply-motor" type="button">Apply tuning</button>
      <button id="motor-start" type="button">Start motor</button>
      <button id="motor-stop" type="button">Stop motor</button>
    </div>
  </section>

  <section>
    <h2>LED</h2>
    <div class="row">
      <label>Red (0-255) <input id="red" type="number" min="0" max="255" value="255"></label>
      <label>Green (0-255) <input id="green" type="number" min="0" max="255" value="0"></label>
      <label>Blue (0-255) <input id="blue" type="number" min="0" max="255" value="0"></label>
      <button id="show-led" type="button">Show colour</button>
    </div>
  </section>

  <section>
    <h2>Buzzer</h2>
    <div class="row">
      <label>Volume (0 = mute)
        <select id="volume">
          <option value="0">0</option><option value="1">1</option>
          <option value="2" selected>2</option><option value="3">3</option>
        </select>
      </label>
      <button id="apply-volume" type="button">Apply volume</button>
      <label>Sound
        <select id="sound">
          <option>APPROVAL</option><option>SUCCESS</option><option>WARNING</option>
        </select>
      </label>
      <button id="play-sound" type="button">Play sound</button>
    </div>
  </section>

  <section>
    <h2>Device replies</h2>
    <pre id="replies" aria-live="polite"></pre>
  </section>

  <script>
    const $ = (id) => document.getElementById(id);
    let panelStatus = { connected: false, testMode: false };

    async function request(path, body) {
      const options = body === undefined ? {} : {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body)
      };
      const response = await fetch(path, options);
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || `HTTP ${response.status}`);
      $("error").textContent = "";
      return result;
    }

    async function action(path, body) {
      try {
        await request(path, body);
        await refreshStatus();
      } catch (error) {
        $("error").textContent = error.message;
      }
    }

    async function refreshPorts() {
      try {
        const result = await request("/api/ports");
        const select = $("port");
        const previous = select.value;
        select.replaceChildren();
        for (const item of result.ports) {
          const option = document.createElement("option");
          option.value = item.device;
          option.textContent = `${item.device} - ${item.description || "USB serial"}`;
          select.append(option);
        }
        if (result.ports.some((item) => item.device === previous)) select.value = previous;
        if (result.ports.length === 0) select.add(new Option("No serial devices found", ""));
      } catch (error) {
        $("error").textContent = error.message;
      }
    }

    async function refreshStatus() {
      try {
        panelStatus = await request("/api/status");
        const connection = panelStatus.connected
          ? `Connected: ${panelStatus.port} - ${panelStatus.firmware || "firmware version unknown"}`
          : "Not connected";
        $("status").textContent = `${connection}${panelStatus.testMode ? " - TEST MODE" : ""}`;
        if (panelStatus.error) $("error").textContent = panelStatus.error;
        $("replies").textContent = panelStatus.messages.join("\\n");
        for (const id of ["connect", "disconnect", "test-start", "test-stop"]) {
          $(id).disabled =
            (id === "connect" && panelStatus.connected) ||
            (id === "disconnect" && !panelStatus.connected) ||
            (id === "test-start" && (!panelStatus.connected || panelStatus.testMode)) ||
            (id === "test-stop" && !panelStatus.testMode);
        }
        for (const element of document.querySelectorAll(
          "#states button, #speed, #frequency, #kick, #decay, #apply-motor, " +
          "#motor-start, #motor-stop, #red, #green, #blue, #show-led, " +
          "#volume, #apply-volume, #sound, #play-sound"
        )) {
          element.disabled = !panelStatus.testMode;
        }
        if (panelStatus.connected) {
          await request("/api/heartbeat", {});
        }
      } catch (error) {
        $("error").textContent = error.message;
      }
    }

    async function applyMotor() {
      await action("/api/test/speed", { value: Number($("speed").value) });
      if ($("error").textContent) return;
      await action("/api/test/frequency", { value: Number($("frequency").value) });
      if ($("error").textContent) return;
      await action("/api/test/kick", { value: Number($("kick").value) });
      if ($("error").textContent) return;
      await action("/api/test/decay", { value: $("decay").value });
    }

    $("refresh-ports").addEventListener("click", refreshPorts);
    $("connect").addEventListener("click", () =>
      action("/api/connect", { port: $("port").value })
    );
    $("disconnect").addEventListener("click", () => action("/api/disconnect", {}));
    $("test-start").addEventListener("click", () => action("/api/test/start", {}));
    $("test-stop").addEventListener("click", () => action("/api/test/stop", {}));
    for (const state of ["IDLE", "WORKING", "THINKING", "APPROVAL", "COMPLETED", "ERROR"]) {
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = state;
      button.addEventListener("click", () => action("/api/test/state", { value: state }));
      $("states").append(button);
    }
    $("apply-motor").addEventListener("click", applyMotor);
    $("motor-start").addEventListener("click", async () => {
      await applyMotor();
      if (!$("error").textContent) await action("/api/test/motor", { value: "START" });
    });
    $("motor-stop").addEventListener("click", () =>
      action("/api/test/motor", { value: "STOP" })
    );
    $("show-led").addEventListener("click", () =>
      action("/api/test/led", {
        r: Number($("red").value),
        g: Number($("green").value),
        b: Number($("blue").value)
      })
    );
    $("apply-volume").addEventListener("click", () =>
      action("/api/test/volume", { value: Number($("volume").value) })
    );
    $("play-sound").addEventListener("click", async () => {
      await action("/api/test/volume", { value: Number($("volume").value) });
      if (!$("error").textContent) {
        await action("/api/test/beep", { value: $("sound").value });
      }
    });
    window.addEventListener("pagehide", () => {
      navigator.sendBeacon("/api/close", new Blob(["{}"], { type: "application/json" }));
    });

    refreshPorts();
    refreshStatus();
    window.setInterval(refreshStatus, 2000);
  </script>
</body>
</html>
"""


class PanelError(Exception):
    """An expected panel, device, or serial-protocol error."""


class PanelController:
    def __init__(
        self,
        serial_factory: Any = serial.Serial,
        ports_provider: Any = list_ports.comports,
    ) -> None:
        self._serial_factory = serial_factory
        self._ports_provider = ports_provider
        self._serial: Any | None = None
        self._port: str | None = None
        self._firmware = ""
        self._test_mode = False
        self._last_reply = ""
        self._last_error = ""
        self._messages: deque[str] = deque(maxlen=30)
        self._lock = threading.Lock()
        self._last_ui_heartbeat = 0.0
        self._watchdog_stop = threading.Event()
        self._watchdog_thread: threading.Thread | None = None

    def ports(self) -> list[dict[str, str]]:
        return [
            {"device": port.device, "description": port.description or ""}
            for port in self._ports_provider()
        ]

    def status(self) -> dict[str, Any]:
        with self._lock:
            return {
                "connected": self._serial is not None,
                "port": self._port or "",
                "firmware": self._firmware,
                "testMode": self._test_mode,
                "lastReply": self._last_reply,
                "error": self._last_error,
                "messages": list(self._messages),
            }

    def connect(self, port: str) -> dict[str, str]:
        if not isinstance(port, str) or not port:
            raise ValueError("Select a serial device.")
        if port not in {item["device"] for item in self.ports()}:
            raise ValueError("The selected serial device is not available.")
        with self._lock:
            if self._serial is not None:
                raise PanelError("A serial device is already connected.")
            try:
                self._serial = self._serial_factory(
                    port, baudrate=BAUDRATE, timeout=0.1, write_timeout=1, exclusive=True
                )
                self._port = port
                version_error = ""
                for _ in range(3):
                    self._firmware = ""
                    try:
                        self._transact_locked("VERSION?")
                    except PanelError as exc:
                        version_error = str(exc)
                        if self._serial is None:
                            break
                        continue
                    if self._firmware:
                        break
                if not self._firmware:
                    message = version_error or "The device did not report its firmware version."
                    raise PanelError(message)
                self._last_ui_heartbeat = time.monotonic()
            except (serial.SerialException, OSError, PanelError) as exc:
                self._last_error = str(exc)
                self._close_locked()
                raise PanelError(str(exc)) from exc
            self._last_error = ""
            self._append_locked(f"Connected to {port}")
            return {"port": port, "firmware": self._firmware}

    def disconnect(self) -> None:
        with self._lock:
            if self._serial is None:
                return
            if self._test_mode:
                try:
                    self._transact_locked("TEST:STOP")
                except PanelError as exc:
                    self._last_error = str(exc)
            self._test_mode = False
            self._close_locked()
            self._append_locked("Serial connection closed")

    def start_test(self) -> str:
        with self._lock:
            self._require_connection_locked()
            self._transact_locked("TEST:START")
            self._test_mode = True
            self._last_ui_heartbeat = time.monotonic()
            self._last_error = ""
            return self._last_reply

    def stop_test(self) -> str:
        with self._lock:
            self._require_connection_locked()
            self._transact_locked("TEST:STOP")
            self._test_mode = False
            self._last_error = ""
            return self._last_reply

    def test_command(self, command: str) -> str:
        with self._lock:
            self._require_connection_locked()
            if not self._test_mode:
                raise PanelError("Enter test mode before using peripheral controls.")
            self._transact_locked(command)
            self._last_error = ""
            return self._last_reply

    def heartbeat(self) -> None:
        with self._lock:
            if self._serial is None:
                return
            self._last_ui_heartbeat = time.monotonic()
            self._transact_locked("PING")
            self._last_error = ""

    def dispatch(self, path: str, data: dict[str, Any]) -> dict[str, str]:
        if path == "/api/connect":
            result = self.connect(data.get("port"))
            return {"reply": f"Connected: {result['firmware']}"}
        if path == "/api/disconnect":
            self.disconnect()
            return {"reply": "Disconnected"}
        if path == "/api/test/start":
            return {"reply": self.start_test()}
        if path == "/api/test/stop":
            return {"reply": self.stop_test()}
        if path == "/api/heartbeat":
            self.heartbeat()
            return {"reply": self._last_reply}
        if path == "/api/close":
            self.disconnect()
            return {"reply": "Panel session closed"}

        value = data.get("value")
        command = ""
        if path == "/api/test/state":
            command = f"TEST:STATE:{_choice(value, STATES, 'state')}"
        elif path == "/api/test/motor":
            command = f"TEST:MOTOR:{_choice(value, ('START', 'STOP'), 'motor action')}"
        elif path == "/api/test/speed":
            command = f"TEST:SPEED:{_integer(value, 0, 255, 'speed')}"
        elif path == "/api/test/frequency":
            command = f"TEST:FREQ:{_integer(value, 1000, 30000, 'frequency')}"
        elif path == "/api/test/kick":
            command = f"TEST:KICK:{_integer(value, 0, 100, 'kick duration')}"
        elif path == "/api/test/decay":
            command = f"TEST:DECAY:{_choice(value, ('FAST', 'SLOW'), 'decay mode')}"
        elif path == "/api/test/volume":
            command = f"TEST:VOLUME:{_integer(value, 0, 3, 'volume')}"
        elif path == "/api/test/led":
            command = "TEST:LED:" + ",".join(
                str(_integer(data.get(channel), 0, 255, channel)) for channel in ("r", "g", "b")
            )
        elif path == "/api/test/beep":
            command = f"TEST:BEEP:{_choice(value, SOUNDS, 'sound')}"
        else:
            raise ValueError("Unknown panel action.")
        return {"reply": self.test_command(command)}

    def start_watchdog(self) -> None:
        if self._watchdog_thread is not None:
            return
        self._watchdog_thread = threading.Thread(target=self._watchdog_loop, daemon=True)
        self._watchdog_thread.start()

    def close(self) -> None:
        self._watchdog_stop.set()
        self.disconnect()

    def _watchdog_loop(self) -> None:
        while not self._watchdog_stop.wait(1):
            self.watchdog_once()

    def watchdog_once(self, now: float | None = None) -> None:
        with self._lock:
            current = time.monotonic() if now is None else now
            if self._serial is None or current - self._last_ui_heartbeat <= HEARTBEAT_TIMEOUT_S:
                return
            self._append_locked("Browser panel heartbeat expired")
            if self._test_mode:
                try:
                    self._transact_locked("TEST:STOP")
                except PanelError as exc:
                    self._last_error = str(exc)
            self._test_mode = False
            self._close_locked()

    def _require_connection_locked(self) -> None:
        if self._serial is None:
            raise PanelError("Connect to the USB serial device first.")

    def _transact_locked(self, command: str) -> None:
        self._require_connection_locked()
        assert self._serial is not None
        try:
            self._serial.write(f"{command}\n".encode("ascii"))
            deadline = time.monotonic() + SERIAL_TIMEOUT_S
            while time.monotonic() < deadline:
                raw = self._serial.readline()
                if not raw:
                    continue
                line = raw.decode("ascii", errors="replace").strip()
                self._append_locked(line)
                if line.startswith("VERSION "):
                    self._firmware = line.removeprefix("VERSION ")
                if line == "OK":
                    self._last_reply = line
                    return
                if line.startswith("ERR "):
                    self._last_reply = line
                    self._last_error = line
                    raise PanelError(line)
        except (serial.SerialException, OSError) as exc:
            self._last_error = str(exc)
            self._close_locked()
            raise PanelError(f"Serial connection lost: {exc}") from exc
        self._last_error = f"No reply to {command}."
        self._close_locked()
        raise PanelError(self._last_error)

    def _append_locked(self, message: str) -> None:
        self._messages.append(message)

    def _close_locked(self) -> None:
        device = self._serial
        self._serial = None
        self._port = None
        self._test_mode = False
        if device is not None:
            try:
                device.close()
            except (serial.SerialException, OSError) as exc:
                self._last_error = str(exc)


def _integer(value: Any, minimum: int, maximum: int, name: str) -> int:
    if type(value) is not int or not minimum <= value <= maximum:
        raise ValueError(f"{name.capitalize()} must be an integer from {minimum} to {maximum}.")
    return value


def _choice(value: Any, choices: tuple[str, ...], name: str) -> str:
    if not isinstance(value, str) or value not in choices:
        raise ValueError(f"Choose a valid {name}.")
    return value


def _make_handler(controller: PanelController) -> type[BaseHTTPRequestHandler]:
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self) -> None:
            path = urlsplit(self.path).path
            if path == "/":
                self._send(200, HTML_PAGE.encode(), "text/html; charset=utf-8")
            elif path == "/api/status":
                self._send_json(200, controller.status())
            elif path == "/api/ports":
                self._send_json(200, {"ports": controller.ports()})
            else:
                self._send_json(404, {"error": "Not found."})

        def do_POST(self) -> None:
            if not self._same_origin():
                self._send_json(403, {"error": "Cross-origin requests are not allowed."})
                return
            try:
                length = int(self.headers.get("Content-Length", "0"))
                if not 0 <= length <= MAX_REQUEST_BYTES:
                    raise ValueError("Request is too large.")
                raw = self.rfile.read(length) if length else b"{}"
                data = json.loads(raw)
                if not isinstance(data, dict):
                    raise ValueError("Expected a JSON object.")
                result = controller.dispatch(urlsplit(self.path).path, data)
            except ValueError as exc:
                self._send_json(400, {"error": str(exc)})
                return
            except PanelError as exc:
                self._send_json(409, {"error": str(exc)})
                return
            self._send_json(200, result)

        def _same_origin(self) -> bool:
            origin = self.headers.get("Origin", "")
            parsed = urlsplit(origin)
            request_host = self.headers.get("Host", "")
            host_name = urlsplit(f"//{request_host}").hostname
            return (
                parsed.scheme == "http"
                and parsed.netloc == request_host
                and host_name in ("localhost", "127.0.0.1")
            )

        def _send_json(self, status: int, value: dict[str, Any]) -> None:
            self._send(status, json.dumps(value).encode(), "application/json; charset=utf-8")

        def _send(self, status: int, body: bytes, content_type: str) -> None:
            self.send_response(status)
            self.send_header("Content-Type", content_type)
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            self.send_header("X-Content-Type-Options", "nosniff")
            self.send_header("X-Frame-Options", "DENY")
            self.send_header(
                "Content-Security-Policy",
                "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'",
            )
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, format_string: str, *args: Any) -> None:
            logging.info("%s - %s", self.address_string(), format_string % args)

    return Handler


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--port", type=int, default=PORT, help="local browser port")
    args = parser.parse_args(argv)
    if not 1 <= args.port <= 65535:
        parser.error("--port must be between 1 and 65535")

    controller = PanelController()
    server = ThreadingHTTPServer(("127.0.0.1", args.port), _make_handler(controller))
    controller.start_watchdog()
    print(f"Cody hardware test panel: http://127.0.0.1:{args.port}/")
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        controller.close()
        server.server_close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

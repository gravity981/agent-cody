"""The serial connection to the device (BRG-03, BRG-05, BRG-06, BRG-07).

`DeviceLink.step(now)` does all the work without blocking: discover, open,
handshake, resend settings and state, heartbeat, read replies, reconnect.
The bridge's loop calls it a few times per second; tests drive it with a
fake port and fake time.
"""

from __future__ import annotations

import contextlib
import logging
from collections.abc import Callable, Iterable
from dataclasses import dataclass
from typing import Protocol

from .sessions import State

log = logging.getLogger(__name__)

ESPRESSIF_VID = 0x303A
HEARTBEAT_S = 5.0  # BRG-05: at least every 10 s
HANDSHAKE_TIMEOUT_S = 2.0
RETRY_S = 1.0  # BRG-06: found within 10 s


class Port(Protocol):
    def write(self, data: bytes) -> object: ...
    def read_all(self) -> bytes: ...
    def close(self) -> None: ...


@dataclass(frozen=True)
class PortInfo:
    device: str
    vid: int | None


Lister = Callable[[], Iterable[PortInfo]]
Opener = Callable[[str], Port]


def pyserial_lister() -> list[PortInfo]:
    from serial.tools import list_ports

    return [PortInfo(p.device, p.vid) for p in list_ports.comports()]


def pyserial_opener(device: str) -> Port:
    import serial

    # dsrdtr/rtscts off, DTR/RTS not toggled: opening must not reset the ESP32
    port = serial.Serial()
    port.port = device
    port.baudrate = 115200
    port.timeout = 0
    port.write_timeout = 1
    port.dtr = False
    port.rts = False
    port.open()
    return port


class DeviceLink:
    def __init__(
        self,
        lister: Lister,
        opener: Opener,
        speed: int,
        volume: int,
        serial_port: str | None = None,
    ) -> None:
        self._lister = lister
        self._opener = opener
        self._configured = serial_port
        self.speed = speed
        self.volume = volume
        self.state = State.IDLE
        self._port: Port | None = None
        self.device: str | None = None
        self.firmware: str | None = None  # "<version> <commit>" once accepted
        self._buffer = b""
        self._handshake_since: float | None = None
        self._next_try = 0.0
        self._last_sent = 0.0
        self._rejected: set[str] = set()  # answered no VERSION?: not a Cody

    @property
    def connected(self) -> bool:
        return self._port is not None and self.firmware is not None

    # --- what the bridge wants on the device -------------------------------------

    def set_state(self, state: State, now: float) -> None:
        """Sends STATE only when it changes (BRG-03)."""
        if state == self.state:
            return
        self.state = state
        if self.connected:
            self._send(f"STATE:{state.value}", now)

    # --- the loop ------------------------------------------------------------

    def step(self, now: float) -> None:
        if self._port is None:
            if now >= self._next_try:
                self._next_try = now + RETRY_S
                self._open(now)
            return
        try:
            self._read(now)
            if self._port is None:
                return
            if self.firmware is None:
                assert self._handshake_since is not None
                if now - self._handshake_since >= HANDSHAKE_TIMEOUT_S:
                    log.info("%s didn't answer VERSION?; not a Cody", self.device)
                    if self.device is not None:
                        self._rejected.add(self.device)
                    self._close(now)
            elif now - self._last_sent >= HEARTBEAT_S:
                self._send("PING", now)
        except OSError as exc:  # unplugged, reset (serial.SerialException is an OSError)
            log.info("lost %s: %s", self.device, exc)
            self._close(now)

    def _candidates(self) -> list[str]:
        if self._configured:
            return [self._configured]
        found = sorted(p.device for p in self._lister() if p.vid == ESPRESSIF_VID)
        self._rejected &= set(found)  # a device that went away may come back as a Cody
        return [d for d in found if d not in self._rejected]

    def _open(self, now: float) -> None:
        for device in self._candidates():
            try:
                self._port = self._opener(device)
            except OSError as exc:
                log.debug("can't open %s: %s", device, exc)
                continue
            self.device = device
            self.firmware = None
            self._buffer = b""
            self._handshake_since = now
            try:
                self._send("VERSION?", now)
            except OSError:
                self._close(now)
                continue
            return

    def _on_connected(self, now: float) -> None:
        """Settings and the current state after every connect (BRG-06, BRG-07)."""
        log.info("connected to %s (firmware %s)", self.device, self.firmware)
        self._send(f"SPEED:{self.speed}", now)
        self._send(f"VOLUME:{self.volume}", now)
        self._send(f"STATE:{self.state.value}", now)

    def _read(self, now: float) -> None:
        assert self._port is not None
        self._buffer += self._port.read_all() or b""
        while b"\n" in self._buffer:
            raw, self._buffer = self._buffer.split(b"\n", 1)
            line = raw.decode("ascii", errors="replace").strip()
            log.debug("<- %s", line)
            if line.startswith("VERSION ") and self.firmware is None:
                self.firmware = line[len("VERSION ") :]
                self._on_connected(now)
            elif line.startswith("READY") and self.firmware is not None:
                # the device restarted without the port disappearing
                log.info("%s restarted", self.device)
                self._on_connected(now)
            elif line.startswith("ERR"):
                log.warning("device: %s", line)

    def _send(self, command: str, now: float) -> None:
        assert self._port is not None
        self._port.write(f"{command}\n".encode("ascii"))
        log.debug("-> %s", command)
        self._last_sent = now

    def _close(self, now: float) -> None:
        if self._port is not None:
            with contextlib.suppress(OSError):
                self._port.close()
        self._port = None
        self.firmware = None
        self.device = None
        self._next_try = now + RETRY_S

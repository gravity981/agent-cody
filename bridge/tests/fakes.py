"""A fake device that answers like the firmware (spec/architecture.md 4.2)."""

from __future__ import annotations

from cody_bridge.device import ESPRESSIF_VID, DeviceLink, PortInfo


class FakePort:
    def __init__(self, cody: bool = True, version: str = "1.2.3 abc") -> None:
        self.cody = cody
        self.version = version
        self.commands: list[str] = []
        self.unplugged = False
        self.closed = False
        self._out = b""

    def write(self, data: bytes) -> int:
        if self.unplugged:
            raise OSError("device disconnected")
        for line in data.decode().splitlines():
            self.commands.append(line)
            if not self.cody:
                continue
            if line == "VERSION?":
                self._out += f"VERSION {self.version}\n".encode()
            self._out += b"OK\n"
        return len(data)

    def read_all(self) -> bytes:
        if self.unplugged:
            raise OSError("device disconnected")
        out, self._out = self._out, b""
        return out

    def close(self) -> None:
        self.closed = True

    def restart(self) -> None:
        """The device rebooted without the port disappearing."""
        self._out += b"READY 1.2.3\n"


class FakeSystem:
    """Serial ports that can be plugged and unplugged."""

    def __init__(self) -> None:
        self.ports: dict[str, FakePort] = {}
        self.vids: dict[str, int | None] = {}
        self.opened: list[str] = []

    def plug(self, device: str, port: FakePort, vid: int | None = ESPRESSIF_VID) -> FakePort:
        self.ports[device] = port
        self.vids[device] = vid
        return port

    def unplug(self, device: str) -> None:
        self.ports[device].unplugged = True
        del self.ports[device]
        del self.vids[device]

    def lister(self) -> list[PortInfo]:
        return [PortInfo(d, self.vids[d]) for d in self.ports]

    def opener(self, device: str) -> FakePort:
        if device not in self.ports:
            raise OSError(f"no such device {device}")
        self.opened.append(device)
        return self.ports[device]

    def link(self, speed: int = 128, volume: int = 2, serial_port: str | None = None) -> DeviceLink:
        return DeviceLink(self.lister, self.opener, speed, volume, serial_port)


def run(link: DeviceLink, start: float, end: float, dt: float = 0.1) -> float:
    """Steps the link from start to end; returns end."""
    t = start
    while t <= end + 1e-9:
        link.step(t)
        t += dt
    return end

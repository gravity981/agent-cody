"""Bridge configuration (BRG-07, spec/architecture.md 4.4).

A TOML file; a missing file or key means the default.
"""

from __future__ import annotations

import ipaddress
import tomllib
from dataclasses import dataclass, field
from pathlib import Path

DEFAULT_LISTEN = ["127.0.0.1", "172.17.0.1"]
# Docker's default address pool for bridge networks; LAN ranges are refused
DOCKER_NETWORKS = ipaddress.ip_network("172.16.0.0/12")


class ConfigError(ValueError):
    pass


@dataclass(frozen=True)
class Config:
    serial_port: str | None = None
    speed: int = 128
    volume: int = 2
    session_timeout_min: float = 10
    listen: list[str] = field(default_factory=lambda: list(DEFAULT_LISTEN))
    port: int = 8765
    log_level: str = "info"


def _int(data: dict, key: str, default: int, low: int, high: int) -> int:
    value = data.get(key, default)
    if not isinstance(value, int) or isinstance(value, bool) or not low <= value <= high:
        raise ConfigError(f"{key} must be an integer from {low} to {high}")
    return value


def _listen(data: dict) -> list[str]:
    value = data.get("listen", DEFAULT_LISTEN)
    if not isinstance(value, list) or not value:
        raise ConfigError("listen must be a non-empty list of IP addresses")
    for item in value:
        try:
            address = ipaddress.ip_address(item)
        except TypeError, ValueError:
            raise ConfigError(f"listen: '{item}' is not an IP address") from None
        # never the LAN (BRG-08): only loopback or a Docker bridge network
        if not (address.is_loopback or address in DOCKER_NETWORKS):
            raise ConfigError(f"listen: '{item}' is not a local address")
    return list(value)


def parse(data: dict) -> Config:
    known = {"serial_port", "speed", "volume", "session_timeout_min", "listen", "log_level"}
    unknown = set(data) - known
    if unknown:
        raise ConfigError(f"unknown keys: {', '.join(sorted(unknown))}")
    port = data.get("serial_port")
    if port is not None and (not isinstance(port, str) or not port):
        raise ConfigError("serial_port must be a device path")
    timeout = data.get("session_timeout_min", 10)
    if not isinstance(timeout, int | float) or isinstance(timeout, bool) or timeout <= 0:
        raise ConfigError("session_timeout_min must be a positive number")
    log_level = data.get("log_level", "info")
    if log_level not in ("info", "debug"):
        raise ConfigError('log_level must be "info" or "debug"')
    return Config(
        log_level=log_level,
        serial_port=port,
        speed=_int(data, "speed", 128, 0, 255),
        volume=_int(data, "volume", 2, 0, 3),
        session_timeout_min=timeout,
        listen=_listen(data),
    )


def load(path: Path) -> Config:
    """Reads the config file; defaults if it doesn't exist."""
    if not path.exists():
        return Config()
    try:
        data = tomllib.loads(path.read_text(encoding="utf-8"))
    except tomllib.TOMLDecodeError as exc:
        raise ConfigError(f"{path}: {exc}") from None
    return parse(data)

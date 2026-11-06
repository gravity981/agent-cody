"""Bridge configuration (BRG-07) and listen addresses (BRG-08)."""

from __future__ import annotations

from pathlib import Path

import pytest
from cody_bridge.config import Config, ConfigError, load, parse


def test_req_BRG_07_missing_file_means_defaults(tmp_path: Path) -> None:
    assert load(tmp_path / "bridge.toml") == Config()
    assert Config().listen == ["127.0.0.1", "172.17.0.1"]
    assert (Config().speed, Config().volume, Config().session_timeout_min) == (128, 2, 10)


def test_req_BRG_07_settings_are_read_from_the_file(tmp_path: Path) -> None:
    path = tmp_path / "bridge.toml"
    path.write_text(
        'serial_port = "/dev/ttyACM3"\nspeed = 90\nvolume = 0\n'
        'session_timeout_min = 2.5\nlisten = ["127.0.0.1"]\n'
    )
    config = load(path)
    assert config.serial_port == "/dev/ttyACM3"
    assert (config.speed, config.volume, config.session_timeout_min) == (90, 0, 2.5)
    assert config.listen == ["127.0.0.1"]


@pytest.mark.parametrize(
    "data,message",
    [
        ({"speed": 256}, "speed"),
        ({"speed": "fast"}, "speed"),
        ({"volume": 4}, "volume"),
        ({"session_timeout_min": 0}, "session_timeout_min"),
        ({"serial_port": ""}, "serial_port"),
        ({"colour": "red"}, "unknown keys"),
    ],
)
def test_req_BRG_07_invalid_settings_are_refused(data: dict, message: str) -> None:
    with pytest.raises(ConfigError, match=message):
        parse(data)


def test_req_BRG_07_broken_toml_is_refused(tmp_path: Path) -> None:
    path = tmp_path / "bridge.toml"
    path.write_text("speed = \n")
    with pytest.raises(ConfigError):
        load(path)


@pytest.mark.parametrize(
    "address", ["0.0.0.0", "::", "192.168.1.20", "10.0.0.5", "8.8.8.8", "localhost"]
)
def test_req_BRG_08_lan_and_wildcard_addresses_are_refused(address: str) -> None:
    with pytest.raises(ConfigError, match="listen"):
        parse({"listen": [address]})


@pytest.mark.parametrize("address", ["127.0.0.1", "::1", "172.17.0.1", "172.18.0.1"])
def test_req_BRG_08_loopback_and_docker_addresses_are_accepted(address: str) -> None:
    assert parse({"listen": [address]}).listen == [address]

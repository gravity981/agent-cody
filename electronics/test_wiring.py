"""Wiring diagram (ELE-07): matches the firmware's pins and is up to date."""

from __future__ import annotations

import importlib.util
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
_spec = importlib.util.spec_from_file_location("wiring", Path(__file__).with_name("wiring.py"))
wiring = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(wiring)

# firmware/src/pins.h constant -> net in the diagram
FIRMWARE_PINS = {
    "kMotorIn1": "MOTOR_IN1",
    "kMotorIn2": "MOTOR_IN2",
    "kLed": "LED_DIN",
    "kBuzzer": "BUZZER",
}


def _firmware_pins() -> dict[str, int]:
    text = (ROOT / "firmware/src/pins.h").read_text(encoding="utf-8")
    return {m[1]: int(m[2]) for m in re.finditer(r"constexpr int (\w+) = (\d+);", text)}


def test_req_ELE_07_gpios_match_the_firmware() -> None:
    pins = _firmware_pins()
    for const, net in FIRMWARE_PINS.items():
        assert wiring.GPIO[net] == pins[const], net
        assert ("ESP32", f"GPIO{pins[const]}") in wiring.NETS[net], net


def test_req_ELE_07_every_pin_is_on_exactly_one_net() -> None:
    seen = [pin for pins in wiring.NETS.values() for pin in pins]
    assert len(seen) == len(set(seen))
    # a net with a single pin would be a dangling wire
    for net, pins in wiring.NETS.items():
        assert len(pins) >= 2, net


def test_req_ELE_07_diagram_names_every_pin_and_net() -> None:
    svg = wiring.SVG.read_text(encoding="utf-8")
    for net, pins in wiring.NETS.items():
        assert f">{net}<" in svg, net
        for _module, pin in pins:
            assert f">{pin}<" in svg, pin


def test_req_ELE_07_committed_svg_is_up_to_date() -> None:
    assert wiring.main(["--check"]) == 0, "run: uv run python electronics/wiring.py"

"""Wiring diagram of the status light (ELE-07; spec/architecture.md 4.3).

Draws the modules as blocks with named pins, the signal wires with their net
names, and the power nets as labelled stubs, and writes wiring.svg next to this
file. The GPIO numbers must match firmware/src/pins.h (checked by the tests).

  uv run python electronics/wiring.py           write wiring.svg
  uv run python electronics/wiring.py --check   fail if wiring.svg is outdated
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import schemdraw
import schemdraw.elements as elm
from schemdraw.elements import IcPin

SVG = Path(__file__).with_name("wiring.svg")

# GPIOs of the ESP32-C3 SuperMini, as in firmware/src/pins.h
GPIO = {"MOTOR_IN1": 3, "MOTOR_IN2": 4, "LED_DIN": 5, "BUZZER": 6}

# Every connection: net -> pins as (module, pin). The drawing below follows it,
# and the tests check it against the firmware and the parts.
NETS: dict[str, list[tuple[str, str]]] = {
    "5V": [("ESP32", "5V"), ("DRV8833", "VM"), ("WS2812", "5V")],
    "3V3": [("ESP32", "3V3"), ("DRV8833", "nSLEEP")],
    "GND": [("ESP32", "GND"), ("DRV8833", "GND"), ("WS2812", "GND"), ("KY-006", "-")],
    "MOTOR_IN1": [("ESP32", "GPIO3"), ("DRV8833", "AIN1")],
    "MOTOR_IN2": [("ESP32", "GPIO4"), ("DRV8833", "AIN2")],
    "MOTOR_A": [("DRV8833", "AOUT1"), ("MOTOR", "M1")],
    "MOTOR_B": [("DRV8833", "AOUT2"), ("MOTOR", "M2")],
    "LED_DIN": [("ESP32", "GPIO5"), ("WS2812", "DIN")],
    "BUZZER": [("ESP32", "GPIO6"), ("KY-006", "S")],
}

POWER = ("5V", "3V3", "GND")
STUB = 0.75  # length of a power stub
NET_COLOR = "#1f5fa8"  # net names, so they stand out from pin names


def _pins(left: list[str], right: list[str] = ()) -> list[IcPin]:
    # schemdraw lists left and right pins from bottom to top; we write them top-down
    return [IcPin(name=p, side="left", anchorname=p) for p in reversed(left)] + [
        IcPin(name=p, side="right", anchorname=p) for p in reversed(right)
    ]


def _net_of(module: str, pin: str) -> str:
    return next(net for net, pins in NETS.items() if (module, pin) in pins)


def _power_stub(d: schemdraw.Drawing, module: str, ic: elm.Ic, pin: str, side: str) -> None:
    net = _net_of(module, pin)
    if net not in POWER:
        return
    d.add(
        (elm.Line().left() if side == "left" else elm.Line().right())
        .at(getattr(ic, pin))
        .length(STUB)
        .label(
            net,
            loc="end",
            halign="right" if side == "left" else "left",
            fontsize=11,
            color=NET_COLOR,
        )
    )


def draw() -> schemdraw.Drawing:
    schemdraw.use("svg")
    d = schemdraw.Drawing(show=False)
    d.config(fontsize=12, unit=2, lw=1.2, font="sans-serif", bgcolor="white", margin=0.3)

    esp_left = ["5V", "GND", "3V3"]
    esp_right = ["GPIO3", "GPIO4", "GPIO5", "GPIO6"]
    esp = d.add(
        elm.Ic(pins=_pins(esp_left, esp_right), size=(3, 6), pinspacing=1.2, edgepadding=0.9)
        .at((0, 0))
        .anchor("center")
        .label("ESP32-C3\nSuperMini", loc="top", fontsize=13)
    )

    drv_left = ["AIN1", "AIN2", "nSLEEP", "VM", "GND"]
    drv = d.add(
        elm.Ic(pins=_pins(drv_left, ["AOUT1", "AOUT2"]), size=(2.6, 5), pinspacing=0.9)
        .at((9, 6.5))
        .anchor("center")
        .label("DRV8833", loc="top", fontsize=13)
    )
    motor = d.add(
        elm.Ic(pins=_pins(["M1", "M2"]), size=(2.2, 2.2), pinspacing=0.9)
        .at((14.5, 6.5))
        .anchor("center")
        .label("N20\nmotor", loc="top", fontsize=13)
    )
    led = d.add(
        elm.Ic(pins=_pins(["DIN", "5V", "GND"]), size=(2.6, 2.6), pinspacing=0.8)
        .at((9, 0))
        .anchor("center")
        .label("WS2812", loc="top", fontsize=13)
    )
    buz = d.add(
        elm.Ic(pins=_pins(["S", "(n.c.)", "-"]), size=(2.6, 2.6), pinspacing=0.8)
        .at((9, -5))
        .anchor("center")
        .label("KY-006\n(piezo)", loc="top", fontsize=13)
    )

    # Signal wires: right from the ESP32, then up or down to the module pin
    signals = [
        ("GPIO3", drv.AIN1, 4.4),
        ("GPIO4", drv.AIN2, 5.0),
        ("GPIO5", led.DIN, 5.6),
        ("GPIO6", buz.S, 4.4),
    ]
    for pin, target, x in signals:
        start = getattr(esp, pin)
        net = _net_of("ESP32", pin)
        d.add(
            elm.Line()
            .at(start)
            .to((x, start[1]))
            .label(net, loc="top", fontsize=10, color=NET_COLOR)
        )
        d.add(elm.Line().at((x, start[1])).to((x, target[1])))
        d.add(elm.Line().at((x, target[1])).to(target))

    # Motor outputs
    for out, m in (("AOUT1", "M1"), ("AOUT2", "M2")):
        a, b = getattr(drv, out), getattr(motor, m)
        d.add(
            elm.Line()
            .at(a)
            .to(b)
            .label(_net_of("DRV8833", out), loc="top", fontsize=10, color=NET_COLOR)
        )

    # Power nets as labelled stubs: pins with the same label are connected
    for pin in esp_left:
        _power_stub(d, "ESP32", esp, pin, "left")
    for pin in drv_left:
        _power_stub(d, "DRV8833", drv, pin, "left")
    for pin in ("5V", "GND"):
        _power_stub(d, "WS2812", led, pin, "left")
    _power_stub(d, "KY-006", buz, "-", "left")

    d.add(
        elm.Label()
        .at((-2.5, -7.4))
        .label(
            "Pins with the same net name are connected. 5V comes from USB through the "
            "SuperMini's 5V pin.\nDRV8833 board labels vary: IN1/IN2 = AIN1/AIN2, "
            "OUT1/OUT2 = AOUT1/AOUT2, VCC = VM, EEP or SLP = nSLEEP.\n"
            "Coil buzzer instead of a piezo: S and - to BOUT1/BOUT2, GPIO6 to BIN1, "
            "GPIO7 to BIN2 (architecture.md 4.3).",
            halign="left",
            valign="top",
            fontsize=10,
        )
    )
    return d


def render() -> str:
    data = draw().get_imagedata("svg")
    return data.decode() if isinstance(data, bytes) else data


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--check", action="store_true", help="fail if wiring.svg is outdated")
    args = parser.parse_args(argv)
    svg = render()
    if args.check:
        if not SVG.exists() or SVG.read_text(encoding="utf-8") != svg:
            print(f"{SVG} is outdated: run uv run python electronics/wiring.py", file=sys.stderr)
            return 1
        return 0
    SVG.write_text(svg, encoding="utf-8")
    print(f"wrote {SVG}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

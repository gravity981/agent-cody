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
GPIO = {
    "MOTOR_IN1": 3,
    "MOTOR_IN2": 4,
    "LED_DIN": 5,
    "BUZZER_IN1": 6,
    "BUZZER_IN2": 7,
}

# Every connection: net -> pins as (module, pin). The drawing below follows it,
# and the tests check it against the firmware and the parts.
NETS: dict[str, list[tuple[str, str]]] = {
    "5V": [("ESP32", "5V"), ("DRV8833", "VM"), ("DRV8833", "STBY"), ("WS2812", "5V")],
    "GND": [("ESP32", "GND"), ("DRV8833", "GND"), ("WS2812", "GND")],
    "MOTOR_IN1": [("ESP32", "GPIO3"), ("DRV8833", "AIN1")],
    "MOTOR_IN2": [("ESP32", "GPIO4"), ("DRV8833", "AIN2")],
    "MOTOR_A": [("DRV8833", "AOUT1"), ("MOTOR", "M1")],
    "MOTOR_B": [("DRV8833", "AOUT2"), ("MOTOR", "M2")],
    "LED_DIN": [("ESP32", "GPIO5"), ("WS2812", "DIN")],
    "BUZZER_IN1": [("ESP32", "GPIO6"), ("DRV8833", "BIN1")],
    "BUZZER_IN2": [("ESP32", "GPIO7"), ("DRV8833", "BIN2")],
    "BUZZER_A": [("DRV8833", "BOUT1"), ("KY-006", "S")],
    "BUZZER_B": [("DRV8833", "BOUT2"), ("KY-006", "-")],
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

    esp_left = ["5V", "GND"]
    esp_right = ["GPIO3", "GPIO4", "GPIO5", "GPIO6", "GPIO7"]
    esp = d.add(
        elm.Ic(pins=_pins(esp_left, esp_right), size=(3, 6), pinspacing=1.2, edgepadding=0.9)
        .at((0, 0))
        .anchor("center")
        .label("ESP32-C3\nSuperMini", loc="top", fontsize=13)
    )

    drv_left = ["AIN1", "AIN2", "BIN1", "BIN2", "STBY", "VM", "GND"]
    drv = d.add(
        elm.Ic(
            pins=_pins(drv_left, ["BOUT2", "BOUT1", "AOUT2", "AOUT1"]),
            size=(3.4, 6.8),
            pinspacing=0.9,
        )
        .at((9, 6.5))
        .anchor("center")
        .label("DRV8833", loc="top", fontsize=13)
    )
    motor = d.add(
        elm.Ic(pins=_pins(["M1", "M2"]), size=(2.2, 2.2), pinspacing=0.9)
        .at((14.5, 7.5))
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
        .at((14.5, -5))
        .anchor("center")
        .label("KY-006\n(coil)", loc="top", fontsize=13)
    )

    # Signal wires: right from the ESP32, then up or down to the module pin
    signals = [
        ("GPIO3", drv.AIN1, 4.4),
        ("GPIO4", drv.AIN2, 5.0),
        ("GPIO5", led.DIN, 5.6),
        ("GPIO6", drv.BIN1, 6.2),
        ("GPIO7", drv.BIN2, 6.8),
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

    # Driver outputs
    for out, module, pin in (
        ("AOUT1", motor, "M1"),
        ("AOUT2", motor, "M2"),
        ("BOUT1", buz, "S"),
        ("BOUT2", buz, "-"),
    ):
        a, b = getattr(drv, out), getattr(module, pin)
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
    d.add(
        elm.Label()
        .at((-2.5, -7.4))
        .label(
            "Pins with the same net name are connected. 5V comes from USB through the "
            "SuperMini's 5V pin.\nDRV8833 board labels vary: AO1/AO2 = AOUT1/AOUT2, "
            "BO1/BO2 = BOUT1/BOUT2, STBY/SLP = nSLEEP, VCC = VM.\n"
            "KY-006 coil buzzer: GPIO6/7 to BIN1/BIN2; S and - to BOUT1/BOUT2.",
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

# Cody parts list

**Stage 2 draft.** This list covers the purchased bench electronics; printed parts, fasteners and consumables will be added when the mechanism and figure are designed in stages 4 and 5. DOC-01 is complete only after that final list is delivered in step 5d.

Prices are what was paid on 2026-11-06, including the price of a whole pack where parts only come in packs. The budget for one Cody is CHF 25 for the purchased parts without filament and tools (ELE-06).

## Purchased parts

| Picture | Part | Qty per Cody | Variant to order | Source | Price paid (CHF) | Per Cody (CHF) |
|---|---|---|---|---|---|---|
| ![ESP32-C3 SuperMini](images/parts/esp32-c3.jpeg) | ESP32-C3 SuperMini development board (controller, TD-01) | 1 | "C3 Super mini" | TENSTAR Store: "TENSTAR ROBOT ESP32-C3 ESP32-S3 ESP32-H2 ESP32-C6 Development …" | 2.06 each (2 bought: 1 spare) | 2.06 |
| ![DRV8833 board](images/parts/drv8833.jpeg) | DRV8833 dual H-bridge motor driver board (ELE-03) | 1 | "DRV8833" | Realpoy Electronic Components Wholesale Store: "TB6612 DRV8833 Dual Motor Driver 1A TB6612FNG for Arduino …" | 0.85 each (2 bought: 1 spare) | 0.85 |
| ![N20 geared motor](images/parts/n20-500rpm.jpeg) | N20 geared DC motor, 6 V (GA12-N20) | 1 | "500RPM, Single motor, 6V" | Bobo01 Store: "GA12-N20 DC 3V 6V 12V Micro Metal Gear Motor Gear Wheel Gear Mot…" | 2.66 | 2.66 |
| ![WS2812 module](images/parts/ws2812.jpeg) | WS2812 single RGB LED module, 5 V (status LED) | 1 | "5pcs" | HHKFYD Module Store: "1-10pcs Electronic building block one RBG 1Bit low power long life WS28…" | 1.42 per pack of 5 | 1.42 |
| ![KY-006 passive buzzer](images/parts/ky006.jpeg) | KY-006 passive coil buzzer module | 1 | "Passive Buzzer, 5PCS" | ZC Technology Store: "Passive Speaker Buzzer Module for Arduino Boards KY-006" | 1.21 per pack of 5 | 1.21 |

**Subtotal for one Cody:** CHF 8.20 (whole packs counted; CHF 6.09 counting only the pieces used). Spares are not counted.

**Still missing:** USB-C data cable; connectors and wires; printed parts; screws; steel wire for the shoulder axle; optionally a level shifter board (74AHCT125) for the LED data line.

## Printed parts

Follows in stage 5 (file, material, quantity).

## Fasteners and consumables

Follows in stages 4 and 5.

## Notes

- **Buzzer:** "passive" says nothing about how it makes sound. Measure between `S` and `-` before wiring: open circuit = piezo, which is not compatible with the coil wiring and firmware; about 16–42 Ω = coil, as used in the wiring diagram. The pictured module looks like a coil type.
- **Sources:** seller names and listing titles are recorded above; product URLs were not retained.
- Pictures: photos of the actual parts.

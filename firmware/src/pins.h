// Default pins of the ESP32-C3 SuperMini (spec/architecture.md 4.3),
// confirmed with the wiring diagram in step 2d (ELE-07).
#pragma once

namespace cody::pins {

constexpr int kMotorIn1 = 3;   // DRV8833 AIN1 (PWM)
constexpr int kMotorIn2 = 4;   // DRV8833 AIN2 (held low: one direction)
constexpr int kLed = 5;        // WS2812 data
constexpr int kBuzzer = 6;     // piezo directly, or DRV8833 BIN1 for a coil buzzer
constexpr int kBuzzerLow = 7;  // DRV8833 BIN2, held low (only used with a coil buzzer)

}  // namespace cody::pins

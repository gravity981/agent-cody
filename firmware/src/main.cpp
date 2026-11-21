// Cody firmware (spec/architecture.md 4.2, 4.3): reads command lines from USB
// serial, lets cody::Device decide, and drives LED, buzzer and motor.
#include <Arduino.h>

#include "device.h"
#include "pins.h"
#include "version.h"

namespace {

constexpr uint8_t kMotorBits = 8;    // duty 0-255
constexpr uint8_t kBuzzerBits = 10;  // duty 0-1023 (cody::kDutyMax)

cody::Device device(cody::version(), cody::commit());
cody::LineReader reader;
cody::Outputs applied;
bool first = true;

void apply(const cody::Outputs& out) {
  if (first || !(out.led == applied.led)) {
    rgbLedWrite(cody::pins::kLed, out.led.r, out.led.g, out.led.b);
  }
  if (first || out.toneHz != applied.toneHz || out.toneDuty != applied.toneDuty) {
    if (out.toneHz > 0 && out.toneDuty > 0) {
      ledcChangeFrequency(cody::pins::kBuzzer, out.toneHz, kBuzzerBits);
      ledcWrite(cody::pins::kBuzzer, out.toneDuty);
    } else {
      ledcWrite(cody::pins::kBuzzer, 0);
    }
  }
  if (first || out.motorFrequencyHz != applied.motorFrequencyHz) {
    ledcChangeFrequency(cody::pins::kMotorIn1, out.motorFrequencyHz, kMotorBits);
    ledcChangeFrequency(cody::pins::kMotorIn2, out.motorFrequencyHz, kMotorBits);
  }
  if (first || out.motorDuty != applied.motorDuty || out.motorDecay != applied.motorDecay) {
    if (!first && out.motorDecay != applied.motorDecay) {
      ledcWrite(cody::pins::kMotorIn1, 0);
      ledcWrite(cody::pins::kMotorIn2, 0);
    }
    if (out.motorDuty == 0) {
      ledcWrite(cody::pins::kMotorIn1, 0);
      ledcWrite(cody::pins::kMotorIn2, 0);
    } else if (out.motorDecay == cody::MotorDecay::Fast) {
      ledcWrite(cody::pins::kMotorIn1, out.motorDuty);
      ledcWrite(cody::pins::kMotorIn2, 0);
    } else {
      ledcWrite(cody::pins::kMotorIn1, 255);
      ledcWrite(cody::pins::kMotorIn2, out.motorDuty);
    }
  }
  applied = out;
  first = false;
}

}  // namespace

void setup() {
  pinMode(cody::pins::kBuzzerLow, OUTPUT);
  digitalWrite(cody::pins::kBuzzerLow, LOW);
  ledcAttach(cody::pins::kMotorIn1, cody::kDefaultMotorFrequencyHz, kMotorBits);
  ledcAttach(cody::pins::kMotorIn2, cody::kDefaultMotorFrequencyHz, kMotorBits);
  ledcAttach(cody::pins::kBuzzer, 2000, kBuzzerBits);
  apply(device.update(millis()));

  Serial.begin(115200);
  // USB CDC: give a host up to 3 s to open the port, so the line isn't lost
  const unsigned long start = millis();
  while (!Serial && millis() - start < 3000) {
    apply(device.update(millis()));
    delay(10);
  }
  Serial.printf("READY %s\n", cody::version());
  // keep the build info marker in the image (the linker drops unreferenced data)
  asm volatile("" : : "r"(cody::kBuildInfo));
}

void loop() {
  while (Serial.available() > 0) {
    const char c = static_cast<char>(Serial.read());
    switch (reader.feed(c)) {
      case cody::LineReader::Result::Line:
        Serial.print(device.handleLine(reader.line(), millis()).c_str());
        break;
      case cody::LineReader::Result::TooLong:
        Serial.print(cody::Device::lineTooLong().c_str());
        break;
      case cody::LineReader::Result::None:
        break;
    }
  }
  apply(device.update(millis()));
  delay(5);
}

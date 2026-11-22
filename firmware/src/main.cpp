// Cody firmware (spec/architecture.md 4.2, 4.3): reads command lines from USB
// serial, lets cody::Device decide, and drives LED, buzzer and motor.
#include <Arduino.h>
#include <Preferences.h>

#include <cstring>
#include <string>

#include "device.h"
#include "pins.h"
#include "version.h"

namespace {

constexpr uint8_t kMotorBits = 8;    // duty 0-255
constexpr uint8_t kBuzzerBits = 10;  // duty 0-1023 (cody::kDutyMax)

constexpr char kNvsNamespace[] = "cody";
constexpr char kTuningKey[] = "tuning";
constexpr uint32_t kTuningMagic = 0x434F4459;
constexpr uint8_t kTuningRecordVersion = 1;

struct TuningRecord {
  uint32_t magic;
  uint8_t version;
  uint8_t speed;
  uint8_t decay;
  uint8_t kickStartMs;
  uint32_t frequencyHz;
  char timestamp[40];
};

class PreferencesTuningStore final : public cody::TuningStore {
 public:
  cody::TuningLoadResult load(cody::MotorTuning& tuning, std::string& timestamp) override {
    Preferences preferences;
    // A read-only open fails when the namespace is absent on a fresh device.
    if (!preferences.begin(kNvsNamespace, false)) return cody::TuningLoadResult::Error;
    if (!preferences.isKey(kTuningKey)) {
      preferences.end();
      return cody::TuningLoadResult::Missing;
    }

    TuningRecord record{};
    const size_t size = preferences.getBytes(kTuningKey, &record, sizeof(record));
    preferences.end();
    if (size != sizeof(record) || record.magic != kTuningMagic ||
        record.version != kTuningRecordVersion || record.decay > 1 || record.frequencyHz < 1000 ||
        record.frequencyHz > 30000 || record.kickStartMs > 100 ||
        std::memchr(record.timestamp, '\0', sizeof(record.timestamp)) == nullptr ||
        !cody::isUtcTimestamp(record.timestamp)) {
      return cody::TuningLoadResult::Error;
    }

    tuning.speed = record.speed;
    tuning.frequencyHz = record.frequencyHz;
    tuning.kickStartMs = record.kickStartMs;
    tuning.decay = record.decay == 0 ? cody::MotorDecay::Fast : cody::MotorDecay::Slow;
    timestamp = record.timestamp;
    return cody::TuningLoadResult::Loaded;
  }

  bool save(const cody::MotorTuning& tuning, const std::string& timestamp) override {
    if (timestamp.size() >= sizeof(TuningRecord::timestamp) ||
        !cody::isUtcTimestamp(timestamp.c_str())) {
      return false;
    }

    TuningRecord record{};
    record.magic = kTuningMagic;
    record.version = kTuningRecordVersion;
    record.speed = tuning.speed;
    record.decay = tuning.decay == cody::MotorDecay::Fast ? 0 : 1;
    record.kickStartMs = tuning.kickStartMs;
    record.frequencyHz = tuning.frequencyHz;
    std::memcpy(record.timestamp, timestamp.c_str(), timestamp.size() + 1);

    Preferences preferences;
    if (!preferences.begin(kNvsNamespace, false)) return false;
    const bool saved = preferences.putBytes(kTuningKey, &record, sizeof(record)) == sizeof(record);
    preferences.end();
    return saved;
  }
};

cody::Device device(cody::version(), cody::commit());
PreferencesTuningStore tuningStore;
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
  device.loadTuning(tuningStore);
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

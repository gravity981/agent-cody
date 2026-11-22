// The device logic (spec/architecture.md 4.2, 4.3): handles command lines and
// computes what LED, buzzer and motor do at a given time. No Arduino calls;
// main.cpp feeds it serial lines and millis() and applies the outputs.
#pragma once

#include <cstdint>
#include <string>

#include "protocol.h"

namespace cody {

struct Rgb {
  uint8_t r = 0, g = 0, b = 0;
  bool operator==(const Rgb& o) const { return r == o.r && g == o.g && b == o.b; }
};

struct Outputs {
  Rgb led;
  uint16_t toneHz = 0;    // 0 = silent
  uint16_t toneDuty = 0;  // of kDutyMax; 0 = silent (muted or no tone)
  uint8_t motorDuty = 0;  // 0-255; 0 = stopped
  uint32_t motorFrequencyHz = 10000;
  MotorDecay motorDecay = MotorDecay::Fast;
};

constexpr uint16_t kDutyMax = 1023;          // buzzer duty resolution (10 bit)
constexpr uint32_t kTimeoutMs = 30000;       // FW-06
constexpr uint32_t kDefaultKickStartMs = 1;  // brief motor kick
constexpr uint32_t kDefaultMotorFrequencyHz = 10000;
constexpr uint8_t kDefaultMotorSpeed = 224;
constexpr uint8_t kLedCap = 127;  // 50 % of full scale (ELE-04)
constexpr uint8_t kDefaultBrightness = 100;
constexpr uint8_t kDefaultVolume = 50;

struct MotorTuning {
  uint8_t speed = kDefaultMotorSpeed;
  uint32_t frequencyHz = kDefaultMotorFrequencyHz;
  uint8_t kickStartMs = kDefaultKickStartMs;
  MotorDecay decay = MotorDecay::Fast;
};

enum class TuningLoadResult : uint8_t { Missing, Loaded, Error };

class TuningStore {
 public:
  virtual ~TuningStore() = default;
  virtual TuningLoadResult load(MotorTuning& tuning, std::string& timestamp) = 0;
  virtual bool save(const MotorTuning& tuning, const std::string& timestamp) = 0;
};

class Device {
 public:
  Device(const char* version, const char* commit) : version_(version), commit_(commit) {}
  void loadTuning(TuningStore& store);

  // Handles one received line; returns the response (one or more lines, each
  // ending with '\n'). Errors never change the state (FW-10).
  std::string handleLine(const char* line, uint32_t nowMs);
  // Response for a line that was too long (dropped).
  static std::string lineTooLong() { return "ERR line too long\n"; }

  // What the outputs should be at nowMs (effects run without commands, FW-03).
  Outputs update(uint32_t nowMs);

  State state() const { return state_; }
  bool connected(uint32_t nowMs) const;
  bool testMode() const { return testMode_; }
  int speed() const { return tuning_.speed; }
  int volume() const { return volume_; }
  int brightness() const { return brightness_; }
  const MotorTuning& tuning() const { return tuning_; }
  const std::string& tuningTimestamp() const { return tuningTimestamp_; }

 private:
  bool validTuning(const MotorTuning& tuning) const;
  std::string tuningReply() const;
  void enterState(State next, uint32_t nowMs);
  void clearTestOverrides();
  void startSound(Sound sound, uint32_t nowMs);
  Rgb stateLed(uint32_t nowMs) const;
  void soundAt(uint32_t nowMs, Outputs& out);

  const char* version_;
  const char* commit_;
  State state_ = State::Idle;
  uint32_t stateSinceMs_ = 0;
  bool everConnected_ = false;
  uint32_t lastCommandMs_ = 0;
  bool wasConnected_ = false;
  bool testMode_ = false;
  bool testMotorOverride_ = false;
  bool testMotorEnabled_ = false;
  MotorTuning tuning_;
  TuningStore* tuningStore_ = nullptr;
  bool tuningStoreReady_ = false;
  std::string tuningTimestamp_;
  int volume_ = kDefaultVolume;
  int brightness_ = kDefaultBrightness;
  int testLedBrightness_ = kDefaultBrightness;
  bool testLedBrightnessOverride_ = false;
  bool ledOverride_ = false;
  Rgb ledOverrideColour_;
  Sound sound_ = Sound::None;
  uint32_t soundSinceMs_ = 0;
  uint32_t motorSinceMs_ = 0;
  bool motorRunning_ = false;
};

}  // namespace cody

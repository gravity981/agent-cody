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
};

constexpr uint16_t kDutyMax = 1023;                    // buzzer duty resolution (10 bit)
constexpr uint32_t kTimeoutMs = 30000;                 // FW-06
constexpr uint32_t kKickStartMs = 100;                 // motor kick start
constexpr uint8_t kLedCap = 127;                       // 50 % of full scale (ELE-04)
constexpr uint16_t kVolumeDuty[] = {0, 30, 120, 512};  // VOLUME 0-3

class Device {
 public:
  Device(const char* version, const char* commit) : version_(version), commit_(commit) {}

  // Handles one received line; returns the response (one or more lines, each
  // ending with '\n'). Errors never change the state (FW-10).
  std::string handleLine(const char* line, uint32_t nowMs);
  // Response for a line that was too long (dropped).
  static std::string lineTooLong() { return "ERR line too long\n"; }

  // What the outputs should be at nowMs (effects run without commands, FW-03).
  Outputs update(uint32_t nowMs);

  State state() const { return state_; }
  bool connected(uint32_t nowMs) const;
  int speed() const { return speed_; }
  int volume() const { return volume_; }

 private:
  void enterState(State next, uint32_t nowMs);
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
  int speed_ = 128;
  int volume_ = 2;
  bool ledOverride_ = false;
  Rgb ledOverrideColour_;
  Sound sound_ = Sound::None;
  uint32_t soundSinceMs_ = 0;
  uint32_t motorSinceMs_ = 0;
  bool motorRunning_ = false;
};

}  // namespace cody

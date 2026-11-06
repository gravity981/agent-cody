#include "device.h"

#include <cmath>

namespace cody {

namespace {

struct Note {
  uint16_t hz;  // 0 = pause
  uint16_t ms;
};

// Sounds (spec/architecture.md 4.3)
const Note kApproval[] = {{2700, 100}, {0, 100}, {2700, 100}};
const Note kSuccess[] = {{2000, 100}, {2500, 100}, {3000, 100}};
const Note kWarning[] = {{3000, 200}, {2000, 200}, {3000, 200}, {2000, 200}};

struct Tune {
  const Note* notes;
  int count;
};

Tune tuneOf(Sound sound) {
  switch (sound) {
    case Sound::Approval:
      return {kApproval, 3};
    case Sound::Success:
      return {kSuccess, 3};
    case Sound::Warning:
      return {kWarning, 4};
    default:
      return {nullptr, 0};
  }
}

Sound soundOf(State state) {
  switch (state) {
    case State::Approval:
      return Sound::Approval;
    case State::Completed:
      return Sound::Success;
    case State::Error:
      return Sound::Warning;
    default:
      return Sound::None;
  }
}

constexpr double kPi = 3.14159265358979323846;

// colour at percent of the capped brightness
Rgb scaled(uint8_t r, uint8_t g, uint8_t b, double percent) {
  const double f = percent / 100.0 * kLedCap / 255.0;
  auto c = [f](uint8_t v) { return static_cast<uint8_t>(std::lround(v * f)); };
  return {c(r), c(g), c(b)};
}

// 0..1..0 over one period (cosine), starting at the low point
double pulse(uint32_t ms, uint32_t periodMs) {
  const double phase = static_cast<double>(ms % periodMs) / periodMs;
  return (1.0 - std::cos(2.0 * kPi * phase)) / 2.0;
}

bool blinkOn(uint32_t ms, uint32_t periodMs) { return (ms % periodMs) < periodMs / 2; }

}  // namespace

bool Device::connected(uint32_t nowMs) const {
  return everConnected_ && (nowMs - lastCommandMs_) < kTimeoutMs;
}

std::string Device::handleLine(const char* line, uint32_t nowMs) {
  Command cmd;
  if (const char* error = parseCommand(line, cmd)) {
    return std::string("ERR ") + error + "\n";
  }
  if (!connected(nowMs)) ledOverride_ = false;  // a disconnect ends the LED test
  everConnected_ = true;
  lastCommandMs_ = nowMs;

  std::string response;
  switch (cmd.kind) {
    case Command::Kind::State:
      enterState(cmd.state, nowMs);
      break;
    case Command::Kind::Ping:
      break;
    case Command::Kind::Speed:
      speed_ = cmd.value;
      break;
    case Command::Kind::Volume:
      volume_ = cmd.value;
      break;
    case Command::Kind::VersionQuery:
      response = std::string("VERSION ") + version_ + " " + commit_ + "\n";
      break;
    case Command::Kind::Led:
      ledOverride_ = true;
      ledOverrideColour_ = {cmd.r, cmd.g, cmd.b};
      break;
    case Command::Kind::Beep:
      startSound(cmd.sound, nowMs);
      break;
  }
  return response + "OK\n";
}

void Device::enterState(State next, uint32_t nowMs) {
  ledOverride_ = false;
  if (next == state_) return;  // re-sent state: no new sound (FW-04)
  state_ = next;
  stateSinceMs_ = nowMs;
  startSound(soundOf(next), nowMs);
}

void Device::startSound(Sound sound, uint32_t nowMs) {
  sound_ = sound;
  soundSinceMs_ = nowMs;
}

Rgb Device::stateLed(uint32_t nowMs) const {
  const uint32_t t = nowMs - stateSinceMs_;
  switch (state_) {
    case State::Idle:
      return scaled(0, 0, 255, 15);
    case State::Working:
      return scaled(0, 0, 255, 20 + 80 * pulse(t, 2000));
    case State::Thinking:
      return scaled(255, 160, 0, 50);
    case State::Approval:
      return blinkOn(t, 1000) ? scaled(255, 80, 0, 100) : Rgb{};
    case State::Completed:
      return scaled(0, 255, 0, 50);
    case State::Error:
      return blinkOn(t, 500) ? scaled(255, 0, 0, 100) : Rgb{};
  }
  return {};
}

void Device::soundAt(uint32_t nowMs, Outputs& out) {
  const Tune tune = tuneOf(sound_);
  uint32_t t = nowMs - soundSinceMs_;
  for (int i = 0; i < tune.count; ++i) {
    if (t < tune.notes[i].ms) {
      out.toneHz = tune.notes[i].hz;
      out.toneDuty = tune.notes[i].hz ? kVolumeDuty[volume_] : 0;
      return;
    }
    t -= tune.notes[i].ms;
  }
  sound_ = Sound::None;  // finished: plays once
}

Outputs Device::update(uint32_t nowMs) {
  Outputs out;
  if (!connected(nowMs)) {
    // FW-06, FW-07: motor stopped, slow dim white pulse
    if (wasConnected_) ledOverride_ = false;
    wasConnected_ = false;
    motorRunning_ = false;
    sound_ = Sound::None;
    out.led = scaled(255, 255, 255, 10 * pulse(nowMs, 4000));
    return out;
  }
  wasConnected_ = true;

  out.led = ledOverride_
                ? scaled(ledOverrideColour_.r, ledOverrideColour_.g, ledOverrideColour_.b, 100)
                : stateLed(nowMs);
  soundAt(nowMs, out);

  const bool run = state_ == State::Working && speed_ > 0;
  if (run && !motorRunning_) motorSinceMs_ = nowMs;
  motorRunning_ = run;
  if (run) out.motorDuty = (nowMs - motorSinceMs_) < kKickStartMs ? 255 : speed_;
  return out;
}

}  // namespace cody

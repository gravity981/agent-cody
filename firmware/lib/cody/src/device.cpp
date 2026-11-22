#include "device.h"

#include <cmath>

namespace cody {

namespace {

constexpr uint16_t kMaximumVolumeDuty = 512;
constexpr double kBrightnessGamma = 2.2;
constexpr double kVolumeGamma = 2.5;

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

// Apply state brightness and the hardware cap before the user's gamma setting.
Rgb scaled(uint8_t r, uint8_t g, uint8_t b, double percent, int brightness) {
  const double f = percent / 100.0 * kLedCap / 255.0;
  const double gamma = std::pow(brightness / 100.0, kBrightnessGamma);
  auto c = [f, gamma](uint8_t v) { return static_cast<uint8_t>(std::lround(v * f * gamma)); };
  return {c(r), c(g), c(b)};
}

uint16_t volumeDuty(int volume) {
  const double level = volume / 100.0;
  return static_cast<uint16_t>(std::lround(kMaximumVolumeDuty * std::pow(level, kVolumeGamma)));
}

// 0..1..0 over one period (cosine), starting at the low point
double pulse(uint32_t ms, uint32_t periodMs) {
  const double phase = static_cast<double>(ms % periodMs) / periodMs;
  return (1.0 - std::cos(2.0 * kPi * phase)) / 2.0;
}

bool blinkOn(uint32_t ms, uint32_t periodMs) { return (ms % periodMs) < periodMs / 2; }

}  // namespace

void Device::loadTuning(TuningStore& store) {
  tuningStore_ = &store;
  MotorTuning loaded;
  std::string timestamp;
  switch (store.load(loaded, timestamp)) {
    case TuningLoadResult::Missing:
      tuning_ = MotorTuning{};
      tuningTimestamp_.clear();
      tuningStoreReady_ = true;
      return;
    case TuningLoadResult::Loaded:
      if (!validTuning(loaded) || !isUtcTimestamp(timestamp.c_str())) {
        tuningStoreReady_ = false;
        return;
      }
      tuning_ = loaded;
      tuningTimestamp_ = timestamp;
      tuningStoreReady_ = true;
      return;
    case TuningLoadResult::Error:
      tuningStoreReady_ = false;
      return;
  }
}

bool Device::validTuning(const MotorTuning& tuning) const {
  return tuning.frequencyHz >= 1000 && tuning.frequencyHz <= 30000 && tuning.kickStartMs <= 100 &&
         (tuning.decay == MotorDecay::Fast || tuning.decay == MotorDecay::Slow);
}

std::string Device::tuningReply() const {
  if (!tuningStoreReady_) return "ERR tuning storage unavailable\n";
  return "TUNING:" + std::to_string(tuning_.speed) + "," + std::to_string(tuning_.frequencyHz) +
         "," + std::to_string(tuning_.kickStartMs) + "," +
         (tuning_.decay == MotorDecay::Fast ? "FAST" : "SLOW") + "," +
         (tuningTimestamp_.empty() ? "UNSAVED" : tuningTimestamp_) + "\n";
}

bool Device::connected(uint32_t nowMs) const {
  return everConnected_ && (nowMs - lastCommandMs_) < kTimeoutMs;
}

std::string Device::handleLine(const char* line, uint32_t nowMs) {
  Command cmd;
  if (const char* error = parseCommand(line, cmd)) {
    return std::string("ERR ") + error + "\n";
  }

  const bool normalControl = cmd.kind == Command::Kind::State || cmd.kind == Command::Kind::Speed ||
                             cmd.kind == Command::Kind::Volume ||
                             cmd.kind == Command::Kind::Brightness ||
                             cmd.kind == Command::Kind::Led || cmd.kind == Command::Kind::Beep;
  const bool testControl =
      cmd.kind == Command::Kind::TestState || cmd.kind == Command::Kind::TestMotorStart ||
      cmd.kind == Command::Kind::TestMotorStop || cmd.kind == Command::Kind::TestSpeed ||
      cmd.kind == Command::Kind::TestFrequency || cmd.kind == Command::Kind::TestKick ||
      cmd.kind == Command::Kind::TestDecay || cmd.kind == Command::Kind::TestVolume ||
      cmd.kind == Command::Kind::TestTuningSave || cmd.kind == Command::Kind::TestTuningReset ||
      cmd.kind == Command::Kind::TestLed || cmd.kind == Command::Kind::TestBeep;
  if (testMode_ && normalControl) return "ERR test mode active\n";
  if (!testMode_ && testControl) return "ERR test mode inactive\n";

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
      tuning_.speed = static_cast<uint8_t>(cmd.value);
      break;
    case Command::Kind::Volume:
      volume_ = cmd.value;
      break;
    case Command::Kind::Brightness:
      brightness_ = cmd.value;
      break;
    case Command::Kind::VersionQuery:
      response = std::string("VERSION ") + version_ + " " + commit_ + "\n";
      break;
    case Command::Kind::TuningQuery:
      response = tuningReply();
      if (response.rfind("ERR ", 0) == 0) return response;
      break;
    case Command::Kind::Led:
      ledOverride_ = true;
      ledOverrideColour_ = {cmd.r, cmd.g, cmd.b};
      break;
    case Command::Kind::Beep:
      startSound(cmd.sound, nowMs);
      break;
    case Command::Kind::TestStart:
      if (!testMode_) {
        testMode_ = true;
        clearTestOverrides();
        enterState(State::Idle, nowMs);
        sound_ = Sound::None;
        motorRunning_ = false;
      }
      break;
    case Command::Kind::TestStop:
      testMode_ = false;
      clearTestOverrides();
      enterState(State::Idle, nowMs);
      sound_ = Sound::None;
      motorRunning_ = false;
      break;
    case Command::Kind::TestState:
      clearTestOverrides();
      enterState(cmd.state, nowMs);
      break;
    case Command::Kind::TestMotorStart:
      testMotorOverride_ = true;
      testMotorEnabled_ = true;
      break;
    case Command::Kind::TestMotorStop:
      testMotorOverride_ = true;
      testMotorEnabled_ = false;
      break;
    case Command::Kind::TestSpeed:
      tuning_.speed = static_cast<uint8_t>(cmd.value);
      break;
    case Command::Kind::TestFrequency:
      tuning_.frequencyHz = static_cast<uint32_t>(cmd.value);
      break;
    case Command::Kind::TestKick:
      tuning_.kickStartMs = static_cast<uint8_t>(cmd.value);
      break;
    case Command::Kind::TestDecay:
      tuning_.decay = cmd.decay;
      break;
    case Command::Kind::TestVolume:
      volume_ = cmd.value;
      break;
    case Command::Kind::TestTuningSave: {
      const std::string timestamp(cmd.timestamp);
      if (!tuningStore_ || !tuningStore_->save(tuning_, timestamp)) {
        tuningStoreReady_ = false;
        return "ERR tuning save failed\n";
      }
      tuningTimestamp_ = timestamp;
      tuningStoreReady_ = true;
      break;
    }
    case Command::Kind::TestTuningReset: {
      const MotorTuning defaults;
      const std::string timestamp(cmd.timestamp);
      if (!tuningStore_ || !tuningStore_->save(defaults, timestamp)) {
        tuningStoreReady_ = false;
        return "ERR tuning reset failed\n";
      }
      tuning_ = defaults;
      tuningTimestamp_ = timestamp;
      tuningStoreReady_ = true;
      break;
    }
    case Command::Kind::TestLed:
      ledOverride_ = true;
      ledOverrideColour_ = {cmd.r, cmd.g, cmd.b};
      break;
    case Command::Kind::TestBeep:
      startSound(cmd.sound, nowMs);
      break;
  }
  return response + "OK\n";
}

void Device::clearTestOverrides() {
  testMotorOverride_ = false;
  testMotorEnabled_ = false;
  ledOverride_ = false;
  sound_ = Sound::None;
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
      return scaled(0, 0, 255, 15, brightness_);
    case State::Working:
      return scaled(0, 0, 255, 20 + 80 * pulse(t, 2000), brightness_);
    case State::Thinking:
      return scaled(255, 160, 0, 50, brightness_);
    case State::Approval:
      return blinkOn(t, 1000) ? scaled(255, 60, 0, 100, brightness_) : Rgb{};
    case State::Completed:
      return scaled(0, 255, 0, 50, brightness_);
    case State::Error:
      return blinkOn(t, 500) ? scaled(255, 0, 0, 100, brightness_) : Rgb{};
  }
  return {};
}

void Device::soundAt(uint32_t nowMs, Outputs& out) {
  const Tune tune = tuneOf(sound_);
  uint32_t t = nowMs - soundSinceMs_;
  for (int i = 0; i < tune.count; ++i) {
    if (t < tune.notes[i].ms) {
      out.toneHz = tune.notes[i].hz;
      out.toneDuty = tune.notes[i].hz ? volumeDuty(volume_) : 0;
      return;
    }
    t -= tune.notes[i].ms;
  }
  sound_ = Sound::None;  // finished: plays once
}

Outputs Device::update(uint32_t nowMs) {
  Outputs out;
  out.motorFrequencyHz = tuning_.frequencyHz;
  out.motorDecay = tuning_.decay;
  if (!connected(nowMs)) {
    // FW-06, FW-07: motor stopped, slow dim white pulse
    if (wasConnected_ || testMode_) {
      clearTestOverrides();
      testMode_ = false;
    }
    wasConnected_ = false;
    motorRunning_ = false;
    sound_ = Sound::None;
    out.led = scaled(255, 255, 255, 10 * pulse(nowMs, 4000), brightness_);
    return out;
  }
  wasConnected_ = true;

  out.led = ledOverride_ ? scaled(ledOverrideColour_.r, ledOverrideColour_.g, ledOverrideColour_.b,
                                  100, brightness_)
                         : stateLed(nowMs);
  soundAt(nowMs, out);

  const bool manualMotor = testMode_ && testMotorOverride_;
  const bool run =
      (manualMotor ? testMotorEnabled_ : state_ == State::Working) && tuning_.speed > 0;
  if (run && !motorRunning_) motorSinceMs_ = nowMs;
  motorRunning_ = run;
  if (run) {
    out.motorDuty = (nowMs - motorSinceMs_) < tuning_.kickStartMs ? 255 : tuning_.speed;
  }
  return out;
}

}  // namespace cody

#include "protocol.h"

#include <cstdlib>
#include <cstring>

namespace cody {

namespace {

const char* const kStateNames[] = {"IDLE", "WORKING", "THINKING", "APPROVAL", "COMPLETED", "ERROR"};

// Parses a decimal integer in [min, max] that makes up all of text.
bool parseInt(const char* text, int min, int max, int& out) {
  if (*text == '\0') return false;
  int value = 0;
  for (const char* p = text; *p != '\0'; ++p) {
    if (*p < '0' || *p > '9') return false;
    const int digit = *p - '0';
    if (digit > max || value > (max - digit) / 10) return false;
    value = value * 10 + digit;
  }
  if (value < min) return false;
  out = value;
  return true;
}

bool isDigit(char c) { return c >= '0' && c <= '9'; }

// The argument after "NAME:" if line starts with it, else nullptr.
const char* argumentOf(const char* line, const char* name) {
  const size_t n = std::strlen(name);
  if (std::strncmp(line, name, n) == 0 && line[n] == ':') return line + n + 1;
  return nullptr;
}

}  // namespace

const char* stateName(State state) { return kStateNames[static_cast<int>(state)]; }

bool parseState(const char* text, State& out) {
  for (int i = 0; i < 6; ++i) {
    if (std::strcmp(text, kStateNames[i]) == 0) {
      out = static_cast<State>(i);
      return true;
    }
  }
  return false;
}

bool parseSound(const char* text, Sound& out) {
  if (std::strcmp(text, "APPROVAL") == 0) {
    out = Sound::Approval;
  } else if (std::strcmp(text, "SUCCESS") == 0) {
    out = Sound::Success;
  } else if (std::strcmp(text, "WARNING") == 0) {
    out = Sound::Warning;
  } else {
    return false;
  }
  return true;
}

namespace {

bool parseDecay(const char* text, MotorDecay& out) {
  if (std::strcmp(text, "FAST") == 0) {
    out = MotorDecay::Fast;
  } else if (std::strcmp(text, "SLOW") == 0) {
    out = MotorDecay::Slow;
  } else {
    return false;
  }
  return true;
}

bool copyTimestamp(const char* text, Command& out) {
  const size_t length = std::strlen(text);
  if (length >= sizeof(out.timestamp) || !isUtcTimestamp(text)) return false;
  std::memcpy(out.timestamp, text, length + 1);
  return true;
}

bool parseRgb(const char* arg, Command& out) {
  char parts[3][4] = {};
  int part = 0;
  size_t len = 0;
  for (const char* p = arg;; ++p) {
    if (*p == ',' || *p == '\0') {
      if (part > 2) return false;
      parts[part][len] = '\0';
      ++part;
      len = 0;
      if (*p == '\0') break;
    } else {
      if (part > 2 || len >= 3) return false;
      parts[part][len++] = *p;
    }
  }
  if (part != 3) return false;
  int rgb[3];
  for (int i = 0; i < 3; ++i) {
    if (!parseInt(parts[i], 0, 255, rgb[i])) return false;
  }
  out.r = static_cast<uint8_t>(rgb[0]);
  out.g = static_cast<uint8_t>(rgb[1]);
  out.b = static_cast<uint8_t>(rgb[2]);
  return true;
}

}  // namespace

bool isUtcTimestamp(const char* text) {
  const size_t length = std::strlen(text);
  if (length < 20 || length > 39 || text[4] != '-' || text[7] != '-' || text[10] != 'T' ||
      text[13] != ':' || text[16] != ':') {
    return false;
  }
  for (size_t i = 0; i < 19; ++i) {
    if (i != 4 && i != 7 && i != 10 && i != 13 && i != 16 && !isDigit(text[i])) return false;
  }
  size_t suffix = 19;
  if (text[suffix] == '.') {
    ++suffix;
    const size_t fractionStart = suffix;
    while (suffix < length && isDigit(text[suffix])) ++suffix;
    if (suffix == fractionStart) return false;
  }
  return suffix + 1 == length && text[suffix] == 'Z';
}

const char* parseCommand(const char* line, Command& out) {
  const char* arg = nullptr;
  if (std::strcmp(line, "PING") == 0) {
    out.kind = Command::Kind::Ping;
  } else if (std::strcmp(line, "VERSION?") == 0) {
    out.kind = Command::Kind::VersionQuery;
  } else if (std::strcmp(line, "TUNING?") == 0) {
    out.kind = Command::Kind::TuningQuery;
  } else if (std::strcmp(line, "TEST:START") == 0) {
    out.kind = Command::Kind::TestStart;
  } else if (std::strcmp(line, "TEST:STOP") == 0) {
    out.kind = Command::Kind::TestStop;
  } else if (std::strcmp(line, "TEST:MOTOR:START") == 0) {
    out.kind = Command::Kind::TestMotorStart;
  } else if (std::strcmp(line, "TEST:MOTOR:STOP") == 0) {
    out.kind = Command::Kind::TestMotorStop;
  } else if ((arg = argumentOf(line, "STATE")) != nullptr) {
    if (!parseState(arg, out.state)) return "unknown state";
    out.kind = Command::Kind::State;
  } else if ((arg = argumentOf(line, "SPEED")) != nullptr) {
    if (!parseInt(arg, 0, 255, out.value)) return "bad value";
    out.kind = Command::Kind::Speed;
  } else if ((arg = argumentOf(line, "VOLUME")) != nullptr) {
    if (!parseInt(arg, 0, 100, out.value)) return "bad value";
    out.kind = Command::Kind::Volume;
  } else if ((arg = argumentOf(line, "BRIGHTNESS")) != nullptr) {
    if (!parseInt(arg, 0, 100, out.value)) return "bad value";
    out.kind = Command::Kind::Brightness;
  } else if ((arg = argumentOf(line, "BEEP")) != nullptr) {
    if (!parseSound(arg, out.sound)) return "unknown sound";
    out.kind = Command::Kind::Beep;
  } else if ((arg = argumentOf(line, "LED")) != nullptr) {
    if (!parseRgb(arg, out)) return "bad value";
    out.kind = Command::Kind::Led;
  } else if ((arg = argumentOf(line, "TEST:STATE")) != nullptr) {
    if (!parseState(arg, out.state)) return "unknown state";
    out.kind = Command::Kind::TestState;
  } else if ((arg = argumentOf(line, "TEST:SPEED")) != nullptr) {
    if (!parseInt(arg, 0, 255, out.value)) return "bad value";
    out.kind = Command::Kind::TestSpeed;
  } else if ((arg = argumentOf(line, "TEST:FREQ")) != nullptr) {
    if (!parseInt(arg, 1000, 30000, out.value)) return "bad value";
    out.kind = Command::Kind::TestFrequency;
  } else if ((arg = argumentOf(line, "TEST:KICK")) != nullptr) {
    if (!parseInt(arg, 0, 100, out.value)) return "bad value";
    out.kind = Command::Kind::TestKick;
  } else if ((arg = argumentOf(line, "TEST:DECAY")) != nullptr) {
    if (!parseDecay(arg, out.decay)) return "bad decay mode";
    out.kind = Command::Kind::TestDecay;
  } else if ((arg = argumentOf(line, "TEST:VOLUME")) != nullptr) {
    if (!parseInt(arg, 0, 100, out.value)) return "bad value";
    out.kind = Command::Kind::TestVolume;
  } else if ((arg = argumentOf(line, "TEST:TUNING:SAVE")) != nullptr) {
    if (!copyTimestamp(arg, out)) return "bad timestamp";
    out.kind = Command::Kind::TestTuningSave;
  } else if ((arg = argumentOf(line, "TEST:TUNING:RESET")) != nullptr) {
    if (!copyTimestamp(arg, out)) return "bad timestamp";
    out.kind = Command::Kind::TestTuningReset;
  } else if ((arg = argumentOf(line, "TEST:LED")) != nullptr) {
    if (!parseRgb(arg, out)) return "bad value";
    out.kind = Command::Kind::TestLed;
  } else if ((arg = argumentOf(line, "TEST:BEEP")) != nullptr) {
    if (!parseSound(arg, out.sound)) return "unknown sound";
    out.kind = Command::Kind::TestBeep;
  } else {
    return "unknown command";
  }
  return nullptr;
}

LineReader::Result LineReader::feed(char c) {
  if (c == '\r') return Result::None;
  if (c == '\n') {
    const bool tooLong = overflow_;
    buffer_[length_] = '\0';
    length_ = 0;
    overflow_ = false;
    return tooLong ? Result::TooLong : Result::Line;
  }
  if (length_ < kMaxLine) {
    buffer_[length_++] = c;
  } else {
    overflow_ = true;
  }
  return Result::None;
}

}  // namespace cody

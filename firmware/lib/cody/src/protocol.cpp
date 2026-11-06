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
    value = value * 10 + (*p - '0');
    if (value > max) return false;
  }
  if (value < min) return false;
  out = value;
  return true;
}

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

const char* parseCommand(const char* line, Command& out) {
  const char* arg = nullptr;
  if (std::strcmp(line, "PING") == 0) {
    out.kind = Command::Kind::Ping;
  } else if (std::strcmp(line, "VERSION?") == 0) {
    out.kind = Command::Kind::VersionQuery;
  } else if ((arg = argumentOf(line, "STATE")) != nullptr) {
    if (!parseState(arg, out.state)) return "unknown state";
    out.kind = Command::Kind::State;
  } else if ((arg = argumentOf(line, "SPEED")) != nullptr) {
    if (!parseInt(arg, 0, 255, out.value)) return "bad value";
    out.kind = Command::Kind::Speed;
  } else if ((arg = argumentOf(line, "VOLUME")) != nullptr) {
    if (!parseInt(arg, 0, 3, out.value)) return "bad value";
    out.kind = Command::Kind::Volume;
  } else if ((arg = argumentOf(line, "BEEP")) != nullptr) {
    if (!parseSound(arg, out.sound)) return "unknown sound";
    out.kind = Command::Kind::Beep;
  } else if ((arg = argumentOf(line, "LED")) != nullptr) {
    char parts[3][4] = {};
    int part = 0;
    size_t len = 0;
    for (const char* p = arg;; ++p) {
      if (*p == ',' || *p == '\0') {
        if (part > 2) return "bad value";
        parts[part][len] = '\0';
        ++part;
        len = 0;
        if (*p == '\0') break;
      } else {
        if (part > 2 || len >= 3) return "bad value";
        parts[part][len++] = *p;
      }
    }
    if (part != 3) return "bad value";
    int rgb[3];
    for (int i = 0; i < 3; ++i) {
      if (!parseInt(parts[i], 0, 255, rgb[i])) return "bad value";
    }
    out.r = static_cast<uint8_t>(rgb[0]);
    out.g = static_cast<uint8_t>(rgb[1]);
    out.b = static_cast<uint8_t>(rgb[2]);
    out.kind = Command::Kind::Led;
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

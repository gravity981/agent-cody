// Serial protocol (spec/architecture.md 4.2): text lines from the bridge,
// parsed into commands. No Arduino calls, so it runs in the host tests.
#pragma once

#include <cstddef>
#include <cstdint>

namespace cody {

enum class State : uint8_t { Idle, Working, Thinking, Approval, Completed, Error };
enum class Sound : uint8_t { None, Approval, Success, Warning };

const char* stateName(State state);
bool parseState(const char* text, State& out);
bool parseSound(const char* text, Sound& out);

struct Command {
  enum class Kind : uint8_t { State, Ping, Speed, Volume, VersionQuery, Led, Beep };
  Kind kind = Kind::Ping;
  State state = State::Idle;
  int value = 0;  // SPEED 0-255, VOLUME 0-3
  uint8_t r = 0, g = 0, b = 0;
  Sound sound = Sound::None;
};

// Parses one line (without the line ending). Returns nullptr on success, or
// the reason for "ERR <reason>".
const char* parseCommand(const char* line, Command& out);

// Collects characters into lines: '\n' ends a line, '\r' is ignored, lines
// longer than kMaxLine are reported as too long and dropped.
class LineReader {
 public:
  static constexpr size_t kMaxLine = 64;
  enum class Result : uint8_t { None, Line, TooLong };

  Result feed(char c);
  const char* line() const { return buffer_; }

 private:
  char buffer_[kMaxLine + 1] = {};
  size_t length_ = 0;
  bool overflow_ = false;
};

}  // namespace cody

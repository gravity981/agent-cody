// Host unit tests of the device logic (PlatformIO native environment, Unity).
// Tag convention: spec/verification.md 3 (test_req_FW_nn_...).
#include <unity.h>

#include <string>

#include "device.h"
#include "protocol.h"

using cody::Device;
using cody::LineReader;
using cody::MotorDecay;
using cody::MotorTuning;
using cody::Outputs;
using cody::Rgb;
using cody::State;
using cody::TuningLoadResult;
using cody::TuningStore;

namespace {

class FakeTuningStore : public TuningStore {
 public:
  TuningLoadResult load(MotorTuning& out, std::string& outTimestamp) override {
    ++loadCount;
    if (loadResult == TuningLoadResult::Loaded) {
      out = storedTuning;
      outTimestamp = storedTimestamp;
    }
    return loadResult;
  }

  bool save(const MotorTuning& tuning, const std::string& timestamp) override {
    ++saveCount;
    if (!saveResult) return false;
    storedTuning = tuning;
    storedTimestamp = timestamp;
    loadResult = TuningLoadResult::Loaded;
    return true;
  }

  TuningLoadResult loadResult = TuningLoadResult::Missing;
  bool saveResult = true;
  MotorTuning storedTuning;
  std::string storedTimestamp;
  int loadCount = 0;
  int saveCount = 0;
};

std::string send(Device& d, const char* line, uint32_t t) { return d.handleLine(line, t); }

void assertLed(const Rgb& expected, const Rgb& actual) {
  TEST_ASSERT_EQUAL_UINT8(expected.r, actual.r);
  TEST_ASSERT_EQUAL_UINT8(expected.g, actual.g);
  TEST_ASSERT_EQUAL_UINT8(expected.b, actual.b);
}

std::string feedAll(LineReader& reader, const char* text, int* tooLong = nullptr) {
  std::string lines;
  for (const char* p = text; *p != '\0'; ++p) {
    const auto result = reader.feed(*p);
    if (result == LineReader::Result::Line) lines += std::string(reader.line()) + "|";
    if (result == LineReader::Result::TooLong && tooLong) ++*tooLong;
  }
  return lines;
}

}  // namespace

void setUp() {}
void tearDown() {}

// --- FW-02: newline-terminated text commands ---

void test_req_FW_02_lines_end_with_newline_and_ignore_cr() {
  LineReader reader;
  TEST_ASSERT_EQUAL_STRING("PING|STATE:IDLE|", feedAll(reader, "PING\nSTATE:IDLE\r\n").c_str());
  TEST_ASSERT_EQUAL_STRING("", feedAll(reader, "SPEED:1").c_str());  // no newline yet
  TEST_ASSERT_EQUAL_STRING("SPEED:12|", feedAll(reader, "2\n").c_str());
}

void test_req_FW_02_too_long_line_is_reported_and_dropped() {
  LineReader reader;
  std::string longLine(LineReader::kMaxLine + 1, 'A');
  longLine += "\nPING\n";
  int tooLong = 0;
  TEST_ASSERT_EQUAL_STRING("PING|", feedAll(reader, longLine.c_str(), &tooLong).c_str());
  TEST_ASSERT_EQUAL_INT(1, tooLong);
  TEST_ASSERT_EQUAL_STRING("ERR line too long\n", Device::lineTooLong().c_str());
}

// --- FW-03: STATE switches; effects run without further commands ---

void test_req_FW_03_state_command_switches_state() {
  Device d("1.2.3", "abc");
  const char* names[] = {"IDLE", "WORKING", "THINKING", "APPROVAL", "COMPLETED", "ERROR"};
  for (int i = 0; i < 6; ++i) {
    TEST_ASSERT_EQUAL_STRING("OK\n",
                             send(d, (std::string("STATE:") + names[i]).c_str(), 0).c_str());
    TEST_ASSERT_EQUAL_INT(i, static_cast<int>(d.state()));
  }
}

void test_req_FW_03_effects_run_without_commands() {
  Device d("1.2.3", "abc");
  send(d, "STATE:WORKING", 0);
  const Outputs low = d.update(0);
  const Outputs high = d.update(1000);
  TEST_ASSERT_TRUE(high.led.b > low.led.b);         // the pulse moves on its own
  TEST_ASSERT_TRUE(d.update(20000).motorDuty > 0);  // motor keeps running
}

// --- FW-05: COMPLETED and ERROR stay until the next state command ---

void test_req_FW_05_completed_and_error_stay_until_next_state() {
  const struct {
    const char* line;
    State state;
  } cases[] = {{"STATE:COMPLETED", State::Completed}, {"STATE:ERROR", State::Error}};
  for (const auto& c : cases) {
    Device d("1.2.3", "abc");
    send(d, c.line, 0);
    for (uint32_t t = 10000; t <= 600000; t += 10000) {  // 10 min of heartbeats
      send(d, "PING", t);
      d.update(t);
      TEST_ASSERT_TRUE(d.state() == c.state);
    }
    send(d, "STATE:IDLE", 610000);
    TEST_ASSERT_TRUE(d.state() == State::Idle);
  }
}

// --- FW-08: SPEED sets the motor speed used in WORKING ---

void test_req_FW_08_speed_sets_motor_duty_in_working() {
  Device d("1.2.3", "abc");
  send(d, "STATE:WORKING", 0);
  TEST_ASSERT_EQUAL_UINT8(255, d.update(0).motorDuty);  // brief kick
  TEST_ASSERT_EQUAL_UINT8(224, d.update(1).motorDuty);  // default
  send(d, "SPEED:90", 100);
  TEST_ASSERT_EQUAL_UINT8(90, d.update(100).motorDuty);
  send(d, "SPEED:200", 300);
  TEST_ASSERT_EQUAL_UINT8(200, d.update(300).motorDuty);
  send(d, "SPEED:0", 400);
  TEST_ASSERT_EQUAL_UINT8(0, d.update(400).motorDuty);
}

void test_req_FW_08_motor_stopped_outside_working() {
  Device d("1.2.3", "abc");
  send(d, "SPEED:200", 0);
  const char* states[] = {"STATE:IDLE", "STATE:THINKING", "STATE:APPROVAL", "STATE:COMPLETED",
                          "STATE:ERROR"};
  for (const char* line : states) {
    send(d, line, 0);
    TEST_ASSERT_EQUAL_UINT8(0, d.update(500).motorDuty);
  }
}

// --- FW-10: OK or an error line; errors don't change the state ---

void test_req_FW_10_every_command_is_answered_ok() {
  Device d("1.2.3", "abc");
  for (const char* line : {"PING", "STATE:THINKING", "SPEED:10", "VOLUME:50", "BRIGHTNESS:100",
                           "LED:1,2,3", "BEEP:SUCCESS"}) {
    TEST_ASSERT_EQUAL_STRING("OK\n", send(d, line, 0).c_str());
  }
}

void test_req_FW_10_errors_keep_the_state() {
  Device d("1.2.3", "abc");
  send(d, "STATE:THINKING", 0);
  send(d, "SPEED:77", 0);
  const char* bad[][2] = {
      {"HELLO", "ERR unknown command\n"},
      {"state:working", "ERR unknown command\n"},
      {"STATE:SLEEPING", "ERR unknown state\n"},
      {"STATE:", "ERR unknown state\n"},
      {"SPEED:256", "ERR bad value\n"},
      {"SPEED:-1", "ERR bad value\n"},
      {"SPEED:", "ERR bad value\n"},
      {"VOLUME:101", "ERR bad value\n"},
      {"LED:1,2", "ERR bad value\n"},
      {"LED:1,2,3,4", "ERR bad value\n"},
      {"LED:300,0,0", "ERR bad value\n"},
      {"BEEP:HONK", "ERR unknown sound\n"},
      {"", "ERR unknown command\n"},
  };
  for (const auto& pair : bad) {
    TEST_ASSERT_EQUAL_STRING(pair[1], send(d, pair[0], 0).c_str());
  }
  TEST_ASSERT_TRUE(d.state() == State::Thinking);
  TEST_ASSERT_EQUAL_INT(77, d.speed());
}

// --- FW-11: version and commit on request ---

void test_req_FW_11_reports_version_and_commit() {
  Device d("1.2.3", "0123abc");
  TEST_ASSERT_EQUAL_STRING("VERSION 1.2.3 0123abc\nOK\n", send(d, "VERSION?", 0).c_str());
}

// --- FW-12: LED and BEEP test commands ---

void test_req_FW_12_led_shows_colour_until_next_state() {
  Device d("1.2.3", "abc");
  send(d, "STATE:IDLE", 0);
  send(d, "LED:255,0,0", 0);
  assertLed({127, 0, 0}, d.update(100).led);  // capped at 50 % (ELE-04)
  assertLed({127, 0, 0}, d.update(5000).led);
  send(d, "STATE:THINKING", 6000);
  assertLed({64, 40, 0}, d.update(6000).led);
}

void test_req_FW_12_beep_plays_a_sound() {
  Device d("1.2.3", "abc");
  send(d, "STATE:IDLE", 0);
  send(d, "BEEP:SUCCESS", 1000);
  TEST_ASSERT_EQUAL_UINT16(2000, d.update(1050).toneHz);
  TEST_ASSERT_EQUAL_UINT16(3000, d.update(1250).toneHz);
  TEST_ASSERT_EQUAL_UINT16(0, d.update(1400).toneHz);
}

// --- TST: permanent, all-peripheral test mode ---

void test_req_TST_01_test_mode_owns_controls_until_stopped() {
  Device d("1.2.3", "abc");
  send(d, "STATE:WORKING", 0);
  TEST_ASSERT_TRUE(d.update(1).motorDuty > 0);

  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:START", 10).c_str());
  TEST_ASSERT_TRUE(d.testMode());
  TEST_ASSERT_TRUE(d.state() == State::Idle);
  TEST_ASSERT_EQUAL_UINT8(0, d.update(10).motorDuty);
  TEST_ASSERT_EQUAL_STRING("ERR test mode active\n", send(d, "STATE:WORKING", 20).c_str());
  TEST_ASSERT_TRUE(d.state() == State::Idle);

  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:STOP", 30).c_str());
  TEST_ASSERT_FALSE(d.testMode());
  TEST_ASSERT_TRUE(d.state() == State::Idle);
  TEST_ASSERT_EQUAL_STRING("ERR test mode inactive\n", send(d, "TEST:STATE:WORKING", 40).c_str());
}

void test_req_TST_02_state_presets_run_while_test_mode_stays_active() {
  Device d("1.2.3", "abc");
  send(d, "TEST:START", 0);
  send(d, "TEST:STATE:WORKING", 1);
  TEST_ASSERT_TRUE(d.testMode());
  TEST_ASSERT_TRUE(d.update(2).motorDuty > 0);
  send(d, "TEST:STATE:APPROVAL", 10);
  TEST_ASSERT_TRUE(d.testMode());
  TEST_ASSERT_EQUAL_UINT16(2700, d.update(20).toneHz);
  TEST_ASSERT_TRUE(d.state() == State::Approval);
}

void test_req_TST_03_motor_tunings_apply_to_test_and_normal_working() {
  Device d("1.2.3", "abc");
  send(d, "TEST:START", 0);
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:SPEED:100", 1).c_str());
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:FREQ:30000", 2).c_str());
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:KICK:5", 3).c_str());
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:DECAY:SLOW", 4).c_str());
  send(d, "TEST:MOTOR:START", 5);
  Outputs out = d.update(5);
  TEST_ASSERT_EQUAL_UINT8(255, out.motorDuty);
  TEST_ASSERT_EQUAL_UINT32(30000, out.motorFrequencyHz);
  TEST_ASSERT_TRUE(out.motorDecay == MotorDecay::Slow);
  TEST_ASSERT_EQUAL_UINT8(100, d.update(10).motorDuty);

  TEST_ASSERT_EQUAL_STRING("ERR bad value\n", send(d, "TEST:SPEED:256", 11).c_str());
  TEST_ASSERT_EQUAL_STRING("ERR bad value\n", send(d, "TEST:FREQ:30001", 12).c_str());
  TEST_ASSERT_EQUAL_STRING("ERR bad value\n", send(d, "TEST:FREQ:999", 13).c_str());
  TEST_ASSERT_EQUAL_STRING("ERR bad value\n", send(d, "TEST:FREQ:999999999999999999", 14).c_str());
  TEST_ASSERT_EQUAL_STRING("ERR bad value\n", send(d, "TEST:KICK:101", 15).c_str());
  TEST_ASSERT_EQUAL_STRING("ERR bad decay mode\n", send(d, "TEST:DECAY:MIDDLE", 16).c_str());

  send(d, "TEST:STOP", 20);
  send(d, "STATE:WORKING", 21);
  out = d.update(21);
  TEST_ASSERT_EQUAL_UINT8(255, out.motorDuty);
  TEST_ASSERT_EQUAL_UINT32(30000, out.motorFrequencyHz);
  TEST_ASSERT_TRUE(out.motorDecay == MotorDecay::Slow);
  TEST_ASSERT_EQUAL_UINT8(100, d.update(26).motorDuty);
}

void test_req_TST_03_motor_tuning_bounds_include_zero_kick_and_30_khz() {
  Device d("1.2.3", "abc");
  send(d, "TEST:START", 0);
  send(d, "TEST:SPEED:70", 1);
  send(d, "TEST:FREQ:1000", 2);
  send(d, "TEST:KICK:0", 3);
  send(d, "TEST:DECAY:FAST", 4);
  send(d, "TEST:MOTOR:START", 5);
  Outputs out = d.update(5);
  TEST_ASSERT_EQUAL_UINT8(70, out.motorDuty);
  TEST_ASSERT_EQUAL_UINT32(1000, out.motorFrequencyHz);
  TEST_ASSERT_TRUE(out.motorDecay == MotorDecay::Fast);

  send(d, "TEST:MOTOR:STOP", 10);
  TEST_ASSERT_EQUAL_UINT8(0, d.update(10).motorDuty);
  send(d, "TEST:SPEED:255", 11);
  TEST_ASSERT_EQUAL_INT(255, d.speed());
  send(d, "TEST:SPEED:70", 12);
  send(d, "TEST:FREQ:30000", 13);
  send(d, "TEST:KICK:100", 14);
  send(d, "TEST:DECAY:SLOW", 15);
  send(d, "TEST:MOTOR:START", 16);
  out = d.update(16);
  TEST_ASSERT_EQUAL_UINT8(255, out.motorDuty);
  TEST_ASSERT_EQUAL_UINT32(30000, out.motorFrequencyHz);
  TEST_ASSERT_TRUE(out.motorDecay == MotorDecay::Slow);
  TEST_ASSERT_EQUAL_UINT8(255, d.update(115).motorDuty);
  TEST_ASSERT_EQUAL_UINT8(70, d.update(116).motorDuty);
}

void test_req_TST_04_raw_led_test_is_capped_and_cleared_by_state_preset() {
  Device d("1.2.3", "abc");
  send(d, "TEST:START", 0);
  send(d, "TEST:LED:255,0,0", 1);
  assertLed({127, 0, 0}, d.update(2).led);
  send(d, "TEST:STATE:THINKING", 3);
  assertLed({64, 40, 0}, d.update(3).led);
  TEST_ASSERT_TRUE(d.testMode());
}

void test_req_TST_05_test_volume_applies_to_normal_state_sounds() {
  Device d("1.2.3", "abc");
  send(d, "TEST:START", 0);
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:VOLUME:100", 1).c_str());
  send(d, "TEST:BEEP:SUCCESS", 10);
  TEST_ASSERT_EQUAL_UINT16(512, d.update(20).toneDuty);
  send(d, "TEST:STOP", 500);
  send(d, "STATE:APPROVAL", 501);
  TEST_ASSERT_EQUAL_UINT16(512, d.update(510).toneDuty);
  TEST_ASSERT_EQUAL_STRING("ERR bad value\n", send(d, "TEST:VOLUME:101", 511).c_str());
}

void test_req_TST_07_timeout_stops_test_motor_and_keeps_volatile_tunings() {
  Device d("1.2.3", "abc");
  send(d, "TEST:START", 0);
  send(d, "TEST:SPEED:90", 1);
  send(d, "TEST:MOTOR:START", 2);
  TEST_ASSERT_TRUE(d.update(3).motorDuty > 0);
  const Outputs disconnected = d.update(cody::kTimeoutMs + 2);
  TEST_ASSERT_FALSE(d.testMode());
  TEST_ASSERT_EQUAL_UINT8(0, disconnected.motorDuty);
  TEST_ASSERT_EQUAL_INT(90, d.speed());
}

// --- Completed on the bench in step 2d; logic verified here ---

void test_req_FW_01_state_led_and_arms_follow_the_table() {
  Device d("1.2.3", "abc");
  send(d, "STATE:IDLE", 0);
  assertLed({0, 0, 19}, d.update(10).led);
  send(d, "STATE:WORKING", 0);
  assertLed({0, 0, 25}, d.update(0).led);      // pulse low: 20 %
  assertLed({0, 0, 127}, d.update(1000).led);  // pulse high: 100 %
  send(d, "STATE:THINKING", 0);
  assertLed({64, 40, 0}, d.update(10).led);
  send(d, "STATE:APPROVAL", 0);
  assertLed({127, 30, 0}, d.update(100).led);  // blink on
  assertLed({0, 0, 0}, d.update(600).led);     // blink off
  send(d, "STATE:COMPLETED", 0);
  assertLed({0, 64, 0}, d.update(10).led);
  send(d, "STATE:ERROR", 0);
  assertLed({127, 0, 0}, d.update(100).led);
  assertLed({0, 0, 0}, d.update(300).led);
}

void test_req_FW_04_sound_plays_once_when_a_state_is_entered() {
  Device d("1.2.3", "abc");
  send(d, "STATE:APPROVAL", 0);
  TEST_ASSERT_EQUAL_UINT16(2700, d.update(50).toneHz);   // first beep
  TEST_ASSERT_EQUAL_UINT16(0, d.update(150).toneHz);     // pause
  TEST_ASSERT_EQUAL_UINT16(2700, d.update(250).toneHz);  // second beep
  TEST_ASSERT_EQUAL_UINT16(0, d.update(400).toneHz);     // done
  TEST_ASSERT_EQUAL_UINT16(0, d.update(2000).toneHz);    // not repeated
  send(d, "STATE:APPROVAL", 3000);                       // re-sent: no new sound
  TEST_ASSERT_EQUAL_UINT16(0, d.update(3050).toneHz);
  send(d, "STATE:ERROR", 4000);
  TEST_ASSERT_EQUAL_UINT16(3000, d.update(4050).toneHz);
  send(d, "STATE:COMPLETED", 5000);
  TEST_ASSERT_EQUAL_UINT16(2000, d.update(5050).toneHz);
  send(d, "STATE:WORKING", 6000);
  TEST_ASSERT_EQUAL_UINT16(0, d.update(6050).toneHz);  // silent states
}

void test_req_FW_06_timeout_stops_motor_and_shows_disconnected() {
  Device d("1.2.3", "abc");
  send(d, "STATE:WORKING", 0);
  TEST_ASSERT_TRUE(d.update(29000).motorDuty > 0);
  send(d, "PING", 29000);  // heartbeat keeps it alive
  TEST_ASSERT_TRUE(d.update(58000).motorDuty > 0);
  const Outputs off = d.update(59000);  // 30 s after the last PING
  TEST_ASSERT_EQUAL_UINT8(0, off.motorDuty);
  TEST_ASSERT_FALSE(d.connected(59000));
  assertLed({13, 13, 13}, d.update(62000).led);  // white pulse at its peak (10 %)
  send(d, "PING", 63000);                        // back again
  TEST_ASSERT_TRUE(d.update(63500).motorDuty > 0);
}

void test_req_FW_07_disconnected_until_first_command() {
  Device d("1.2.3", "abc");
  TEST_ASSERT_FALSE(d.connected(0));
  assertLed({13, 13, 13}, d.update(2000).led);
  assertLed({0, 0, 0}, d.update(4000).led);
  send(d, "PING", 5000);
  assertLed({0, 0, 19}, d.update(5000).led);  // IDLE
}

void test_req_FW_09_volume_levels_and_mute() {
  Device d("1.2.3", "abc");
  const struct {
    int percent;
    uint16_t duty;
  } cases[] = {{0, 0}, {50, 91}, {100, 512}};
  for (const auto& test : cases) {
    send(d, (std::string("VOLUME:") + std::to_string(test.percent)).c_str(), 0);
    send(d, "BEEP:WARNING", 0);
    const Outputs out = d.update(10);
    TEST_ASSERT_EQUAL_UINT16(3000, out.toneHz);
    TEST_ASSERT_EQUAL_UINT16(test.duty, out.toneDuty);
  }
  TEST_ASSERT_EQUAL_STRING("ERR bad value\n", send(d, "VOLUME:101", 20).c_str());
}

void test_req_FW_14_brightness_uses_gamma_after_the_led_cap() {
  Device d("1.2.3", "abc");
  send(d, "STATE:ERROR", 0);
  TEST_ASSERT_EQUAL_UINT8(127, d.update(10).led.r);
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "BRIGHTNESS:50", 20).c_str());
  TEST_ASSERT_EQUAL_UINT8(28, d.update(20).led.r);
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "BRIGHTNESS:0", 30).c_str());
  assertLed({}, d.update(30).led);
  TEST_ASSERT_EQUAL_STRING("ERR bad value\n", send(d, "BRIGHTNESS:101", 40).c_str());
}

void test_req_FW_13_tuning_defaults_are_unsaved_and_save_load_persistently() {
  constexpr const char* savedAt = "2026-03-16T12:34:56.789Z";
  FakeTuningStore store;
  Device d("1.2.3", "abc");
  d.loadTuning(store);
  TEST_ASSERT_EQUAL_INT(1, store.loadCount);
  TEST_ASSERT_EQUAL_STRING("TUNING:224,10000,1,FAST,UNSAVED\nOK\n", send(d, "TUNING?", 0).c_str());
  TEST_ASSERT_EQUAL_INT(0, store.saveCount);

  TEST_ASSERT_EQUAL_STRING("ERR test mode inactive\n",
                           send(d, "TEST:TUNING:SAVE:2026-03-16T12:34:56Z", 1).c_str());
  send(d, "TEST:START", 2);
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:SPEED:100", 3).c_str());
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:FREQ:30000", 4).c_str());
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:KICK:5", 5).c_str());
  TEST_ASSERT_EQUAL_STRING("OK\n", send(d, "TEST:DECAY:SLOW", 6).c_str());
  TEST_ASSERT_EQUAL_STRING(
      "OK\n", send(d, (std::string("TEST:TUNING:SAVE:") + savedAt).c_str(), 7).c_str());
  TEST_ASSERT_EQUAL_INT(1, store.saveCount);
  TEST_ASSERT_EQUAL_STRING(savedAt, store.storedTimestamp.c_str());
  TEST_ASSERT_EQUAL_STRING("TUNING:100,30000,5,SLOW,2026-03-16T12:34:56.789Z\nOK\n",
                           send(d, "TUNING?", 8).c_str());

  Device rebooted("1.2.3", "abc");
  rebooted.loadTuning(store);
  TEST_ASSERT_EQUAL_INT(2, store.loadCount);
  TEST_ASSERT_EQUAL_INT(100, rebooted.speed());
  TEST_ASSERT_EQUAL_UINT32(30000, rebooted.tuning().frequencyHz);
  TEST_ASSERT_EQUAL_STRING(savedAt, rebooted.tuningTimestamp().c_str());
}

void test_req_FW_15_factory_reset_persists_defaults_and_timestamp() {
  FakeTuningStore store;
  store.loadResult = TuningLoadResult::Loaded;
  store.storedTuning.speed = 100;
  store.storedTuning.frequencyHz = 30000;
  store.storedTuning.kickStartMs = 5;
  store.storedTuning.decay = MotorDecay::Slow;
  store.storedTimestamp = "2026-03-16T12:34:56Z";
  Device d("1.2.3", "abc");
  d.loadTuning(store);
  send(d, "TEST:START", 0);
  const char* resetAt = "2026-03-16T13:00:00Z";
  TEST_ASSERT_EQUAL_STRING(
      "OK\n", send(d, (std::string("TEST:TUNING:RESET:") + resetAt).c_str(), 1).c_str());
  TEST_ASSERT_EQUAL_INT(1, store.saveCount);
  TEST_ASSERT_EQUAL_INT(224, store.storedTuning.speed);
  TEST_ASSERT_EQUAL_UINT32(10000, store.storedTuning.frequencyHz);
  TEST_ASSERT_EQUAL_UINT8(1, store.storedTuning.kickStartMs);
  TEST_ASSERT_TRUE(store.storedTuning.decay == MotorDecay::Fast);
  TEST_ASSERT_EQUAL_STRING(resetAt, store.storedTimestamp.c_str());
  TEST_ASSERT_EQUAL_STRING("TUNING:224,10000,1,FAST,2026-03-16T13:00:00Z\nOK\n",
                           send(d, "TUNING?", 2).c_str());
}

void test_req_FW_13_req_FW_15_storage_failures_are_reported() {
  FakeTuningStore store;
  store.loadResult = TuningLoadResult::Error;
  store.saveResult = false;
  Device d("1.2.3", "abc");
  d.loadTuning(store);
  TEST_ASSERT_EQUAL_STRING("ERR tuning storage unavailable\n", send(d, "TUNING?", 0).c_str());
  send(d, "TEST:START", 1);
  TEST_ASSERT_EQUAL_STRING("ERR tuning save failed\n",
                           send(d, "TEST:TUNING:SAVE:2026-03-16T12:34:56Z", 2).c_str());
  TEST_ASSERT_EQUAL_STRING("ERR tuning reset failed\n",
                           send(d, "TEST:TUNING:RESET:2026-03-16T12:34:56Z", 3).c_str());
}

int main() {
  UNITY_BEGIN();
  RUN_TEST(test_req_FW_02_lines_end_with_newline_and_ignore_cr);
  RUN_TEST(test_req_FW_02_too_long_line_is_reported_and_dropped);
  RUN_TEST(test_req_FW_03_state_command_switches_state);
  RUN_TEST(test_req_FW_03_effects_run_without_commands);
  RUN_TEST(test_req_FW_05_completed_and_error_stay_until_next_state);
  RUN_TEST(test_req_FW_08_speed_sets_motor_duty_in_working);
  RUN_TEST(test_req_FW_08_motor_stopped_outside_working);
  RUN_TEST(test_req_FW_10_every_command_is_answered_ok);
  RUN_TEST(test_req_FW_10_errors_keep_the_state);
  RUN_TEST(test_req_FW_11_reports_version_and_commit);
  RUN_TEST(test_req_FW_12_led_shows_colour_until_next_state);
  RUN_TEST(test_req_FW_12_beep_plays_a_sound);
  RUN_TEST(test_req_TST_01_test_mode_owns_controls_until_stopped);
  RUN_TEST(test_req_TST_02_state_presets_run_while_test_mode_stays_active);
  RUN_TEST(test_req_TST_03_motor_tunings_apply_to_test_and_normal_working);
  RUN_TEST(test_req_TST_03_motor_tuning_bounds_include_zero_kick_and_30_khz);
  RUN_TEST(test_req_TST_04_raw_led_test_is_capped_and_cleared_by_state_preset);
  RUN_TEST(test_req_TST_05_test_volume_applies_to_normal_state_sounds);
  RUN_TEST(test_req_TST_07_timeout_stops_test_motor_and_keeps_volatile_tunings);
  RUN_TEST(test_req_FW_01_state_led_and_arms_follow_the_table);
  RUN_TEST(test_req_FW_04_sound_plays_once_when_a_state_is_entered);
  RUN_TEST(test_req_FW_06_timeout_stops_motor_and_shows_disconnected);
  RUN_TEST(test_req_FW_07_disconnected_until_first_command);
  RUN_TEST(test_req_FW_09_volume_levels_and_mute);
  RUN_TEST(test_req_FW_14_brightness_uses_gamma_after_the_led_cap);
  RUN_TEST(test_req_FW_13_tuning_defaults_are_unsaved_and_save_load_persistently);
  RUN_TEST(test_req_FW_15_factory_reset_persists_defaults_and_timestamp);
  RUN_TEST(test_req_FW_13_req_FW_15_storage_failures_are_reported);
  return UNITY_END();
}

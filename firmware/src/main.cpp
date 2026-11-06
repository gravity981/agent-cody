// Cody firmware. Skeleton from step 1c: reports its version (REL-02,
// spec/architecture.md 4.2 and 6). The protocol follows in stage 2.
#include <Arduino.h>

#include "version.h"

void setup() {
  Serial.begin(115200);
  // USB CDC: give a host up to 3 s to open the port, so the line isn't lost
  const unsigned long start = millis();
  while (!Serial && millis() - start < 3000) {
    delay(10);
  }
  Serial.printf("READY %s\n", cody::version());
  // keep the build info marker in the image (the linker drops unreferenced data)
  asm volatile("" : : "r"(cody::kBuildInfo));
}

void loop() {}

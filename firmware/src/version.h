// Version and commit of the build (REL-02), passed in as build flags.
#pragma once

#ifndef CODY_VERSION
#define CODY_VERSION ""
#endif
#ifndef CODY_COMMIT
#define CODY_COMMIT ""
#endif

namespace cody {

// "0.0.0+unknown" / "unknown" when built without VERSION / COMMIT
inline const char* version() { return CODY_VERSION[0] ? CODY_VERSION : "0.0.0+unknown"; }
inline const char* commit() { return CODY_COMMIT[0] ? CODY_COMMIT : "unknown"; }

// Keeps both strings in the image, so a released .bin reveals them
// (marker "CODY_BUILD <version> <commit>").
extern const char kBuildInfo[];

}  // namespace cody

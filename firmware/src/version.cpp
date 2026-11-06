#include "version.h"

namespace cody {

__attribute__((used)) const char kBuildInfo[] = "CODY_BUILD " CODY_VERSION " " CODY_COMMIT;

}  // namespace cody

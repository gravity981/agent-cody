# Shared helpers for the ci/ entry points (sourced, not executed).
# shellcheck shell=bash

# Version and commit of this build (spec/delivery.md 2): from the environment
# in CI, determined locally otherwise.
cody_version() { "$(dirname "${BASH_SOURCE[0]}")/version"; }
cody_commit() { echo "${COMMIT:-$(git rev-parse HEAD)}"; }

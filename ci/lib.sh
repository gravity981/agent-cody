# Shared helpers for the ci/ entry points (sourced, not executed).
# shellcheck shell=bash

# Version and commit of this build (spec/delivery.md 2): from the environment
# in CI, determined locally otherwise.
cody_version() { "$(dirname "${BASH_SOURCE[0]}")/version"; }
cody_commit() { echo "${COMMIT:-$(git rev-parse HEAD)}"; }

# owner/repo on GitHub: $GITHUB_REPOSITORY in CI, the origin remote locally.
cody_repository() {
  if [[ -n "${GITHUB_REPOSITORY:-}" ]]; then
    echo "$GITHUB_REPOSITORY"
  else
    git remote get-url origin | sed -E 's#^(git@github\.com:|https://github\.com/)##; s#\.git$##'
  fi
}

# The bridge image on GHCR (spec/architecture.md TD-04), lowercase as GHCR requires.
# CODY_BRIDGE_IMAGE overrides it, e.g. with a local registry for testing.
cody_bridge_image() {
  if [[ -n "${CODY_BRIDGE_IMAGE:-}" ]]; then
    echo "$CODY_BRIDGE_IMAGE"
    return
  fi
  local owner
  owner=$(cody_repository | cut -d/ -f1)
  echo "ghcr.io/${owner,,}/cody-bridge"
}

# Logs in to GHCR with the workflow's token (GHCR_USER, GHCR_TOKEN); other
# registries (local tests) need no login.
cody_ghcr_login() {
  [[ "$(cody_bridge_image)" == ghcr.io/* ]] || return 0
  if [[ -z "${GHCR_TOKEN:-}" || -z "${GHCR_USER:-}" ]]; then
    echo "GHCR_USER and GHCR_TOKEN must be set (spec/delivery.md 2)" >&2
    return 1
  fi
  echo "$GHCR_TOKEN" | docker login ghcr.io --username "$GHCR_USER" --password-stdin
}

# Whether an image reference exists in its registry: returns 0 if it does, 1 if
# it doesn't; any other error (network, permissions) stops the entry point.
cody_image_exists() {
  local out
  if out=$(docker buildx imagetools inspect "$1" 2>&1); then
    return 0
  fi
  if grep -qiE "not found|manifest unknown" <<< "$out"; then
    return 1
  fi
  echo "Can't check whether $1 exists:" >&2
  echo "$out" >&2
  exit 1
}

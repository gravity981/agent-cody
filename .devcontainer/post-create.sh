#!/usr/bin/env bash
# postCreateCommand (devcontainer.json): runs when the container is created,
# locally and in CI (devcontainers/ci).
set -euo pipefail

# devcontainers/ci gives the vscode user the runner's UID. pioarduino modifies
# files in its own package folder during every build (shutil.copy2, which only
# the owner may do), so the toolchain must belong to the current user. Locally
# the UIDs match and nothing is changed.
for dir in /opt/platformio /opt/cody-venv; do
  if [[ "$(stat -c %u "$dir")" != "$(id -u)" ]]; then
    echo "Taking ownership of $dir for UID $(id -u)"
    sudo chown -R "$(id -u):$(id -g)" "$dir"
  fi
done

# Install the workspace members (bridge, cad, tools) into the prebuilt environment
uv sync --frozen

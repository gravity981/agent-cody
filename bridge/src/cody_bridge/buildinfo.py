"""Version and commit of this build, set in the image (REL-02, spec/architecture.md 6)."""

from __future__ import annotations

import os


def version() -> str:
    return os.environ.get("CODY_VERSION") or "0.0.0+unknown"


def commit() -> str:
    return os.environ.get("CODY_COMMIT") or "unknown"

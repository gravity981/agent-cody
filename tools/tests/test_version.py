"""Version determination and the version in the firmware image (REL-02, spec/delivery.md 1)."""

from __future__ import annotations

import os
import subprocess
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[2]
VERSION_SCRIPT = ROOT / "ci" / "version"
FIRMWARE_BIN = ROOT / "firmware" / ".pio" / "build" / "esp32c3" / "firmware.bin"


def git(repo: Path, *args: str) -> str:
    env = {
        **os.environ,
        "GIT_AUTHOR_NAME": "t",
        "GIT_AUTHOR_EMAIL": "t@example.com",
        "GIT_COMMITTER_NAME": "t",
        "GIT_COMMITTER_EMAIL": "t@example.com",
    }
    out = subprocess.run(["git", *args], cwd=repo, env=env, check=True, capture_output=True)
    return out.stdout.decode().strip()


@pytest.fixture
def repo(tmp_path: Path) -> Path:
    git(tmp_path, "init", "-q")
    git(tmp_path, "commit", "-q", "--allow-empty", "-m", "first")
    return tmp_path


def version_in(repo: Path, version_env: str | None = None) -> str:
    env = {k: v for k, v in os.environ.items() if k != "VERSION"}
    if version_env is not None:
        env["VERSION"] = version_env
    out = subprocess.run([str(VERSION_SCRIPT)], cwd=repo, env=env, check=True, capture_output=True)
    return out.stdout.decode().strip()


def test_req_REL_02_local_version_from_last_release(repo: Path) -> None:
    git(repo, "tag", "-a", "v0.1.0", "-m", "release")
    git(repo, "commit", "-q", "--allow-empty", "-m", "second")
    git(repo, "tag", "-a", "v0.2.0-rc.1", "-m", "candidate")  # candidates don't count
    short = git(repo, "rev-parse", "--short", "HEAD")
    assert version_in(repo) == f"0.1.0+local.{short}"


def test_req_REL_02_local_version_without_release_is_0_0_0(repo: Path) -> None:
    short = git(repo, "rev-parse", "--short", "HEAD")
    assert version_in(repo) == f"0.0.0+local.{short}"


def test_req_REL_02_version_from_environment_wins(repo: Path) -> None:
    git(repo, "tag", "-a", "v0.1.0", "-m", "release")
    assert version_in(repo, "1.2.0+build.7") == "1.2.0+build.7"


def test_req_REL_02_firmware_image_contains_version_and_commit() -> None:
    version, commit = os.environ.get("VERSION"), os.environ.get("COMMIT")
    if not (version and commit and FIRMWARE_BIN.exists()):
        pytest.skip("needs VERSION, COMMIT and a firmware build (run by ci/check)")
    image = FIRMWARE_BIN.read_bytes()
    assert f"CODY_BUILD {version} {commit}".encode() in image
    assert b"READY %s\n" in image

"""cody-bridge command line (skeleton): version and commit (REL-02, spec/architecture.md 6)."""

from __future__ import annotations

import pytest
from cody_bridge.__main__ import main


def test_req_REL_02_bridge_reports_version_and_commit(
    monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    monkeypatch.setenv("CODY_VERSION", "1.2.0")
    monkeypatch.setenv("CODY_COMMIT", "0123abc")
    assert main(["--version"]) == 0
    assert capsys.readouterr().out == "cody-bridge 1.2.0 0123abc\n"


def test_req_REL_02_bridge_without_build_info_says_unknown(
    monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
) -> None:
    monkeypatch.delenv("CODY_VERSION", raising=False)
    monkeypatch.delenv("CODY_COMMIT", raising=False)
    assert main(["--version"]) == 0
    assert capsys.readouterr().out == "cody-bridge 0.0.0+unknown unknown\n"

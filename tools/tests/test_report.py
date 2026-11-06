"""Tests of the requirement report (TRC-01 – TRC-07). Tag convention: spec/verification.md 3."""

from __future__ import annotations

from pathlib import Path

import pytest
from cody_tools import report

REQUIREMENTS = """# Requirements

## 1. Alpha (ABC)

| ID | Requirement | Prio |
|---|---|---|
| ABC-01 | First. | Must |
| ABC-02 | Second. | Should |
| ABC-03 | Third. | Could |
| ~~ABC-04~~ | ~~Dropped.~~ (withdrawn: not needed) | – |
| ABC-05 | Fifth. | Must |
"""

VERIFICATION = """# Verification

## 5. Not yet implemented

| Prefix | IDs |
|---|---|
| ABC | {nyi} |

## 6. Manual verification
"""

PLAN = """# Plan

## 3. Stages and steps

### Stage 1 – One

**1a – Step.** Text.
Requirements: {stage1}

### Stage 2 – Two

**2a – Step.** Text.
Requirements: {stage2}

## 4. Status report template

**Requirements:** <implemented> of <planned>
"""

LOG = """# Log

## Records

| Requirements | Date | Environment | Result | Notes |
|---|---|---|---|---|
{rows}
"""


def junit(*cases: tuple[str, str]) -> str:
    body = []
    for name, outcome in cases:
        inner = {"pass": "", "fail": "<failure message='x'/>", "skipped": "<skipped/>"}[outcome]
        body.append(f'<testcase classname="t" name="{name}">{inner}</testcase>')
    return f'<testsuites><testsuite name="s">{"".join(body)}</testsuite></testsuites>'


def make_repo(
    tmp_path: Path,
    *,
    requirements: str = REQUIREMENTS,
    nyi: str = "ABC-02 – ABC-03, ABC-05",
    stage1: str = "ABC-01 – ABC-03.",
    stage2: str = "ABC-05.",
    log_rows: str = "",
    results: str | None = None,
) -> Path:
    spec = tmp_path / "spec"
    spec.mkdir()
    (spec / "requirements.md").write_text(requirements)
    (spec / "verification.md").write_text(VERIFICATION.format(nyi=nyi))
    (spec / "implementation-plan.md").write_text(PLAN.format(stage1=stage1, stage2=stage2))
    (spec / "verification-log.md").write_text(LOG.format(rows=log_rows))
    reports = tmp_path / "reports"
    reports.mkdir()
    if results is not None:
        (reports / "unit.xml").write_text(results)
    return tmp_path


OK_RESULTS = junit(("test_req_ABC_01_works", "pass"))


# --- TRC-01 ---------------------------------------------------------------------


def test_req_TRC_01_reads_all_rows_including_withdrawn(tmp_path: Path) -> None:
    rep = report.build(make_repo(tmp_path, results=OK_RESULTS))
    assert list(rep.requirements) == ["ABC-01", "ABC-02", "ABC-03", "ABC-04", "ABC-05"]
    assert rep.requirements["ABC-04"].withdrawn
    assert not rep.requirements["ABC-01"].withdrawn
    assert rep.errors == []


def test_req_TRC_01_duplicate_id_is_error(tmp_path: Path) -> None:
    reqs = REQUIREMENTS + "| ABC-01 | Again. | Must |\n"
    rep = report.build(make_repo(tmp_path, requirements=reqs, results=OK_RESULTS))
    assert any("ABC-01: duplicate ID" in e for e in rep.errors)


def test_req_TRC_01_unknown_priority_is_error(tmp_path: Path) -> None:
    reqs = REQUIREMENTS.replace("| Second. | Should |", "| Second. | Maybe |")
    rep = report.build(make_repo(tmp_path, requirements=reqs, results=OK_RESULTS))
    assert any("ABC-02: unknown priority 'Maybe'" in e for e in rep.errors)


# --- TRC-02 ---------------------------------------------------------------------


def test_req_TRC_02_reads_list_with_ranges(tmp_path: Path) -> None:
    rep = report.build(make_repo(tmp_path, results=OK_RESULTS))
    assert rep.not_implemented == {"ABC-02", "ABC-03", "ABC-05"}


def test_req_TRC_02_unknown_id_on_list_is_error(tmp_path: Path) -> None:
    rep = report.build(
        make_repo(tmp_path, nyi="ABC-02 – ABC-03, ABC-05, ABC-09", results=OK_RESULTS)
    )
    assert any("unknown requirement ABC-09" in e for e in rep.errors)


def test_req_TRC_02_withdrawn_id_on_list_is_error(tmp_path: Path) -> None:
    rep = report.build(make_repo(tmp_path, nyi="ABC-02 – ABC-05", results=OK_RESULTS))
    assert any("verification.md 5: withdrawn requirement ABC-04" in e for e in rep.errors)


# --- TRC-03 ---------------------------------------------------------------------


def test_req_TRC_03_reads_results_with_tags(tmp_path: Path) -> None:
    results = junit(("test_req_ABC_01_req_ABC_02_both", "pass"), ("test_untagged", "fail"))
    rep = report.build(make_repo(tmp_path, results=results))
    assert [e.source for e in rep.evidence["ABC-01"]] == ["test_req_ABC_01_req_ABC_02_both"]
    assert [e.result for e in rep.evidence["ABC-02"]] == ["pass"]


# ids keep the malformed tags out of this test's own name in the JUnit results
@pytest.mark.parametrize(
    "name",
    ["test_req_abc_01_x", "test_req_ABC_1_x", "test_req_ABC_001_x"],
    ids=["lowercase", "one-digit", "three-digits"],
)
def test_req_TRC_03_malformed_tag_is_error(tmp_path: Path, name: str) -> None:
    rep = report.build(
        make_repo(tmp_path, results=junit(("test_req_ABC_01_ok", "pass"), (name, "pass")))
    )
    assert any("malformed tag" in e and name in e for e in rep.errors)


def test_req_TRC_03_unknown_and_withdrawn_tags_are_errors(tmp_path: Path) -> None:
    results = junit(
        ("test_req_ABC_01_ok", "pass"), ("test_req_XYZ_01_x", "pass"), ("test_req_ABC_04_x", "pass")
    )
    rep = report.build(make_repo(tmp_path, results=results))
    assert any("unknown requirement XYZ-01" in e for e in rep.errors)
    assert any("withdrawn requirement ABC-04" in e for e in rep.errors)


def test_req_TRC_03_skipped_test_is_no_evidence(tmp_path: Path) -> None:
    rep = report.build(make_repo(tmp_path, results=junit(("test_req_ABC_01_x", "skipped"))))
    assert report.status_of(rep, "ABC-01") == ("implemented", "no evidence")
    assert "ABC-01: implemented but no passing evidence" in rep.errors


# --- TRC-04 ---------------------------------------------------------------------


def test_req_TRC_04_newest_record_counts(tmp_path: Path) -> None:
    rows = "| ABC-01 | 2026-02-01 | bench | pass | |\n| ABC-01 | 2026-01-01 | bench | fail | |"
    rep = report.build(make_repo(tmp_path, log_rows=rows))
    assert [(e.source, e.result) for e in rep.evidence["ABC-01"]] == [("log: 2026-02-01", "pass")]
    assert rep.errors == []


def test_req_TRC_04_same_date_lower_row_counts(tmp_path: Path) -> None:
    rows = (
        "| ABC-01 | 2026-02-01 | bench | pass | |\n| ABC-01, ABC-02 | 2026-02-01 | bench | fail | |"
    )
    rep = report.build(make_repo(tmp_path, log_rows=rows))
    assert [e.result for e in rep.evidence["ABC-01"]] == ["fail"]
    assert "ABC-01: failing evidence log: 2026-02-01" in rep.errors


def test_req_TRC_04_invalid_rows_are_errors(tmp_path: Path) -> None:
    rows = "| ABC-01 | 1.2.2026 | bench | pass | |\n| ABC-01 | 2026-02-01 | bench | ok | |"
    rep = report.build(make_repo(tmp_path, log_rows=rows, results=OK_RESULTS))
    assert any("invalid date" in e for e in rep.errors)
    assert any("result must be pass or fail" in e for e in rep.errors)


# --- TRC-05 ---------------------------------------------------------------------


def test_req_TRC_05_implemented_without_passing_evidence_is_error(tmp_path: Path) -> None:
    rep = report.build(make_repo(tmp_path, results=None))
    assert rep.errors == ["ABC-01: implemented but no passing evidence"]


def test_req_TRC_05_failing_evidence_is_error_even_if_not_implemented(tmp_path: Path) -> None:
    results = junit(("test_req_ABC_01_ok", "pass"), ("test_req_ABC_02_broken", "fail"))
    rep = report.build(make_repo(tmp_path, results=results))
    assert rep.errors == ["ABC-02: failing evidence test_req_ABC_02_broken"]


def test_req_TRC_05_not_implemented_without_evidence_is_fine(tmp_path: Path) -> None:
    rep = report.build(make_repo(tmp_path, results=OK_RESULTS))
    assert rep.errors == []


# --- TRC-06 ---------------------------------------------------------------------


def test_req_TRC_06_passing_but_not_implemented_is_warning(tmp_path: Path) -> None:
    results = junit(("test_req_ABC_01_ok", "pass"), ("test_req_ABC_02_ok", "pass"))
    rep = report.build(make_repo(tmp_path, results=results))
    assert rep.warnings == ["ABC-02: has passing evidence but is still 'not yet implemented'"]
    assert rep.errors == []


def test_req_TRC_06_no_stage_and_several_stages_are_warnings(tmp_path: Path) -> None:
    rep = report.build(
        make_repo(tmp_path, stage1="ABC-01 – ABC-02.", stage2="ABC-02, ABC-05.", results=OK_RESULTS)
    )
    assert "ABC-03: planned in no stage" in rep.warnings
    assert "ABC-02: planned in several stages (1, 2)" in rep.warnings
    assert not any("ABC-04" in w for w in rep.warnings)  # withdrawn: not planned


def test_req_TRC_06_none_lines_and_template_are_ignored(tmp_path: Path) -> None:
    rep = report.build(
        make_repo(
            tmp_path, stage2="none (part of ABC-05, verified elsewhere). ", results=OK_RESULTS
        )
    )
    assert "ABC-05: planned in no stage" in rep.warnings
    assert rep.errors == []


# --- TRC-07 ---------------------------------------------------------------------


def test_req_TRC_07_writes_report_and_exits_zero_without_errors(
    tmp_path: Path, monkeypatch
) -> None:
    monkeypatch.setenv("VERSION", "1.2.3")
    monkeypatch.setenv("COMMIT", "abc123")
    root = make_repo(tmp_path, results=OK_RESULTS)
    assert report.main(["--root", str(root)]) == 0
    text = (root / "reports" / "requirements.md").read_text()
    assert "Version `1.2.3`, commit `abc123`" in text
    assert "**Result: OK** (0 errors, 0 warnings)" in text
    for heading in ("## Totals", "## Per stage", "## Errors", "## Warnings", "## Requirements"):
        assert heading in text
    assert "| Active | 4 |" in text and "| Withdrawn | 1 |" in text
    assert "| 1 | 3 | 1 | 1 |" in text  # stage 1: 3 planned, 1 implemented, 1 passing
    assert "| ABC-01 | Must | 1 | implemented | `test_req_ABC_01_works`: pass | pass |" in text


def test_req_TRC_07_exits_non_zero_on_error(tmp_path: Path) -> None:
    root = make_repo(tmp_path, results=None)
    assert report.main(["--root", str(root)]) == 1
    text = (root / "reports" / "requirements.md").read_text()
    assert "**Result: FAILED**" in text
    assert "- ABC-01: implemented but no passing evidence" in text

"""Build documentation checks (DOC-02, DOC-05, DOC-06)."""

from __future__ import annotations

import csv
import re
from pathlib import Path

from cody_tools.build_docs import DOCS_DIR, build_release_docs

ROOT = DOCS_DIR.parent
STEP_HEADING = re.compile(r"^## Step \d+: .+$", re.MULTILINE)
IMAGE_LINK = re.compile(r"!\[[^\]]+\]\(([^)]+)\)")


def _step_sections(markdown: str) -> list[str]:
    starts = [match.start() for match in STEP_HEADING.finditer(markdown)]
    return [
        markdown[start : starts[index + 1] if index + 1 < len(starts) else len(markdown)]
        for index, start in enumerate(starts)
    ]


def test_req_DOC_02_instructions_cover_stage_two_setup() -> None:
    instructions = (DOCS_DIR / "assembly-instructions.md").read_text(encoding="utf-8").lower()
    for topic in (
        "solder",
        "wire the modules",
        "flash the firmware",
        "install and start the bridge",
        "install claude code hooks",
    ):
        assert topic in instructions


def test_req_DOC_05_each_step_names_parts_tools_and_existing_images() -> None:
    markdown = (DOCS_DIR / "assembly-instructions.md").read_text(encoding="utf-8")
    sections = _step_sections(markdown)
    assert sections
    for section in sections:
        assert re.search(r"^\*\*Parts:\*\* .+", section, re.MULTILINE)
        assert re.search(r"^\*\*Tools:\*\* .+", section, re.MULTILINE)
        images = IMAGE_LINK.findall(section)
        assert images
        for image in images:
            assert (DOCS_DIR / image).resolve().is_file()


def test_req_DOC_06_builds_versioned_parts_csv_and_instructions_pdf(tmp_path: Path) -> None:
    parts_csv, instructions_pdf = build_release_docs("1.2.3-test", tmp_path)
    assert parts_csv.name == "cody-parts-list-1.2.3-test.csv"
    assert instructions_pdf.name == "cody-assembly-instructions-1.2.3-test.pdf"
    with parts_csv.open(encoding="utf-8", newline="") as source:
        rows = list(csv.DictReader(source))
    assert len(rows) == 5
    assert rows[0]["Part"] == "ESP32-C3 SuperMini"
    assert rows[0]["Source"].startswith("TENSTAR Store")
    assert instructions_pdf.read_bytes().startswith(b"%PDF-")
    for target in (parts_csv, instructions_pdf):
        assert target.stat().st_size > 0

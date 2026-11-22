"""Build versioned documentation release files from the Markdown sources."""

from __future__ import annotations

import argparse
import csv
import re
import subprocess
import tempfile
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[3]
DOCS_DIR = REPO_ROOT / "docs"
CSV_FIELDS = (
    "Part",
    "Qty per Cody",
    "Variant to order",
    "Source",
    "Price paid (CHF)",
    "Per Cody (CHF)",
)
IMAGE_ALT = re.compile(r"!\[([^\]]+)\]\([^)]+\)")


def _parts_rows(markdown: str) -> list[list[str]]:
    purchased_section = False
    table_started = False
    rows = []
    for line in markdown.splitlines():
        if line.strip() == "## Purchased parts":
            purchased_section = True
            continue
        if purchased_section and line.startswith("|"):
            cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
            if not table_started:
                table_started = True
                if cells[1:3] != ["Part", "Qty per Cody"]:
                    raise ValueError("unexpected purchased-parts table header")
                continue
            if all(set(cell) <= {"-", ":"} for cell in cells):
                continue
            if len(cells) != 7:
                raise ValueError("purchased-parts table row must have seven columns")
            image = IMAGE_ALT.fullmatch(cells[0])
            if image is None:
                raise ValueError("purchased-parts row has no picture description")
            rows.append([image.group(1), *cells[2:]])
        elif table_started:
            break
    if not rows:
        raise ValueError("purchased-parts table has no data rows")
    return rows


def _write_parts_csv(source: Path, destination: Path) -> None:
    rows = _parts_rows(source.read_text(encoding="utf-8"))
    with destination.open("w", encoding="utf-8", newline="") as output:
        writer = csv.writer(output, lineterminator="\n")
        writer.writerow(CSV_FIELDS)
        writer.writerows(rows)


def build_release_docs(version: str, release_dir: Path) -> tuple[Path, Path]:
    if not version or "/" in version or "\\" in version:
        raise ValueError("version must be a non-empty filename component")
    release_dir.mkdir(parents=True, exist_ok=True)
    parts_csv = release_dir / f"cody-parts-list-{version}.csv"
    instructions_pdf = release_dir / f"cody-assembly-instructions-{version}.pdf"
    _write_parts_csv(DOCS_DIR / "parts-list.md", parts_csv)

    with tempfile.TemporaryDirectory(prefix="cody-docs-") as temporary:
        html_path = Path(temporary) / "assembly-instructions.html"
        subprocess.run(
            [
                "pandoc",
                "--standalone",
                "--self-contained",
                "--resource-path=.:..",
                "--css=print.css",
                "--metadata",
                "title=Cody Stage 2 Assembly Instructions",
                "--output",
                str(html_path),
                "assembly-instructions.md",
            ],
            cwd=DOCS_DIR,
            check=True,
        )
        subprocess.run(
            ["weasyprint", str(html_path), str(instructions_pdf)],
            cwd=DOCS_DIR,
            check=True,
        )
    return parts_csv, instructions_pdf


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True)
    parser.add_argument("--release-dir", type=Path, required=True)
    args = parser.parse_args(argv)
    parts_csv, instructions_pdf = build_release_docs(args.version, args.release_dir)
    print(parts_csv)
    print(instructions_pdf)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

"""Requirement report (TRC-01 – TRC-07, spec/verification.md 3).

Reads the requirements, the "not yet implemented" list, the stage plan, the
automated results (JUnit XML in reports/) and the manual verification log, and
writes reports/requirements.md. Exits non-zero on any error.
"""

from __future__ import annotations

import argparse
import os
import re
import sys
import xml.etree.ElementTree as ET
from dataclasses import dataclass, field
from datetime import date
from pathlib import Path

PRIORITIES = {"Must", "Should", "Could"}
WITHDRAWN_PRIO = {"–", "-"}
ID_RE = re.compile(r"^[A-Z]{2,4}-\d{2}$")
RANGE_RE = re.compile(r"^([A-Z]{2,4})-(\d{2})\s*[–-]\s*([A-Z]{2,4})-(\d{2})$")
WITHDRAWN_RE = re.compile(r"\(withdrawn\b")
TAG_RE = re.compile(r"req_([A-Z]{2,4})_(\d{2})(?!\d)")

PASS, FAIL, SKIP = "pass", "fail", "skipped"


@dataclass
class Requirement:
    id: str
    text: str
    prio: str
    withdrawn: bool


@dataclass
class Evidence:
    source: str  # test name or "log: <date>"
    result: str  # pass / fail / skipped


@dataclass
class Report:
    requirements: dict[str, Requirement] = field(default_factory=dict)
    not_implemented: set[str] = field(default_factory=set)
    stages: dict[str, set[str]] = field(default_factory=dict)  # requirement -> stages
    evidence: dict[str, list[Evidence]] = field(default_factory=dict)
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


def _cells(line: str) -> list[str] | None:
    line = line.strip()
    if not (line.startswith("|") and line.endswith("|")):
        return None
    return [c.strip() for c in line[1:-1].split("|")]


def _strip_strike(text: str) -> str:
    return text.replace("~~", "").strip()


def _section(text: str, heading_prefix: str) -> str:
    """Text of the level-2 section whose heading starts with heading_prefix."""
    lines = text.splitlines()
    out: list[str] = []
    inside = False
    for line in lines:
        if line.startswith("## "):
            inside = line.startswith(heading_prefix)
            continue
        if inside:
            out.append(line)
    return "\n".join(out)


def expand_ids(spec: str) -> tuple[list[str], list[str]]:
    """Expand a comma-separated list of IDs and ranges. Returns (ids, problems)."""
    ids: list[str] = []
    problems: list[str] = []
    for part in (p.strip() for p in spec.split(",")):
        if not part:
            continue
        if ID_RE.match(part):
            ids.append(part)
            continue
        m = RANGE_RE.match(part)
        if m and m.group(1) == m.group(3) and int(m.group(2)) <= int(m.group(4)):
            prefix = m.group(1)
            ids.extend(f"{prefix}-{n:02d}" for n in range(int(m.group(2)), int(m.group(4)) + 1))
            continue
        problems.append(part)
    return ids, problems


# --- TRC-01: requirements -------------------------------------------------------


def read_requirements(path: Path, rep: Report) -> None:
    for line in path.read_text(encoding="utf-8").splitlines():
        cells = _cells(line)
        if not cells or len(cells) != 3:
            continue
        rid = _strip_strike(cells[0])
        if not ID_RE.match(rid):
            continue
        text, prio = cells[1], _strip_strike(cells[2])
        withdrawn = prio in WITHDRAWN_PRIO and WITHDRAWN_RE.search(text) is not None
        if rid in rep.requirements:
            rep.errors.append(f"{rid}: duplicate ID in requirements.md")
            continue
        if not withdrawn and prio not in PRIORITIES:
            rep.errors.append(f"{rid}: unknown priority '{prio}'")
        rep.requirements[rid] = Requirement(rid, text, prio, withdrawn)


def _check_known(rep: Report, rid: str, where: str) -> bool:
    req = rep.requirements.get(rid)
    if req is None:
        rep.errors.append(f"{where}: unknown requirement {rid}")
        return False
    if req.withdrawn:
        rep.errors.append(f"{where}: withdrawn requirement {rid}")
        return False
    return True


# --- TRC-02: "not yet implemented" list -------------------------------------------


def read_not_implemented(path: Path, rep: Report) -> None:
    section = _section(path.read_text(encoding="utf-8"), "## 5. Not yet implemented")
    for line in section.splitlines():
        cells = _cells(line)
        if not cells or len(cells) != 2 or cells[0] in ("Prefix", "") or set(cells[0]) <= {"-"}:
            continue
        ids, problems = expand_ids(cells[1])
        for p in problems:
            rep.errors.append(f"verification.md 5: malformed entry '{p}'")
        for rid in ids:
            if _check_known(rep, rid, "verification.md 5"):
                rep.not_implemented.add(rid)


# --- Stage plan (TRC-06) -------------------------------------------------------------


STAGE_RE = re.compile(r"^### Stage (\d+)\b")


def read_stages(path: Path, rep: Report) -> None:
    section = _section(path.read_text(encoding="utf-8"), "## 3.")
    stage: str | None = None
    for line in section.splitlines():
        m = STAGE_RE.match(line)
        if m:
            stage = m.group(1)
            continue
        if stage is None or not line.startswith("Requirements:"):
            continue
        spec = line[len("Requirements:") :].strip().rstrip(".")
        if spec.lower().startswith("none"):
            continue
        ids, problems = expand_ids(spec)
        for p in problems:
            rep.errors.append(f"implementation-plan.md stage {stage}: malformed entry '{p}'")
        for rid in ids:
            if _check_known(rep, rid, f"implementation-plan.md stage {stage}"):
                rep.stages.setdefault(rid, set()).add(stage)


# --- TRC-03: automated results -----------------------------------------------------


def tags_in(name: str) -> tuple[list[str], list[str]]:
    """Requirement IDs tagged in a test name, and malformed tags."""
    ids = [f"{m.group(1)}-{m.group(2)}" for m in TAG_RE.finditer(name)]
    starts = (m.start() for m in re.finditer("req_", name))
    malformed = [name[i : i + 12] for i in starts if not TAG_RE.match(name, i)]
    return ids, malformed


def _outcome(case: ET.Element) -> str:
    if case.find("failure") is not None or case.find("error") is not None:
        return FAIL
    if case.find("skipped") is not None:
        return SKIP
    return PASS


def read_results(reports_dir: Path, rep: Report) -> None:
    for xml in sorted(reports_dir.rglob("*.xml")):
        try:
            root = ET.parse(xml).getroot()
        except ET.ParseError as exc:
            rep.errors.append(f"{xml.name}: not readable ({exc})")
            continue
        if root.tag not in ("testsuites", "testsuite"):
            continue
        for case in root.iter("testcase"):
            name = case.get("name", "")
            ids, malformed = tags_in(name)
            for bad in malformed:
                rep.errors.append(f"{xml.name}: malformed tag '{bad}…' in test '{name}'")
            outcome = _outcome(case)
            for rid in ids:
                if _check_known(rep, rid, f"{xml.name} test '{name}'"):
                    rep.evidence.setdefault(rid, []).append(Evidence(name, outcome))


# --- TRC-04: manual verification log --------------------------------------------------


def read_log(path: Path, rep: Report) -> None:
    section = _section(path.read_text(encoding="utf-8"), "## Records")
    latest: dict[str, tuple[date, int, Evidence]] = {}
    row = 0
    for line in section.splitlines():
        cells = _cells(line)
        if not cells or len(cells) != 5 or cells[0] == "Requirements" or set(cells[0]) <= {"-"}:
            continue
        row += 1
        req_cell, date_cell, _env, result, _notes = cells
        try:
            when = date.fromisoformat(date_cell)
        except ValueError:
            rep.errors.append(f"verification-log.md row {row}: invalid date '{date_cell}'")
            continue
        if result not in (PASS, FAIL):
            rep.errors.append(f"verification-log.md row {row}: result must be pass or fail")
            continue
        ids, problems = expand_ids(req_cell)
        for p in problems:
            rep.errors.append(f"verification-log.md row {row}: malformed entry '{p}'")
        for rid in ids:
            if not _check_known(rep, rid, f"verification-log.md row {row}"):
                continue
            ev = Evidence(f"log: {date_cell}", result)
            prev = latest.get(rid)
            if prev is None or (when, row) >= (prev[0], prev[1]):
                latest[rid] = (when, row, ev)
    for rid, (_, _, ev) in latest.items():
        rep.evidence.setdefault(rid, []).append(ev)


# --- TRC-05, TRC-06: evaluation ------------------------------------------------------


def evaluate(rep: Report) -> None:
    for rid, req in rep.requirements.items():
        if req.withdrawn:
            continue
        evs = rep.evidence.get(rid, [])
        failing = [e for e in evs if e.result == FAIL]
        passing = [e for e in evs if e.result == PASS]
        for e in failing:
            rep.errors.append(f"{rid}: failing evidence {e.source}")
        if rid not in rep.not_implemented and not passing:
            rep.errors.append(f"{rid}: implemented but no passing evidence")
        if rid in rep.not_implemented and passing:
            rep.warnings.append(f"{rid}: has passing evidence but is still 'not yet implemented'")
        stages = rep.stages.get(rid, set())
        if not stages:
            rep.warnings.append(f"{rid}: planned in no stage")
        elif len(stages) > 1:
            rep.warnings.append(f"{rid}: planned in several stages ({', '.join(sorted(stages))})")


def status_of(rep: Report, rid: str) -> tuple[str, str]:
    """(status, result) of a requirement for the table."""
    req = rep.requirements[rid]
    if req.withdrawn:
        return "withdrawn", "–"
    evs = rep.evidence.get(rid, [])
    if any(e.result == FAIL for e in evs):
        result = "FAIL"
    elif any(e.result == PASS for e in evs):
        result = "pass"
    else:
        result = "no evidence"
    status = "not yet implemented" if rid in rep.not_implemented else "implemented"
    return status, result


# --- TRC-07: output --------------------------------------------------------------


def render(rep: Report, version: str, commit: str) -> str:
    active = [r for r in rep.requirements.values() if not r.withdrawn]
    implemented = [r for r in active if r.id not in rep.not_implemented]
    with_pass = [r for r in active if status_of(rep, r.id)[1] == "pass"]
    out = ["# Requirement report", ""]
    if version or commit:
        out += [f"Version `{version or '–'}`, commit `{commit or '–'}`.", ""]
    out += [
        f"**Result: {'FAILED' if rep.errors else 'OK'}** "
        f"({len(rep.errors)} errors, {len(rep.warnings)} warnings)",
        "",
        "## Totals",
        "",
        "| Requirements | Count |",
        "|---|---|",
        f"| Active | {len(active)} |",
        f"| Withdrawn | {len(rep.requirements) - len(active)} |",
        f"| Implemented | {len(implemented)} |",
        f"| Not yet implemented | {len(active) - len(implemented)} |",
        f"| With passing evidence | {len(with_pass)} |",
        "",
        "## Per stage",
        "",
        "| Stage | Planned | Implemented | With passing evidence |",
        "|---|---|---|---|",
    ]
    stage_names = sorted({s for ss in rep.stages.values() for s in ss}, key=int)
    for stage in stage_names:
        planned = [r for r in active if stage in rep.stages.get(r.id, set())]
        impl = [r for r in planned if r.id not in rep.not_implemented]
        ok = [r for r in planned if status_of(rep, r.id)[1] == "pass"]
        out.append(f"| {stage} | {len(planned)} | {len(impl)} | {len(ok)} |")
    unplanned = [r for r in active if not rep.stages.get(r.id)]
    if unplanned:
        out.append(f"| none | {len(unplanned)} | – | – |")
    out += ["", "## Errors", ""]
    out += [f"- {e}" for e in rep.errors] or ["None."]
    out += ["", "## Warnings", ""]
    out += [f"- {w}" for w in rep.warnings] or ["None."]
    out += [
        "",
        "## Requirements",
        "",
        "| ID | Prio | Stage | Status | Evidence | Result |",
        "|---|---|---|---|---|---|",
    ]
    for rid, req in rep.requirements.items():
        status, result = status_of(rep, rid)
        evs = rep.evidence.get(rid, [])
        ev_text = "<br>".join(f"`{e.source}`: {e.result}" for e in evs) or "–"
        stages = ", ".join(sorted(rep.stages.get(rid, set()), key=int)) or "–"
        out.append(f"| {rid} | {req.prio} | {stages} | {status} | {ev_text} | {result} |")
    return "\n".join(out) + "\n"


def build(root: Path) -> Report:
    rep = Report()
    spec = root / "spec"
    read_requirements(spec / "requirements.md", rep)
    read_not_implemented(spec / "verification.md", rep)
    read_stages(spec / "implementation-plan.md", rep)
    reports = root / "reports"
    if reports.is_dir():
        read_results(reports, rep)
    read_log(spec / "verification-log.md", rep)
    evaluate(rep)
    return rep


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--root", type=Path, default=Path.cwd(), help="repository root")
    args = parser.parse_args(argv)
    rep = build(args.root)
    out_dir = args.root / "reports"
    out_dir.mkdir(exist_ok=True)
    text = render(rep, os.environ.get("VERSION", ""), os.environ.get("COMMIT", ""))
    (out_dir / "requirements.md").write_text(text, encoding="utf-8")
    for e in rep.errors:
        print(f"error: {e}", file=sys.stderr)
    for w in rep.warnings:
        print(f"warning: {w}", file=sys.stderr)
    print(f"requirement report: {len(rep.errors)} errors, {len(rep.warnings)} warnings")
    return 1 if rep.errors else 0


if __name__ == "__main__":
    sys.exit(main())

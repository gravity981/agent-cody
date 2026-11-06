# Cody – Verification

How every requirement in [requirements.md](requirements.md) is shown to be met, and how the evidence is traced back to it.

## 1. Principles

- Every active requirement is verified by at least one piece of evidence, and every piece of evidence names the requirement IDs it verifies.
- Automate what can be automated and run it in CI. What can't (real hardware, physical inspection, real environments) is verified manually and recorded in [verification-log.md](verification-log.md).
- A requirement counts as implemented only when it is off the "not yet implemented" list (section 5) and has passing evidence.

## 2. Verification methods and levels

| Method | Meaning | Evidence |
|---|---|---|
| Test | Executed automatically, result pass/fail | Test results read by the requirement report |
| Analysis | Calculation, simulation, static check | Automated where possible (a check that passes/fails), otherwise a log record |
| Inspection | Review of a file, drawing, part or document | Log record |
| Demonstration | Shown working in a real environment | Log record |

| Level | Covers | Tool | Result format |
|---|---|---|---|
| Unit (firmware) | Parser, state machine, timeouts, LED/sound sequencing, without hardware | PlatformIO `native` environment, Unity | JUnit XML in `reports/` |
| Unit / integration (bridge, hooks) | Session merging, timeouts, reconnect, config, heartbeat, listen addresses; mapping of recorded hook payloads | pytest with a fake serial device | JUnit XML in `reports/` |
| Model checks (CAD) | Height, fit on the MK3S+ bed, part count, files present | pytest over the build123d models | JUnit XML in `reports/` |
| Requirement report | TRC | pytest | JUnit XML in `reports/` |
| Bench and real use | Visible/audible behaviour, real agents, current, endurance, look, service | Manual | verification-log.md |

## 3. Requirement traceability

### Tagging evidence

- pytest and Unity test names are identifiers and use `req_ABC_01` in the name: `test_req_BRG_02_most_urgent_wins`, `test_req_FW_10_unknown_command_keeps_state`. Several IDs: `test_req_BRG_03_req_FW_04_…`.
- Manual checks name their IDs in the "Requirements" column of verification-log.md.
- Every ID must exist in requirements.md and must not be withdrawn.
- A tag is malformed if `req_` in a test name is not followed by 2–4 capital letters, `_` and two digits (`req_BRG_02` is valid; `req_brg_2` and `req_BRG_2` are errors).
- Results are read from the JUnit XML files in `reports/`. A failed or erroring test is failing evidence; a skipped test counts as no evidence (neither passing nor failing).
- For manual records, the latest record of a requirement is the one with the newest date; on the same date, the row further down the table wins.

### Requirement report

Built in stage 1 in the project's own technology (requirements TRC-01 – TRC-07), run as the last step of `ci/check`, output in `reports/requirements.md`. It reads:

1. the requirement tables in [requirements.md](requirements.md),
2. the "not yet implemented" list in section 5 of this document,
3. the stage plan (`Requirements:` lines per stage) in [implementation-plan.md](implementation-plan.md),
4. the results of all automated verification in `reports/`,
5. the manual records in [verification-log.md](verification-log.md),

and fails if an implemented requirement has no passing evidence, if any evidence fails, or if a tag is malformed, unknown or withdrawn.

## 4. Verification areas per topic

| Topic | Method and level | Notes |
|---|---|---|
| TRC | Test (the report's own tests) | |
| REL | REL-01: green CI run on a pull request (log). REL-02: tests of the version determination, the bridge `--version` and the version and commit in the firmware image. REL-03 – REL-06: first candidate, promotion, tag ruleset and delivery done by the owner and recorded in the log | REL-03 – REL-06 stay on the "not yet implemented" list until their log rows exist. |
| FW | Test (unit, host); demonstration on the bench (log) | Visible/audible behaviour (FW-01, FW-04, FW-06, FW-07) also demonstrated on hardware. |
| BRG | Test (pytest, fake serial); demonstration of autostart and replug (log) | BRG-06 replug and BRG-09 autostart need the real container and device. |
| AGT | Test (recorded payloads); demonstration with real Claude Code and Copilot (log) | AGT-02 needs Linux, WSL2 and devcontainer runs. |
| ELE | Inspection (BOM, wiring, modules); measurement (USB meter); demonstration (8 h run) | All manual. |
| FIG | Analysis (CI model checks: FIG-02 – FIG-04, FIG-17); inspection and demonstration (log) | Look, travel, noise, stability, endurance and service are judged on the printed assembly. |

## 5. Not yet implemented

Requirements that are not implemented yet. The requirement report doesn't expect evidence for them. Remove IDs as they are delivered; withdrawn requirements are never listed.

| Prefix | IDs |
|---|---|
| FW | FW-01 – FW-12 |
| BRG | BRG-01 – BRG-12 |
| AGT | AGT-01 – AGT-09 |
| ELE | ELE-01 – ELE-07 |
| FIG | FIG-01 – FIG-17 |

## 6. Manual verification

Requirements that need real hardware, real environments or human judgement are recorded in [verification-log.md](verification-log.md): one row per check with requirement IDs, date, verifier, environment and result. For each requirement only the latest record counts; older rows stay as history.

Verified manually because they need real hardware, real agents or the owner's judgement:

- FW-01, FW-04, FW-06, FW-07: behaviour seen and heard on the device (also unit-tested).
- BRG-06, BRG-09: replug and autostart with the real container and device.
- AGT-01 – AGT-04: real agent sessions in each environment; AGT-09: setup guide followed once.
- ELE-01 – ELE-07: modules, soldering, current, stability, cost, documents.
- FIG-01, FIG-05 – FIG-16: printed assembly, motion, endurance, noise, service.

## 7. Release checklist

- [ ] CI green on the candidate's commit, requirement report OK.
- [ ] Manual checks for this release recorded in verification-log.md.
- [ ] Candidate verified as described in delivery.md.
- [ ] Firmware from the candidate flashed; bridge image of the candidate pulled and running; one real Claude Code session drives the device.

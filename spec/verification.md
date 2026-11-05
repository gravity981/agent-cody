# <Project> – Verification

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

<!-- TODO(blueprint): the project's levels and tools, e.g.

| Level | Covers | Tool | Result format |
|---|---|---|---|
| Unit | Single functions/modules | … | JUnit XML in reports/ |
| Integration | Components together | … | … |
| System / hardware-in-the-loop | Whole product | … | … |
| Model checks | Geometry, tolerances, DRC/ERC, clearances | … | … |
-->

## 3. Requirement traceability

### Tagging evidence

<!-- TODO(blueprint): fix the tag convention per tool, with one example each. Defaults: -->

- Where test names allow it, the name starts with the IDs in square brackets: `[ABC-01] rejects an empty input`, `[ABC-01][ABC-02] …`.
- Where test names must be identifiers, use `req_ABC_01` in the name: `test_req_ABC_01_rejects_empty_input`.
- Where the tool supports metadata, a property `requirements` with the IDs is also accepted.
- Every ID must exist in requirements.md and must not be withdrawn.

### Requirement report

Built in stage 1 in the project's own technology (requirements TRC-01 – TRC-07), run as the last step of `ci/check`, output in `reports/requirements.md`. It reads:

1. the requirement tables in [requirements.md](requirements.md),
2. the "not yet implemented" list in section 5 of this document,
3. the stage plan (`Requirements:` lines per stage) in [implementation-plan.md](implementation-plan.md),
4. the results of all automated verification in `reports/`,
5. the manual records in [verification-log.md](verification-log.md),

and fails if an implemented requirement has no passing evidence, if any evidence fails, or if a tag is malformed, unknown or withdrawn.

## 4. Verification areas per topic

<!-- TODO(blueprint): per topic, what is verified how. -->

| Topic | Method and level | Notes |
|---|---|---|
| TRC | Test (the report's own tests) | |
| REL | Test where scriptable; first real candidate and promotion recorded in the log | |

## 5. Not yet implemented

Requirements that are not implemented yet. The requirement report doesn't expect evidence for them. Remove IDs as they are delivered; withdrawn requirements are never listed.

| Prefix | IDs |
|---|---|
| TRC | TRC-01 – TRC-07 |
| REL | REL-01 – REL-06 |

## 6. Manual verification

Requirements that need real hardware, real environments or human judgement are recorded in [verification-log.md](verification-log.md): one row per check with requirement IDs, date, verifier, environment and result. For each requirement only the latest record counts; older rows stay as history.

<!-- TODO(blueprint): list which requirements are verified manually, and why they can't be automated. -->

## 7. Release checklist

<!-- TODO(blueprint): what is checked before a candidate is promoted. -->

- [ ] CI green on the candidate's commit, requirement report OK.
- [ ] Manual checks for this release recorded in verification-log.md.
- [ ] Candidate verified as described in delivery.md.

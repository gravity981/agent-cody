# <Project> – Specification

<!-- TODO(blueprint): one paragraph on what the project is, for whom, and what it delivers. -->

**Status: draft.** Implementation follows [implementation-plan.md](implementation-plan.md) once the spec is approved.

This folder is the **source of truth**. The project's files (code, models, drawings, configuration) implement the spec. When they and the spec disagree, either the files get fixed or the spec gets changed explicitly. Silent deviations are not allowed.

## Document index

| Document | Content |
|---|---|
| [README.md](README.md) | Purpose, document index, conventions, topic status, decisions, open points |
| [requirements.md](requirements.md) | All requirement tables, one section per topic |
| [architecture.md](architecture.md) | Building blocks, interfaces, technology stack with decision records (TD) |
| [delivery.md](delivery.md) | Version scheme, CI environment and entry points, release candidates and promotion, delivery to users |
| [verification.md](verification.md) | Verification methods, tagging of evidence, requirement report, "not yet implemented" list |
| [verification-log.md](verification-log.md) | Records of manual verification (inspections, bench tests, real environments) |
| [implementation-plan.md](implementation-plan.md) | Stages and steps, preconditions, milestones, status reports |

## Conventions

### Requirement IDs

Requirements live in tables in [requirements.md](requirements.md), one section per topic:

```
| ID | Requirement | Prio |
|---|---|---|
| ABC-01 | … | Must |
```

- An ID is `<PREFIX>-<nn>`. The prefix has 2–4 capital letters, one per topic.
- A table has exactly these three columns; the requirement report reads them.
- IDs are never renumbered or reused. New requirements are added at the end of their topic.
- A dropped requirement stays in the table, struck through, marked `(withdrawn)` with the reason, and gets `–` as priority, so its ID is never reused.
- A requirement states *what* is needed and is verifiable. *How* it is achieved belongs in [architecture.md](architecture.md).

### Priorities

| Prio | Meaning |
|---|---|
| Must | Required for the first release that the topic belongs to. |
| Should | Expected. Can be deferred with an explicit decision. |
| Could | Optional and nice to have. |

### Decisions

Every gap or ambiguity that was decided is recorded: in the requirement or section it belongs to, and with a line in the table below. Decisions are never made silently.

## Topics and status

Status values: **specified** (requirements written, not implemented) · **in progress** · **implemented** (all active requirements have passing evidence).

| Prefix | Topic | Section in requirements.md | Reqs (active) | Stage | Status |
|---|---|---|---|---|---|
| TRC | Requirement traceability | 1 | 7 | 1 | specified |
| REL | CI, versions and releases | 2 | 6 | 1 | specified |
<!-- TODO(blueprint): one row per project topic -->

## Decisions

| Date | Decision | Recorded in |
|---|---|---|
<!-- | 2026-01-31 | Example: chose X over Y because … | architecture.md 3.2 | -->

## Open points

<!-- Questions that are not decided yet, each with options and a recommendation. Move them to "Decisions" once settled. -->

- None yet.

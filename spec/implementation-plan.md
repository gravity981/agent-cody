# Cody – Implementation Plan

The spec is implemented in stages, each split into steps. This plan lists what must be decided or provided before each stage and collects a status report after every completed stage.

## 1. How we work

- One stage at a time, on a branch `feature/stage-<n>` (or `feature/stage-<n>-<text>`) that the project owner creates. Claude never commits and never creates or switches branches.
- **Before a stage starts:** its preconditions are settled, and its steps are confirmed or adjusted with the project owner.
- **Before each step:** the branch matches the stage and the working tree is clean (the previous step is committed).
- **Each step** ends with a short summary: what changed, verification and its results, anything open. The project owner reviews and commits before the next step starts.
- Every step adds evidence tagged with the requirement IDs it implements and removes those IDs from the "not yet implemented" list in [verification.md](verification.md). The requirement report must pass.
- **Each completed stage** ends with a status report (section 4), appended to section 5 and shown in the conversation.
- If a step shows that the spec is wrong or incomplete, Claude proposes a spec change before deviating.

## 2. Milestones

<!-- TODO(blueprint): what users get, and after which stage. -->

| Milestone | After stage | What users get |
|---|---|---|
| | | |

## 3. Stages and steps

### Overview

| Stage | Content | Milestone | Status |
|---|---|---|---|
| 1 | Walking skeleton: toolchain, CI, requirement report, first release candidate | | next |
<!-- TODO(blueprint): one row per stage; status: open / next / in progress / done -->

### Stage 1 – Walking Skeleton

**Before the stage starts:**
- Spec approved enough to choose the toolchain; key technologies decided (architecture.md 5).
- CI environment decided (delivery.md 2).
- Delivery decided (delivery.md 4) and release files defined.
- GitHub repository with Actions enabled; tag ruleset for `v*` (REL-05) set up by the project owner.

**Steps:**

**1a – Devcontainer and toolchain.** Base image and project tools with pinned versions in `.devcontainer/`; CI environment as decided (delivery.md 2); top-level folders as in architecture.md 3; formatting and static checks for the chosen technology.
Requirements: none (foundation).

**1b – Requirement report and CI.** The requirement report in the project's technology with its own tests; `ci/check` runs all automated verification and then the report; CI green on a pull request.
Requirements: TRC-01 – TRC-07, REL-01.

**1c – Release.** Version determination (`<last release>+build.<N>` in CI, `+local.<commit>` locally) and how artifacts reveal it; `ci/build-release` (and `ci/publish` / `ci/promote` if delivery needs them); first candidate and promotion done by the project owner and recorded in the verification log; delivery as in delivery.md 4.
Requirements: REL-02 – REL-06.

<!-- TODO(blueprint): further stages, same structure:

### Stage <n> – <Title>

**Before the stage starts:**
- decisions, material, access needed

**Steps:**

**<n>a – <Title>.** What is built.
Requirements: ABC-01, ABC-03 – ABC-05.
-->

## 4. Status report template

After each completed stage, Claude appends a report with this structure to section 5 and shows it in the conversation:

```markdown
### Stage <n> – <title> (<date>)

**Summary:** one or two sentences on what users can do now.

**Steps:** <step> – done / changed / moved, one line each.

**Requirements:** <implemented in this stage> of <planned for this stage>, all with passing evidence (requirement report). Overall: <implemented> of <active> requirements.

**Verification:** automated results per level; manual checks recorded.

**Spec changes and deviations:** what changed in the spec during the stage and why.

**Overall status:**

| Stage | Content | Status |
|---|---|---|
| 1 | … | done / next / open |

**Remaining requirements by stage:** stage → number of requirements (from the requirement report).

**Waiting on the project owner:** decisions, material or access needed for the next stage.

**Risks and notes:** anything that could affect the next stages or milestones.
```

## 5. Status reports

No stage completed yet.

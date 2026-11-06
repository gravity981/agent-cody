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

| Milestone | After stage | What users get |
|---|---|---|
| M1 | 2 | Desk status light: LED and sound (and a bare motor) show what Claude Code is doing; documented so others can build it. |
| M2 | 3 | Both Claude Code and GitHub Copilot drive the status light. |
| M3 | 5 | Complete Cody: printed figure whose arms type while the agent works; parts list and assembly instructions for a full rebuild. |

## 3. Stages and steps

### Overview

| Stage | Content | Milestone | Status |
|---|---|---|---|
| 1 | Walking skeleton: toolchain, CI, requirement report, first release candidate | | done |
| 2 | Status light on the bench with Claude Code | M1 | next |
| 3 | Copilot integration | M2 | open |
| 4 | Mechanism prototype | | open |
| 5 | Character and assembly | M3 | open |

### Stage 1 – Walking Skeleton

**Before the stage starts:**
- Spec approved enough to choose the toolchain; key technologies decided (architecture.md 5).
- CI environment decided (delivery.md 2).
- Delivery decided (delivery.md 4) and release files defined.
- GitHub repository with Actions enabled; tag ruleset for `v*` (REL-05) set up by the project owner. *Status at stage start (2026-11-05): no remote yet; the owner sets up the repository, Actions, the ruleset and package permissions before step 1b. Step 1a needs no GitHub.*

**Steps:**

**1a – Devcontainer and toolchain.** Base image and project tools with pinned versions in `.devcontainer/`: PlatformIO with the pioarduino platform, uv with Python, build123d, manifold3d, pytest, ruff, clang-format, shellcheck, and the docker-outside-of-docker feature; Python dependencies locked with uv. Top-level folders as in architecture.md 3; formatting and static checks (ruff, clang-format, shellcheck); `.gitignore`, `.gitattributes`, `.editorconfig` for the toolchain.
Requirements: none (foundation).

**1b – Requirement report and CI.** The requirement report in Python (`tools/`) with its own pytest tests (JUnit XML); `ci/check` runs the static checks, all tests and then the report; CI green on a pull request.
Requirements: TRC-01 – TRC-07, REL-01.

**1c – Release.** Version determination (`<last release>+build.<N>` in CI, `+local.<commit>` locally) and how artifacts reveal it; skeleton firmware (`.bin` that sends `READY <version>` at boot) and skeleton bridge image (answers `--version`) with its compose file, as the first release files; `ci/build-release`, `ci/publish` (bridge image to GHCR with the candidate tag) and `ci/promote` (retag without rebuilding); workflow permissions `packages: write` and action SHA pins; first candidate and promotion done by the project owner and recorded in the verification log; delivery as in delivery.md 4.
Requirements: REL-02 – REL-06.

### Stage 2 – Status light on the bench

**Before the stage starts:**
- Electronics bought: ESP32-C3 SuperMini (two), WS2812 module, passive buzzer module, motor driver breakout, a small N20 geared motor for the bench, cables (models chosen with the owner, within ELE-06).
- On WSL2: usbipd-win installed.

**Steps:**

**2a – Firmware.** Serial protocol, state machine, LED effects, sounds with volume, motor PWM, heartbeat fallback, version; unit tests on the host; bench demonstration.
Requirements: FW-01 – FW-12.

**2b – Bridge.** HTTP endpoints, session merging, timeouts, serial connection with reconnect, heartbeat, config, status and manual state commands; container with autostart and replug proven on the device.
Requirements: BRG-01 – BRG-12.

**2c – Claude Code integration.** User-level hooks that find the bridge on `localhost` or the Docker host, event mapping with recorded payloads, timeout behaviour; real sessions on Linux, WSL2 and in a devcontainer.
Requirements: AGT-01 – AGT-03, AGT-07, AGT-11.

**2d – Bench electronics documented.** Modules, connections and soldering as built; wiring diagram.
Requirements: ELE-01 – ELE-03, ELE-07.

**2e – Build documentation for the status light.** Parts list; assembly instructions for the electronics and the setup (soldering, wiring, flashing, bridge, hooks) with pictures; both as release files.
Requirements: DOC-01, DOC-02, DOC-05, DOC-06.

### Stage 3 – Copilot integration

**Before the stage starts:**
- Current state of VS Code agent hooks checked (Preview; events, config location).

**Steps:**

**3a – Copilot hooks.** User-level hooks, mapping, session key by workspace folder, deduplication against Claude Code hooks; setup guide for both agents.
Requirements: AGT-04 – AGT-06, AGT-08, AGT-09.

**3b – One-command setup.** `cody setup`: starts the bridge with autostart, installs the hooks of both agents at user level, checks the device connection and reports what is missing.
Requirements: AGT-10.

### Stage 4 – Mechanism prototype

**Before the stage starts:**
- Motor model and gear ratio chosen; filament and screws available.

**Steps:**

**4a – Crank and arms.** Code-CAD test rig: motor mount, crankshaft with two pins at 180°, rods, shoulder pivots, stops, test arms; tuned to the travel and stroke rate.
Requirements: FIG-06 – FIG-09, FIG-11.

**4b – Power and endurance.** USB current measured incl. motor start; stability of ESP32 and USB; 8 h run with motor temperature and noise.
Requirements: FIG-12, FIG-13, ELE-04, ELE-05.

**4c – Mechanism instructions.** Assembly instructions for motor, crank, rods and arms; parts list updated.
Requirements: DOC-03.

### Stage 5 – Character and assembly

**Before the stage starts:**
- Meshy.ai licence terms checked (README.md "Open points"); head generated.
- A test builder for DOC-07 found.

**Steps:**

**5a – Head pipeline.** Meshy head committed with its prompt; manifold3d repair, scaling, socket; printable STL.
Requirements: none (part of FIG-01, verified in 5b).

**5b – Figure and scene.** Torso with hidden mechanism, table, laptop, platform with electronics compartment; model checks in CI; print notes.
Requirements: FIG-01 – FIG-05, FIG-10, FIG-17.

**5c – Assembly and service.** Full assembly; service checks, stability, no glue; final parts cost (from the parts list).
Requirements: FIG-14 – FIG-16, ELE-06.

**5d – Complete build documentation.** Printing and full assembly instructions; parts list final; a test build by someone other than the owner.
Requirements: DOC-04, DOC-07.

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

### Stage 1 – Walking Skeleton (2026-11-06)

**Summary:** The project has its toolchain, CI and release path: every pull request and push to `main` runs all checks and the requirement report in the devcontainer, and release `v0.1.0` (skeleton firmware, skeleton bridge image on GHCR, compose file) was built, promoted and downloaded.

**Steps:**
- 1a – done: devcontainer with pinned toolchain (PlatformIO/pioarduino for the ESP32-C3, uv with Python 3.14, build123d, manifold3d, ruff, clang-format, shellcheck), top-level folders, static checks.
- 1b – done, extended: requirement report with its tests, `ci/check`, CI green on PR #1. Extended by fixes found in CI: post-create step for the runner's UID, devcontainer image slimmed (PlatformIO 8.1 → 2.9 GB), devcontainer image cached on GHCR.
- 1c – done: version determination, skeleton firmware and bridge, `ci/build-release`, `ci/publish`, `ci/promote`; first candidate and promotion by the owner. A bug in the image existence check (images overwritten) was found in local tests and fixed before the first release.

**Requirements:** 13 of 13 planned for this stage implemented, all with passing evidence (requirement report). Overall: 13 of 79 active requirements.

**Verification:** automated: 29 tests passing (requirement report 23, version and release 6), static checks, firmware build; manual: 5 records (REL-01, REL-03 – REL-06).

**Spec changes and deviations:**
- Plug & play for the device connection: BRG-06 reworded, BRG-07 port setting optional.
- REL-05 reworded: the tag ruleset can't restrict creation, because GitHub can't put the GitHub Actions actor on a bypass list.
- New topic DOC (parts list, assembly instructions, 7 requirements): others can rebuild Cody; ELE-07 reduced to the wiring diagram.
- New AGT-10 (one-command setup) and AGT-11 (same hooks on host and in devcontainers) to reduce setup effort; the bridge stays.
- Spec review findings recorded as risks R1 – R8 (architecture.md 7.1); R1 addressed by AGT-11.
- `spec/idea.md` removed (conflicted with the spec); "Decisions" limited to decisions that change requirements or architecture.

**Overall status:**

| Stage | Content | Status |
|---|---|---|
| 1 | Walking skeleton | done |
| 2 | Status light on the bench with Claude Code | next |
| 3 | Copilot integration | open |
| 4 | Mechanism prototype | open |
| 5 | Character and assembly | open |

**Remaining requirements by stage:** 2 → 37, 3 → 6, 4 → 10, 5 → 13.

**Waiting on the project owner:**
- Branch `feature/stage-2`, then `/stage-start 2`.
- Parts delivered (ordered 2026-11-06: ESP32-C3 ×2, DRV8833 ×2, N20 6 V 300 and 500 rpm, WS2812, KY-006); usbipd-win installed on Windows.
- Decide at stage start: how the CHF 25 budget is counted (risk R6); release format of the build documentation (open point).
- Open the project once with "Reopen in Container" in VS Code (needed for flashing).

**Risks and notes:**
- Plug & play of the serial device inside the bridge container (TD-04) is unproven; step 2b must show it, otherwise a native service.
- CI runs that rebuild the toolchain layers (first run, toolchain changes in a pull request) take long; the ownership step in the post-create adds a few minutes to every run.
- The KY-006 may be a coil buzzer: measure before wiring; then it is driven through the DRV8833's second channel.
- Product photos in the parts list are the shops' images; replace them with own photos before a release.
- GitHub Copilot's agent hooks are still in Preview (stage 3).

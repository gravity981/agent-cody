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
| M1 | 2 | Desk status light: LED and sound (and a bare motor) show what Claude Code is doing; volume and brightness are configurable, motor tuning is saved per device, and the build is documented. |
| M2 | 3 | Both Claude Code and GitHub Copilot drive the status light. |
| M3 | 5 | Complete Cody: printed figure whose arms type while the agent works; parts list and assembly instructions for a full rebuild. |

## 3. Stages and steps

### Overview

| Stage | Content | Milestone | Status |
|---|---|---|---|
| 1 | Walking skeleton: toolchain, CI, requirement report, first release candidate | | done |
| 2 | Status light on the bench with Claude Code | M1 | done (open points) |
| 3 | Copilot integration | M2 | done (open points) |
| 4 | Mechanism prototype | | next |
| 5 | Character and assembly | M3 | open |
| 6 | Native bridge for Linux and macOS, firmware updater | | open |

### Stage 1 – Walking Skeleton

**Before the stage starts:**
- Spec approved enough to choose the toolchain; key technologies decided (architecture.md 5).
- CI environment decided (delivery.md 2).
- Delivery decided (delivery.md 4) and release files defined.
- GitHub repository with Actions enabled; tag ruleset for `v*` (REL-05) set up by the project owner. *Status at stage start (2026-11-05): no remote yet; the owner sets up the repository, Actions, the ruleset and package permissions before step 1b. Step 1a needs no GitHub.*

**Steps:**

**1a – Devcontainer and toolchain — done.** Base image and project tools with pinned versions in `.devcontainer/`: PlatformIO with the pioarduino platform, uv with Python, build123d, manifold3d, pytest, ruff, clang-format, shellcheck, and the docker-outside-of-docker feature; Python dependencies locked with uv. Top-level folders as in architecture.md 3; formatting and static checks (ruff, clang-format, shellcheck); `.gitignore`, `.gitattributes`, `.editorconfig` for the toolchain.
Requirements: none (foundation).

**1b – Requirement report and CI — done.** The requirement report in Python (`tools/`) with its own pytest tests (JUnit XML); `ci/check` runs the static checks, all tests and then the report; CI green on a pull request.
Requirements: TRC-01 – TRC-07, REL-01.

**1c – Release — done.** Version determination (`<last release>+build.<N>` in CI, `+local.<commit>` locally) and how artifacts reveal it; skeleton firmware (`.bin` that sends `READY <version>` at boot) and skeleton bridge image (answers `--version`) with its compose file, as the first release files; `ci/build-release`, `ci/publish` (bridge image to GHCR with the candidate tag) and `ci/promote` (retag without rebuilding); workflow permissions `packages: write` and action SHA pins; first candidate and promotion done by the project owner and recorded in the verification log; delivery as in delivery.md 4.
Requirements: REL-02 – REL-06.

### Stage 2 – Status light on the bench

**Before the stage starts:**
- Electronics bought: ESP32-C3 SuperMini (two), WS2812 module, passive buzzer module, motor driver breakout, a small N20 geared motor for the bench, cables (models chosen with the owner, within ELE-06).
- *Status at stage start (2026-11-06): parts ordered (docs/parts-list.md), not delivered yet; cables, breadboard and multimeter to be provided by the owner; usbipd-win to be installed before the bench checks. Steps 2a – 2c start without hardware; the bench work is step 2d.*
- *Status after step 2c (2026-11-06): parts not delivered yet, usbipd-win not set up; step 2d waits for both. Decided for 2d: the devcontainer gets `/dev` for initial bench flashing (architecture.md 4.2), the wiring diagram is an SVG block diagram with named pins and nets, generated with schemdraw (architecture.md 3, 5.1), the stall current for ELE-03 is measured on the bench (verification.md 4).*
- *Redesign of the bridge (2026-11-21, TD-06): the Windows host uses the Windows app for serial access; usbipd-win is not needed for that workflow. Steps 2g – 2i were added after 2d. Local image flashing from the test panel, with no devcontainer USB access, was decided on 2026-11-22 (TD-11).*

**Steps:**

**Owner decision (2026-11-22):** Stage 2 is marked done with open points. All remaining manual verification is deferred, not passed; see the Stage 2 report in section 5.

Steps 2a – 2c need no hardware; step 2d starts when the parts have arrived. Order after 2d (decided 2026-11-21, TD-06, TD-08, TD-09; updated 2026-11-22, TD-11, TD-13): 2g, 2h, 2i, 2j, 2k, 2n, 2o, 2m, 2p, then 2e. Step 2l was moved to Stage 6 on 2026-11-22; Stage 2 adds local image flashing, while Stage 6 retains the GitHub release updater.

**2a – Firmware logic — done.** Serial protocol, state machine, LED effects, sounds with volume and motor control as logic with host unit tests (defaults in architecture.md 4.3); firmware builds for the ESP32-C3.
Requirements: FW-02, FW-03, FW-05, FW-10 – FW-12.

**2b – Bridge — done.** HTTP endpoints, session merging, timeouts, heartbeat, serial connection tested with a fake serial device, config file, autostart, log with rotation; host command `cody` (status, manual state).
Requirements: BRG-01 – BRG-06, BRG-08 – BRG-14.

**2c – Claude Code integration — done.** User-level curl command hooks that find the bridge on the host and in containers, event mapping with recorded payloads, timeout behaviour; real sessions on WSL2 and in a devcontainer (the bridge without a device is enough).
Requirements: AGT-01 – AGT-03, AGT-07, AGT-11.

**2d – Bench and initial hardware test panel.** Parts arrived: firmware flashed, modules wired and soldered; LED effects, sounds and the disconnected pattern shown and tuned on the device; wiring diagram. Includes the permanent, all-peripheral firmware test mode and the initial devcontainer browser panel for state presets and independent motor/LED/buzzer testing. The initial panel owns USB serial directly; step 2k replaces it with the bridge-backed website panel. Motor tuning can be changed in RAM; raw LED and sound tests are temporary. (Plug & play in the bridge container dropped on 2026-11-21: BRG-06 is shown with the Windows app in 2h.)
Requirements: FW-01, FW-04, FW-06, FW-07, ELE-01 – ELE-03, ELE-07, TST-01 – TST-04.

**2f – Clearing sessions — done.** Added during the stage (2026-11-06), done before 2d: `cody session clear` and `DELETE /sessions` remove all sessions (architecture.md 4.1).
Requirements: BRG-15.

**2g – Go bridge core — done.** Added 2026-11-21 (TD-06). Go toolchain and gotestsum in the devcontainer; gofmt and go vet in `tools/static-checks`; Go tests in `ci/check`. In `bridge-go/`: the bridge logic ported from Python (HTTP endpoints, session merging, Claude Code mapping tested with the shared recorded payloads, timeouts, heartbeat, serial connection with a fake device, config, log with rotation, version); builds for Windows and Linux without cgo, including the tray library. The Python bridge stays as is (frozen).
Requirements: BRG-01 – BRG-05, BRG-10, BRG-13 – BRG-15.

**2h – Windows app and installer — implementation done; manual verification pending.** Tray icon in the state colour with menu, "Start automatically" (Run key), single instance, COM port discovery and plug & play (first plug-in, replug and reset), config and log under `%APPDATA%`/`%LOCALAPPDATA%`; NSIS installer `cody-<version>-windows-setup.exe` as a release file; shown on Windows with the device. Autostart is enabled by the installer by default. The 2h installer contains `cody.exe`; step 2i adds `cody-linux` when the WSL relay is implemented, and step 2m adds the pinned firmware-flashing helper and its license notices.
*Implementation status (2026-11-21): Linux CI, Windows cross-build and installer compilation pass; demonstration of the tray app and install/uninstall on Windows with the device remains.*
Requirements: BRG-06, BRG-09, BRG-16 – BRG-18.

**2i – WSL relay and reachability — implementation done; manual verification pending.** `cody-linux relay` started by the app through `wsl.exe`, tunnel over stdin/stdout; real Claude Code sessions drive the device from the Windows host (Git Bash), WSL2 in NAT and mirrored mode, a devcontainer in Docker Engine in WSL2 and a devcontainer on Docker Desktop; `cody` command from WSL; setup guide updated.
*Implementation status (2026-11-21): CI, Windows/Linux builds and installer compilation pass; manual Claude Code sessions across Windows, WSL2 and both devcontainer environments remain.*
Requirements: AGT-02, AGT-12, BRG-08.

**2j – Local web interface — implementation done; manual verification pending.** Serve the status/configuration website from the Windows Go bridge, embedded in the binary; add the tray action that opens it in the default browser. Show device, session and relay information and the correct hook-routing hints; edit and save the selected settings, clearly state that restart is required, and offer Restart now. Keep the site loopback-only and protect browser mutations with same-origin checks.
*Implementation status (2026-11-21): automated `ci/check` passes; the user confirmed the tray launch, hook/relay guidance and session timeout input. Manual Windows verification of saving settings and applying them through Restart now remains.*
Requirements: BRG-20 – BRG-21.

**2k – Bridge-backed hardware test panel — implementation done; manual verification pending.** Replace the devcontainer's direct-serial panel with the test section of the Windows bridge website. The bridge remains the sole serial owner, suppresses normal device writes during testing, continues merging agent events, and restores the latest merged state on stop or after the 10 s panel-heartbeat timeout. Remove the old panel's serial ownership and direct-serial tests.
*Implementation status (2026-11-22): bridge-owned test commands, reply display, heartbeat timeout and state restoration are covered by Go tests; the Python panel, direct-serial tests and forwarded devcontainer port are removed. Manual Windows/bench verification of connection status, panel controls and replies/errors; agent-event merging during testing; and state restoration on panel close or heartbeat expiry remains.*
Requirements: TST-06, TST-08.

**2n – Device tuning and output settings — implementation done; bench verification pending.** Add per-device motor tuning in ESP32 NVS, including explicit save, bridge-supplied UTC timestamps, firmware defaults and factory reset. Add percent-based volume and brightness commands; volume maps slider value `x` with `V=(x/100)^2.5`, and brightness uses `b=(x/100)^2.2` on channels after the existing LED cap. The frozen Python bridge is not made compatible with this firmware change.
Requirements: FW-08 – FW-09, FW-13 – FW-15.
Implementation status: Firmware protocol, NVS adapter, tuning save/load/reset and output curves are implemented; host tests and the ESP32-C3 build pass. Bench verification of volume/brightness and NVS persistence/reset remains.

**2o – Tuning panel and user settings — implementation done; manual verification pending.** Extend the Windows Go bridge config and website with 1%-step volume and brightness sliders, and the bridge-backed panel with tuning readout, last-write time, explicit Save and Factory reset. Display local time and identify unsaved firmware defaults. Volume defaults to 50%; brightness defaults to 100%. Clearly identify the frozen Python Docker bridge as incompatible with the new firmware.
Requirements: BRG-22, TST-05, TST-07, TST-09.
Implementation status: Go tests, firmware native tests and the ESP32-C3 build pass. Manual Windows/bench verification of sliders, preference restoration, tuning display and timestamps, save/reset, and temporary test volume remains.

**2m – Local firmware flashing — implementation done; manual verification pending.** Provide Manual ROM bootloader flash for first-time or existing ESP32-C3 devices, with explicit Espressif port selection, manual ROM entry, and a warning that motor tuning is not backed up and may be lost. Keep Update connected Cody device for recognized firmware that supports `TUNING?`; snapshot and restore tuning and timestamp, leaving `UNSAVED` unchanged. Accept local non-empty `.bin` images up to 4 MiB and explain that the user must choose a combined Cody image. Confirm in the browser with the selected filename, target port and firmware-replacement warning. Bundle the pinned `espflash` helper and its license notices in the Windows installer. Stop active test mode, suspend device writes, release the COM port, flash at `0x0`, report the current phase and captured output/errors, reacquire the port and leave test mode while restoring current preferences and merged state on success or failure when the device reconnects. Report disconnection and allow retry. No USB device access or bind mount in the devcontainer.
Implementation status: `ci/check` passes, including bridge API/lifecycle tests, Windows app build and installer packaging. Manual bootloader flashing of a first-time or legacy device, tuning-preserving update, and recovery still need demonstration on the Windows host and device.
Requirements: TST-10.

**2p – Temporary LED test brightness — implementation done; manual verification pending.** Add a 1%-step brightness slider and Apply button to the bridge-backed LED panel. Applying brightness sends the entered raw RGB colour first, then applies the temporary test brightness with the existing gamma curve and 50% cap. Initialize the slider from the saved bridge preference without changing that preference; clear the override when test mode ends.
Implementation status: Firmware behavior and bridge command mapping are covered by requirement-tagged tests; manual Windows/bench verification of the slider, command order, brightness levels, and preference restoration remains.
Requirements: TST-11.

**2e – Build documentation for the status light — implementation done; manual verification pending.** Done after 2p. Stage 2 electronics parts-list draft; assembly instructions for soldering, wiring, flashing a combined firmware image, installing the Windows bridge or compatible legacy Docker bridge, installing Claude Code hooks, and using the hardware test panel and settings. Each step names parts and tools and has an illustration. `ci/build-release` emits a versioned CSV and PDF from the Markdown sources.
Implementation status: Documentation exports and their release filenames are covered by CI; the owner still needs to inspect the instructions and follow them on the Windows host and bench. DOC-01's complete parts list moves to step 5d because printed parts and fasteners are not yet defined.
Requirements: DOC-02, DOC-05, DOC-06.

### Stage 3 – Copilot integration

**Before the stage starts:**
- Checked 2026-11-22 against the [VS Code agent hooks documentation](https://code.visualstudio.com/docs/agent-customization/hooks), the [Copilot hooks reference](https://docs.github.com/en/copilot/reference/hooks-reference), and the user's VS Code 1.147.0 hook picker. User hooks are in `~/.copilot/hooks/` or `$COPILOT_HOME/hooks/`; Copilot also reads repository `.claude/settings.json` and `.claude/settings.local.json`, but does not list Claude Code's user-level `~/.claude/settings.json`. The picker exposes `ErrorOccurred` but no approval/permission-request event: step 3b maps ERROR and leaves APPROVAL conditional. Payloads include a session ID, but AGT-06 continues to key by workspace folder (`cwd`).

**Steps:**

**3a – Retire the Python Docker bridge — done.** Removed the `bridge/` Python package and tests, its uv workspace/pytest configuration, Docker image and compose asset generation, bridge-only helpers in `ci/lib.sh`, and `ci/publish`/`ci/promote`; regenerated `uv.lock`. Updated `ci/README.md`, `docs/assembly-instructions.md`, and `delivery.md`; marked TD-03 and TD-04 retired. Native Linux bridge support will not return until step 6a (TD-15).
Requirements: none (retirement of the Stage 2 BRG-07 implementation).

**3b – Copilot hooks — done for available events.** Added user-level hooks, mapping, session key by workspace folder, deduplication against all applicable hook sources, and the setup guide for both agents. The VS Code 1.147.0 picker provides `ErrorOccurred` but no approval event; ERROR is mapped and AGT-05 remains open for APPROVAL.
Requirements: AGT-04 – AGT-06, AGT-08, AGT-09.

**3c – Setup through Cody's web UI — implementation done; manual verification pending.** The Windows installer starts the tray app with autostart. The web UI installs both agents' hooks at user level on Windows and in each WSL distro configured for Cody's relay, checks the device connection and reports what is missing, including guidance for devcontainer routing. Existing hook scripts and manual installation instructions remain available. This devcontainer bind-mounts the WSL user's Copilot configuration; independent devcontainer volumes still need manual installation. Native Linux reports the bridge as unsupported until step 6a (TD-15).
Requirements: AGT-10.

**Owner decision (2026-11-22):** Mark Stage 3 done with AGT-05 open because VS Code does not yet provide an approval event. Keep AGT-05 on the "not yet implemented" list; this is a deferral, not a claim that Copilot drives APPROVAL. The fresh-host AGT-10 demonstration also remains pending and is not recorded as passing evidence.

### Stage 4 – Mechanism prototype

**Before the stage starts:**
- The 6 V, 500 rpm N20 is selected; filament and screws are available.
- The owner's [chest cam concept](../cad/concept/Cody%20Chest%20Cam%20Drive.html) is reviewed and its dimensions are integrated below.

**Steps:**

**4a – Chest cam and arms.** Code-CAD test rig based on the concept: a 2 mm steel shoulder axle at mid-depth about 10 mm below the shoulder top; action-figure sleeve joints, with the left arm's sleeve extending along the axle so both levers sit side by side; 8 mm levers pointing forward, with hands about 37 mm forward; and stops above both lever tips. Place the 34 mm motor across the chest, with its axis about 8 mm in front of and 6 mm below the axle. Fit one printed double eccentric to the motor's Ø3 × 10 mm D-shaft, with two side-by-side Ø10 × 4 mm lobes offset 0.65 mm and phased 180° apart; arm weight holds each lever tip on its lobe. Check the 62 × 38 mm shoulder envelope (about 58 × 34 mm inside with 2 mm walls), the motor surface about 4 mm from the axle center, about 1.5 mm clearance to the arm sleeves, and its 5 mm recess from the chest surface (2 mm wall plus 3 mm gap). Tune target hand travel to about 6 mm and verify alternating motion, follower contact at up to 4 strokes/s, and 1–4 strokes/s per arm. The owner measured about 0.52 s per revolution (about 1.9 strokes/s per arm); the motor can run faster, so tune to the fastest non-distracting rate at 1 m within FIG-08.
Requirements: FIG-06 – FIG-09, FIG-11.

**4b – Power and endurance.** Measure USB current including motor start; check ESP32 and USB stability; run for 8 h without failure with the motor below 60 °C; assess noise at 1 m.
Requirements: FIG-12, FIG-13, ELE-04, ELE-05.

**4c – Mechanism instructions.** Assembly instructions for motor, double eccentric, shoulder axle, arm sleeves, levers, stops and arms; parts list updated.
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

**5d – Complete build documentation.** Printing and full assembly instructions; complete parts list including printed parts, fasteners and consumables; a test build by someone other than the owner.
Requirements: DOC-01, DOC-04, DOC-07.

### Stage 6 – Native bridge for Linux and macOS, firmware updater

Added 2026-11-21 (TD-06).
Step 6b was added on 2026-11-22 by moving GitHub release checking from Stage 2. Local image flashing is delivered in Stage 2 (TD-11); the release updater remains Windows-first and its setup instructions are added to the build documentation here. The Python Docker bridge is retired in Stage 3, so native Linux bridge support is unavailable between that retirement and step 6a (TD-15).

**Before the stage starts:**
- A Mac (or a macOS CI runner) for building the tray with cgo and for the demonstrations.
- Autostart on Linux (systemd user service or desktop autostart, with or without tray) and on macOS (login item, menu bar app) decided.

**Steps:**

**6a – Linux and macOS builds.** Native builds of the Go bridge with autostart and plug & play; delivered as release files; shown on a Linux host and a Mac.
Requirements: BRG-19.

**6b – GitHub release updater.** Add a user-initiated check of the latest stable Cody release to the Windows bridge website, with explicit confirmation, SHA-256 verification, serial handoff, the pinned espflash helper bundled in step 2m, progress, automatic bootloader entry and manual recovery/retry on failure; reconnect the bridge and report the result. Unknown current firmware can be updated only after explicit confirmation; same/newer versions do not offer an update. Add the updater setup instructions to the build documentation.
Requirements: BRG-23, BRG-24.

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
- New AGT-10 setup flow and AGT-11 (same hooks on host and in devcontainers) to reduce setup effort; AGT-10's flow was revised in Stage 3.
- Spec review findings recorded as risks R1 – R8 (architecture.md 7.1); R1 addressed by AGT-11.
- `spec/idea.md` removed (conflicted with the spec); "Decisions" limited to decisions that change requirements or architecture.

**Overall status:**

| Stage | Content | Status |
|---|---|---|
| 1 | Walking skeleton | done |
| 2 | Status light on the bench with Claude Code | done (open points) |
| 3 | Copilot integration | in progress |
| 4 | Mechanism prototype | open |
| 5 | Character and assembly | open |

**Remaining requirements by stage:** 2 → 38, 3 → 6, 4 → 10, 5 → 13.

**Waiting on the project owner:**
- Parts delivered (ordered 2026-11-06: ESP32-C3 ×2, DRV8833 ×2, N20 6 V 300 and 500 rpm, WS2812, KY-006).
- The Windows host needs the device available for serial and first-flash verification; no devcontainer USB access or bind mount is needed.
- Open the project with "Reopen in Container" in VS Code for the pinned build and verification tools.

**Risks and notes:**
- Plug & play of the serial device inside the bridge container (TD-04) is unproven; step 2b must show it, otherwise a native service.
- CI runs that rebuild the toolchain layers (first run, toolchain changes in a pull request) take long; the ownership step in the post-create adds a few minutes to every run.
- The KY-006 measured 15 Ω across its signal and ground pins on the bench and is driven through the DRV8833's second channel (2026-11-21).
- Product photos in the parts list are the shops' images; replace them with own photos before a release.
- GitHub Copilot's agent hooks are still in Preview (stage 3).

### Stage 2 – Status light on the bench with Claude Code (2026-11-22)

**Summary:** Stage 2's firmware, Windows bridge, test panel, local firmware flashing, LED brightness control and electronics documentation are implemented and merged to `main`. At the project owner's request, manual verification not recorded in the verification log is deferred. Its absence is not a claim that a check was not done, but unrecorded checks are not counted as passing evidence.

**Steps:**
- 2a, 2b, 2c, 2f, 2g – done.
- 2d – implemented; ELE-01 – ELE-03 remain open.
- 2h, 2i, 2j, 2k, 2m, 2n, 2o, 2p, 2e – implementation done; remaining hands-on checks deferred by the owner.
- 2l – moved to Stage 6; local image flashing was added as 2m, while GitHub release checking remains a separate Stage 6 feature.
- Python Docker bridge retirement – moved from Stage 6 step 6b to Stage 3 step 3a; the accepted native-Linux support gap is recorded in TD-15.

**Requirements:** 53 of 60 planned for this stage are implemented; 56 have passing evidence. Seven remain open: FW-15, BRG-18, AGT-02, AGT-12 and ELE-01 – ELE-03. Four have no evidence in the report (BRG-18, ELE-01 – ELE-03); three have passing evidence but remain listed as not implemented because the required manual confirmation is not recorded (FW-15, AGT-02, AGT-12). Overall: 66 of 106 active requirements implemented.

**Verification:** `ci/check` passed with 140 Python tests, 29 firmware native tests, 114 Go tests, static checks and firmware/Go/Windows installer builds. The requirement report has 0 errors and 3 warnings (FW-15, AGT-02, AGT-12). Fifteen manual records were added: FW-01, FW-04, FW-06, FW-07, FW-09, FW-13, FW-14, BRG-06, BRG-16, BRG-17, ELE-07 and TST-01 – TST-04. Unrecorded checks include factory reset, the Windows installer lifecycle, Windows-native Claude Code, WSL2/Docker Desktop routing, electronics inspections/measurements, first-time/update/recovery flashing and remaining website/panel/documentation demonstrations; these are deferred and are not counted as passing evidence.

**Spec changes and deviations:**
- Added decisions TD-10 – TD-14 for tuning/output preferences, local firmware flashing, manual bootloader flashing, temporary LED brightness and versioned documentation exports.
- Moved the GitHub release updater from Stage 2 to Stage 6; moved the complete parts list DOC-01 to Stage 5 step 5d.
- TD-15 moves Python Docker bridge retirement to Stage 3 step 3a. Native Linux bridge support is intentionally absent until Stage 6 step 6a; WSL and devcontainers continue to use the Windows bridge relay.
- Stage 2 is marked done with seven requirements and additional manual evidence still open, at the owner's request. Passing automated evidence for a requirement does not replace the required manual record.

**Overall status:**

| Stage | Content | Status |
|---|---|---|
| 1 | Walking skeleton | done |
| 2 | Status light on the bench with Claude Code | done (open points) |
| 3 | Copilot integration | next |
| 4 | Mechanism prototype | open |
| 5 | Character and assembly | open |
| 6 | Native bridge for Linux and macOS, firmware updater | open |

**Remaining requirements by stage:** 1 → 0, 2 → 7, 3 → 6, 4 → 10, 5 → 14, 6 → 3.

**Waiting on the project owner:**
- Before Stage 3, check the current VS Code agent-hook behavior (Preview, available events and configuration location).
- Stage 3 decisions: retain the `cody` shell command and manual hook scripts, and provide host hook installation through the Windows bridge website (AGT-10); Copilot's hook sources and deduplication are specified under AGT-08.
- Existing open points before later stages remain in `spec/README.md`.

**Risks and notes:**
- Native Linux hosts lose the Python Docker bridge in Stage 3 and have no native replacement until Stage 6 step 6a. The owner accepted this gap; Windows, WSL and devcontainers use the Windows bridge/relay in the interim.
- Deferred verifications remain open in `spec/verification.md`; the latest requirement report has 3 warnings and no errors.

### Stage 3 – Copilot integration (2026-11-22)

**Summary:** Claude Code and GitHub Copilot hooks now drive Cody through the Windows bridge. The Windows web UI installs hooks on Windows and in each configured WSL relay distro; the repository devcontainer shares the WSL Copilot configuration. Manual scripts and the `cody` shell command remain available.

**Steps:**
- 3a – done: retired the Python Docker bridge; native Linux bridge support remains deferred to Stage 6 step 6a.
- 3b – done for available events: Copilot hooks, mapping, workspace session keys, cross-source deduplication and setup documentation. AGT-05 remains open because the current VS Code hook picker has no approval event.
- 3c – implementation done: added Windows and configured-WSL hook setup in Cody's web UI, with status/error reporting and the devcontainer Copilot bind mount. The fresh-host manual demonstration remains pending.

**Requirements:** 5 of 6 planned for this stage are implemented; all 6 have passing evidence. AGT-05 remains open at the owner's request. Overall: 70 of 105 active requirements are implemented.

**Verification:** `ci/check` passed: 38 Python tests, 29 firmware native tests, 131 Go tests, static checks, firmware build, Windows app/installer and Linux build. The requirement report has 0 errors and 4 warnings (FW-15, AGT-02, AGT-05, AGT-12). One manual record was added: AGT-04, a real Copilot session in VS Code 1.147.0 in the WSL2 devcontainer driving IDLE, THINKING, WORKING and COMPLETED. The fresh-host Windows plus configured-WSL setup check for AGT-10 is still pending.

**Spec changes and deviations:**
- Retired the Python Docker bridge in Stage 3a and accepted the native-Linux support gap until Stage 6 step 6a (TD-15); withdrew BRG-07.
- Added Copilot hook behavior, source deduplication and setup guidance. VS Code 1.147.0 exposes `ErrorOccurred` but no approval event, so ERROR is supported and APPROVAL remains conditional.
- Replaced the planned `cody setup` command with setup through the Windows bridge website. The UI installs hooks on Windows and in each configured WSL relay distro; the `cody` shell command and manual hook scripts remain.
- Added a WSL-home bind mount for this repository's Copilot configuration so the WSL-installed hooks are visible to the devcontainer. Separate devcontainer hook volumes remain a manual-installation case.
- At the owner's request, Stage 3 is marked done with AGT-05 unimplemented and AGT-10's fresh-host demonstration pending. Neither is counted as passing manual evidence.

**Overall status:**

| Stage | Content | Status |
|---|---|---|
| 1 | Walking skeleton | done |
| 2 | Status light on the bench with Claude Code | done (open points) |
| 3 | Copilot integration | done (open points) |
| 4 | Mechanism prototype | next |
| 5 | Character and assembly | open |
| 6 | Native bridge for Linux and macOS, firmware updater | open |

**Remaining requirements by stage:** 1 → 0, 2 → 7, 3 → 1, 4 → 10, 5 → 14, 6 → 3.

**Waiting on the project owner:**
- Before Stage 5, choose the figure-construction approach: Meshy outer shapes with code-CAD functional geometry (recommended), or full code CAD.

**Risks and notes:**
- AGT-05 depends on VS Code exposing an approval hook event; keep it open until that event is available and verified.
- AGT-10's fresh-host Windows and configured-WSL installation demonstration remains pending.
- AGT topic status remains in progress because AGT-05 is not implemented.

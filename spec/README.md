# Cody – Specification

Cody (Coding Agent Desk Companion) is a small 3D-printed desk figure of a cartoon developer sitting at a laptop. It shows developers who work with AI coding agents (Claude Code, GitHub Copilot in VS Code) what their agent is doing: while the agent works, the figure's two arms type, driven by a single geared DC motor; an RGB LED and a buzzer signal the other states (idle, thinking, approval needed, completed, error). The project delivers printable CAD parts, ESP32 firmware driving ready-made electronics modules over USB, and a computer-side bridge with the agent hooks.

**Status: approved (2026-11-06).** Implementation follows [implementation-plan.md](implementation-plan.md). Changes to requirements or the architecture are still possible; they are decided with the project owner and recorded (see "Decisions").

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
| [mechanical-workflow.md](mechanical-workflow.md) | Nine-phase mechanical workflow, review gates, FDM rules, and shared assistant skills |

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

Every gap or ambiguity that was decided is recorded in the requirement or section it belongs to. Decisions are never made silently. The table below lists only the important ones: decisions that change requirements (scope, cuts, new or reworded requirements) or the architecture (building blocks, interfaces, key technologies, delivery). Smaller choices (tool versions, conventions, verification details, CI setup) are recorded only in their section.

## Topics and status

Status values: **specified** (requirements written, not implemented) · **in progress** · **implemented** (all active requirements have passing evidence).

| Prefix | Topic | Section in requirements.md | Reqs (active) | Stage | Status |
|---|---|---|---|---|---|
| TRC | Requirement traceability | 1 | 7 | 1 | implemented |
| REL | CI, versions and releases | 2 | 6 | 1 | implemented |
| FW | Firmware | 3 | 15 | 2 | in progress |
| BRG | Host bridge | 4 | 24 | 2, 6 | in progress |
| AGT | Agent integrations | 5 | 12 | 2, 3 | in progress |
| ELE | Electronics | 6 | 7 | 2, 4, 5 | in progress |
| FIG | Figure and mechanism | 7 | 18 | 4, 5 | specified |
| DOC | Build documentation | 8 | 7 | 2, 4, 5 | in progress |
| TST | Hardware test panel | 9 | 11 | 2 | in progress |

## Decisions

| Date | Decision | Recorded in |
|---|---|---|
| 2026-11-05 | Built for the owner's own desk; repo open, no build kit for others (no polished BOM, assembly guide or toolchain-free initial flashing). *Build kit part superseded on 2026-11-06 (DOC); local image flashing from the test panel was added in Stage 2, while GitHub release checking remains a separate Stage 6 feature.* | README.md (this table) |
| 2026-11-05 | Claude Code and GitHub Copilot (VS Code) integrations are both Must. | requirements.md (AGT) |
| 2026-11-05 | A host-side bridge (localhost HTTP in, USB serial out) is used, because hooks run in containers/WSL that can't open the serial port, and it merges sessions. Alternatives rejected: hook writes serial directly; Wi-Fi on the ESP32. | requirements.md (BRG) |
| 2026-11-05 | The bridge runs on Linux and WSL2 (USB attached with usbipd-win on WSL); containerizing it is evaluated in architecture. | architecture.md 2 |
| 2026-11-05 | Out of scope: arm home-position sensor; Wi-Fi/Bluetooth on the device. | architecture.md 1 |
| 2026-11-05 | Topics FW, BRG, AGT, ELE, FIG (mechanism and printed figure merged into FIG). | requirements.md |
| 2026-11-05 | Initial FW decisions: COMPLETED/ERROR stay until the next state; heartbeat with fallback to "disconnected" after 30 s; volume; SPEED Must and LED/BEEP test commands Could. Runtime test tuning was initially volatile; per-device NVS tuning and bridge-owned volume/brightness are decided on 2026-11-22 (TD-10). | requirements.md 3, architecture.md 4.2, TD-10 |
| 2026-11-05 | BRG: several sessions are merged, most urgent state wins; stale sessions (WORKING/THINKING/APPROVAL) dropped after a configurable timeout; listens on local interfaces only, no auth; autostart. | requirements.md 4 |
| 2026-11-05 | AGT: Copilot drives IDLE/THINKING/WORKING/COMPLETED (Must); APPROVAL/ERROR are Could until VS Code provides events; Copilot sessions keyed by workspace folder. | requirements.md 5 |
| 2026-11-22 | VS Code 1.147.0 exposes `ErrorOccurred` but no approval/permission-request event; map ERROR and leave APPROVAL conditional. Although the payload includes a session ID, Copilot sessions remain keyed by `cwd`. | requirements.md 5, architecture.md 4.1 |
| 2026-11-05 | ELE: whole device powered from one USB port, ≤ 500 mA (USB 2.0 guarantee; the ESP32 is a USB 2.0 device); soldering of header pins and wires allowed; parts budget ≤ CHF 25. | requirements.md 6 |
| 2026-11-05 | FIG: figure simplified to a bust (head, torso, arms) behind table and laptop on a platform; originally ≤ 12 printed parts (withdrawn 2026-11-23); hand travel 3–8 mm at 1–4 strokes/s; 8 h endurance run; noise judged by the owner. | requirements.md 7 |
| 2026-11-05 | Controller ESP32-C3 SuperMini. | architecture.md TD-01 |
| 2026-11-05 | Firmware with Arduino-ESP32 and PlatformIO (pioarduino platform). | architecture.md TD-02 |
| 2026-11-05 | Bridge in Python. | architecture.md TD-03 |
| 2026-11-05 | Bridge runs as a Docker container; image published on GHCR (ci/publish, ci/promote). | architecture.md TD-04 |
| 2026-11-05 | CAD in build123d; head originally generated with Meshy.ai (revised 2026-11-23). | architecture.md TD-05 |
| 2026-11-05 | Initial interfaces: hooks forward the agent's own payload to the bridge (port 8765), all mapping in the bridge; serial text protocol with OK/ERR replies, VOLUME 0–3 (0 = mute; superseded by TD-10), READY line at boot. | architecture.md 4 |
| 2026-11-05 | Delivery: download from the GitHub release, plus the bridge image on GHCR. | delivery.md 4 |
| 2026-11-06 | Plug & play for the device connection, incl. replug: BRG-06 reworded (no port setting, no manual step after a one-time host setup); serial port in BRG-07 is an optional override; on WSL2 a one-time usbipd setup with auto-attach is accepted; Docker stays, plug & play in the container must be shown in step 2b. | requirements.md 4, architecture.md 2 and 7 |
| 2026-11-06 | Tag ruleset for `v*` restricts updates and deletions and blocks force pushes, but not creation: GitHub can't add the built-in GitHub Actions actor to a bypass list. Rejected: own GitHub App with token in the workflows (setup, secrets, workflow changes). REL-05 reworded. | requirements.md 2, delivery.md 3 |
| 2026-11-06 | Others can rebuild Cody: the project delivers a parts list and assembly instructions (new topic DOC), written for a maker without the owner's help, in the repository and as release files, growing with stages 2, 4 and 5. ELE-07 reduced to the wiring diagram. | requirements.md 8 |
| 2026-11-06 | The bridge stays (dropping it would need the device and a hook program in every devcontainer); setup effort is reduced instead: one command sets up bridge and hooks (AGT-10), and the same hook configuration works on the host and in devcontainers (AGT-11). | requirements.md 5, architecture.md 4.1 |
| 2026-11-06 | Stage 2 start: AGT-02 counts WSL2 as Linux; ELE-03 margin 1.5 × stall current; ELE-06 counted per piece without spares and shipping; host command `cody` as a shell script (status and manual state in stage 2, setup in stage 3); hooks are curl command hooks with fallback addresses (Docker Engine in WSL2 has no `host.docker.internal`); build documentation released as CSV and PDF. | requirements.md 5–6, architecture.md 3–4, delivery.md 2 |
| 2026-11-06 | The bridge logs state changes (also without a device) and, at debug level, events and serial traffic (BRG-13); the log goes to Docker's log, rotated by Docker at 10 MB × 2 files (BRG-14). | requirements.md 4, architecture.md 4.4 |
| 2026-11-06 | Wiring diagram (ELE-07) as an SVG block diagram with named pins and nets, generated from Python with schemdraw (added in step 2d). | architecture.md 3, 5.1 |
| 2026-11-06 | New BRG-15: `cody session clear` (`DELETE /sessions`) clears all sessions, to reset the bridge when a session is stale (e.g. Claude Code killed without `SessionEnd`). | requirements.md 4, architecture.md 4.1 |
| 2026-11-21 | Bridge redesigned (TD-06): a Go tray app on Windows with "Start automatically" (no Windows service), installed with an NSIS installer without admin rights; WSL2 and its devcontainers reach it through a relay that the app starts with `wsl.exe`, so no WSL networking mode is forced. New BRG-16 – BRG-19, AGT-12; AGT-02 includes Windows. The Python Docker bridge stays for Linux, frozen; native Linux and macOS were planned for stage 6. Retirement timing was later changed by TD-15. Steps 2g – 2i added to stage 2 after 2d; Docker replug dropped from 2d. `cody` shell command and Python hook installer stay; consolidation decided at stage 3 start. | requirements.md 4–5, architecture.md 2–4, 5, TD-06, implementation-plan.md |
| 2026-11-21 | Initial TST design: a dev-only browser panel in the devcontainer directly owns USB serial. Superseded by the local bridge website decision below; the firmware test mode remains, while tuning persistence is decided separately in TD-10. | requirements.md 9, architecture.md TD-07, implementation-plan.md |
| 2026-11-21 | In WSL mirrored networking, Windows owns bridge port 8765, so the Docker-gateway relay uses the first available port in 8766–8770. The hook installer can set the gateway target port explicitly (`--port`, default 8765); hooks require the expected bridge response and no longer probe a port range. | architecture.md 4.1, 4.6 |
| 2026-11-21 | The Windows tray app starts its WSL child processes without showing console windows. | requirements.md 4, architecture.md 4.3, 4.6 |
| 2026-11-21 | The Windows tray remains a launcher/status indicator; Cody serves one embedded local website for status, configuration, the bridge-backed test panel and firmware updates. The site is loopback-only with same-origin checks for browser mutations; settings require restart. The updater is Windows-first and uses a pinned espflash helper. | requirements.md 4, 9; architecture.md 4, 5, TD-08, TD-09; delivery.md 2; implementation-plan.md |
| 2026-11-22 | Motor tuning is per-device NVS data, saved explicitly with a bridge-supplied UTC timestamp; the website shows it and can save or restore firmware defaults. Volume and brightness are Windows Go bridge preferences with 1%-step sliders; volume uses `V=(x/100)^2.5`, defaults to 50%, and brightness uses `b=(x/100)^2.2` on channels after the existing 50% cap, defaulting to 100%. The frozen Python bridge is knowingly incompatible with the new firmware until its Stage 3 retirement. | requirements.md 3–4, 9; architecture.md 4.2–4.4, TD-10; verification.md; implementation-plan.md |
| 2026-11-22 | Add first-time and update `.bin` flashing through the bridge, with explicit ESP32-C3 bootloader selection, preserving saved motor tuning through protocol snapshot/restore; no devcontainer USB bind mount. Keep the GitHub release updater as a separate Stage 6 feature. | requirements.md 9; architecture.md 4.2, 4.4, TD-11; implementation-plan.md |
| 2026-11-22 | Add Manual ROM bootloader flash for existing Cody devices whose firmware lacks `TUNING?`; this path warns that motor tuning may be lost, while the separate Update path continues to require and restore a tuning snapshot. | requirements.md 9; architecture.md 4.2, 4.4, TD-12; implementation-plan.md |
| 2026-11-22 | Add a temporary LED test brightness control that applies after the raw RGB test colour and restores the saved bridge preference on test exit. | requirements.md 9; architecture.md 4.2, TD-13; implementation-plan.md |
| 2026-11-22 | Keep the Stage 2 parts list as an electronics draft and complete DOC-01 in Stage 5, after the printed parts and fasteners are defined (originally 5d, now 5g). Use pinned Pandoc and WeasyPrint to publish versioned documentation files. | implementation-plan.md 2e, 5g; architecture.md 5.1, TD-14; delivery.md 2 |
| 2026-11-22 | Retire the frozen Python Docker bridge in Stage 3 step 3a, before native Linux support is available in Stage 6. The owner accepts the native-Linux support gap; Windows, WSL and devcontainers use the Windows Go bridge and relay in the interim. | requirements.md BRG-07; architecture.md TD-15; delivery.md 2, 4; implementation-plan.md 3a, 6a |
| 2026-11-22 | Withdraw BRG-07 because it described configuration behavior of the retired Python Docker bridge; retain its ID and historical evidence. | requirements.md BRG-07; architecture.md TD-03, TD-04; implementation-plan.md 3a |
| 2026-11-22 | VS Code's Copilot harness reads repository `.claude/settings.json` and `.claude/settings.local.json`, but its documented user hook sources do not include Claude Code's `~/.claude/settings.json`; deduplicate events across all applicable sources. The Go program takes over `cody` and the hook installer. Step 3c supports Windows and its WSL/devcontainer relay; native Linux setup is deferred until step 6a. | requirements.md AGT-08, AGT-10; architecture.md 2, 4.1; implementation-plan.md 3b, 3c, 6a |
| 2026-11-22 | Replace the planned `cody setup` command with setup in the Windows bridge website: the installer enables autostart, and the UI installs both agents' hooks on Windows and in each configured WSL relay distro, checks device connectivity and reports missing prerequisites. Keep the current `cody` shell command and hook scripts for manual use. | requirements.md AGT-10; architecture.md 3, 4.1; implementation-plan.md 3c |
| 2026-11-22 | Select the 6 V, 500 rpm N20 and the chest cam drive with action-figure shoulder joints. Adapt FIG-10/11 and Stage 4 to the cam, levers and axle; tune the motor to the fastest non-distracting speed within FIG-08. The owner reports the selected motor currently runs at about 120 rpm (about 2 strokes/s per arm) and can run faster. | requirements.md 7; architecture.md 8.2–8.3; docs/parts-list.md; implementation-plan.md 4 |
| 2026-11-23 | Stage 4a printability: align each arm's outward side as a flat bed-contact face with the knob up; make the cam-side lever root cover the full axle end face while keeping the arm socket open; support the motor mount from the ground plate without overhangs. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a |
| 2026-11-23 | Increase the Stage 4a axle running clearance for undersized FDM holes: use a Ø4.6 mm crossbar bore around the Ø4 mm axle (0.3 mm radial clearance), retaining the Ø6 mm outside diameter for motor clearance. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a |
| 2026-11-22 | Use a fixed, rectangular 3D-printed shoulder crossbar with relieved arm sleeves and concentric joint surfaces guiding rotation; keep the N20 factory shaft. Update FIG-11 to require the printed axle as well as the cam and levers. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a |
| 2026-11-22 | Set the Stage 4a crossbar to 2.4 × 3.2 mm with 0.2 mm relief and 0.6 mm sleeve walls; retain it with removable M3 end caps using Ø4.5 × 6 mm inserts, within the 62 mm shoulder envelope. Use the permitted 1.2 mm lower motor position to meet the 1.5 mm sleeve clearance; the frame uses four inserts total for the caps and stop bridge. | architecture.md 8.3; implementation-plan.md 4a; docs/parts-list.md |
| 2026-11-23 | Use the supplied GA12-N20 dimension drawing (12 × 10 mm body envelope, 9 mm gearbox, 15 mm motor can, Ø3 × 10 mm shaft); adjust the motor axis and lever to 9.7 mm forward to preserve at least 1.5 mm sleeve clearance. Exclude the separate shaft-fit coupon from the assembled mechanism and require pairwise collision evidence for all assembly components. | architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Replace the rectangular shoulder crossbar with a Ø6 × 60 mm circular crossbar captured by blind sockets in frame-mounted end caps; use Ø6.4 mm circular arm-sleeve bores and cam levers at least 4 mm thick. Lower the motor axis to 8.8 mm below the axle to preserve at least 1.5 mm clearance to the larger round sleeves. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Replace the end-cap layout with a frame-integrated hollow Ø6 outer axle with a continuous front assembly slot, Ø4 inner arm axles with captured split collars, and outside push-fit arms. Remove the stop bridge in favor of local frame/joint stops; integrate a provisional base flush with the frame back. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Supersede the slotted integrated axle with two separate closed Ø6 shoulder tubes of unequal lengths. Each tube and its rotating axle and outside arm form a module secured to the frame's inner face with two M3 screws into Ø4.5 × 6 mm inserts; the rear stays flush. The arms use male axle knobs, and the frame has local upward travel stops. | requirements.md FIG-03, FIG-11, FIG-15, FIG-18; architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Replace the backplate inserts with captured standard M3 hex nuts; use two top-access M3 × 6 socket-head screws per crossbar with head recesses and Allen-key clearance. Set the backplate to 5 mm while keeping the rear face flush. Extend both integral lever roots into the axle and make each arm's male knob 4 mm long. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Correct the crossbar screw direction: use two rear-access M3 × 8 socket-head screws per tube, passing through backplate clearance holes and engaging nuts captured in each crossbar's upper mounting rail. Recess screw heads from the back; retain the 5 mm backplate without inserts. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Replace the bulky upper blocks and captured nuts with a full-length rear crossbar shelf and sturdy upward lip with through-holes; use rear-access screws and loose, wrench-held nuts. Remove the frame endstops and withdraw FIG-09; orient arms flat for printing with knobs up and reduce the shoulder hub to Ø8 × 2 mm for frame clearance. | requirements.md FIG-09, FIG-11; architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Approve the Stage 5 frame-and-covers direction: removable frame/platform connection and keyed push-on head, separate single-color tabletop and one-piece notebook; all printed parts are single-color. Withdraw the fixed part-count cap. Use a simple code-CAD head by default; Meshy is optional. | requirements.md FIG-04, FIG-19, FIG-20; architecture.md TD-05, 8.4; implementation-plan.md 5; verification.md |
| 2026-11-23 | Integrate the back into the Stage 5 frame and leave only the chest as a removable torso cover. Redesign shoulder fasteners for front access in Stage 5; retain the rear-access Stage 4 prototype and scope its detailed fasteners in FIG-11 accordingly. | requirements.md FIG-11, FIG-20; architecture.md 8.3–8.4; implementation-plan.md 5 |
| 2026-11-23 | Include every printable CAD part as a versioned STL in the flat release directory; keep the assembly STEP reference out of downloadable release files. | delivery.md 2; ci/build-release |
| 2026-11-23 | Replace the motor-mount support lattice with one solid block on the ground plate; leave filament-use optimization to slicer infill instead of adding material-saving pockets to the CAD. | architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Enlarge Stage 4 crossbars to Ø10/Ø8.6 mm and axles to Ø8 mm; reverse the arm interface to an outward Ø8 × 8 mm axle peg and an Ø8.2 × 8.3 mm blind arm hole in an Ø11 × 9.3 mm hub. Keep the existing motor position and accept ≈0.30 mm calculated clearance provisionally, pending a physical no-contact check; print revised crossbars, axles and arms in PLA. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-23 | Raise the Stage 4 crossbar/axle axes by 2 mm without changing the frame, motor position or frame screw holes; add local crossbar reliefs for M3 nut and wrench access. Lengthen the cam levers diagonally to the existing follower location and reduce the outward axle pegs to Ø6 × 8 mm with matching Ø6.2 × 8.3 mm arm holes. | requirements.md FIG-11; architecture.md 8.3; implementation-plan.md 4a; cad/README.md |
| 2026-11-22 | Update DOC-03's mechanism assembly instructions from the superseded crank and connecting rods to the selected chest-cam mechanism: motor, cam drive, shoulder axle and arm joints, levers, stops, and arms. | requirements.md DOC-03; implementation-plan.md 4c; architecture.md 8.3 |
<!-- | 2026-01-31 | Example: chose X over Y because … | architecture.md 3.2 | -->

## Open points

<!-- Questions that are not decided yet, each with options and a recommendation. Move them to "Decisions" once settled. -->

- **Optional Meshy.ai licence** (only if a generated model is used or distributed). Check the terms applicable to that output before committing or releasing it; the code-CAD head remains the baseline.
- **Frame, chest cover and neck engineering** (before detailed Stage 5 CAD). Determine the front-access shoulder fasteners, joints, clearances, wiring and access using architecture.md 8.4; validate retention and fit with a representative printed coupon. Evaluate the chair backrest and tabletop cutout without assuming their final shape.

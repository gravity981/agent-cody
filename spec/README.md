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
| FW | Firmware | 3 | 12 | 2 | in progress |
| BRG | Host bridge | 4 | 14 | 2 | in progress |
| AGT | Agent integrations | 5 | 11 | 2, 3 | in progress |
| ELE | Electronics | 6 | 7 | 2, 4, 5 | specified |
| FIG | Figure and mechanism | 7 | 17 | 4, 5 | specified |
| DOC | Build documentation | 8 | 7 | 2, 4, 5 | specified |

## Decisions

| Date | Decision | Recorded in |
|---|---|---|
| 2026-11-05 | Built for the owner's own desk; repo open, no build kit for others (no polished BOM, assembly guide or toolchain-free flashing). *Build kit part superseded on 2026-11-06 (DOC).* | README.md (this table) |
| 2026-11-05 | Claude Code and GitHub Copilot (VS Code) integrations are both Must. | requirements.md (AGT) |
| 2026-11-05 | A host-side bridge (localhost HTTP in, USB serial out) is used, because hooks run in containers/WSL that can't open the serial port, and it merges sessions. Alternatives rejected: hook writes serial directly; Wi-Fi on the ESP32. | requirements.md (BRG) |
| 2026-11-05 | The bridge runs on Linux and WSL2 (USB attached with usbipd-win on WSL); containerizing it is evaluated in architecture. | architecture.md 2 |
| 2026-11-05 | Out of scope: arm home-position sensor; Wi-Fi/Bluetooth on the device. | architecture.md 1 |
| 2026-11-05 | Topics FW, BRG, AGT, ELE, FIG (mechanism and printed figure merged into FIG). | requirements.md |
| 2026-11-05 | FW: COMPLETED/ERROR stay until the next state; heartbeat with fallback to "disconnected" after 30 s; mute and volume; SPEED Must, LED/BEEP test commands Could; settings kept by the bridge, not the firmware. | requirements.md 3 |
| 2026-11-05 | BRG: several sessions are merged, most urgent state wins; stale sessions (WORKING/THINKING/APPROVAL) dropped after a configurable timeout; listens on local interfaces only, no auth; autostart. | requirements.md 4 |
| 2026-11-05 | AGT: Copilot drives IDLE/THINKING/WORKING/COMPLETED (Must); APPROVAL/ERROR are Could until VS Code provides events; Copilot sessions keyed by workspace folder. | requirements.md 5 |
| 2026-11-05 | ELE: whole device powered from one USB port, ≤ 500 mA (USB 2.0 guarantee; the ESP32 is a USB 2.0 device); soldering of header pins and wires allowed; parts budget ≤ CHF 25. | requirements.md 6 |
| 2026-11-05 | FIG: figure simplified to a bust (head, torso, arms) behind table and laptop on a platform; ≤ 12 printed parts; hand travel 3–8 mm at 1–4 strokes/s; 8 h endurance run; noise judged by the owner. | requirements.md 7 |
| 2026-11-05 | Controller ESP32-C3 SuperMini. | architecture.md TD-01 |
| 2026-11-05 | Firmware with Arduino-ESP32 and PlatformIO (pioarduino platform). | architecture.md TD-02 |
| 2026-11-05 | Bridge in Python. | architecture.md TD-03 |
| 2026-11-05 | Bridge runs as a Docker container; image published on GHCR (ci/publish, ci/promote). | architecture.md TD-04 |
| 2026-11-05 | CAD in build123d; head generated with Meshy.ai and processed with manifold3d. | architecture.md TD-05 |
| 2026-11-05 | Interfaces: hooks forward the agent's own payload to the bridge (port 8765), all mapping in the bridge; serial text protocol with OK/ERR replies, VOLUME 0–3 (0 = mute), READY line at boot. | architecture.md 4 |
| 2026-11-05 | Delivery: download from the GitHub release, plus the bridge image on GHCR. | delivery.md 4 |
| 2026-11-06 | Plug & play for the device connection, incl. replug: BRG-06 reworded (no port setting, no manual step after a one-time host setup); serial port in BRG-07 is an optional override; on WSL2 a one-time usbipd setup with auto-attach is accepted; Docker stays, plug & play in the container must be shown in step 2b. | requirements.md 4, architecture.md 2 and 7 |
| 2026-11-06 | Tag ruleset for `v*` restricts updates and deletions and blocks force pushes, but not creation: GitHub can't add the built-in GitHub Actions actor to a bypass list. Rejected: own GitHub App with token in the workflows (setup, secrets, workflow changes). REL-05 reworded. | requirements.md 2, delivery.md 3 |
| 2026-11-06 | Others can rebuild Cody: the project delivers a parts list and assembly instructions (new topic DOC), written for a maker without the owner's help, in the repository and as release files, growing with stages 2, 4 and 5. ELE-07 reduced to the wiring diagram. | requirements.md 8 |
| 2026-11-06 | The bridge stays (dropping it would need the device and a hook program in every devcontainer); setup effort is reduced instead: one command sets up bridge and hooks (AGT-10), and the same hook configuration works on the host and in devcontainers (AGT-11). | requirements.md 5, architecture.md 4.1 |
| 2026-11-06 | Stage 2 start: AGT-02 counts WSL2 as Linux; ELE-03 margin 1.5 × stall current; ELE-06 counted per piece without spares and shipping; host command `cody` as a shell script (status and manual state in stage 2, setup in stage 3); hooks are curl command hooks with fallback addresses (Docker Engine in WSL2 has no `host.docker.internal`); build documentation released as CSV and PDF. | requirements.md 5–6, architecture.md 3–4, delivery.md 2 |
| 2026-11-06 | The bridge logs state changes (also without a device) and, at debug level, events and serial traffic (BRG-13); the log goes to Docker's log, rotated by Docker at 10 MB × 2 files (BRG-14). | requirements.md 4, architecture.md 4.4 |
| 2026-11-06 | Wiring diagram (ELE-07) as an SVG block diagram with named pins and nets, generated from Python with schemdraw (added in step 2d). | architecture.md 3, 5.1 |
<!-- | 2026-01-31 | Example: chose X over Y because … | architecture.md 3.2 | -->

## Open points

<!-- Questions that are not decided yet, each with options and a recommendation. Move them to "Decisions" once settled. -->

- **Meshy.ai licence of generated models** (before stage 5). Free-tier outputs are believed to be CC BY 4.0 (to be checked). Options: (a) free tier with attribution in the repo (the mesh then stays under CC BY 4.0 next to the MIT-licensed project); (b) paid tier for private ownership; (c) sculpt the head yourself. Recommendation: (a) if the terms confirm it.
- **N20 gear ratio** (before stage 4): 300 or 500 rpm, both bought; chosen with the crank prototype. The other modules are chosen (docs/parts-list.md).
- **Docker container and serial replug** (TD-04, stage 2). Reaching the bridge from devcontainers was shown in step 2c. Recommendation: prove first plug-in and replug in step 2d; fall back to a native systemd service if it doesn't work.
- **Does Copilot read user-level Claude Code hook configuration?** (AGT-08, stage 3). Recommendation: check at stage 3 start; dedupe in the bridge either way.

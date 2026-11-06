# Cody – Specification

Cody (Coding Agent Desk Companion) is a small 3D-printed desk figure of a cartoon developer sitting at a laptop. It shows developers who work with AI coding agents (Claude Code, GitHub Copilot in VS Code) what their agent is doing: while the agent works, the figure's two arms type, driven by a single geared DC motor; an RGB LED and a buzzer signal the other states (idle, thinking, approval needed, completed, error). The project delivers printable CAD parts, ESP32 firmware driving ready-made electronics modules over USB, and a computer-side bridge with the agent hooks.

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
| FW | Firmware | 3 | 12 | 2 | specified |
| BRG | Host bridge | 4 | 12 | 2 | specified |
| AGT | Agent integrations | 5 | 9 | 2, 3 | specified |
| ELE | Electronics | 6 | 7 | 2, 4, 5 | specified |
| FIG | Figure and mechanism | 7 | 17 | 4, 5 | specified |

## Decisions

| Date | Decision | Recorded in |
|---|---|---|
| 2026-11-05 | Built for the owner's own desk; repo open, no build kit for others (no polished BOM, assembly guide or toolchain-free flashing). | README.md (this table) |
| 2026-11-05 | Claude Code and GitHub Copilot (VS Code) integrations are both Must. | requirements.md (AGT) |
| 2026-11-05 | Status light (ESP32, LED, buzzer, bridge, hooks) comes before the arm mechanism and the character, for value early. | implementation-plan.md |
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
| 2026-11-05 | CI runs in the devcontainer. | delivery.md 2 |
| 2026-11-05 | Delivery: download from the GitHub release, plus the bridge image on GHCR. | delivery.md 4 |
| 2026-11-05 | MAJOR version for serial protocol or bridge HTTP API breaks, or printed parts that no longer fit the previous release. | delivery.md 1 |
| 2026-11-05 | Verification: unit tests and CAD model checks in CI (JUnit XML, names with `req_ABC_01`); hardware, real agents and look verified manually. | verification.md |
| 2026-11-05 | Stages: 1 walking skeleton, 2 status light with Claude Code (M1), 3 Copilot (M2), 4 mechanism prototype, 5 character and assembly (M3). | implementation-plan.md |
| 2026-11-05 | Stage 1: first release contains skeletons (firmware sends `READY <version>`, bridge image answers `--version`, compose file); Python dependencies with uv and lockfiles; static checks ruff, clang-format, shellcheck; Docker in the devcontainer via docker-outside-of-docker; bridge image on GHCR is public. | implementation-plan.md stage 1, delivery.md 2, architecture.md 5.1 |
| 2026-11-06 | Step 1a: Python 3.14 (newest supported by build123d/OCP); ESP32 toolchain baked into the devcontainer image; one uv workspace with a single lockfile instead of one lockfile per project; binary files in plain git, marked binary (no LFS). | architecture.md 5.1 |
| 2026-11-06 | Plug & play for the device connection, incl. replug: BRG-06 reworded (no port setting, no manual step after a one-time host setup); serial port in BRG-07 is an optional override; on WSL2 a one-time usbipd setup with auto-attach is accepted; Docker stays, plug & play in the container must be shown in step 2b. | requirements.md 4, architecture.md 2 and 7 |
| 2026-11-06 | Tag ruleset for `v*` restricts updates and deletions and blocks force pushes, but not creation: GitHub can't add the built-in GitHub Actions actor to a bypass list. Rejected: own GitHub App with token in the workflows (setup, secrets, workflow changes). REL-05 reworded. | requirements.md 2, delivery.md 3 |
| 2026-11-06 | Step 1b: REL-01 is verified by a manual log row (a green CI run on a pull request) and stays on the "not yet implemented" list until that row exists; skipped tests count as no evidence; a `req_` token not followed by `<2–4 capitals>_<2 digits>` is a malformed tag; latest manual record = newest date, then lowest row. | verification.md 3 |
| 2026-11-06 | CI caches the devcontainer image on GHCR (`<repo>-devcontainer`, pushed on pushes to `main`, used as cache by every run); devcontainers/ci and docker/login-action pinned to commit SHAs; the devcontainer image slimmed (PlatformIO only for the ESP32-C3, no download caches). | delivery.md 2 |
| 2026-11-06 | The project is licensed under MIT, copyright Nicolas Marty (`LICENSE`). Third-party assets keep their own licence and are marked where they are stored. | README.md (repository) |
| 2026-11-06 | Step 1c: REL-03 – REL-06 verified by log rows after the first candidate and promotion and stay on the "not yet implemented" list until then; firmware released as one combined image; `LICENSE` in the release files and licence label on the bridge image; GHCR access with `GITHUB_TOKEN`. | verification.md 4, delivery.md 2 |
| 2026-11-06 | The original idea file (`spec/idea.md`) is removed because it conflicted with the spec; the spec is the only description of the project. It remains in the git history. | README.md (this document) |
<!-- | 2026-01-31 | Example: chose X over Y because … | architecture.md 3.2 | -->

## Open points

<!-- Questions that are not decided yet, each with options and a recommendation. Move them to "Decisions" once settled. -->

- **Meshy.ai licence of generated models** (before stage 5). Free-tier outputs are believed to be CC BY 4.0 (to be checked). Options: (a) free tier with attribution in the repo (the mesh then stays under CC BY 4.0 next to the MIT-licensed project); (b) paid tier for private ownership; (c) sculpt the head yourself. Recommendation: (a) if the terms confirm it.
- **Exact module models** (before stage 2): motor driver breakout (DRV8833 candidate), WS2812 module, buzzer module, N20 motor and gear ratio (before stage 4). Recommendation: choose together at stage start within ELE-04 and ELE-06.
- **Docker container and serial replug / reachability from devcontainers** (TD-04, stage 2). Recommendation: prove in step 2b; fall back to a native systemd service if it doesn't work.
- **Does Copilot read user-level Claude Code hook configuration?** (AGT-08, stage 3). Recommendation: check at stage 3 start; dedupe in the bridge either way.

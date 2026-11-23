# Cody – Requirements

All requirements of the project, one section per topic. Conventions (IDs, priorities, withdrawn requirements) are in [README.md](README.md). Topics 1 and 2 come from the project blueprint and apply to every project; adapt them where the project needs it.

## 1. Requirement traceability (TRC)

The requirement report is built in the project's own technology during stage 1. It links every requirement to its evidence (see [verification.md](verification.md)).

| ID | Requirement | Prio |
|---|---|---|
| TRC-01 | The requirement report reads every requirement row of `spec/requirements.md`, including withdrawn ones; duplicate IDs and unknown priorities are errors. | Must |
| TRC-02 | It reads the "not yet implemented" list in `spec/verification.md`, including ranges (`ABC-01 – ABC-05`); unknown or withdrawn IDs on the list are errors. | Must |
| TRC-03 | It reads the results of every automated verification the project uses, with the requirement IDs each result is tagged with (tag convention in `spec/verification.md`); malformed, unknown or withdrawn tags are errors. | Must |
| TRC-04 | It reads `spec/verification-log.md`; for each requirement only its latest manual record counts. | Must |
| TRC-05 | Errors are: an implemented requirement (not on the "not yet implemented" list) without passing evidence, and any failing evidence of a requirement. | Must |
| TRC-06 | Warnings are: a requirement with passing evidence that is still on the "not yet implemented" list, and a requirement planned in no stage or in several stages of `spec/implementation-plan.md`. | Should |
| TRC-07 | It writes `reports/requirements.md` (totals, totals per stage, errors, warnings, and requirement → evidence → result) and exits non-zero on any error. | Must |

## 2. CI, versions and releases (REL)

Details in [delivery.md](delivery.md). The GitHub workflows come with the blueprint; the project provides the entry points in `ci/`.

| ID | Requirement | Prio |
|---|---|---|
| REL-01 | On every pull request and every push to `main`, CI runs `ci/check` in the CI environment (delivery.md 2); it includes all automated verification and the requirement report, and fails if any of them fails. | Must |
| REL-02 | Every build carries a version according to the version scheme (delivery.md 1) and its commit; how a built artifact reveals both is defined in architecture.md. | Must |
| REL-03 | A release candidate is built from `main` by a manually started workflow, only for a commit whose CI passed, after the full check passed again; it is tagged `v<version>-rc.<M>` and published as a GitHub pre-release with the release files. | Must |
| REL-04 | Only the latest candidate of a version can be promoted; promotion does not rebuild, tags `v<version>` on the candidate's commit, and publishes a GitHub release with exactly the candidate's release files. | Must |
| REL-05 | Tags `v*` can't be moved or deleted (repository tag ruleset: restrict updates, restrict deletions, block force pushes); the release workflows create them and never reuse an existing tag. Creating `v*` tags by hand is not blocked by GitHub. | Must |
| REL-06 | Releases are delivered as defined in delivery.md 4 (deployment, distribution, or download from the GitHub release only). | Must |

## 3. Firmware (FW)

The ESP32 firmware turns simple commands from the bridge into physical behaviour. It knows nothing about agents or IDEs. Protocol details in [architecture.md](architecture.md). Motor tuning is device-specific and stored by the firmware; sound volume and LED brightness are user preferences stored by the Windows Go bridge and sent after every (re)connect.

| ID | Requirement | Prio |
|---|---|---|
| FW-01 | The firmware has the states IDLE, WORKING, THINKING, APPROVAL, COMPLETED and ERROR, with arm, LED and sound behaviour as follows: IDLE – arms stopped, dim blue, no sound; WORKING – alternating typing, pulsating blue, no sound; THINKING – arms stopped, yellow, no sound; APPROVAL – arms stopped, blinking orange, double beep; COMPLETED – arms stopped, green, success sound; ERROR – arms stopped, blinking red, warning sound. | Must |
| FW-02 | It accepts newline-terminated, human-readable text commands over USB serial. | Must |
| FW-03 | `STATE:<name>` switches to the named state; LED effects and the motor keep running independently of incoming commands. | Must |
| FW-04 | A state's sound plays once when the state is entered. | Must |
| FW-05 | COMPLETED and ERROR stay until the next state command. | Must |
| FW-06 | After 30 s without any command or heartbeat, it stops the motor and shows a distinct "disconnected" LED pattern. | Must |
| FW-07 | After power-up it shows "disconnected" until the first command arrives. | Must |
| FW-08 | WORKING uses the motor speed from the device's current motor tuning; a bridge setting cannot override it. | Must |
| FW-09 | Sound volume is set by serial command from 0% (mute) to 100% (maximum), with 1% increments; the firmware maps slider value `x` with `V=(x/100)^2.5`. | Must |
| FW-10 | Every command is answered with `OK` or an error line; unknown or malformed commands don't change the state. | Must |
| FW-11 | On request, it reports its firmware version and commit (REL-02). | Must |
| FW-12 | Test commands `LED:r,g,b` (show a colour) and `BEEP:<sound>` (play a sound). | Could |
| FW-13 | The firmware stores motor speed, PWM frequency, startup kick and decay mode with a UTC write timestamp in NVS on explicit save, and loads saved values after reboot; before the first save, firmware defaults are active. A failed save is reported. | Must |
| FW-14 | LED brightness can be set from 0% to 100% in 1% increments; for slider value `x`, the firmware uses `b=(x/100)^2.2` to scale each channel after the existing 50% brightness cap. 100% is the default. | Must |
| FW-15 | The firmware can restore motor tuning to firmware defaults and persist those defaults with the supplied UTC timestamp; a failed write is reported. | Must |

## 4. Host bridge (BRG)

A background process on the host that receives agent events from the hooks over local HTTP, merges the state of all sessions and drives the device over USB serial (FW). On Windows it is a tray app with a local web interface (BRG-16 – BRG-18, BRG-20 – BRG-24); WSL2 and devcontainers use its relay. Native Linux support returns in Stage 6 (BRG-19), leaving an intentional gap after the Stage 3 retirement (TD-15).

| ID | Requirement | Prio |
|---|---|---|
| BRG-01 | The bridge receives agent events over local HTTP, each with agent, session id and event. | Must |
| BRG-02 | It tracks the state per session; the device shows the most urgent one, in the order APPROVAL > ERROR > WORKING > THINKING > COMPLETED > IDLE; with no sessions it shows IDLE. | Must |
| BRG-03 | It sends a state command only when the merged state changes, so sounds are not repeated. | Must |
| BRG-04 | It drops a session in WORKING, THINKING or APPROVAL after no event for a configurable time (default 10 min); a session in COMPLETED or ERROR stays until its next event or its end. | Must |
| BRG-05 | It sends a heartbeat to the device at least every 10 s. | Must |
| BRG-06 | Plug & play: it runs without the device; when the device is plugged in (first time or replug) or reset, it finds and uses it within 10 s without a port setting or any manual step on the host (after a one-time host setup), and then resends the settings and the current state. | Must |
| ~~BRG-07~~ | ~~The frozen Python Docker bridge reads its legacy settings (including motor speed, volume, session timeout and optional serial-port override) from its config file. It is not required to work with firmware implementing revised FW-08, FW-09 and FW-13–FW-15; Windows Go bridge settings are specified in BRG-22.~~ (withdrawn: retired with the Python Docker bridge in Stage 3; native Linux support is deferred to Stage 6.) | – |
| BRG-08 | It listens only on the loopback interface and the local Docker/WSL interfaces, never on the LAN. | Must |
| BRG-09 | It starts automatically and runs in the background. | Should |
| BRG-10 | It reports its version and commit (REL-02). | Must |
| BRG-11 | A status command shows whether the device is connected, its firmware version, the sessions with their states, and the merged state. | Should |
| BRG-12 | A state can be set manually from the command line, for testing and demos. | Could |
| BRG-13 | The bridge writes a formatted log (time, level, message): every change of the merged state with its cause and whether it reached the device, also without a device; device connects and losses; at the configurable debug level also every received event and every command and reply on the serial line. | Must |
| BRG-14 | The bridge's log takes at most 2 files of at most 10 MB each. | Must |
| BRG-15 | All sessions, including a manual state, can be cleared from the command line, which sets the merged state to IDLE; for resetting the bridge when a session is stale. | Could |
| BRG-16 | On Windows, the bridge runs as an app in the notification area and starts WSL processes without showing console windows; the user turns its start at login on or off there, without admin rights. | Must |
| BRG-17 | The notification-area icon shows the merged state by its colour. | Should |
| BRG-18 | On Windows, the bridge is installed and uninstalled with an installer, without admin rights. | Must |
| BRG-19 | The bridge runs natively on Linux and macOS. | Should |
| BRG-20 | The Windows tray menu opens Cody's local web interface in the default browser; the bridge does not open a native Cody window. | Must |
| BRG-21 | The web interface shows bridge version and commit, firmware version, device connection, merged state, sessions, each configured WSL distro's relay port, and routing guidance distinguishing WSL-host hooks from Docker Engine devcontainer hooks. | Must |
| BRG-22 | The Windows Go bridge web interface edits and saves the WSL distro list, sound volume and LED brightness as 0–100% sliders with 1% steps, session timeout, optional serial-port override and log level; it has no motor-tuning setting. It clearly indicates that changes require a restart and offers Restart now. | Must |
| BRG-23 | On user request, the updater checks the latest stable Cody GitHub release and shows installed and available firmware versions; release/network errors are reported, not treated as "no update". Flashing requires confirmation before download. It offers an update only when the release is newer, except that an unknown installed version may be updated after explicit confirmation. | Must |
| BRG-24 | The updater verifies the firmware asset's SHA-256 digest before flashing, shows progress and errors, and restores bridge device operation afterward; if automatic bootloader entry fails, it provides manual recovery instructions and retry. | Must |

## 5. Agent integrations (AGT)

Hooks that turn coding-agent lifecycle events into events for the bridge (BRG). The mapping of each agent's events to states is in [architecture.md](architecture.md). The user checked the Copilot hook picker in VS Code 1.147.0 on 2026-11-22: it exposes `ErrorOccurred` but no approval/permission-request event. Copilot payloads include a session ID, but Cody deliberately keys them by workspace folder (`cwd`), as required by AGT-06.

| ID | Requirement | Prio |
|---|---|---|
| AGT-01 | Claude Code sessions drive all six states. | Must |
| AGT-02 | The Claude Code integration works with Claude Code running on Windows, on Linux (including WSL2) and in a devcontainer. | Must |
| AGT-03 | The Claude Code hooks are set up once at user level and apply to every project. | Must |
| AGT-04 | GitHub Copilot agent mode in VS Code drives the states IDLE, THINKING, WORKING and COMPLETED. | Must |
| AGT-05 | GitHub Copilot drives APPROVAL and ERROR, once VS Code provides events for them. | Could |
| AGT-06 | Copilot sessions are told apart by workspace folder. | Must |
| AGT-07 | If the bridge is unreachable, a hook returns within 1 s and never blocks or fails the agent. | Must |
| AGT-08 | No agent event is counted twice (e.g. Copilot also reads Claude Code hook configuration). | Must |
| AGT-09 | The hook setup for both agents is described step by step in the repository. | Should |
| AGT-10 | The Windows Cody installer enables bridge autostart, and the bridge web UI installs Claude Code and Copilot hooks at user level on Windows and in each configured WSL relay distro, checks device connectivity, and reports what is missing. | Must |
| AGT-11 | The same hook configuration works on the host and in devcontainers; the hooks find the bridge on `localhost` or on the Docker host. | Must |
| AGT-12 | The hooks reach the bridge in every WSL2 networking mode and from devcontainers on Docker Desktop and on Docker Engine in WSL2, without changing network settings. | Must |

## 6. Electronics (ELE)

Electronics built from ready-made modules only, powered from the host's USB port.

| ID | Requirement | Prio |
|---|---|---|
| ELE-01 | The electronics consist only of ready-made development boards and breakout modules; no custom PCB and no discrete circuits. | Must |
| ELE-02 | Soldering is limited to header pins and wires onto modules; all other connections are pluggable (JST, Dupont, screw terminals). | Must |
| ELE-03 | The motor is driven only through a ready-made motor driver breakout with PWM, whose continuous current rating is at least 1.5 × the motor's stall current. | Must |
| ELE-04 | The whole device is powered from one USB port of the host through a single cable; its total current, including motor start, stays at or below 500 mA. There is no second supply. | Must |
| ELE-05 | Motor start and switching never reset the ESP32 or interrupt the USB connection. | Must |
| ELE-06 | The purchased parts of one Cody (electronics and mechanics, without filament and tools) cost at most CHF 25, counted per piece used (pack price divided by pack size), without spares and shipping. | Must |
| ELE-07 | A wiring diagram (modules, pins, connectors) is in the repository; the purchased parts are listed in the parts list (DOC-01). | Must |

## 7. Figure and mechanism (FIG)

The printed desk scene and the arm mechanism. The figure is simplified to a bust (head, torso, arms) behind the table; the arms stop anywhere in their cycle (no home position).

| ID | Requirement | Prio |
|---|---|---|
| FIG-01 | The figure is a cartoon developer bust (oversized head, torso, hoodie, two arms) behind a table with a laptop on a platform; no lower body is needed (judged by the owner). | Must |
| FIG-02 | The whole assembly is at most 150 mm high (target 120–130 mm). | Must |
| FIG-03 | All printed parts fit the Prusa MK3S+ build volume (250 × 210 × 210 mm) and are designed for FDM printing in PLA; PETG is allowed for mechanism parts. Prefer a 0.6 mm nozzle with PrusaSlicer 2.9.4; smaller nozzles are possible but should be avoided. | Must |
| FIG-04 | There are at most 12 printed parts. | Must |
| FIG-05 | Parts print without support material, except where noted in the print notes. | Should |
| FIG-06 | Both arms move alternately, driven by one continuously rotating geared DC motor; no servos. | Must |
| FIG-07 | The hands stay close to the laptop keyboard; each hand moves 3–8 mm up and down. | Must |
| FIG-08 | Each arm makes 1–4 strokes per second, set via the motor speed (FW-08). | Must |
| ~~FIG-09~~ | ~~Mechanical stops limit the arm travel.~~ (withdrawn: the owner chose gravity and cam contact without a separate frame endstop.) | – |
| FIG-10 | Motor and cam drive are hidden inside the torso. | Must |
| FIG-11 | No lubrication-intensive or precision-machined parts; two separate, closed, hollow Ø6 mm shoulder tubes mount to the frame, each guiding a circular Ø4 mm rotating arm axle in a Ø4.6 mm bore (0.3 mm radial clearance). Each axle has an integral cam lever at least 4 mm thick whose root covers the full axle end face on the cam side; the arm-knob socket remains open. Each arm has a 4 mm-long male knob that fits into its axle socket. Each tube has a full-length rear mounting shelf and a sturdy upward lip with two through-holes. Two rear-access M3 × 8 socket-head screws per lip pass through Ø3.4 mm clearance holes in the 5 mm backplate and lip; loose M3 hex nuts on the inside are accessible to a wrench. Ø6 × 3.1 mm rear head recesses keep screw heads flush. Each arm's outward-facing side is planar and lies flat on the print bed with its knob pointing up; arms have Ø8 mm, 2 mm-thick shoulder hubs that clear the frame. The motor mount stands on the ground plate without overhangs when the frame is printed flat on that plate. | Must |
| FIG-12 | The mechanism runs 8 h continuously in WORKING without failure; the motor stays below 60 °C. | Must |
| FIG-13 | In WORKING, the noise is not distracting at 1 m in a quiet office (judged by the owner). | Should |
| FIG-14 | Serviceable: the torso opens; the motor and each arm can be replaced without damage; the electronics are reachable through the platform; modules can be swapped without changing printed parts; the USB port is accessible. | Must |
| FIG-15 | The assembly uses screws or snap fits with standard hand tools, no glue. | Should |
| FIG-16 | The assembly stands stable on the desk and doesn't move or tip while the arms move. | Must |
| FIG-17 | CAD sources and print files with their print settings are in the repository. | Must |
| FIG-18 | The mechanism can be assembled step by step without blocking or trapping a part before it is installed. | Must |

## 8. Build documentation (DOC)

Written so that a maker with a 3D printer and a soldering iron can build a Cody without the owner's help. Markdown in the repository; every release carries it as release files. It grows with the stages: the status light (stage 2), the mechanism (stage 4), the full figure (stage 5).

| ID | Requirement | Prio |
|---|---|---|
| DOC-01 | A parts list names every part of one Cody: purchased parts (with source and price), printed parts (file, material, quantity), fasteners and consumables. | Must |
| DOC-02 | Assembly instructions cover the electronics and the setup: soldering, wiring, flashing the firmware, installing the bridge and the hooks. | Must |
| DOC-03 | Assembly instructions cover the mechanism: motor, cam drive, shoulder axle and arm joints, levers, and arms. | Must |
| DOC-04 | Assembly instructions cover printing the parts and assembling the full figure. | Must |
| DOC-05 | Each step of the instructions names the parts and tools it needs and has a picture or render. | Must |
| DOC-06 | The parts list and the assembly instructions are release files, with the version in their names. | Must |
| DOC-07 | Someone other than the owner builds a Cody from the parts list and the instructions. | Should |

## 9. Hardware test panel (TST)

A developer-only bench tool for exercising the firmware and its peripherals. Its controls and local firmware flashing are served by the Windows bridge's local web interface; it does not open USB serial directly.

| ID | Requirement | Prio |
|---|---|---|
| TST-01 | The firmware provides an explicit test mode that gives test commands exclusive control of the motor, LED and buzzer, separate from normal agent state commands. | Must |
| TST-02 | The test panel can apply each of the six normal states as a preset without leaving test mode. | Must |
| TST-03 | The panel can start and stop the motor and set speed (0–255), PWM frequency (1–30 kHz), startup kick (0–100 ms) and DRV8833 fast or slow decay. Changes take effect in RAM for testing and normal WORKING behavior; they are not persisted until explicit Save (TST-09). | Must |
| TST-04 | The panel can show a raw RGB test colour (0–255 per channel), subject to the existing 50% brightness cap and active brightness level; raw LED tests do not change normal state colours. | Must |
| TST-05 | The panel can play the APPROVAL, SUCCESS and WARNING sounds and set volume with a 0–100% slider in 1% steps; 0% is mute and test sounds are one-shot. | Must |
| TST-06 | The developer test panel is part of the Windows bridge's local web interface, uses the bridge's device connection, and displays connection status, replies and errors without opening USB serial itself. | Must |
| TST-07 | Unsaved motor tuning is volatile; saved motor tuning persists across reboot. Exiting test mode clears temporary outputs and stops the test motor while retaining the current motor tuning. Volume and brightness preferences are stored by the bridge and reapplied after reconnect; test-panel volume and LED test brightness changes are temporary. Loss of the device connection stops the motor and enters disconnected behavior. | Must |
| TST-08 | During test mode the bridge remains the sole serial owner and suspends normal device writes while continuing to merge agent events; on panel stop or 10 s without a panel heartbeat, it exits test mode and applies the latest merged state. | Must |
| TST-09 | The bridge website shows the current motor tuning and last-write date, allows tuning changes followed by explicit Save, and provides a factory-reset control that restores firmware defaults. Before the first NVS write it identifies the firmware defaults as never saved; timestamps are shown in local date and time. | Must |
| TST-10 | The bridge test panel supports manual ROM-bootloader flashing of a first-time or existing ESP32-C3, and a tuning-preserving update of a recognized Cody device, without devcontainer USB access. Manual flashing lets the user select an Espressif serial port and guides entry to ROM bootloader mode without requiring a Cody handshake, whether or not the current firmware is recognized. It warns that manual flashing does not back up motor tuning and saved tuning or its timestamp may be lost. The preserving update snapshots motor tuning and timestamp before flashing and restores them after Cody firmware reconnects; `UNSAVED` remains unsaved. The panel accepts a non-empty `.bin` up to 4 MiB, explains that the user must select a combined Cody image, and asks browser confirmation naming the image and target port before flashing it at `0x0`. It reports flash phases, captured output and errors, exits test mode, and restores the latest bridge preferences and merged state after reconnect. | Must |
| TST-11 | The panel can apply a temporary 0–100% brightness override in 1% steps to its current raw RGB test colour, using the existing brightness curve and 50% LED cap. It does not change the bridge's saved brightness preference; ending test mode restores that preference. | Must |

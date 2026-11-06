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

The ESP32 firmware turns simple commands from the bridge into physical behaviour. It knows nothing about agents or IDEs. State table: [idea.md](idea.md) §6.1; protocol details in [architecture.md](architecture.md). Settings (speed, volume, mute) are kept by the bridge and sent after every (re)connect; the firmware stores none.

| ID | Requirement | Prio |
|---|---|---|
| FW-01 | The firmware has the states IDLE, WORKING, THINKING, APPROVAL, COMPLETED and ERROR, with arm, LED and sound behaviour as follows: IDLE – arms stopped, dim blue, no sound; WORKING – alternating typing, pulsating blue, no sound; THINKING – arms stopped, yellow, no sound; APPROVAL – arms stopped, blinking orange, double beep; COMPLETED – arms stopped, green, success sound; ERROR – arms stopped, blinking red, warning sound. | Must |
| FW-02 | It accepts newline-terminated, human-readable text commands over USB serial. | Must |
| FW-03 | `STATE:<name>` switches to the named state; LED effects and the motor keep running independently of incoming commands. | Must |
| FW-04 | A state's sound plays once when the state is entered. | Must |
| FW-05 | COMPLETED and ERROR stay until the next state command. | Must |
| FW-06 | After 30 s without any command or heartbeat, it stops the motor and shows a distinct "disconnected" LED pattern. | Must |
| FW-07 | After power-up it shows "disconnected" until the first command arrives. | Must |
| FW-08 | `SPEED:<n>` sets the motor speed (PWM) used in WORKING. | Must |
| FW-09 | Sound can be muted, and its volume set in at least 3 levels, by serial command. | Should |
| FW-10 | Every command is answered with `OK` or an error line; unknown or malformed commands don't change the state. | Must |
| FW-11 | On request, it reports its firmware version and commit (REL-02). | Must |
| FW-12 | Test commands `LED:r,g,b` (show a colour) and `BEEP:<sound>` (play a sound). | Could |

## 4. Host bridge (BRG)

A background process on the host (Linux, WSL2) that receives agent events from the hooks over local HTTP, merges the state of all sessions and drives the device over USB serial (FW).

| ID | Requirement | Prio |
|---|---|---|
| BRG-01 | The bridge receives agent events over local HTTP, each with agent, session id and event. | Must |
| BRG-02 | It tracks the state per session; the device shows the most urgent one, in the order APPROVAL > ERROR > WORKING > THINKING > COMPLETED > IDLE; with no sessions it shows IDLE. | Must |
| BRG-03 | It sends a state command only when the merged state changes, so sounds are not repeated. | Must |
| BRG-04 | It drops a session in WORKING, THINKING or APPROVAL after no event for a configurable time (default 10 min); a session in COMPLETED or ERROR stays until its next event or its end. | Must |
| BRG-05 | It sends a heartbeat to the device at least every 10 s. | Must |
| BRG-06 | Plug & play: it runs without the device; when the device is plugged in (first time or replug) or reset, it finds and uses it within 10 s without a port setting or any manual step on the host (after a one-time host setup), and then resends the settings and the current state. | Must |
| BRG-07 | Settings (motor speed, volume, mute, session timeout, and optionally a serial port that overrides auto-detection) are read from a config file; the device settings are sent to the device on every connect. | Must |
| BRG-08 | It listens only on the loopback interface and the local Docker/WSL interfaces, never on the LAN. | Must |
| BRG-09 | It starts automatically and runs in the background. | Should |
| BRG-10 | It reports its version and commit (REL-02). | Must |
| BRG-11 | A status command shows whether the device is connected, its firmware version, the sessions with their states, and the merged state. | Should |
| BRG-12 | A state can be set manually from the command line, for testing and demos. | Could |

## 5. Agent integrations (AGT)

Hooks that turn coding-agent lifecycle events into events for the bridge (BRG). The mapping of each agent's events to states is in [architecture.md](architecture.md). GitHub Copilot's agent hooks in VS Code are in Preview and provide no events for "waiting for approval" or "turn failed", and no session id.

| ID | Requirement | Prio |
|---|---|---|
| AGT-01 | Claude Code sessions drive all six states. | Must |
| AGT-02 | The Claude Code integration works with Claude Code running on Linux, in WSL2 and in a devcontainer. | Must |
| AGT-03 | The Claude Code hooks are set up once at user level and apply to every project. | Must |
| AGT-04 | GitHub Copilot agent mode in VS Code drives the states IDLE, THINKING, WORKING and COMPLETED. | Must |
| AGT-05 | GitHub Copilot drives APPROVAL and ERROR, once VS Code provides events for them. | Could |
| AGT-06 | Copilot sessions are told apart by workspace folder. | Must |
| AGT-07 | If the bridge is unreachable, a hook returns within 1 s and never blocks or fails the agent. | Must |
| AGT-08 | No agent event is counted twice (e.g. Copilot also reads Claude Code hook configuration). | Must |
| AGT-09 | The hook setup for both agents is described step by step in the repository. | Should |

## 6. Electronics (ELE)

Electronics built from ready-made modules only, powered from the host's USB port.

| ID | Requirement | Prio |
|---|---|---|
| ELE-01 | The electronics consist only of ready-made development boards and breakout modules; no custom PCB and no discrete circuits. | Must |
| ELE-02 | Soldering is limited to header pins and wires onto modules; all other connections are pluggable (JST, Dupont, screw terminals). | Must |
| ELE-03 | The motor is driven only through a ready-made motor driver breakout with PWM, rated for the motor's stall current with margin. | Must |
| ELE-04 | The whole device is powered from one USB port of the host through a single cable; its total current, including motor start, stays at or below 500 mA. There is no second supply. | Must |
| ELE-05 | Motor start and switching never reset the ESP32 or interrupt the USB connection. | Must |
| ELE-06 | The purchased parts (electronics and mechanics, without filament and tools) cost at most CHF 25. | Must |
| ELE-07 | A bill of materials (part, source, price) and a wiring diagram (modules, pins, connectors) are in the repository. | Must |

## 7. Figure and mechanism (FIG)

The printed desk scene and the arm mechanism. The figure is simplified to a bust (head, torso, arms) behind the table; the arms stop anywhere in their cycle (no home position).

| ID | Requirement | Prio |
|---|---|---|
| FIG-01 | The figure is a cartoon developer bust (oversized head, torso, hoodie, two arms) behind a table with a laptop on a platform; no lower body is needed (judged by the owner). | Must |
| FIG-02 | The whole assembly is at most 150 mm high (target 120–130 mm). | Must |
| FIG-03 | All printed parts print on a Prusa MK3S+ (FDM, 0.4 mm nozzle) in PLA; PETG is allowed for mechanism parts. | Must |
| FIG-04 | There are at most 12 printed parts. | Must |
| FIG-05 | Parts print without support material, except where noted in the print notes. | Should |
| FIG-06 | Both arms move alternately, driven by one continuously rotating geared DC motor; no servos. | Must |
| FIG-07 | The hands stay close to the laptop keyboard; each hand moves 3–8 mm up and down. | Must |
| FIG-08 | Each arm makes 1–4 strokes per second, set via the motor speed (FW-08). | Must |
| FIG-09 | Mechanical stops limit the arm travel. | Must |
| FIG-10 | Motor and crank are hidden inside the torso. | Must |
| FIG-11 | No lubrication-intensive or precision-machined parts; connecting rods are simple wire or printed parts. | Must |
| FIG-12 | The mechanism runs 8 h continuously in WORKING without failure; the motor stays below 60 °C. | Must |
| FIG-13 | In WORKING, the noise is not distracting at 1 m in a quiet office (judged by the owner). | Should |
| FIG-14 | Serviceable: the torso opens; the motor and each arm can be replaced without damage; the electronics are reachable through the platform; modules can be swapped without changing printed parts; the USB port is accessible. | Must |
| FIG-15 | The assembly uses screws or snap fits, no glue. | Should |
| FIG-16 | The assembly stands stable on the desk and doesn't move or tip while the arms move. | Must |
| FIG-17 | CAD sources and print files with their print settings are in the repository. | Must |

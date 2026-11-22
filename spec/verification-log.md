# Cody – Verification Log

Manual verification: inspections, demonstrations, checks on real hardware or in real environments (see [verification.md](verification.md), section 6). The requirement report reads the table below as an extra source of evidence.

- One row per check. **Requirements:** one or more IDs, separated by commas. **Date:** `YYYY-MM-DD`. **Result:** `pass` or `fail`.
- For each requirement only its latest record counts. A failed check is fixed and checked again with a new row; old rows stay as history.
- Add a row only after the check was actually done. Claude never adds rows for checks it didn't observe.

## Records

| Requirements | Date | Environment | Result | Notes |
|---|---|---|---|---|
| REL-01 | 2026-11-06 | GitHub Actions, PR #1 | pass | ci/check ran in the devcontainer on the PR and passed; requirement report in the run summary |
| REL-03 | 2026-11-06 | GitHub Actions, Release candidate 0.1.0 | pass | v0.1.0-rc.1 tagged, pre-release with firmware, compose file and LICENSE; image cody-bridge:0.1.0-rc.1 on GHCR |
| REL-04 | 2026-11-06 | GitHub Actions, Promote v0.1.0-rc.1 | pass | v0.1.0 on the candidate's commit, release with the same files; image 0.1.0 and latest with the candidate's digest |
| REL-05 | 2026-11-06 | GitHub, tag ruleset | pass | deleting and moving v0.1.0-rc.1 rejected; a second candidate for the released 0.1.0 refused |
| REL-06 | 2026-11-06 | WSL2, Docker | pass | release files downloaded; docker compose pulled ghcr.io/gravity981/cody-bridge:0.1.0 and printed "cody-bridge 0.1.0 71d95bc0405f938c3f134de0fdcf8457f483a805" |
| AGT-01, AGT-02, AGT-03, AGT-11 | 2026-11-06 | WSL2 host, Claude Code, bridge container without device | pass | hooks installed at user level with hooks/install-claude-code; a real session drove the states at the bridge (/status); payloads recorded as test fixtures |
| AGT-01, AGT-02, AGT-03, AGT-11 | 2026-11-06 | Devcontainer (Docker Engine in WSL2, shared claude-config volume), Claude Code, bridge container on the host without device | pass | hooks installed once in the claude-config volume with `uv run python hooks/install-claude-code`; bridge found on 172.17.0.1 (host.docker.internal not defined); session and its states shown in /status |
| AGT-04 | 2026-11-22 | VS Code 1.147.0 in WSL2 devcontainer, Windows Cody app | pass | User confirms a real Copilot agent session reached the updated bridge and drove IDLE, THINKING, WORKING, and COMPLETED through the installed hooks |
| FW-07 | 2026-11-21 | Assembled bench, ESP32-C3 firmware | pass | After flashing/rebooting, with no serial command or bridge connection, the LED showed the white disconnected pulse |
| FW-04 | 2026-11-21 | Assembled bench, ESP32-C3 firmware over USB serial | pass | APPROVAL played its double beep on entry; sending `STATE:APPROVAL` again did not repeat the sound |
| FW-06 | 2026-11-21 | Assembled bench, ESP32-C3 firmware over USB serial | pass | After the 30 s command timeout, the white disconnected pulse appeared as specified |
| FW-09 | 2026-11-21 | Assembled bench, ESP32-C3 firmware over USB serial | pass | Volume levels and mute worked; after wiring the coil buzzer through DRV8833 channel B, LED changes produced no audible buzzer noise |
| BRG-16 | 2026-11-21 | Windows, installed Cody app | pass | Selecting "Open config" opened the config and created an empty TOML file |
| AGT-11, AGT-12 | 2026-11-21 | Devcontainer (Docker Engine in WSL2), Claude Code, Windows Cody app, Ubuntu WSL relay | pass | User confirmed hook delivery after installing with `--port 8767`; relay log showed listener at `172.17.0.1:8767` |
| BRG-21, BRG-22 | 2026-11-21 | Windows Cody website | pass | User confirmed the updated hook/relay setup guidance works and session timeout `10` is accepted |
| BRG-20 | 2026-11-21 | Windows Cody tray app | pass | User confirmed "Open Cody" in the tray opened the website |
| FW-01 | 2026-11-22 | Bench assembly | pass | User reports all six state behaviors matched: arms, LED colors/patterns and sounds |
| FW-04 | 2026-11-22 | Bench assembly | pass | User reports APPROVAL, COMPLETED and ERROR each sounded once on entry, repeats did not replay sounds, and IDLE/WORKING/THINKING stayed silent |
| FW-06 | 2026-11-22 | Bench assembly | pass | User confirms after 30 s without commands or heartbeat, the motor stopped and the disconnected LED pattern appeared |
| FW-07 | 2026-11-22 | Bench assembly | pass | User confirms the disconnected pattern appeared after power-up and the first state command switched to its requested state |
| FW-09 | 2026-11-22 | Bench assembly | pass | User confirms 0% was silent, 50% quieter than 100%, and 100% loudest; below 20% was inaudible on this buzzer, consistent with the curve's very low output |
| FW-13 | 2026-11-22 | Bench assembly | pass | User reports saving changed tuning, then power-cycling; all saved values and the UTC write timestamp persisted |
| FW-14 | 2026-11-22 | Bench assembly | pass | User confirms 0%, 50%, and 100% brightness behaved as expected; below 20% appeared off and below 50% the pulsating effects looked stepped |
| BRG-06 | 2026-11-22 | Windows Cody app with assembled ESP32-C3 | pass | User reports the app stayed running without the device and detected first plug, replug and reset within 10 s without manual port selection; settings and current state returned |
| BRG-16 | 2026-11-22 | Windows Cody app | pass | User reports the app appeared in the tray, launched WSL relay without a console window, and allowed autostart to be toggled without admin rights |
| BRG-17 | 2026-11-22 | Windows Cody app | pass | User reports the tray icon changed to the corresponding color for multiple bridge states |
| ELE-07 | 2026-11-22 | Assembled bench, wiring diagram and parts list inspection | pass | User confirms the diagram and listed electronics are correct; the coil wires cross motor connections in the SVG, and the layout could be improved |
| TST-01 | 2026-11-22 | Bench assembly, Windows Cody hardware test panel | pass | User reports test mode retained control while a normal agent state was triggered, then normal control resumed after test mode stopped |
| TST-02 | 2026-11-22 | Bench assembly, Windows Cody hardware test panel | pass | User reports all six state presets applied without leaving test mode |
| TST-03 | 2026-11-22 | Bench assembly, Windows Cody hardware test panel | pass | User reports motor start/stop, tuning controls and decay modes worked, changes affected WORKING, and unsaved tuning did not persist after power-cycle |
| TST-04 | 2026-11-22 | Bench assembly, Windows Cody hardware test panel | pass | User reports raw RGB tests displayed the selected hues and normal state color/brightness returned unchanged after test mode |

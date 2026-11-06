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

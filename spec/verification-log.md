# Cody – Verification Log

Manual verification: inspections, demonstrations, checks on real hardware or in real environments (see [verification.md](verification.md), section 6). The requirement report reads the table below as an extra source of evidence.

- One row per check. **Requirements:** one or more IDs, separated by commas. **Date:** `YYYY-MM-DD`. **Result:** `pass` or `fail`.
- For each requirement only its latest record counts. A failed check is fixed and checked again with a new row; old rows stay as history.
- Add a row only after the check was actually done, by the person who did it. Claude never adds rows for checks it didn't observe.

## Records

| Requirements | Date | Verifier | Environment | Result | Notes |
|---|---|---|---|---|---|
| REL-01 | 2026-11-06 | Nicolas Marty | GitHub Actions, PR #1 | pass | ci/check ran in the devcontainer on the PR and passed; requirement report in the run summary |
| REL-03 | 2026-11-06 | Nicolas Marty | GitHub Actions, Release candidate 0.1.0 | pass | v0.1.0-rc.1 tagged, pre-release with firmware, compose file and LICENSE; image cody-bridge:0.1.0-rc.1 on GHCR |
| REL-04 | 2026-11-06 | Nicolas Marty | GitHub Actions, Promote v0.1.0-rc.1 | pass | v0.1.0 on the candidate's commit, release with the same files; image 0.1.0 and latest with the candidate's digest |
| REL-05 | 2026-11-06 | Nicolas Marty | GitHub, tag ruleset | pass | deleting and moving v0.1.0-rc.1 rejected; a second candidate for the released 0.1.0 refused |
| REL-06 | 2026-11-06 | Nicolas Marty | WSL2, Docker | pass | release files downloaded; docker compose pulled ghcr.io/gravity981/cody-bridge:0.1.0 and printed "cody-bridge 0.1.0 71d95bc0405f938c3f134de0fdcf8457f483a805" |


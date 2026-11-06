# Cody – Verification Log

Manual verification: inspections, demonstrations, checks on real hardware or in real environments (see [verification.md](verification.md), section 6). The requirement report reads the table below as an extra source of evidence.

- One row per check. **Requirements:** one or more IDs, separated by commas. **Date:** `YYYY-MM-DD`. **Result:** `pass` or `fail`.
- For each requirement only its latest record counts. A failed check is fixed and checked again with a new row; old rows stay as history.
- Add a row only after the check was actually done, by the person who did it. Claude never adds rows for checks it didn't observe.

## Records

| Requirements | Date | Verifier | Environment | Result | Notes |
|---|---|---|---|---|---|
| REL-01 | 2026-11-06 | Nicolas Marty | GitHub Actions, PR #<n> (run <link or number>) | pass | ci/check ran in the devcontainer on the PR and passed; requirement report in the run summary |


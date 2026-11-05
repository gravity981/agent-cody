---
name: stage-report
description: Writes the status report for a completed stage using the template in spec/implementation-plan.md, appends it to section 5, updates the stage overview and the topic status, and shows the report in the conversation. Use when all steps of a stage are done and committed.
---

# Stage report

## 1. Check that the stage is complete

- Same preflight as `/step`: branch matches the stage `in progress`, working tree clean.
- Every step of the stage is done, and every requirement planned for the stage is off the "not yet implemented" list.
- Run `ci/check` and read `reports/requirements.md`. It must have no errors. If something is missing or failing, report it and stop: the stage isn't complete.

## 2. Gather

- From the requirement report: requirements implemented in this stage vs. planned; overall implemented vs. active; remaining requirements per stage (take the numbers from the report, don't count by hand).
- From the verification results: counts per level; manual records added during the stage (`spec/verification-log.md`).
- From `git log` of the stage branch and the step summaries: steps done, changed or moved.
- Spec changes during the stage: `git diff <stage base>..HEAD -- spec/`.
- Anything waiting on the user for the next stage: its "Before the stage starts" list and open points in `spec/README.md`.

## 3. Write

- Fill the template in `spec/implementation-plan.md`, section 4, and append it to section 5 (replace "No stage completed yet." on the first report).
- Overview table (section 3): this stage `done`, the next one `next`.
- `spec/README.md`, topics table: status and counts of the topics this stage touched.

## 4. Show

Show the full report in the conversation. Stop; the user reviews and commits, and creates the branch for the next stage.

---
name: step
description: Implements one step of the current stage from spec/implementation-plan.md - preflight checks, implementation, evidence tagged with requirement IDs, "not yet implemented" list updated, ci/check run, then a summary for the user to review and commit. Use for every implementation step. Optional argument - the step ID (e.g. 2b); default is the next open step.
---

# Implement a step

## 1. Preflight (before changing any file)

- Stage `n` = the stage `in progress` in the overview of `spec/implementation-plan.md`. If none: ask, and suggest `/stage-start`.
- `git rev-parse --abbrev-ref HEAD` is `feature/stage-<n>` or `feature/stage-<n>-<text>` (exactly `n`).
- `git status --porcelain` is empty: the previous step is committed.
- If a check fails: stop and ask the user. Don't fix it yourself.
- Step: from the argument, otherwise the first step of stage `n` not covered by earlier summaries or commits (check `git log`). Confirm with the user if unclear.

## 2. Understand

- Read the step and every requirement it lists, plus the design sections they touch.
- List anything unclear, contradictory or missing. Ask all of it now, in one round, with options and a recommendation. Record the answers in the spec before implementing.
- If the step turns out too big or depends on something missing, propose a split or reorder instead of starting.

## 3. Implement

- Implement only this step. Follow `spec/architecture.md` and the conventions of the files around you.
- Add evidence for every requirement of the step, tagged as in `spec/verification.md` (e.g. `[ABC-01] …`). Each piece of evidence verifies what the requirement says, not just that the code runs.
- Requirements that can only be verified manually: describe the check for the user; the user records it in `spec/verification-log.md` after doing it.
- Remove the step's requirement IDs from "Not yet implemented" in `spec/verification.md` (keep ranges tidy).
- If the spec turns out wrong or incomplete: stop, explain, propose the spec change, and continue only after the user agrees.

## 4. Verify

- Run `ci/check` (or, before it exists in stage 1, whatever the step builds). Fix failures caused by the step. Report failures you can't fix; never weaken or skip a check to make it pass.
- Read `reports/requirements.md`: no errors; warnings explained.

## 5. Summary, then stop

```
### Step <id> – <title>

**Changed:** files or areas, one line each.
**Requirements:** <IDs> — evidence: <test / check names> — all passing.
**Verification:** ci/check result; counts per level; requirement report OK / errors.
**Spec changes:** what and why (or none).
**Open:** questions, manual checks to do, follow-ups (or none).
```

Don't commit. Wait for the user to review and commit before the next step. If this was the last step of the stage, say so and suggest `/stage-report`.

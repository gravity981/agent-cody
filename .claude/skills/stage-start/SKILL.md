---
name: stage-start
description: Prepares an implementation stage - checks branch and working tree, goes through the stage's preconditions and steps in spec/implementation-plan.md with the user, records the decisions, and marks the stage as in progress. Use before the first step of a stage. Argument - the stage number.
---

# Start a stage

## 1. Preflight

- Stage number `n`: from the argument, otherwise the stage marked `next` in the overview of `spec/implementation-plan.md`. Confirm with the user if unclear.
- `git rev-parse --abbrev-ref HEAD` must be `feature/stage-<n>` or `feature/stage-<n>-<text>` (exactly `n`: `stage-1` doesn't match `stage-10`).
- `git status --porcelain` must be empty.
- If either check fails: stop, tell the user what you found, and ask them to fix it. Don't create or switch branches, don't commit.
- All earlier stages are `done` in the overview. If not, ask how to proceed.

## 2. Preconditions

For each item under "Before the stage starts":

- settled (decision recorded in the spec) → say so, with where;
- open → ask now, with options and a recommendation; record the answer in the spec, and in "Decisions" in `spec/README.md` only if it changes requirements or the architecture;
- needs something only the user can provide (access, hardware, material, accounts) → list it; ask whether the stage can start without it.

## 3. Steps

- Show the planned steps with their requirements. For each requirement, read it and check that it is clear enough to implement. Ask about every gap; never fill one with an assumption.
- Check that the stage's steps cover each requirement planned for the stage exactly once.
- Ask the user to confirm or adjust the steps. Write changes into the plan.

## 4. Start

- Set the stage's status in the overview to `in progress`.
- Summarise: preconditions (settled / open), confirmed steps, decisions recorded, files changed.
- Stop. The user reviews and commits the stage preparation; then `/step` starts the first step.

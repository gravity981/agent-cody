# CLAUDE.md

## Project

<!-- TODO(blueprint): two or three sentences: what the project is, what it delivers, and its main technologies and tools. -->

The specification in `spec/` is the source of truth; the project's files implement it.

| Document | Content |
|---|---|
| `spec/README.md` | Purpose, document index, conventions, topic status, decisions, open points |
| `spec/requirements.md` | All requirement tables |
| `spec/architecture.md` | Building blocks, interfaces, technology stack with decision records |
| `spec/delivery.md` | Version scheme, CI and its entry points (`ci/check`, `ci/build-release`, …), release candidates and promotion, delivery to users |
| `spec/verification.md` | Verification methods, tagging of evidence, requirement report, "not yet implemented" list |
| `spec/verification-log.md` | Manual verification records |
| `spec/implementation-plan.md` | Stages and steps, preconditions per stage, milestones, status reports |

## Working with the spec

- **Ask, don't assume.** When the spec leaves something open or is ambiguous, ask before implementing. Never fill a gap with your own assumption. Offer concrete options with a recommendation (first, marked as recommended) so the question is quick to answer.
- Once a gap is decided, write the decision into the spec: in the requirement or section it belongs to, plus a row in "Decisions" in `spec/README.md`, so the spec stays complete.
- Requirements live in tables in `spec/requirements.md` with IDs (`| ABC-01 | Requirement | Prio |`, prefix of 2–4 capital letters per topic). Never renumber or reuse an ID; add new ones at the end of the topic; withdraw instead of deleting.
- When implementation shows the spec is wrong or incomplete, point it out and propose a spec change. Don't silently deviate.

## Requirement traceability

- Every active requirement is verified by at least one piece of evidence, and each piece of evidence names the requirement IDs it verifies, using the tag convention in `spec/verification.md` (e.g. a test named `[ABC-01] does something`).
- What can't be automated is recorded in `spec/verification-log.md` by the person who did the check. Never add a record for a check you didn't observe.
- The requirement report (`reports/requirements.md`, last step of `ci/check`) lists requirement → evidence → result. CI fails if an implemented requirement has no passing evidence. Requirements not implemented yet are listed in `spec/verification.md` ("Not yet implemented"); shrink that list as stages are delivered.

## Implementation in stages

- The spec is implemented in stages, each split into steps (1a, 1b, …), as planned in `spec/implementation-plan.md`. Before starting a stage, confirm its preconditions and steps with the user (`/stage-start`).
- **The user creates branches and makes all commits. Never commit, push, tag, or create or switch branches.** (`.claude/settings.json` denies these commands.)
- Each stage is built on a branch named `feature/stage-<n>` or `feature/stage-<n>-<text>`. Before changing any file for a step, check:
  - `git rev-parse --abbrev-ref HEAD` matches the current stage (exactly `n`, so `stage-1` does not match `stage-10`);
  - `git status --porcelain` is empty (the previous step is committed).
  If either check fails, stop and ask the user. Don't fix it yourself.
- After each step: stop with a short summary (what changed, verification and its results, anything open) so the user can review and commit (`/step`).
- **When a stage is complete,** write a status report using the template in `spec/implementation-plan.md` (section 4), append it to section 5 there, update the overview table in section 3, and show the report in the conversation (`/stage-report`).

## Releases

- Release candidates and promotions are started by the user in GitHub Actions. Never start workflows, create tags or releases yourself.
- The workflows are generic; project specifics live in the entry points in `ci/` (contract in `spec/delivery.md`, section 2). Change the workflows only where they are marked `TODO(blueprint)` or after the user agreed.

## Environment

- Development happens in the devcontainer (`.devcontainer/`). Every tool is installed there with a pinned version, never only on a developer's machine. Which environment CI uses is decided in `spec/delivery.md`, section 2.
- Monorepo: one top-level folder per building block, as listed in `spec/architecture.md`, section 3.

<!-- TODO(blueprint): commands, filled in during stage 1 -->

| Command | Does |
|---|---|
| `ci/check` | All automated verification and the requirement report, as in CI |
| `ci/build-release` | Release files into `$RELEASE_DIR` (normally run by the release workflow) |

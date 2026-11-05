# Project Blueprint

A starting point for projects built with Claude Code around four ideas:

- **Spec-based development:** `spec/` is the source of truth; the project's files implement it.
- **Ask, don't assume:** gaps in the spec are asked about, decided by the project owner and written into the spec, never filled with guesses. Claude challenges value and cost and evaluates technologies together with the owner.
- **Requirement traceability:** every requirement has an ID, every piece of evidence (automated test or manual check) names the IDs it verifies, and CI fails when an implemented requirement has no passing evidence.
- **Implementation in stages:** stages and steps planned in the spec; one branch per stage; Claude implements a step and stops; the project owner reviews and commits.

The blueprint is **technology agnostic**. It contains no code and no toolchain: a firmware, CAD, software or documentation project starts from the same files and adds its own technology in stage 1.

The blueprint's version is in `BLUEPRINT_VERSION` (Semantic Versioning; releases are tagged `v<version>` in the blueprint repository). Projects keep the file, so it shows which version they started from.

## Using it for a new project

1. Copy this folder into a new repository, `git init`, and commit it unchanged.
2. Open the folder in VS Code, "Reopen in Container", and start Claude Code.
3. `/blueprint-setup idea.md` — reads your idea file, confirms name and purpose, fills in the placeholders.
4. `/spec-interview` — topics, requirements, value and cost, architecture, technology decisions, verification, stages. Repeat until stage 1 can start.
5. Create the branch `feature/stage-1`, then `/stage-start 1`.
6. `/step` for each step; review and commit after each one.
7. `/stage-report` when the stage is done; merge; next stage.

Releases: run **Release candidate** (input `1.2.0`) in GitHub Actions, verify the candidate, then run **Promote** (input `v1.2.0-rc.N`). See `spec/delivery.md`.

Every place that needs a project decision is marked `TODO(blueprint)`: `grep -rn "TODO(blueprint)" .`

## What's inside

| Path | Content |
|---|---|
| `BLUEPRINT_VERSION` | Version of the blueprint |
| `CLAUDE.md` | How Claude works in the project: ask don't assume, traceability, stages, git rules |
| `.claude/settings.json` | Denies git/gh commands that change branches, commits, tags or releases; shows branch and working-tree state at session start |
| `.claude/skills/` | `/blueprint-setup`, `/spec-interview`, `/stage-start`, `/step`, `/stage-report` |
| `spec/` | Templates: README (conventions, decisions), requirements, architecture (with technology decision records), delivery, verification, verification log, implementation plan |
| `spec/requirements.md` | Two topics every project gets: TRC (requirement report) and REL (CI and releases), planned in stage 1 |
| `.github/workflows/` | `ci.yml` (every PR and push to main), `release-candidate.yml`, `promote.yml` |
| `.devcontainer/` | Base image with Claude Code; the project adds its toolchain |

## Design choices

- **No code in the blueprint.** The requirement report is specified as requirements (TRC-01 – TRC-07) and built in stage 1 in the project's own technology, verified like everything else. The workflows contain only shell steps for git tags and GitHub releases.
- **Project specifics behind entry points.** The workflows call executables in `ci/` (`ci/check`, `ci/build-release`, optional `ci/publish`, `ci/promote`), specified in `spec/delivery.md` section 2 and written in stage 1. What gets checked, built and published, and whether there is a deployment at all, is up to the project. Release files are attached to GitHub releases, so "download only" needs nothing else.
- **CI environment is a project decision.** By default the workflows run the entry points in the devcontainer; a project can run them on the runner or on its own runners instead (`spec/delivery.md` section 2).
- **Promotion never rebuilds.** A release has exactly the candidate's files, and `ci/promote` re-labels what `ci/publish` published.
- **Git rules are enforced, not only requested.** `permissions.deny` blocks commit, push, branch, tag and release commands for Claude; CLAUDE.md explains why.

## The blueprint repository itself

Disable GitHub Actions in the blueprint's own repository (Settings → Actions): its workflows are meant for projects and fail without the `ci/` entry points.

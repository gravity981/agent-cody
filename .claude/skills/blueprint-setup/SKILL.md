---
name: blueprint-setup
description: First-time setup of a project created from the project blueprint, starting from the user's idea file (Markdown). Derives name and purpose from the idea, asks only what the idea leaves open, fills in the placeholders, and hands over to /spec-interview. Use once, right after copying the blueprint into a new repository. Argument - path to the idea file.
---

# Blueprint setup

Turn a fresh copy of the blueprint into the start of a real project. The input is the user's rough idea, usually a Markdown file. Keep questions to a minimum: this skill sets up the frame; the content work happens in `/spec-interview`.

## 1. Starting point

- Idea file: from the argument. If none is given, look for a Markdown file in the repository root that isn't part of the blueprint; if there is none, ask for the path (or for the idea in a few sentences).
- Run `git rev-parse --abbrev-ref HEAD` and `git status --porcelain`. If there is no repository, ask the user to `git init` and commit the unchanged blueprint first, so the setup shows up as its own diff. Don't run these yourself.

## 2. Read and derive

From the idea, derive:

- project name (replaces `<Project>`);
- purpose: what it is, for whom, what it delivers, in two or three sentences;
- kind of project (software, embedded, mechanical/CAD, electronics, documents, mixed).

## 3. One confirmation

Show what you derived in one short block and ask a single question (AskUserQuestion): "Use this?", with "Yes" and "Adjust" as options. Ask about anything else only if the idea doesn't answer it **and** the setup can't proceed without it. Everything about content, scope, technology and stages belongs to the spec interview, not here.

## 4. Fill in

- Move the idea file to `spec/idea.md` (unchanged) and add it to the document index in `spec/README.md` as "the original idea; input to the spec, not part of it".
- Replace `<Project>` in every file.
- Purpose paragraph in `spec/README.md` and the project section of `CLAUDE.md`.
- Replace `README.md` with a short project README: purpose, link to `spec/README.md`, how to start (open in the devcontainer, `ci/check`), how releases work (link to `spec/delivery.md`), and "Created from project blueprint <version>" with the version from `BLUEPRINT_VERSION`.
- Keep `BLUEPRINT_VERSION`: it records which blueprint version the project started from.
- Remove the `TODO(blueprint)` markers you resolved; leave the others.

## 5. Hand over

Summarise what changed, then:

- Ask the user to set up in the GitHub repository settings (you can't and shouldn't): Actions enabled, workflow permissions "Read repository contents"; a tag ruleset for `v*` restricting creation, update and deletion, with bypass only for GitHub Actions (REL-05); optionally branch protection on `main` requiring the "Check" job.
- List the remaining `TODO(blueprint)` markers (`grep -rn "TODO(blueprint)" --exclude-dir=.git .`), grouped by where they get resolved: the spec interview (topics, requirements, architecture, technology decisions, verification, stages) or stage 1 (devcontainer toolchain, CI environment, `ci/` entry points, tag convention, delivery, action SHA pins).
- Recommend `/spec-interview` next. Don't commit; the user reviews and commits.

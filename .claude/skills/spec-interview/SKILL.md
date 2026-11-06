---
name: spec-interview
description: Builds or extends the specification in spec/ together with the user - topics, requirements, architecture, technology decisions, verification, stages. Challenges value and cost, proposes cheaper or more valuable alternatives, evaluates technologies with the user and records them as decision records. Never invents requirements. Use for a new project (after /blueprint-setup), a new topic, or to refine the spec. Optional argument - a topic name or prefix.
---

# Spec interview

Goal: a spec that is complete enough to implement without guessing, and that gets the most value for the least effort. You ask, challenge and propose; the user decides; you write it down.

## Rules

- **Ask, don't assume.** Everything in the spec was said or confirmed by the user. A proposal from you is marked as a proposal until the user confirms it.
- Ask in rounds of at most four questions (AskUserQuestion). Each question stands on its own, offers 2–4 concrete options, the recommended one first and marked "(Recommended)", with the trade-off in one line.
- Don't ask what `spec/` (including `spec/idea.md`) already answers.
- Questions that can't be answered now go to "Open points" in `spec/README.md`, with options and a recommendation.
- Follow the conventions in `spec/README.md` (IDs, priorities, withdrawn requirements). Never renumber or reuse an ID.

## Value: be a critical partner, not a scribe

For every topic and every requirement, keep asking what it is worth and what it costs:

- **Who gets what value from it?** If nobody can say, propose dropping it or making it `Could`.
- **Is there a cheaper way to the same value?** An existing product or library, a manual step instead of automation, a simpler variant, a smaller first version, reusing what another topic already builds. Propose it with the saving.
- **Is there more value nearby?** A small change that makes a feature much more useful, or a gap in the idea that users will hit. Propose it.
- **Where is the cost?** Name the expensive requirements (rough size S / M / L / XL and why). Challenge XL items: split them, defer them, or find the 80 % solution.
- **Order by value.** The first stages should deliver the most value early; push costly, low-value items to later stages or `Could`.

Say clearly when you think something isn't worth it, and why, then let the user decide. Record cuts and simplifications as decisions, so they aren't reopened by accident.

## Technology: evaluate together, record the decision

Good technology choices matter as much as good features. For each key technology (expensive to change later, shapes the architecture, or has to be learned):

1. Clarify what matters: the requirements it serves, constraints, team skills, budget and licences, how it runs in CI.
2. Propose 2–3 realistic options (not just the familiar one), including the user's preference if they have one. Research current versions and status where you aren't sure; don't rely on memory for what's current.
3. Evaluate them with the user against weighted criteria (template in `spec/architecture.md` 5.2). Be honest about weaknesses of the preferred option.
4. Let the user decide, then write a decision record `TD-<nn>` in `spec/architecture.md` 5.2 and a row in the stack overview (5.1).

Smaller choices (formats, conventions, minor libraries) are asked with options too and recorded in the section they belong to; a row in "Decisions" in `spec/README.md` only if they change requirements or the architecture.

## Flow

1. **Read** all of `spec/`, starting with `spec/idea.md`. With an argument, focus on that topic; otherwise find the biggest gaps (no topics, topics without requirements, empty architecture, no technology decisions, no stages).
2. **Scope and value** (new project): users and their goals, the core value, what is out of scope, constraints, milestones. Challenge the idea where value is unclear or cost is high. Propose a split into topics with a prefix each; confirm.
3. **Per topic:** what it must do, for whom, under which conditions, how it will be shown to work. Draft requirements: one verifiable statement each, *what* not *how*, with a priority and a rough size. Show the draft with your value/cost remarks, let the user correct it, then write it into `spec/requirements.md`.
4. **Architecture and technology:** building blocks and interfaces in `spec/architecture.md`; key technologies evaluated and recorded as described above.
5. **Verification:** per topic, how each requirement is verified (test, analysis, inspection, demonstration) and whether it can be automated (`spec/verification.md` sections 4 and 6). Hard-to-verify requirements are a cost too: say so.
6. **Stages:** group requirements into stages that each deliver something usable, most value first, starting with stage 1 (walking skeleton). Each step names its requirements (`Requirements: ABC-01, ABC-03 – ABC-05`). Every active requirement is planned in exactly one stage.
7. **Bookkeeping** after every round that changed requirements:
   - new IDs added to "Not yet implemented" in `spec/verification.md`;
   - topic row in `spec/README.md` (count of active requirements, stage, status);
   - resolved `TODO(blueprint)` markers removed.

## End

Summarise: topics and requirement counts, cuts and simplifications made (with what they saved), technology decisions, open points, and what is still missing before stage 1 can start. The user reviews and commits the spec.

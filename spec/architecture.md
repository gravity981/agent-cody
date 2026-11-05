# Cody – Architecture

How the requirements in [requirements.md](requirements.md) are met. Requirements say *what*; this document says *how*. Every choice that was open is recorded: key technologies as decision records in section 5, everything else as a row in "Decisions" in [README.md](README.md).

<!-- TODO(blueprint): fill the sections through the spec interview (/spec-interview). Drop sections that don't apply, add what the project needs. -->

## 1. Context and scope

What the project delivers, who uses it, what is explicitly out of scope, and the external systems, parts or people it interacts with.

## 2. Constraints

Fixed conditions: standards, regulations, budget, target hardware or platforms, deadlines, existing systems, licences, skills available in the team.

## 3. Overview

The main building blocks and how they fit together (a diagram helps). Each block maps to a top-level folder of the repository:

| Folder | Content | Technology / tool |
|---|---|---|
| `spec/` | Specification | Markdown |
| `ci/` | Entry points for CI and releases (delivery.md 2) | |
| … | | |

## 4. Interfaces

Between the building blocks and to the outside: protocols, file formats, connectors, mechanical interfaces, APIs.

## 5. Technology stack

### 5.1 Overview

| Area | Choice | Version | Decision |
|---|---|---|---|
| | | | TD-01 |

Every tool has a pinned version and is installed in the devcontainer (`.devcontainer/`), never only on someone's machine. How CI gets the same tools is decided in delivery.md 2.

### 5.2 Decision records

One record per key technology: anything that is expensive to change later, shapes the architecture, or the team has to learn. Smaller choices go to "Decisions" in README.md. Records are numbered `TD-<nn>` and never renumbered; a revised decision gets a new record and the old one is marked `superseded by TD-<nn>`.

<!-- Template:

#### TD-01 <Area>: <choice>

**Status:** proposed / decided (YYYY-MM-DD) / superseded by TD-<nn>

**Context:** which requirements need this, constraints that apply, what matters most.

**Options evaluated:**

| Criterion (weight) | Option A | Option B | Option C |
|---|---|---|---|
| Fit with the requirements (high) | | | |
| Maturity, support, community (medium) | | | |
| Team knowledge, learning effort (medium) | | | |
| Cost and licence (medium) | | | |
| Toolchain, automation and CI support (medium) | | | |
| Lock-in, ease of replacing later (low) | | | |

**Decision:** the choice and the deciding reasons.

**Consequences:** what follows from it, risks, and what would make us revisit it.
-->

## 6. Version information

How a built artifact reveals its version and commit (REL-02): e.g. a `--version` output, a field in a file header, a title block, a device register, an About page.

## 7. Risks and open technical questions

Known risks with mitigation; questions that are still open go to "Open points" in [README.md](README.md).

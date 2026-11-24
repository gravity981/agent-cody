# Cody - Mechanical design workflow

Claude Code and GitHub Copilot must use this workflow for new or revised mechanical parts, mechanisms, fits, and print preparation. Do not jump from requirements to detailed CAD. This document defines the sequence and gates; six specialist skills supply the working procedures.

This is a development process, not a second product specification. [Requirements](requirements.md), [architecture](architecture.md), [verification](verification.md), and the [implementation plan](implementation-plan.md) remain authoritative. Ask the owner about gaps or conflicts before proceeding.

## 1. Integration and entry

- `/spec-interview` covers requirements analysis. `/stage-start`, `/step`, and `/stage-report` still control stage preparation, branch/clean-tree checks, implementation, checks, review, and commits. Mechanical gates do not replace them or authorize starting another step.
- On entry, identify the affected assembly, active step, requirement IDs, current design revision, and existing evidence. Start at the earliest phase without current, sufficient evidence, not automatically at CAD.
- Load the skill for that phase from the table below using the assistant's supported mechanism; slash-command text alone does not execute another skill. Direct specialist invocation still requires checking prerequisite gates and returning to the earliest missing one.
- Existing work is not automatically approved. Reuse demonstrated evidence and recorded owner decisions; mark missing checks pending. Do not invent historical approvals or repeat sound work merely to populate a template.
- Scale the depth to the risk. A small fit change can use a short calculation and existing concept; a new mechanism needs alternatives and fuller analysis. Every phase is still accounted for.
- Stop at a failed gate. State the missing evidence, decision, measurement, or tool and the next action. An unavailable CAD viewer, slicer, reviewer, or printer is a blocker, not a passing check.
- This workflow does not authorize hardware operation, printing, purchases, tool installation, commits, or releases. Obtain any necessary permission through the existing process.

## 2. Phases and gates

| Phase | Skill / existing process | Work and exit gate |
|---|---|---|
| 1. Requirements analysis | `spec-interview` | Identify functions, interfaces, envelopes, motion, loads, service, manufacturing constraints, and acceptance methods from the spec. Resolve blocking gaps with the owner. Link requirement IDs and measurable criteria; do not invent targets. |
| 2. Mechanical concept development | `mechanical-concept` | Compare feasible concepts, including simpler solutions, against the requirements. Show motion, load paths, component layout, assembly/disassembly, and likely print orientations. Owner confirms the selected concept before dimensioning it in detail. |
| 3. Engineering calculations and dimensioning | `mechanical-engineering` | Establish critical dimensions from sourced inputs, kinematics, loads, drive capability, FDM strength considerations, and tolerance stacks. Record units, uncertainty, margins, and sensitivity. Owner approves the dimensioning basis and explicitly identified provisional fits with their validation plan before detailed CAD. Blocking unknowns remain blockers. |
| 4. CAD implementation | `mechanical-cad` | Implement only the approved concept and dimensioning basis with the existing parametric toolchain. Produce identified parts, assembly reference, print-oriented exports, and focused automated checks. A successful export alone does not pass inspection. |
| 5. CAD inspection | `mechanical-cad` (separate inspection pass) | Inspect dimensions, sections, critical interfaces, motion envelope, assembly/service paths, and fastener/tool/wire access. Inspect actual exported meshes as well as native geometry. Resolve defects and link inspection evidence before independent review. |
| 6. Independent verification | `mechanical-verification` | A fresh read-only reviewer context or a separate human reviewer checks requirements, calculations, CAD, and evidence independently. Resolve blocking findings and obtain the reviewer's disposition on fixes. Record reviewer, revision, scope, results, and physical limitations. Author self-review does not satisfy this gate. |
| 7. Manufacturing review | `mechanical-print-review` | Slice each changed printable part with the specified printer/material/nozzle/profile. Review actual toolpaths, orientation, strength-critical walls, small features, holes, supports, and bed fit. Record settings and outstanding fit-coupon needs. Gate passes for prototype manufacture, not proven physical performance. |
| 8. Physical prototyping and testing | `mechanical-prototype` | The human operator prints coupons or the minimum useful prototype and performs the planned fit, assembly, motion, and applicable bench tests. Record actual conditions, measurements, and outcomes. Missing physical results remain pending; failures go back to the earliest affected phase. |
| 9. Design refinement and release | `mechanical-prototype` plus existing release process | Refine from observed results and rerun affected gates. Reconcile CAD, tested print settings, BOM, assembly/print notes, evidence, and compatibility. Declare mechanical release readiness only when applicable physical checks and blocking reviews pass. The owner alone starts candidates and promotions under [delivery.md](delivery.md). |

Phases 4 and 5 share one skill to avoid duplicate context, but must have separate results. Review of source code alone is not geometric inspection. Phase 6 verifies the claims possible before manufacture; it cannot pass a requirement that needs Phase 8.

Exploratory sketches, calculations, and disposable envelope/kinematic geometry may support Phases 2 and 3. Label them exploratory, not print-ready or approved detailed CAD. If early fit data is needed, run a small coupon iteration through proportionate Phases 1-7, obtain physical measurements in Phase 8, then feed them into the parent design's dimensioning. This is not a shortcut around review.

## 3. FDM rules throughout the workflow

### Concept and dimensioning

- Read FIG-03 and FIG-05 for the current printer, material, nozzle, slicer, and support constraints. The current baseline is Prusa MK3S+, 250 x 210 x 210 mm, PrusaSlicer 2.9.4, preferably a 0.6 mm nozzle; PLA generally, PETG permitted for mechanism parts. Smaller nozzles should be avoided. Reconcile these facts with the spec if it changes.
- Choose part splits and orientations while choosing the concept. Balance load paths and layer adhesion against support avoidance, bed contact, assembly access, service, and the complete figure's part budget. Give each printed part a defined orientation with a planar bed-contact face.
- Check feature feasibility against actual extrusion widths and layer heights, not nozzle diameter alone. Consider thin tube walls, lever roots, small knobs, holes, bridges, overhangs, elephant foot, and seams at sliding contacts.
- Account for directional strength, repeated loading, stress concentrations, sustained-load creep, temperature, wear, and sliding friction as applicable. Bulk or injection-molded material data does not establish FDM performance; document uncertainty and the needed test.
- Distinguish nominal mating geometry, intended functional clearance/interference, manufacturing compensation, and expected process variation. Label radial versus diametral values. Check worst-case tolerance stacks where binding, retention, or assembly depends on them.
- Measure purchased parts and use relevant project evidence or fit coupons for clearances and hole compensation. Printed holes can be undersized. Do not prescribe universal tolerances, a universal hole correction, or untested press-fit retention.

### Assembly and modeling

- Design the assembly and removal sequence before finalizing geometry. No component may block a later installation, tool approach, or required service path.
- Use screws or snap fits, no glue. M3 screws engage metal threads, not printed plastic. Use the specified inserts or nuts, with the full screw stack, engagement, head clearance, driver/wrench access, and no bottoming out.
- For the current Stage 4a layout, follow [architecture.md section 8.3](architecture.md#83-preferred-concept-chest-cam-drive): rear-access crossbar screws and loose wrench-held nuts, not inserts or captured nuts. Do not generalize this stage-specific choice to other joints.
- Retain electronics with snug, removable printed features, not screws or glue, preserving FIG-14 access and replacement.
- Check moving parts, fasteners, wires, connectors, and tools throughout assembly and the full motion cycle. Distinguish intended contact from interference; endpoint-only or single-pose checks do not prove full-cycle clearance.
- Do not add pockets, lattices, thin webs, or hollowing solely to save filament. Keep functional cavities, and keep structural supports solid unless a design requirement says otherwise; let slicer infill manage material use.
- Use the existing build123d/manifold3d toolchain. Separate parts when material, orientation, service, or assembly requires it, while considering total part count. Do not add dependencies without need.

### Slicing and physical evidence

- Inspect the actual exported meshes in their intended orientations for scale, integrity, lost features, and build-volume fit. Inspect sliced layers, not just a shaded exterior or estimated print success.
- Confirm critical walls and small features have usable toolpaths; inspect hole shape, bridging, support contact/removal, mating surfaces, and access after printing. Resolve slicing errors. Record any support exception in the print notes.
- Record the actual printer, slicer version, nozzle, material, layer height, extrusion widths, perimeters, infill, supports, and relevant compensation settings. Record print-time/material estimates when comparing alternatives; never trade away fit or strength just to reduce them.
- A fit coupon must reproduce the relevant feature, orientation, material, and process. Identify coupon variants and record measured outcomes so the chosen compensation is reproducible.
- CAD clearance, an FDM heuristic, a slicer preview, and an AI review are not physical proof of fit, retention, contact, noise, stability, or endurance.

## 4. Lean records and evidence

Keep one compact gate record per affected assembly in its existing design/print notes (currently [cad/README.md](../cad/README.md)). Link authoritative spec sections and actual calculations, inspections, review results, slicer evidence, and tests rather than copying requirements or creating nine reports.

Use this structure when starting real mechanical work; do not prefill passing results:

```markdown
### Mechanical gates - <assembly / change>

**Scope:** <step, requirement IDs, affected parts>
**Revision:** <commit plus identification of relevant uncommitted changes, if any>
**Basis:** <links to requirements, approved concept, dimensions, and input sources>

| Phase | Status | Evidence / limitations | Approval or reviewer |
|---|---|---|---|
| 1-9: one row per phase | pending / passed / blocked / invalidated | link and result | identity/date where required |

**Next gate:** <phase and action>
**Physical checks pending:** <checks, acceptance criteria source, required operator/input>
```

- Each evidence item names the requirement IDs it actually verifies, using [verification.md](verification.md) conventions. A test must evaluate the required behavior, not merely repeat a design constant.
- Calculations record inputs and their sources, units, method/equations, results, limits, and margins. Identify measured, specified, derived, and provisional inputs. Do not invent material allowables, motor torque, friction coefficients, tolerances, or safety factors; agree missing decision-critical values with the owner.
- Review records identify the reviewer and exact reviewed revision/artifacts, method, findings, and disposition. Author inspection and independent review remain distinguishable.
- Physical records identify the tested part revision, print settings/material, setup, operator, criteria, and observations/measurements. Only performed checks belong in [verification-log.md](verification-log.md); pending test plans stay in design notes.
- The gate record supplements the existing requirement report; CI does not currently enforce these gates. Green CI alone does not establish mechanical release readiness. Keep IDs on "Not yet implemented" while required physical or other evidence is missing, even if a partial automated check passes.
- A gate summary at a step boundary is enough: current phase, changed parts, evidence/results, blockers, and next action. Do not claim completion of the whole lifecycle when only a CAD step is complete.

## 5. Independence, iteration, and release boundaries

Use a fresh read-only AI reviewer context when available, or a separate human reviewer. Supply the authoritative requirements, revision/artifacts, input sources, and acceptance criteria; the reviewer inspects the actual design and independently derives critical checks, rather than only endorsing the author's summary or reusing its assertions. A new role label in the author's conversation is not independence. No particular model or agent product is required.

If neither reviewer is available, mark Phase 6 blocked and ask the owner to arrange review. Do not silently substitute self-review. Record AI review as AI review, not professional certification or physical evidence.

When geometry, interfaces, material, orientation, slicer settings, loads, or requirements change, identify and invalidate affected evidence before continuing. For example, an axle bore change reopens tolerance analysis, CAD inspection, independent review, slicing, and physical fit; changing orientation also reopens layer-strength and compensation assumptions. Unaffected evidence can be retained with a reason. Repeat from the earliest affected phase, not always from Phase 1.

Physical tests may be scheduled in later planned steps (for example Stage 4b endurance). A CAD step may end with those lifecycle gates explicitly pending; it does not mark the associated requirements verified. Owner-approved deferral remains a deferral, never a pass or an unqualified mechanical release-readiness claim.

Phase 9 assesses readiness within the active scope. It must not execute a later stage or start a release workflow. The existing candidate verification and owner-controlled promotion remain necessary.

## 6. Assistant discovery and smoke checks

Skill procedures have one canonical copy in [.claude/skills/](../.claude/skills/), the project location for [Claude Code](https://code.claude.com/docs/en/skills). Thin discovery files in [.github/skills/](../.github/skills/), the native project location for [Copilot in VS Code](https://code.visualstudio.com/docs/agent-customization/agent-skills), link to those canonical procedures. This avoids depending on harness-specific discovery of another assistant's directories or filesystem symlinks. [CLAUDE.md](../CLAUDE.md) and [Copilot instructions](../.github/copilot-instructions.md) route mechanical work here; do not duplicate the skill bodies.

After changing skills, check discovery and routing in a fresh chat for each assistant:

1. Confirm the six specialist mechanical skills in the phase table are available.
2. Ask for a new mechanism without a concept: expect requirements/concept work, not detailed CAD.
3. Ask to change an axle fit: expect evidence reuse plus reopened dimensioning and downstream gates.
4. Ask for independent verification without a separate reviewer: expect a blocked gate, not author self-approval.
5. Ask whether passing CAD tests prove a physical fit: expect pending physical validation.

These are manual assistant smoke checks, not checks already performed or automated enforcement. Instruction files guide the assistants; they cannot guarantee model compliance.

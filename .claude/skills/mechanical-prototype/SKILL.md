---
name: mechanical-prototype
description: Plans and evaluates physical FDM coupons/prototypes, routes mechanical refinements through affected gates, and assesses mechanical release readiness. Use for fit, assembly, motion, endurance, service, and print validation; never invent physical results or start releases.
---

# Physical prototypes, refinement, and readiness - Phases 8 and 9

Follow [the mechanical workflow](../../../spec/mechanical-workflow.md). Check current Phase 1-7 evidence before treating a design as ready to print. Work only within the active stage/step.

## Phase 8 - Prototype and test

1. Prepare the minimum useful coupon/prototype test plan. Name requirement IDs, tested revision, setup/tools, operator, controlled conditions, measurements, acceptance criteria sources, and applicable stop conditions. Ask the owner about missing criteria instead of inventing thresholds.
2. Start with unresolved fits and highest-risk interfaces before a full assembly. Use the reviewed material/orientation/profile and record any actual deviations, including relevant printer calibration or post-processing.
3. Have the human operator print, measure, and assemble the prototype in the documented order. Check fit, retention, hand-tool access, removal/service, and motion without forcing or trapping parts.
4. Perform only the physical checks planned for the active scope. Motion/travel/rate/contact, current/startup stability, noise, temperature/endurance, and stability need their applicable bench or complete-assembly tests. Use the existing test panel/process; do not improvise unapproved hardware operation.
5. Capture actual observations and measurements with setup, operator, part revision, and print settings. Request missing measurements explicitly. Never infer physical success from CAD, slicing, or an AI review.
6. Record only performed checks in `spec/verification-log.md` under its conventions. Keep pending plans in design notes, distinguish partial checks from full requirement verification, and leave incomplete requirements on "Not yet implemented."

## Phase 9 - Refine and assess readiness

1. Classify each failure by evidence: concept, sizing/load, geometry, fit compensation, print process, or assembly. Do not compensate for an unexplained failure with arbitrary clearance changes.
2. Propose the smallest corrective change. Obtain approval for changed requirements, architecture, or dimensioning decisions as required. Invalidate affected gates/evidence and repeat from the earliest affected phase, including independent review and physical retest where affected.
3. Reconcile the tested CAD/exports, materials/settings, parts list, assembly order, service access, print notes, and compatibility. Run the active step's required checks and inspect its requirement report.
4. Report mechanical readiness only for the stated scope with passing applicable gates and physical evidence. Later-stage or owner-deferred checks remain explicitly pending, not passed.
5. Hand off to the existing owner-controlled candidate/release process when ready. Do not start workflows, create tags/releases, commit, or execute a later step.

**Exit:** measured results, remaining risks/checks, refinement disposition, and a qualified readiness assessment. Stop for the owner's review and commit.

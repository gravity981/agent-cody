---
name: mechanical-print-review
description: Reviews FDM manufacture using actual oriented exports and sliced toolpaths for Cody's printer, material, and nozzle. Use after independent design verification and before prototype printing; also reopen after relevant geometry or print-setting changes.
---

# Manufacturing review - Phase 7

Follow [the mechanical workflow](../../../spec/mechanical-workflow.md) and its FDM rules. Check current Phase 1-6 evidence first. Earlier concept-phase printability estimates do not replace this review.

1. Identify the exact exports and approved printer/material/profile. Follow FIG-03/05 and the existing print notes; do not silently substitute another nozzle, material, or slicer version.
2. Confirm per-part orientation, planar bed contact, units/scale, mesh integrity, and actual build-volume fit.
3. Slice every changed printable part with PrusaSlicer using the specified profile (prefer the 0.6 mm nozzle). Resolve slicing errors. If the slicer/profile is unavailable, report the blocker; do not substitute a generic overhang calculation for slicing.
4. Inspect layer/toolpath previews, especially tube walls, lever roots, small knobs, bores, mating surfaces, first-layer expansion, bridges, seams, support interfaces/removal, and tool-access holes. Confirm critical features produce usable paths rather than disappearing or becoming unsupported.
5. Reconcile orientation and perimeters with the dimensioning load paths and layer-strength assumptions. Return to engineering/CAD if support avoidance or profile changes undermine them.
6. Record actual nozzle, material, layer height, extrusion widths, perimeter count, infill, support settings/exceptions, and relevant compensation. Link reproducible profile/project files and preview evidence as appropriate; update the existing print notes rather than duplicating them.
7. Select the smallest useful coupon/prototype batch. Coupons reproduce the fit feature, orientation, and process and clearly identify variants. Record print-time/material estimates when they help choose an alternative, without adding material-saving CAD pockets or lattices.

**Exit:** reviewed sliced revision/settings, findings/resolution, print notes, and coupon/test handoff. This gate establishes prototype manufacturing readiness only; physical fit and performance remain unverified.

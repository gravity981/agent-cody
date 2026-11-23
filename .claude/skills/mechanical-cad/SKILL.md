---
name: mechanical-cad
description: Implements an approved mechanical concept and dimensioning basis in parametric CAD, then performs a separate geometric inspection. Use for build123d/manifold3d modeling, exports, sections, motion/collision checks, and assembly access inspection.
---

# CAD implementation and inspection - Phases 4 and 5

Follow [the mechanical workflow](../../../spec/mechanical-workflow.md), especially its assembly and FDM rules. Check current Phase 1-3 approvals before detailed modeling; return to the earliest phase lacking current evidence. An existing model is not a substitute for them.

## Phase 4 - Implement

1. Read the active step, approved dimensions/interfaces, related models/tests, and print notes. Reuse the existing build123d/manifold3d conventions and toolchain.
2. Keep functional dimensions and fit compensation identifiable and parametric. Implement the approved assembly/print strategy; do not silently alter requirements or invent a new joint to make geometry fit.
3. Model enough purchased-part, fastener, connector, wire, and tool envelopes to check access. Preserve intended service/removal paths and distinguish print aids from assembly components.
4. Build the assembly and required exports. Identify part names, units, assembly placement, and per-part print orientation. Add focused requirement-tagged geometric/behavior checks for the active change.
5. Record the implemented revision and exports. A build/export success finishes implementation, not inspection.

## Phase 5 - Inspect separately

1. Inspect native geometry using views and sections, not only code or a shaded assembly. Check critical measured dimensions, wall/root thicknesses, bores, sockets, fastener engagement/head clearance, and the approved envelope.
2. Check all relevant component pairs through the full motion cycle and during assembly/removal. Include wires and tool approach/turn/removal. Classify designed contacts separately from unintended interference; document sampling limits if checks are discrete.
3. Inspect the actual exported meshes for units/scale, validity, missing features, and agreement with native geometry. Check each part's intended bed-contact face, oriented build-volume fit, and product part count.
4. Use the available CAD viewer/inspection tools. If visual or sectional inspection cannot be performed, record it as pending/blocked; do not claim tests or screenshots prove an inspection that did not occur.
5. Fix defects, regenerate affected exports, rerun focused checks, and record separate inspection results and limitations. Run `ci/check` at the step boundary as required by `/step`.

**Exit:** separate Phase 4 and 5 evidence, inspected revision/exports, and unresolved physical assumptions. Hand off to independent verification; the author cannot approve Phase 6 by inspecting their own work.

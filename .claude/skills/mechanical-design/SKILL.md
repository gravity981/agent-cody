---
name: mechanical-design
description: Designs or revises Cody's 3D-printed mechanisms, parts, fits and assembly sequences in CAD. Use for mechanical CAD work and printability reviews.
---

# Mechanical design

## Before designing

- Read the relevant requirements in `spec/requirements.md`, the design in `spec/architecture.md`, and the active step in `spec/implementation-plan.md`. The spec is authoritative; do not silently change a requirement to make a design fit.
- Review the related models and conventions in `cad/` and use the existing build123d/manifold3d toolchain. Do not add a dependency without need.
- If an interface, tolerance, assembly sequence, material, or requirement is unclear or conflicting, ask the owner before modeling it. Offer concrete options and a recommendation, then record the decision in the appropriate spec section.
- When working within an implementation stage, follow the stage and step preflight and verification process.

## Assembly and interfaces

- Design the complete assembly sequence before finalizing geometry. Every part must be installable in sequence without another part blocking access, and no part may be trapped so the mechanism cannot be completed, serviced, or disassembled as required.
- Make the assembly possible with standard hand tools. Use screws or snap fits, never glue.
- M3 screws must engage M3 threaded inserts with a 4.5 mm outside diameter and 6 mm length. Model insert pockets for the actual insert dimensions and installation method; do not use screws directly in printed plastic. Select screw lengths for the full stack without bottoming out.
- Retain electronic modules with snug, removable printed features. Do not use screws or glue for electronic components; preserve the access and replacement requirements in FIG-14.
- Check moving parts, fasteners, wires, connectors, and tool access for collisions throughout assembly and motion. Provide a clear removal path for serviceable parts.

## FDM design rules

- Use the Prusa MK3S+ and PrusaSlicer 2.9.4. Design primarily for a 0.6 mm nozzle.
- FIG-03 currently requires compatibility with a 0.4 mm nozzle. Until that requirement is changed with the owner, keep every part printable with both 0.6 mm and 0.4 mm nozzles. Do not rely on a nozzle smaller than 0.4 mm; smaller nozzles are an exception to avoid and must be justified.
- Verify that each part fits the MK3S+ build volume in its intended print orientation. Check the actual printer limits rather than assuming a bed or height dimension.
- Account for FDM tolerances in every fit: joints, shafts, holes, insert pockets, module mounts, snap fits, and moving clearances. Measure purchased components and use project evidence or fit coupons to determine clearances; do not apply an unverified universal tolerance.
- Consider nozzle width, wall/perimeter thickness, layer direction, bridging, overhangs, elephant foot, and support needs. Follow FIG-05 and record any required supports and print orientation in the print notes.
- Use materials permitted by FIG-03: PLA generally and PETG for mechanism parts where allowed. Avoid geometry that depends on glue, precision machining, or lubrication-intensive contact.

## Model and verify

- Keep the model parametric and buildable with the existing CAD package. Separate parts that need different materials, print orientations, service, or assembly steps.
- Check dimensions, part count, collisions, motion limits, build-volume fit, insert and electronics clearances, and the full assembly order in CAD.
- Export the required print files and settings. Slice every part with the 0.6 mm profile in PrusaSlicer 2.9.4 and verify the 0.4 mm compatibility required by FIG-03. Resolve slicing errors and review supports, perimeters, holes, and fit-critical features.
- Use a small printed fit coupon or prototype when a tolerance or retention force cannot be verified from existing evidence. Report untested physical fit as unverified; do not claim manual checks that were not performed.
- Add or update requirement-tagged automated evidence and print notes as required by the active step. Run its specified checks, including `ci/check` when the step requires it.
- Summarize the assembly sequence, critical fits and tolerances, material and nozzle profiles, changed parts, verification results, and remaining physical checks.

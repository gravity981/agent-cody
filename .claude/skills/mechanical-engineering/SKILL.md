---
name: mechanical-engineering
description: Calculates and dimensions Cody's mechanisms before detailed CAD, including kinematics, drive loads, FDM strength risks, critical fits, and tolerance stacks. Use for sizing changes or validating the engineering basis of a mechanical concept.
---

# Engineering calculations and dimensioning - Phase 3

Follow [the mechanical workflow](../../../spec/mechanical-workflow.md). Check approved requirements and concept evidence first; return to the earliest phase lacking current evidence.

1. List inputs with units, sources, and status: specified, measured, derived, or provisional. Use actual purchased-part drawings/measurements. Ask about decision-critical gaps rather than inventing loads, torque, material allowables, friction, tolerances, or safety factors.
2. Analyze only applicable failure modes, explaining exclusions. For the chest cam, check displacement through the cycle, alternating phase, stroke frequency, follower contact/acceleration, required drive torque including startup/friction, axle/lever loading, and the motor/torso envelope.
3. Derive critical dimensions from requirements and load paths. Show equations/method, results versus limits, uncertainty, margins, and sensitivity to the important inputs. A geometric clearance calculation does not establish adequate torque or physical follower contact.
4. Consider printed-layer strength, stress concentrations, creep, temperature, wear, and repeated loading. Use relevant printed-material evidence where available; do not certify strength from bulk-material values or unsupported simulation precision.
5. For each critical fit, distinguish nominal geometry, functional clearance/interference, process compensation, and variability. State radial versus diametral values and analyze worst-case mating stacks. Do not treat an untested clearance as proven retention or free rotation.
6. Plan representative coupons or measurements for unknown fits; include orientation, material/profile, variants, and acceptance criteria sourced from requirements or confirmed with the owner.
7. Keep calculations reproducible with the existing tools, adding focused requirement-tagged checks when implementing the active step. Link the calculation evidence instead of copying it into several documents.
8. Present the dimensioning basis, margins, blockers, and provisional fits/test plans to the owner for approval before detailed CAD. Record chosen dimensions in the appropriate design section, not as new requirements unless the owner explicitly changes them.

**Exit:** approved dimensioning basis, sourced inputs, calculation results, and explicit validation needs. Missing decisive inputs or failed limits block the gate; route concept failures back to Phase 2.

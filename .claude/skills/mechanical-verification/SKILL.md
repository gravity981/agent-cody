---
name: mechanical-verification
description: Arranges or performs independent mechanical verification against requirements, sourced calculations, and actual CAD evidence. Use after author CAD inspection; requires a fresh read-only reviewer context or separate human reviewer, never author self-approval.
---

# Independent mechanical verification - Phase 6

Follow [the mechanical workflow](../../../spec/mechanical-workflow.md). Confirm current Phase 1-5 evidence and identify the author and reviewed revision.

## Reviewer boundary

- If you authored the design or are continuing its authoring context, arrange a fresh read-only reviewer context using an available review facility, or request a separate human review. A role change in this conversation is not independent.
- Give the reviewer the requirements/criteria, relevant architecture, actual CAD/exports, input sources, calculations, and inspection/test evidence. Do not limit the review to the author's summary or direct it to confirm a desired conclusion.
- If no separate reviewer is available, mark the gate blocked and stop. Do not create a review record implying a reviewer ran.
- When acting as the separate reviewer, remain read-only. Report findings; the author fixes them and returns changed evidence for review.

## Review procedure

1. Trace the claimed results to the actual requirements and inspect the artifacts, not just test names or CAD constants.
2. Independently derive critical checks from sourced inputs: applicable kinematics, units, loads/torque, margins, tolerance stacks, and FDM failure risks. Identify unsupported assumptions and evidence that shares the same erroneous premise as the implementation.
3. Check geometry, sections, motion coverage, assembly/service/tool paths, and consistency of exports with the inspected revision. State any tool or inspection limitation.
4. Distinguish nominal geometric correctness from manufacturing and physical performance. Slicing and physical checks occur later; list them without marking their requirements passed.
5. Report concrete findings with affected part/location, requirement, consequence, and evidence. Identify blocking findings and any missing verification.
6. After fixes, review the changed artifacts and affected checks; record disposition. Approval applies only to the identified revision and reviewed scope.

**Exit:** reviewer identity/context, revision/artifacts, independent methods/results, findings/disposition, and limitations linked in the gate record. Only the separate reviewer can supply the independent review outcome.

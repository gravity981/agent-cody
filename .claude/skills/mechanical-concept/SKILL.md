---
name: mechanical-concept
description: Develops and compares FDM-oriented mechanical concepts before detailed dimensioning or CAD. Use for mechanism selection, part layout, load paths, assembly/service concepts, or revisiting a rejected mechanical approach.
---

# Mechanical concept development - Phase 2

Follow [the mechanical workflow](../../../spec/mechanical-workflow.md). First check Phase 1 evidence; use `spec-interview` if requirements need clarification. A direct invocation does not bypass prerequisites.

1. Read the affected requirements, interfaces, architecture decisions, active step, and existing concept history. Do not reopen an approved concept without a reason.
2. For a new concept, compare 2-3 feasible approaches, including a simpler option. For a small revision, explain why the existing concept still applies instead of inventing alternatives.
3. Sketch motion, component envelopes, load paths, interfaces, and the complete assembly/disassembly sequence. Check fastener, tool, wire, and service access before choosing geometry.
4. Propose part splits, allowed materials, and print orientations together. Compare layer loading, self-supporting geometry, nozzle-resolvable features, critical fits, wear, part count, purchased parts, and prototype effort.
5. Use exploratory geometry only where it resolves a concrete concept question; label it non-production. Do not produce detailed print-ready CAD.
6. Present trade-offs against the existing acceptance criteria, principal risks/unknowns, the cheapest useful experiment, and a recommendation. Do not disguise estimated scores or dimensions as measured evidence.
7. Obtain the owner's concept decision and record it in the appropriate architecture section; add a spec decision row only when required by its conventions.

**Exit:** link the approved concept, rejected alternatives/reasons, assembly/print strategy, and open inputs for dimensioning. If the owner has not chosen, mark the gate pending and stop.

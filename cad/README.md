# cad/

build123d models, head mesh pipeline (Meshy.ai, manifold3d) and model checks ([spec/architecture.md](../spec/architecture.md) TD-05). Requirements: FIG in [spec/requirements.md](../spec/requirements.md). Python package `cody-cad` in the uv workspace.

## Chest-cam prototype (Stage 4a)

The parametric mechanism and export entry point are in `src/cody_cad/mechanism.py`.
From the repository root, export the printable components and assembled STEP reference with:

```sh
uv run --frozen python -m cody_cad.mechanism --output /tmp/cody-mechanism
```

To display the assembled mechanism in the OCP CAD Viewer, run the VS Code task
`CAD: View chest-cam mechanism` (`Terminal > Run Task`). It runs `view_mechanism.py`
with the workspace's pinned Python dependencies. Open the OCP CAD Viewer view first if
it is not already running.
Assembly components appear by part name; the GA12-N20 motor is gray, the Z axis is up,
orbit controls are enabled, and the camera resets to fit the model. The shaft-fit coupon
is a separate print aid, not an assembly component.

The supplied GA12-N20 drawing gives a 12 x 10 mm body envelope, a 9 mm gearbox and
15 mm motor can, and a 3 x 10 mm output shaft (34 mm overall). The CAD motor is modeled
as this conservative envelope for clearance checks. The motor axis is 9.7 mm forward
and 8.8 mm below the shoulder axle; the resulting lever geometry gives about 4.96 mm
calculated hand travel and 1.50 mm minimum outer-axle clearance. The D-shaft profile and
motor cradle still require physical fit validation.

The assembly clearance test checks every pair of the nine assembled components. The frame
includes the provisional base but no arm endstops; gravity keeps the lever tips on the cams.
The two crossbars, arms and arm axles are separate parts. The shaft-fit coupon is a separate
print aid and is not part of the assembly.

Slice the STLs with PrusaSlicer 2.9.4 and the preferred 0.6 mm nozzle; smaller nozzles are
possible but should be avoided. Check each sliced part fits the MK3S+ volume
(250 x 210 x 210 mm). The two closed Ø6 mm crossbars have different lengths and mount to
the frame's inner face along their mounting rails. Separate Ø4 mm arm axles rotate inside
their Ø4.6 mm bores with 0.3 mm radial clearance; an integral Ø5.2 mm collar abuts the
inner end of each tube. Each outside arm has a 4 mm-long male Ø2.6 mm knob that enters its axle's Ø2.7 mm
socket with 0.1 mm diametral clearance. The axle levers are integral, at least 4 mm thick,
and their roots cover the full Ø4 mm axle end face on the cam side while leaving the arm
socket open. Each crossbar has a full-length rear
shelf and a sturdy upward lip with two through-holes. Secure it with two M3 x 8 socket-head
screws inserted from the rear through Ø3.4 mm clearance holes in the backplate and lip. The
screw heads fit Ø6 x 3.1 mm rear counterbores. Hold standard M3 hex nuts (5.5 mm across
flats, 2.4 mm thick) with a wrench on the inside of the lip while tightening from the rear.
The 5 mm backplate has no inserts or nut seats. Print each arm on its planar outward-facing
side with its knob pointing upward; its Ø8 mm shoulder hub is 2 mm thick. The frame is
oriented with its ground plate flat on the print bed; a solid block supports the motor mount
from the plate without overhangs. Leave filament-use optimization to the slicer's infill.
Inspect the lip holes, screw access and base in the slicer preview.

The shaft-fit coupon is a 4 x 4 matrix. Place the single notched corner at front-left: rows from
front to back use flat depths 1.0, 1.1, 1.2 and 1.3 mm; columns from left to right use radial
clearances 0.0, 0.1, 0.2 and 0.3 mm. Press the selected coupon hole onto the motor's D-shaft
before choosing the cam bore. The motor fit, the motor cradle fit, and the clearance estimates
still need physical validation.

Proposed prototype assembly order: for each arm, slide the closed crossbar over the axle
from its outer end until the integral collar meets the tube, then insert the arm's 4 mm
male knob into the axle socket. Place the assembled module against the frame, insert two
M3 x 8 socket-head screws from the rear through the backplate and crossbar lip, and hold
loose M3 hex nuts with a wrench on the inside while tightening with an Allen key. Install
the motor in its cradle and fit the double eccentric to its D-shaft.
Confirm the complete sequence works with standard hand tools without forcing, blocking
tool access or trapping any part. The printed fit, screw access, movement and stability
still need physical validation; record only checks actually performed in
`../spec/verification-log.md`.

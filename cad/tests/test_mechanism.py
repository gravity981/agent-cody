from itertools import combinations
from math import isclose, sqrt
from pathlib import Path

from build123d import Box, Cylinder, Pos, RegularPolygon, Rot, extrude
from cody_cad.mechanism import (
    CAM_CENTERS_X,
    CAM_LEVER_ROOT_FACE_OVERLAP,
    CAM_LEVER_ROOT_FACE_THICKNESS,
    CROSSBAR_MOUNT_LIP_THICKNESS_Y,
    CROSSBAR_NUT_FACE_Y,
    CROSSBAR_SCREW_TIP_Y,
    CROSSBAR_SCREW_Z,
    DIMENSIONS,
    M3_CLEARANCE_DIAMETER,
    M3_NUT_ACROSS_FLATS,
    M3_NUT_HEIGHT,
    M3_SOCKET_HEAD_DIAMETER,
    M3_SOCKET_HEAD_HEIGHT,
    M3_SOCKET_HEAD_POCKET_DEPTH,
    M3_SOCKET_HEAD_POCKET_DIAMETER,
    PRUSA_MK3S_BUILD_VOLUME,
    arm_crossbar_layout,
    build_arm,
    build_arm_axle,
    build_arm_crossbar,
    build_assembly_parts,
    build_cam_lever,
    build_double_eccentric,
    build_motor_reference,
    build_print_parts,
    build_test_frame,
    build_test_rig,
    cam_lobe_centers,
    export_print_parts,
    hand_travel_mm,
    outer_axle_motor_clearance_mm,
    strokes_per_second,
)


def test_req_FIG_06_one_motor_drives_two_counterphased_cam_lobes() -> None:
    centers = cam_lobe_centers()
    offsets = tuple(center[2] - DIMENSIONS.motor_axis_z for center in centers)

    assert len(centers) == 2
    assert offsets[0] == -offsets[1]
    rig = build_test_rig()
    assert rig.is_valid
    assert len(rig.solids()) == len(build_assembly_parts())
    cam = build_double_eccentric()
    arms = (
        build_arm(-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        build_arm(DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
    )
    assert all((arm & cam).volume <= 1e-6 for arm in arms)


def test_req_FIG_07_hand_travel_is_within_the_required_range() -> None:
    assert 3.0 <= hand_travel_mm() <= 8.0
    assert isclose(hand_travel_mm(), 4.958762886597938)


def test_req_FIG_08_each_arm_strokes_within_the_required_rate() -> None:
    assert 1.0 <= strokes_per_second() <= 4.0
    assert isclose(strokes_per_second(), 2.0)


def test_req_FIG_11_two_hollow_crossbars_have_arm_specific_lengths() -> None:
    left = build_arm_crossbar(-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0])
    right = build_arm_crossbar(DIMENSIONS.shoulder_position, CAM_CENTERS_X[1])
    left_bounds = left.bounding_box().size
    right_bounds = right.bounding_box().size
    frame = build_test_frame()
    left_midpoint_x = (
        -DIMENSIONS.shoulder_position - 1.1 + CAM_CENTERS_X[0] - DIMENSIONS.cam_width / 2
    ) / 2
    wall_probe = Pos(left_midpoint_x, 4.6, DIMENSIONS.crossbar_axis_lift_z) * Box(0.1, 0.1, 0.1)
    bore_probe = Pos(left_midpoint_x, 0, DIMENSIONS.crossbar_axis_lift_z) * Box(0.1, 0.1, 0.1)

    assert left.is_valid and right.is_valid
    assert left_bounds.X > right_bounds.X
    assert isclose(left_bounds.X, 36.1, abs_tol=1e-5)
    assert isclose(right_bounds.X, 16.7, abs_tol=1e-5)
    assert (left & wall_probe).volume > 0
    assert (left & bore_probe).volume == 0
    assert (left & frame).volume == 0
    assert (right & frame).volume == 0


def test_req_FIG_11_crossbar_lift_preserves_frame_geometry() -> None:
    frame = build_test_frame()
    bounds = frame.bounding_box()

    assert isclose(DIMENSIONS.crossbar_axis_lift_z, 2.0)
    assert isclose(frame.volume, 38768.325287, abs_tol=0.001)
    assert isclose(bounds.min.X, -40.0)
    assert isclose(bounds.min.Y, -12.5)
    assert isclose(bounds.min.Z, -33.0)
    assert isclose(bounds.max.X, 40.0)
    assert isclose(bounds.max.Y, 47.5)
    assert isclose(bounds.max.Z, 13.0)


def test_req_FIG_11_crossbar_bore_has_0_3mm_radial_axle_clearance() -> None:
    bore_diameter = DIMENSIONS.arm_axle_bore_diameter
    axle_diameter = DIMENSIONS.arm_axle_diameter
    shoulder_x = -DIMENSIONS.shoulder_position
    cam_x = CAM_CENTERS_X[0]
    layout = arm_crossbar_layout(shoulder_x, cam_x)
    crossbar = build_arm_crossbar(shoulder_x, cam_x)
    center_x = (layout.outer_end_x + layout.inner_end_x) / 2
    bore_probe = Pos(center_x, 4.2, DIMENSIONS.crossbar_axis_lift_z) * Box(0.1, 0.1, 0.1)
    wall_probe = Pos(center_x, 4.6, DIMENSIONS.crossbar_axis_lift_z) * Box(0.1, 0.1, 0.1)

    assert isclose(DIMENSIONS.crossbar_diameter, 10.0)
    assert isclose(bore_diameter, 8.6)
    assert isclose(axle_diameter, 8.0)
    assert isclose((bore_diameter - axle_diameter) / 2, 0.3)
    assert (crossbar & bore_probe).volume == 0
    assert (crossbar & wall_probe).volume > 0


def test_req_FIG_11_crossbars_have_full_length_shelves_and_upward_lips() -> None:
    for shoulder_x, cam_x in (
        (-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        (DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
    ):
        layout = arm_crossbar_layout(shoulder_x, cam_x)
        crossbar = build_arm_crossbar(shoulder_x, cam_x)
        lip_y = DIMENSIONS.backplate_inner_face_y + CROSSBAR_MOUNT_LIP_THICKNESS_Y / 2
        shelf_probes = (
            Pos(
                layout.outer_end_x + (layout.inner_end_x - layout.outer_end_x) * fraction,
                -4.5,
                3.5,
            )
            * Box(0.1, 0.1, 0.1)
            for fraction in (0.05, 0.5, 0.95)
        )
        lip_probes = (
            Pos(
                layout.outer_end_x + (layout.inner_end_x - layout.outer_end_x) * fraction,
                lip_y,
                10,
            )
            * Box(0.1, 0.1, 0.1)
            for fraction in (0.05, 0.5, 0.95)
        )

        assert all((crossbar & probe).volume > 0 for probe in shelf_probes)
        assert all((crossbar & probe).volume > 0 for probe in lip_probes)


def test_req_FIG_11_rear_screws_pass_through_backplate_and_crossbar_lip() -> None:
    frame = build_test_frame()
    assert isclose(DIMENSIONS.backplate_thickness, 5.0)
    assert isclose(frame.bounding_box().min.Y, -12.5)

    rear_face_y = DIMENSIONS.backplate_rear_face_y
    inner_face_y = DIMENSIONS.backplate_inner_face_y
    for shoulder_x, cam_x in (
        (-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        (DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
    ):
        layout = arm_crossbar_layout(shoulder_x, cam_x)
        crossbar = build_arm_crossbar(shoulder_x, cam_x)
        for screw_x in layout.screw_positions_x:
            screw_path = (
                Pos(
                    screw_x,
                    (inner_face_y + CROSSBAR_SCREW_TIP_Y) / 2,
                    CROSSBAR_SCREW_Z,
                )
                * Rot(-90, 0, 0)
                * Cylinder(
                    M3_CLEARANCE_DIAMETER / 2,
                    CROSSBAR_SCREW_TIP_Y - inner_face_y,
                )
            )
            backplate_hole = (
                Pos(screw_x, DIMENSIONS.backplate_center_y, CROSSBAR_SCREW_Z)
                * Rot(-90, 0, 0)
                * Cylinder(
                    M3_CLEARANCE_DIAMETER / 2,
                    DIMENSIONS.backplate_thickness + 0.2,
                )
            )
            head_pocket = (
                Pos(
                    screw_x,
                    rear_face_y + M3_SOCKET_HEAD_POCKET_DEPTH / 2,
                    CROSSBAR_SCREW_Z,
                )
                * Rot(-90, 0, 0)
                * Cylinder(
                    M3_SOCKET_HEAD_POCKET_DIAMETER / 2,
                    M3_SOCKET_HEAD_POCKET_DEPTH,
                )
            )
            head_probe = (
                Pos(
                    screw_x,
                    rear_face_y + M3_SOCKET_HEAD_HEIGHT / 2,
                    CROSSBAR_SCREW_Z,
                )
                * Rot(-90, 0, 0)
                * Cylinder(M3_SOCKET_HEAD_DIAMETER / 2, M3_SOCKET_HEAD_HEIGHT)
            )
            nut_fit_probe = (
                Pos(screw_x, CROSSBAR_NUT_FACE_Y, CROSSBAR_SCREW_Z)
                * Rot(-90, 0, 0)
                * extrude(
                    RegularPolygon(M3_NUT_ACROSS_FLATS / sqrt(3), 6),
                    amount=M3_NUT_HEIGHT,
                )
            )
            wrench_access_probe = Pos(
                screw_x,
                CROSSBAR_NUT_FACE_Y + 3,
                CROSSBAR_SCREW_Z,
            ) * Box(7, 6, 5.5)
            tool_probe = (
                Pos(
                    screw_x,
                    rear_face_y + (M3_SOCKET_HEAD_POCKET_DEPTH - 10) / 2,
                    CROSSBAR_SCREW_Z,
                )
                * Rot(-90, 0, 0)
                * Cylinder(1.25, 10 + M3_SOCKET_HEAD_POCKET_DEPTH)
            )
            assert (crossbar & screw_path).volume <= 1e-6
            assert (frame & screw_path).volume <= 1e-6
            assert (frame & backplate_hole).volume <= 1e-6
            assert (frame & head_pocket).volume <= 1e-6
            assert (frame & head_probe).volume <= 1e-6
            assert (crossbar & nut_fit_probe).volume <= 1e-6
            assert (crossbar & wrench_access_probe).volume <= 1e-6
            assert (frame & tool_probe).volume <= 1e-6
            assert CROSSBAR_SCREW_TIP_Y > CROSSBAR_NUT_FACE_Y + M3_NUT_HEIGHT


def test_req_FIG_11_arms_print_flat_with_arm_sockets_up() -> None:
    parts = build_print_parts()
    for shoulder_x, name in (
        (-DIMENSIONS.shoulder_position, "left_arm"),
        (DIMENSIONS.shoulder_position, "right_arm"),
    ):
        side = 1 if shoulder_x > 0 else -1
        arm = parts[name]
        bounds = arm.bounding_box()
        socket_open_probe = Pos(
            shoulder_x,
            0,
            DIMENSIONS.arm_shoulder_hub_thickness - 0.1,
        ) * Box(0.1, 0.1, 0.1)
        socket_floor_probe = Pos(shoulder_x, 0, 0.5) * Box(0.1, 0.1, 0.1)
        forearm_z = -20 + (10 - 1) / (DIMENSIONS.hand_reach - 1) * (-DIMENSIONS.hand_drop + 20)
        outer_side_probes = (
            Pos(shoulder_x, 0, 0.05) * Box(0.1, 0.1, 0.1),
            Pos(shoulder_x + side * -12.25, 1, 0.05) * Box(0.1, 0.1, 0.1),
            Pos(shoulder_x + side * forearm_z, 10, 0.05) * Box(0.1, 0.1, 0.1),
            Pos(shoulder_x + side * -DIMENSIONS.hand_drop, 36, 0.05) * Box(0.1, 0.1, 0.1),
        )

        assert abs(bounds.min.Z) <= 1e-6
        assert (arm & socket_open_probe).volume == 0
        assert (arm & socket_floor_probe).volume > 0
        assert all((arm & probe).volume > 0 for probe in outer_side_probes)


def test_req_FIG_11_arm_shoulder_hubs_clear_the_mounting_lips() -> None:
    for shoulder_x, cam_x in (
        (-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        (DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
    ):
        arm = build_arm(shoulder_x, cam_x)
        crossbar = build_arm_crossbar(shoulder_x, cam_x)

        assert (arm & crossbar).volume <= 1e-6


def test_req_FIG_11_cam_levers_are_at_least_4mm_thick() -> None:
    for cam_x in CAM_CENTERS_X:
        cam_center = next(center for center in cam_lobe_centers() if center[0] == cam_x)
        cam_top_z = cam_center[2] + DIMENSIONS.cam_diameter / 2
        start_y = 0.0
        start_z = DIMENSIONS.crossbar_axis_lift_z
        end_z = cam_top_z + DIMENSIONS.lever_thickness / 2
        beam_length = sqrt((DIMENSIONS.lever_length - start_y) ** 2 + (end_z - start_z) ** 2)
        lever = build_cam_lever(cam_x)

        assert lever.volume / (3 * beam_length) >= 4.0 - 1e-6


def test_req_FIG_11_nested_axles_have_integral_levers_and_collars() -> None:
    parts = build_print_parts()
    for name in ("left_arm_axle", "right_arm_axle"):
        assert parts[name].is_valid
        assert len(parts[name].solids()) == 1


def test_req_FIG_11_levers_cover_the_full_cam_side_axle_face() -> None:
    for shoulder_x, cam_x in (
        (-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        (DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
    ):
        side = 1 if shoulder_x > 0 else -1
        axle = build_arm_axle(shoulder_x, cam_x)
        face_x = cam_x - side * (
            CAM_LEVER_ROOT_FACE_THICKNESS / 2 - CAM_LEVER_ROOT_FACE_OVERLAP / 2
        )
        face_probes = (
            Pos(face_x, 1.8, DIMENSIONS.crossbar_axis_lift_z) * Box(0.1, 0.1, 0.1),
            Pos(face_x, -1.8, DIMENSIONS.crossbar_axis_lift_z) * Box(0.1, 0.1, 0.1),
            Pos(face_x, 0, DIMENSIONS.crossbar_axis_lift_z + 1.8) * Box(0.1, 0.1, 0.1),
            Pos(face_x, 0, DIMENSIONS.crossbar_axis_lift_z - 1.8) * Box(0.1, 0.1, 0.1),
        )

        assert all((axle & probe).volume > 0 for probe in face_probes)


def test_req_FIG_11_arms_are_continuous_push_fit_parts() -> None:
    parts = build_print_parts()
    for name in ("left_arm", "right_arm"):
        assert parts[name].is_valid
        assert len(parts[name].solids()) == 1


def test_req_FIG_11_arm_sockets_fit_outward_axle_knobs() -> None:
    assert isclose(DIMENSIONS.arm_axle_collar_diameter, 9.2)
    assert isclose(DIMENSIONS.arm_knob_diameter, 6.0)
    assert isclose(DIMENSIONS.arm_knob_socket_diameter, 6.2)
    assert isclose(DIMENSIONS.arm_knob_length, 8.0)
    assert isclose(DIMENSIONS.arm_knob_socket_depth, 8.3)
    assert isclose(DIMENSIONS.arm_shoulder_hub_radius * 2, 11.0)
    assert isclose(DIMENSIONS.arm_shoulder_hub_thickness, 9.3)

    for shoulder_x, cam_x in (
        (-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        (DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
    ):
        side = 1 if shoulder_x > 0 else -1
        layout = arm_crossbar_layout(shoulder_x, cam_x)
        arm = build_arm(shoulder_x, cam_x)
        axle = build_arm_axle(shoulder_x, cam_x)
        knob_center_x = layout.outer_end_x + side * DIMENSIONS.arm_knob_length / 2
        knob_probe = Pos(
            knob_center_x,
            0,
            DIMENSIONS.crossbar_axis_lift_z,
        ) * Box(0.1, 0.1, 0.1)
        radial_gap_probe = Pos(
            knob_center_x,
            3.05,
            DIMENSIONS.crossbar_axis_lift_z,
        ) * Box(0.1, 0.02, 0.1)
        hub_wall_probe = Pos(
            knob_center_x,
            3.5,
            DIMENSIONS.crossbar_axis_lift_z,
        ) * Box(0.1, 0.1, 0.1)
        knob_tip_probe = Pos(
            layout.outer_end_x + side * (DIMENSIONS.arm_knob_length - 0.1),
            0,
            DIMENSIONS.crossbar_axis_lift_z,
        ) * Box(0.1, 0.1, 0.1)
        socket_floor_probe = Pos(
            layout.outer_end_x + side * (DIMENSIONS.arm_knob_socket_depth + 0.5),
            0,
            DIMENSIONS.crossbar_axis_lift_z,
        ) * Box(0.1, 0.1, 0.1)

        assert (axle & knob_probe).volume > 0
        assert (arm & knob_probe).volume == 0
        assert (axle & radial_gap_probe).volume == 0
        assert (arm & radial_gap_probe).volume == 0
        assert (arm & hub_wall_probe).volume > 0
        assert (axle & hub_wall_probe).volume == 0
        assert (axle & knob_tip_probe).volume > 0
        assert (arm & socket_floor_probe).volume > 0
        assert (axle & socket_floor_probe).volume == 0


def test_req_FIG_11_assembly_components_have_no_geometric_interference() -> None:
    parts = build_assembly_parts()
    assert set(parts) == {
        "Test frame",
        "Left crossbar",
        "Right crossbar",
        "Left arm",
        "Right arm",
        "Left arm axle",
        "Right arm axle",
        "Double eccentric",
        "GA12-N20 motor",
    }

    for (name, shape), (other_name, other_shape) in combinations(parts.items(), 2):
        overlap_volume = (shape & other_shape).volume
        assert overlap_volume <= 1e-6, (
            f"{name} intersects {other_name} by {overlap_volume:.6f} mm^3"
        )


def test_req_FIG_11_motor_envelope_matches_the_supplied_dimension_drawing() -> None:
    bounds = build_motor_reference().bounding_box().size
    assert isclose(bounds.X, 34.0)
    assert isclose(bounds.Y, 12.0)
    assert isclose(bounds.Z, 10.0)


def test_req_FIG_11_printed_mechanism_parts_are_valid_and_fit_the_prusa_bed() -> None:
    assert isclose(outer_axle_motor_clearance_mm(), 1.8797, abs_tol=0.001)
    assert outer_axle_motor_clearance_mm() >= 1.8

    for part in build_print_parts().values():
        assert part.is_valid
        bounds = part.bounding_box().size
        dimensions = (bounds.X, bounds.Y, bounds.Z)
        assert all(
            size <= limit for size, limit in zip(dimensions, PRUSA_MK3S_BUILD_VOLUME, strict=True)
        )


def test_frame_base_is_lowest_and_has_a_flush_back_face() -> None:
    frame = build_test_frame()
    arms = (
        build_arm(-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        build_arm(DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
    )

    assert min(arm.bounding_box().min.Z for arm in arms) > frame.bounding_box().min.Z
    assert isclose(
        frame.bounding_box().min.Z,
        DIMENSIONS.frame_base_center_z - DIMENSIONS.frame_base_thickness / 2,
    )
    assert isclose(frame.bounding_box().min.Y, -12.5)


def test_req_FIG_11_motor_mount_support_is_solid_and_stands_on_the_ground_plate() -> None:
    frame = build_test_frame()
    base_top_z = DIMENSIONS.frame_base_center_z + DIMENSIONS.frame_base_thickness / 2
    cradle_bottom_z = DIMENSIONS.motor_axis_z - (DIMENSIONS.motor_body_height + 4) / 2
    support_z = (base_top_z + cradle_bottom_z) / 2

    for x_offset in (-12, 0, 12):
        for y_offset in (-7.5, 0, 7.5):
            probe = Pos(
                DIMENSIONS.motor_body_center_x + x_offset,
                DIMENSIONS.motor_axis_y + y_offset,
                support_z,
            ) * Box(0.1, 0.1, 0.1)

            assert (frame & probe).volume > 0

    former_pocket_probe = Pos(
        DIMENSIONS.motor_body_center_x - 9.5,
        DIMENSIONS.motor_axis_y - 5.25,
        support_z,
    ) * Box(0.2, 0.2, 0.2)
    assert isclose((frame & former_pocket_probe).volume, former_pocket_probe.volume)


def test_req_REL_06_versioned_cad_stls_are_flat_and_exclude_step(tmp_path: Path) -> None:
    version = "1.2.3"
    exported = export_print_parts(tmp_path, version=version, include_step=False)

    expected_names = {
        f"cody-{name.replace('_', '-')}-{version}.stl" for name in build_print_parts()
    }
    assert {path.name for path in exported} == expected_names
    assert all(
        path.parent == tmp_path and path.is_file() and path.stat().st_size > 0 for path in exported
    )
    assert not list(tmp_path.glob("*.step"))

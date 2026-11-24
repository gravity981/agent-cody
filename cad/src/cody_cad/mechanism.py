"""Code-CAD model for the Stage 4a chest-cam mechanism test rig."""

from __future__ import annotations

import argparse
from dataclasses import dataclass
from math import asin, degrees, sqrt
from pathlib import Path

from build123d import (
    Box,
    Compound,
    Cylinder,
    Pos,
    Rot,
    Shape,
    export_step,
    export_stl,
)


@dataclass(frozen=True)
class MechanismDimensions:
    chest_width: float = 62.0
    crossbar_diameter: float = 10.0
    crossbar_relief: float = 0.3
    arm_axle_diameter: float = 8.0
    arm_axle_collar_diameter: float = 9.2
    arm_axle_collar_width: float = 1.0
    shoulder_position: float = 32.0
    crossbar_axis_lift_z: float = 2.0
    arm_shoulder_hub_radius: float = 5.5
    arm_shoulder_hub_thickness: float = 9.3
    arm_knob_diameter: float = 6.0
    arm_knob_fit_diametral_clearance: float = 0.2
    arm_knob_length: float = 8.0
    arm_knob_socket_depth: float = 8.3
    backplate_center_y: float = -10.0
    backplate_thickness: float = 5.0
    frame_base_width: float = 80.0
    frame_base_depth: float = 60.0
    frame_base_thickness: float = 4.0
    frame_base_center_y: float = 17.5
    frame_base_center_z: float = -31.0
    motor_body_center_x: float = -7.0
    motor_gearbox_length: float = 9.0
    motor_can_length: float = 15.0
    motor_body_width: float = 12.0
    motor_body_height: float = 10.0
    motor_rpm: float = 120.0
    motor_axis_y: float = 9.7
    motor_axis_z: float = -8.8
    motor_shaft_diameter: float = 3.0
    motor_shaft_length: float = 10.0
    cam_diameter: float = 10.0
    cam_width: float = 4.0
    cam_offset: float = 0.65
    lever_length: float = 9.7
    lever_thickness: float = 4.0
    hand_reach: float = 37.0
    hand_drop: float = 27.0
    shaft_flat_from_center: float = 1.0
    shaft_clearance: float = 0.2

    @property
    def motor_body_length(self) -> float:
        return self.motor_gearbox_length + self.motor_can_length

    @property
    def arm_axle_bore_diameter(self) -> float:
        return self.arm_axle_diameter + 2 * self.crossbar_relief

    @property
    def arm_knob_socket_diameter(self) -> float:
        return self.arm_knob_diameter + self.arm_knob_fit_diametral_clearance

    @property
    def backplate_inner_face_y(self) -> float:
        return self.backplate_center_y + self.backplate_thickness / 2

    @property
    def backplate_rear_face_y(self) -> float:
        return self.backplate_center_y - self.backplate_thickness / 2


DIMENSIONS = MechanismDimensions()
CAM_CENTERS_X = (7.2, 12.2)
SHAFT_FLAT_CANDIDATES = (1.0, 1.1, 1.2, 1.3)
SHAFT_CLEARANCE_CANDIDATES = (0.0, 0.1, 0.2, 0.3)
PRUSA_MK3S_BUILD_VOLUME = (250.0, 210.0, 210.0)
FOLLOWER_RADIUS = 0.8
M3_CLEARANCE_DIAMETER = 3.4
M3_NUT_ACROSS_FLATS = 5.5
M3_NUT_HEIGHT = 2.4
M3_SOCKET_HEAD_DIAMETER = 5.5
M3_SOCKET_HEAD_HEIGHT = 3.0
M3_SOCKET_HEAD_POCKET_DIAMETER = 6.0
M3_SOCKET_HEAD_POCKET_DEPTH = 3.1
CROSSBAR_MOUNT_END_OFFSET = 4.3
CROSSBAR_MOUNT_LIP_THICKNESS_Y = 2.5
CROSSBAR_MOUNT_LIP_BOTTOM_Z = 2.4
CROSSBAR_MOUNT_LIP_TOP_Z = 11.5
CROSSBAR_MOUNT_SHELF_FRONT_Y = 2.2
CROSSBAR_MOUNT_SHELF_BOTTOM_Z = 2.4
CROSSBAR_MOUNT_SHELF_TOP_Z = 4.0
CROSSBAR_REAR_WEB_CENTER_Y = -5.5
CROSSBAR_REAR_WEB_DEPTH_Y = 2.4
CROSSBAR_REAR_WEB_BOTTOM_Z = 1.2
CROSSBAR_REAR_WEB_TOP_Z = 2.6
CROSSBAR_SCREW_Z = 8.0
CROSSBAR_NUT_FACE_Y = DIMENSIONS.backplate_inner_face_y + CROSSBAR_MOUNT_LIP_THICKNESS_Y
M3_SCREW_LENGTH = 8.0
CROSSBAR_SCREW_TIP_Y = (
    DIMENSIONS.backplate_rear_face_y + M3_SOCKET_HEAD_POCKET_DEPTH + M3_SCREW_LENGTH
)
CAM_LEVER_ROOT_FACE_THICKNESS = 1.2
CAM_LEVER_ROOT_FACE_OVERLAP = 0.1
MOTOR_MOUNT_SUPPORT_OVERLAP = 0.1


@dataclass(frozen=True)
class ArmCrossbarLayout:
    side: int
    shoulder_x: float
    cam_x: float
    outer_end_x: float
    inner_end_x: float
    screw_positions_x: tuple[float, float]

    @property
    def length(self) -> float:
        return abs(self.outer_end_x - self.inner_end_x)


def arm_crossbar_layout(shoulder_x: float, cam_x: float) -> ArmCrossbarLayout:
    if shoulder_x == 0:
        raise ValueError("shoulder_x must be non-zero")
    if cam_x not in CAM_CENTERS_X:
        raise ValueError(f"cam_x must match a cam lobe center: {cam_x}")

    side = 1 if shoulder_x > 0 else -1
    outer_end_x = shoulder_x - side * 1.1
    inner_end_x = cam_x + side * DIMENSIONS.cam_width / 2
    screw_positions_x = (
        outer_end_x - side * CROSSBAR_MOUNT_END_OFFSET,
        inner_end_x + side * CROSSBAR_MOUNT_END_OFFSET,
    )
    return ArmCrossbarLayout(
        side,
        shoulder_x,
        cam_x,
        outer_end_x,
        inner_end_x,
        screw_positions_x,
    )


def hand_travel_mm(
    eccentric_offset: float = DIMENSIONS.cam_offset,
    hand_reach: float = DIMENSIONS.hand_reach,
    lever_length: float = DIMENSIONS.lever_length,
) -> float:
    return 2 * eccentric_offset * hand_reach / lever_length


def strokes_per_second(motor_rpm: float = DIMENSIONS.motor_rpm) -> float:
    return motor_rpm / 60


def cam_lobe_centers() -> tuple[tuple[float, float, float], tuple[float, float, float]]:
    return (
        (
            CAM_CENTERS_X[0],
            DIMENSIONS.motor_axis_y,
            DIMENSIONS.motor_axis_z + DIMENSIONS.cam_offset,
        ),
        (
            CAM_CENTERS_X[1],
            DIMENSIONS.motor_axis_y,
            DIMENSIONS.motor_axis_z - DIMENSIONS.cam_offset,
        ),
    )


def highest_cam_surface_z() -> float:
    return DIMENSIONS.motor_axis_z + DIMENSIONS.cam_diameter / 2 + DIMENSIONS.cam_offset


def outer_axle_motor_clearance_mm() -> float:
    distance_y = max(
        abs(DIMENSIONS.motor_axis_y) - DIMENSIONS.motor_body_width / 2,
        0,
    )
    distance_z = max(
        abs(DIMENSIONS.crossbar_axis_lift_z - DIMENSIONS.motor_axis_z)
        - DIMENSIONS.motor_body_height / 2,
        0,
    )
    return sqrt(distance_y**2 + distance_z**2) - DIMENSIONS.crossbar_diameter / 2


def _cylinder_x(radius: float, length: float, center: tuple[float, float, float]) -> Shape:
    return Pos(*center) * Rot(0, 90, 0) * Cylinder(radius, length)


def _d_bore_x(
    center: tuple[float, float, float],
    length: float,
    diameter: float,
    flat_from_center: float,
    clearance: float,
) -> Shape:
    radius = diameter / 2 + clearance
    flat = flat_from_center + clearance
    bore = _cylinder_x(radius, length, center)
    cut_height = radius + 0.5 - flat
    flat_cut = Pos(
        center[0],
        center[1],
        center[2] - (radius + 0.5 + flat) / 2,
    ) * Box(length + 1, diameter + 2, cut_height)
    return bore - flat_cut


def _d_bore_z(
    center: tuple[float, float, float],
    height: float,
    diameter: float,
    flat_from_center: float,
    clearance: float,
) -> Shape:
    radius = diameter / 2 + clearance
    flat = flat_from_center + clearance
    bore = Pos(*center) * Cylinder(radius, height)
    cut_height = radius + 0.5 - flat
    flat_cut = Pos(
        center[0],
        center[1],
        center[2] - (radius + 0.5 + flat) / 2,
    ) * Box(diameter + 2, diameter + 2, cut_height)
    return bore - flat_cut


def build_arm_crossbar(shoulder_x: float, cam_x: float) -> Shape:
    layout = arm_crossbar_layout(shoulder_x, cam_x)
    center_x = (layout.outer_end_x + layout.inner_end_x) / 2
    crossbar = _cylinder_x(
        DIMENSIONS.crossbar_diameter / 2,
        layout.length,
        (center_x, 0, DIMENSIONS.crossbar_axis_lift_z),
    )
    mounting_shelf = Pos(
        center_x,
        (DIMENSIONS.backplate_inner_face_y + CROSSBAR_MOUNT_SHELF_FRONT_Y) / 2,
        (CROSSBAR_MOUNT_SHELF_BOTTOM_Z + CROSSBAR_MOUNT_SHELF_TOP_Z) / 2,
    ) * Box(
        layout.length,
        CROSSBAR_MOUNT_SHELF_FRONT_Y - DIMENSIONS.backplate_inner_face_y,
        CROSSBAR_MOUNT_SHELF_TOP_Z - CROSSBAR_MOUNT_SHELF_BOTTOM_Z,
    )
    mounting_lip = Pos(
        center_x,
        DIMENSIONS.backplate_inner_face_y + CROSSBAR_MOUNT_LIP_THICKNESS_Y / 2,
        (CROSSBAR_MOUNT_LIP_BOTTOM_Z + CROSSBAR_MOUNT_LIP_TOP_Z) / 2,
    ) * Box(
        layout.length,
        CROSSBAR_MOUNT_LIP_THICKNESS_Y,
        CROSSBAR_MOUNT_LIP_TOP_Z - CROSSBAR_MOUNT_LIP_BOTTOM_Z,
    )
    rear_web = Pos(
        center_x,
        CROSSBAR_REAR_WEB_CENTER_Y,
        (CROSSBAR_REAR_WEB_BOTTOM_Z + CROSSBAR_REAR_WEB_TOP_Z) / 2,
    ) * Box(
        layout.length,
        CROSSBAR_REAR_WEB_DEPTH_Y,
        CROSSBAR_REAR_WEB_TOP_Z - CROSSBAR_REAR_WEB_BOTTOM_Z,
    )
    crossbar = crossbar + mounting_shelf + mounting_lip + rear_web
    for screw_x in layout.screw_positions_x:
        screw_hole = (
            Pos(
                screw_x,
                (DIMENSIONS.backplate_inner_face_y + CROSSBAR_SCREW_TIP_Y) / 2,
                CROSSBAR_SCREW_Z,
            )
            * Rot(-90, 0, 0)
            * Cylinder(
                M3_CLEARANCE_DIAMETER / 2,
                CROSSBAR_SCREW_TIP_Y - DIMENSIONS.backplate_inner_face_y + 0.2,
            )
        )
        wrench_access = Pos(
            screw_x,
            CROSSBAR_NUT_FACE_Y + 3.1,
            CROSSBAR_SCREW_Z,
        ) * Box(7.2, 6.2, 5.7)
        crossbar = crossbar - screw_hole - wrench_access
    crossbar = crossbar - _cylinder_x(
        DIMENSIONS.arm_axle_bore_diameter / 2,
        layout.length + 0.2,
        (center_x, 0, DIMENSIONS.crossbar_axis_lift_z),
    )
    return crossbar


def _beam_between_yz(
    x: float,
    start: tuple[float, float],
    end: tuple[float, float],
    width_x: float,
    thickness: float,
) -> Shape:
    delta_y = end[0] - start[0]
    delta_z = end[1] - start[1]
    length = sqrt(delta_y**2 + delta_z**2)
    angle = degrees(asin(delta_z / length))
    center_y = (start[0] + end[0]) / 2
    center_z = (start[1] + end[1]) / 2
    return Pos(x, center_y, center_z) * Rot(angle, 0, 0) * Box(width_x, length, thickness)


def _arm_section_center_x(outer_face_x: float, side: int, width_x: float) -> float:
    return outer_face_x - side * width_x / 2


def build_arm(shoulder_x: float, cam_x: float) -> Shape:
    layout = arm_crossbar_layout(shoulder_x, cam_x)
    side = layout.side
    hub_center_x = layout.outer_end_x + side * DIMENSIONS.arm_shoulder_hub_thickness / 2
    hub_outer_face_x = hub_center_x + side * DIMENSIONS.arm_shoulder_hub_thickness / 2
    arm_hub = _cylinder_x(
        DIMENSIONS.arm_shoulder_hub_radius,
        DIMENSIONS.arm_shoulder_hub_thickness,
        (hub_center_x, 0, 0),
    )
    socket = _cylinder_x(
        DIMENSIONS.arm_knob_socket_diameter / 2,
        DIMENSIONS.arm_knob_socket_depth + 0.2,
        (
            layout.outer_end_x + side * (DIMENSIONS.arm_knob_socket_depth - 0.2) / 2,
            0,
            0,
        ),
    )
    upper_arm = _beam_between_yz(
        _arm_section_center_x(hub_outer_face_x, side, 2.0),
        (0, -2.5),
        (2, -22),
        2.0,
        3,
    )
    forearm = _beam_between_yz(
        _arm_section_center_x(hub_outer_face_x, side, 3.6),
        (1, -20),
        (DIMENSIONS.hand_reach, -DIMENSIONS.hand_drop),
        3.6,
        3,
    )
    hand = Pos(
        _arm_section_center_x(hub_outer_face_x, side, 8.0),
        DIMENSIONS.hand_reach - 1,
        -DIMENSIONS.hand_drop,
    ) * Box(8, 7, 3)

    arm = arm_hub + upper_arm + forearm + hand - socket
    return Pos(0, 0, DIMENSIONS.crossbar_axis_lift_z) * arm


def build_cam_lever(cam_x: float) -> Shape:
    cam_center = next((center for center in cam_lobe_centers() if center[0] == cam_x), None)
    if cam_center is None:
        raise ValueError(f"cam_x must match a cam lobe center: {cam_x}")
    cam_top_z = cam_center[2] + DIMENSIONS.cam_diameter / 2
    start_y = 0.0
    start_z = DIMENSIONS.crossbar_axis_lift_z
    end_z = cam_top_z + DIMENSIONS.lever_thickness / 2
    return _beam_between_yz(
        cam_x,
        (start_y, start_z),
        (DIMENSIONS.lever_length, end_z),
        3,
        DIMENSIONS.lever_thickness,
    )


def build_arm_axle(
    shoulder_x: float,
    cam_x: float,
    angle_degrees: float = 0.0,
) -> Shape:
    layout = arm_crossbar_layout(shoulder_x, cam_x)
    side = layout.side
    outer_end_x = layout.outer_end_x
    shaft_length = abs(outer_end_x - cam_x)
    shaft_center_x = (outer_end_x + cam_x) / 2
    shaft = _cylinder_x(
        DIMENSIONS.arm_axle_diameter / 2,
        shaft_length,
        (shaft_center_x, 0, DIMENSIONS.crossbar_axis_lift_z),
    )
    collar_center_x = layout.inner_end_x - side * DIMENSIONS.arm_axle_collar_width / 2
    collar = _cylinder_x(
        DIMENSIONS.arm_axle_collar_diameter / 2,
        DIMENSIONS.arm_axle_collar_width,
        (collar_center_x, 0, DIMENSIONS.crossbar_axis_lift_z),
    )

    cam_center = next((center for center in cam_lobe_centers() if center[0] == cam_x), None)
    if cam_center is None:
        raise ValueError(f"cam_x must match a cam lobe center: {cam_x}")
    cam_top_z = cam_center[2] + DIMENSIONS.cam_diameter / 2
    follower = _cylinder_x(
        FOLLOWER_RADIUS,
        3,
        (cam_x, DIMENSIONS.lever_length, cam_top_z + FOLLOWER_RADIUS),
    )
    root_face_center_x = cam_x - side * (
        CAM_LEVER_ROOT_FACE_THICKNESS / 2 - CAM_LEVER_ROOT_FACE_OVERLAP
    )
    lever_root_face = _cylinder_x(
        DIMENSIONS.arm_axle_diameter / 2,
        CAM_LEVER_ROOT_FACE_THICKNESS,
        (root_face_center_x, 0, DIMENSIONS.crossbar_axis_lift_z),
    )
    knob = _cylinder_x(
        DIMENSIONS.arm_knob_diameter / 2,
        DIMENSIONS.arm_knob_length + 0.1,
        (
            outer_end_x + side * (DIMENSIONS.arm_knob_length / 2 - 0.05),
            0,
            DIMENSIONS.crossbar_axis_lift_z,
        ),
    )
    axle = shaft + collar + build_cam_lever(cam_x) + lever_root_face + follower + knob
    return Rot(angle_degrees, 0, 0) * axle


def build_double_eccentric(
    shaft_clearance: float = DIMENSIONS.shaft_clearance,
    flat_from_center: float = DIMENSIONS.shaft_flat_from_center,
) -> Shape:
    shaft_center_x = (
        DIMENSIONS.motor_body_center_x
        + DIMENSIONS.motor_body_length / 2
        + DIMENSIONS.motor_shaft_length / 2
    )
    shaft_center = (shaft_center_x, DIMENSIONS.motor_axis_y, DIMENSIONS.motor_axis_z)
    hub = _cylinder_x(4, DIMENSIONS.motor_shaft_length, shaft_center)

    lobes = [
        _cylinder_x(DIMENSIONS.cam_diameter / 2, DIMENSIONS.cam_width, center)
        for center in cam_lobe_centers()
    ]
    body = hub + lobes[0] + lobes[1]
    bore = _d_bore_x(
        shaft_center,
        DIMENSIONS.motor_shaft_length + 2,
        DIMENSIONS.motor_shaft_diameter,
        flat_from_center,
        shaft_clearance,
    )
    return body - bore


def build_motor_cradle() -> Shape:
    center = (
        DIMENSIONS.motor_body_center_x,
        DIMENSIONS.motor_axis_y,
        DIMENSIONS.motor_axis_z,
    )
    cradle = Pos(*center) * Box(
        DIMENSIONS.motor_body_length + 1,
        DIMENSIONS.motor_body_width + 4,
        DIMENSIONS.motor_body_height + 4,
    )
    motor_seat = Pos(*center) * Box(
        DIMENSIONS.motor_body_length + 1,
        DIMENSIONS.motor_body_width + 0.4,
        DIMENSIONS.motor_body_height + 0.4,
    )
    opening = Pos(
        DIMENSIONS.motor_body_center_x,
        DIMENSIONS.motor_axis_y,
        DIMENSIONS.motor_axis_z + DIMENSIONS.motor_body_height / 2 + 5,
    ) * Box(
        DIMENSIONS.motor_body_length + 1,
        DIMENSIONS.motor_body_width + 0.6,
        10,
    )
    axle_passage = _cylinder_x(
        DIMENSIONS.crossbar_diameter / 2 + 0.3,
        DIMENSIONS.motor_body_length + 4,
        (DIMENSIONS.motor_body_center_x, 0, 0),
    )
    return cradle - motor_seat - opening - axle_passage


def build_test_frame() -> Shape:
    backplate = Pos(0, DIMENSIONS.backplate_center_y, -2) * Box(
        DIMENSIONS.chest_width,
        DIMENSIONS.backplate_thickness,
        30,
    )
    base = Pos(0, DIMENSIONS.frame_base_center_y, DIMENSIONS.frame_base_center_z) * Box(
        DIMENSIONS.frame_base_width,
        DIMENSIONS.frame_base_depth,
        DIMENSIONS.frame_base_thickness,
    )
    base_web = Pos(0, DIMENSIONS.backplate_center_y, -23) * Box(
        DIMENSIONS.chest_width,
        DIMENSIONS.backplate_thickness,
        14,
    )
    frame: Shape = backplate + base + base_web

    for shoulder_x, cam_x in (
        (-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        (DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
    ):
        for screw_x in arm_crossbar_layout(shoulder_x, cam_x).screw_positions_x:
            backplate_through_hole = (
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
                    DIMENSIONS.backplate_rear_face_y + M3_SOCKET_HEAD_POCKET_DEPTH / 2,
                    CROSSBAR_SCREW_Z,
                )
                * Rot(-90, 0, 0)
                * Cylinder(
                    M3_SOCKET_HEAD_POCKET_DIAMETER / 2,
                    M3_SOCKET_HEAD_POCKET_DEPTH,
                )
            )
            frame = frame - backplate_through_hole - head_pocket

    cradle = build_motor_cradle()
    base_top_z = DIMENSIONS.frame_base_center_z + DIMENSIONS.frame_base_thickness / 2
    cradle_bottom_z = DIMENSIONS.motor_axis_z - (DIMENSIONS.motor_body_height + 4) / 2
    support_height = cradle_bottom_z - base_top_z + 2 * MOTOR_MOUNT_SUPPORT_OVERLAP
    support_center_z = (base_top_z + cradle_bottom_z) / 2
    support_outer_length_x = DIMENSIONS.motor_body_length + 1
    support_outer_depth_y = DIMENSIONS.motor_body_width + 4
    motor_mount_support = Pos(
        DIMENSIONS.motor_body_center_x,
        DIMENSIONS.motor_axis_y,
        support_center_z,
    ) * Box(support_outer_length_x, support_outer_depth_y, support_height)
    frame = frame + cradle + motor_mount_support
    return frame


def build_shaft_fit_coupon() -> Shape:
    pitch = 10.0
    rows = len(SHAFT_FLAT_CANDIDATES)
    columns = len(SHAFT_CLEARANCE_CANDIDATES)
    width = columns * pitch
    height = rows * pitch
    plate = Pos(0, 0, 0) * Box(width + 8, height + 8, 1.2)
    orientation_notch = Pos(-(width + 8) / 2 + 0.75, -(height + 8) / 2 + 0.75, 0) * Box(
        1.5,
        1.5,
        2,
    )
    coupon: Shape = plate - orientation_notch
    for row, flat in enumerate(SHAFT_FLAT_CANDIDATES):
        y = (row - (rows - 1) / 2) * pitch
        for column, clearance in enumerate(SHAFT_CLEARANCE_CANDIDATES):
            x = (column - (columns - 1) / 2) * pitch
            cell = Pos(x, y, 2.5) * Box(8, 8, 3.8)
            bore = _d_bore_z(
                (x, y, 2.5),
                6,
                DIMENSIONS.motor_shaft_diameter,
                flat,
                clearance,
            )
            coupon = coupon + cell - bore
    return coupon


def build_motor_reference() -> Shape:
    body_front_x = DIMENSIONS.motor_body_center_x + DIMENSIONS.motor_body_length / 2
    gearbox = Pos(
        body_front_x - DIMENSIONS.motor_gearbox_length / 2,
        DIMENSIONS.motor_axis_y,
        DIMENSIONS.motor_axis_z,
    ) * Box(
        DIMENSIONS.motor_gearbox_length,
        DIMENSIONS.motor_body_width,
        DIMENSIONS.motor_body_height,
    )
    motor_can = Pos(
        body_front_x - DIMENSIONS.motor_gearbox_length - DIMENSIONS.motor_can_length / 2,
        DIMENSIONS.motor_axis_y,
        DIMENSIONS.motor_axis_z,
    ) * Box(
        DIMENSIONS.motor_can_length,
        DIMENSIONS.motor_body_width,
        DIMENSIONS.motor_body_height,
    )
    shaft_center = (
        body_front_x + DIMENSIONS.motor_shaft_length / 2,
        DIMENSIONS.motor_axis_y,
        DIMENSIONS.motor_axis_z,
    )
    shaft = _d_bore_x(
        shaft_center,
        DIMENSIONS.motor_shaft_length,
        DIMENSIONS.motor_shaft_diameter,
        DIMENSIONS.shaft_flat_from_center,
        0,
    )
    return gearbox + motor_can + shaft


def build_print_parts() -> dict[str, Shape]:
    return {
        "test_frame": build_test_frame(),
        "left_crossbar": build_arm_crossbar(-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        "right_crossbar": build_arm_crossbar(DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
        "left_arm": _arm_for_print(-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        "right_arm": _arm_for_print(DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
        "left_arm_axle": build_arm_axle(-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        "right_arm_axle": build_arm_axle(DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
        "double_eccentric": build_double_eccentric(),
        "shaft_fit_coupon": build_shaft_fit_coupon(),
    }


def _arm_for_print(shoulder_x: float, cam_x: float) -> Shape:
    side = 1 if shoulder_x > 0 else -1
    axis_z = DIMENSIONS.crossbar_axis_lift_z
    arm = (
        Pos(shoulder_x, 0, axis_z)
        * Rot(0, side * 90, 0)
        * Pos(-shoulder_x, 0, -axis_z)
        * build_arm(shoulder_x, cam_x)
    )
    return Pos(0, 0, -arm.bounding_box().min.Z) * arm


def build_assembly_parts() -> dict[str, Shape]:
    print_parts = build_print_parts()
    return {
        "Test frame": print_parts["test_frame"],
        "Left crossbar": print_parts["left_crossbar"],
        "Right crossbar": print_parts["right_crossbar"],
        "Left arm": build_arm(-DIMENSIONS.shoulder_position, CAM_CENTERS_X[0]),
        "Right arm": build_arm(DIMENSIONS.shoulder_position, CAM_CENTERS_X[1]),
        "Left arm axle": print_parts["left_arm_axle"],
        "Right arm axle": print_parts["right_arm_axle"],
        "Double eccentric": print_parts["double_eccentric"],
        "GA12-N20 motor": build_motor_reference(),
    }


def build_test_rig() -> Compound:
    return Compound(
        children=list(build_assembly_parts().values()),
        label="Cody chest cam test rig",
    )


def export_print_parts(
    output_dir: Path,
    *,
    version: str | None = None,
    include_step: bool = True,
) -> list[Path]:
    output_dir.mkdir(parents=True, exist_ok=True)
    exported: list[Path] = []
    for name, part in build_print_parts().items():
        file_name = f"cody-{name.replace('_', '-')}-{version}.stl" if version else f"{name}.stl"
        file_path = output_dir / file_name
        print_shape = (
            Rot(0, 90, 0) * part
            if name
            in {
                "left_arm_axle",
                "right_arm_axle",
                "double_eccentric",
            }
            else part
        )
        bounds = print_shape.bounding_box()
        print_shape = Pos(-bounds.min.X, -bounds.min.Y, -bounds.min.Z) * print_shape
        if not export_stl(print_shape, file_path):
            raise RuntimeError(f"Failed to export {file_path}")
        exported.append(file_path)
    if include_step:
        assembly_name = f"cody-test-rig-{version}.step" if version else "test_rig.step"
        assembly_path = output_dir / assembly_name
        if not export_step(build_test_rig(), assembly_path):
            raise RuntimeError(f"Failed to export {assembly_path}")
        exported.append(assembly_path)
    return exported


def main() -> None:
    parser = argparse.ArgumentParser(description="Export the Stage 4a chest-cam test rig.")
    parser.add_argument(
        "--output",
        type=Path,
        required=True,
        help="directory for STL and STEP files",
    )
    parser.add_argument("--version", help="include a version in exported file names")
    parser.add_argument(
        "--stl-only",
        action="store_true",
        help="export print-part STLs without the assembled STEP reference",
    )
    args = parser.parse_args()
    if args.version and any(separator in args.version for separator in ("/", "\\")):
        parser.error("--version must not contain path separators")
    export_print_parts(args.output, version=args.version, include_step=not args.stl_only)


if __name__ == "__main__":
    main()

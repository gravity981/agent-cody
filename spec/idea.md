# Coding Agent Desk Companion

## Project Specification – Version 0.2

**Project type:** 3D-printed electromechanical desk gadget  
**Controller:** ESP32  
**Computer connection:** USB  
**Primary purpose:** Physically represent the activity and status of an AI coding agent.

# 1. Project Concept

Build a small cartoon developer sitting at a desk with a laptop. The figure represents an AI coding agent working in an IDE such as Visual Studio Code, using GitHub Copilot or Claude Code.

When the coding agent is working, the figure appears to type on its miniature laptop. Its two arms move alternately up and down, creating the illusion of typing.

Additional feedback is provided through:
- An RGB LED indicating the agent's current state.
- A small buzzer providing notification sounds.
- USB communication between the computer and an ESP32.

The gadget should be compact, inexpensive, mechanically simple, 3D-printable and suitable for long-term use on a desk.

A key design principle is to minimize the number of parts and avoid unnecessary mechanical and electronic complexity.

# 2. Design Goals and Constraints

| Requirement | Target |
|---|---|
| Overall height | Maximum 150 mm |
| Target height | Approximately 120–130 mm |
| 3D printer | Prusa MK3S+ |
| Printing technology | FDM |
| Controller | ESP32 development board |
| Computer connection | USB |
| Moving parts | Two arms |
| Motor count | One |
| Motor type | Continuously rotating geared DC motor |
| Arm movement | Alternating, driven by one motor |
| Lighting | One WS2812-compatible RGB LED |
| Sound | Small buzzer |
| Electronics | Ready-made modules and breakout boards |
| Mechanical complexity | As low as reasonably possible |
| Assembly | Simple, modular and serviceable |

# 3. Physical Design

## 3.1 Overall Appearance

The figure should have a cute, exaggerated cartoon developer appearance.

Suggested characteristics:
- Oversized head.
- Small body.
- Hoodie or casual developer clothing.
- Simplified facial features.
- Seated working posture.
- Two movable arms reaching towards the laptop keyboard.

The design should look like a miniature desk toy, not an industrial robot.

## 3.2 Main Printed Components

The design should consist of as few printed components as possible.

| Component | Description |
|---|---|
| Platform and table | One integrated printed structure, including base and tabletop |
| Laptop | Static miniature laptop on the table |
| Developer body | Integrated head, torso and seated lower body |
| Left arm | One-piece movable arm |
| Right arm | One-piece movable arm |
| Torso back cover | Optional removable cover for mechanical access |

The torso back cover is the only additional structural component currently anticipated.

No separate chair, neck, fingers, articulated hands or decorative mechanical joints are required.

## 3.3 Dimensions

Initial target dimensions:

| Element | Approximate dimensions |
|---|---|
| Platform | 120 × 100 mm |
| Platform thickness | 5 mm |
| Table height | 45 mm |
| Tabletop | 90 × 55 mm |
| Developer height | 70–75 mm |
| Laptop width | 40–55 mm |
| Overall height | 120–130 mm |

These are preliminary design values and can be adjusted during CAD development. The complete assembly must remain within a maximum height of 150 mm.

# 4. Mechanical System

## 4.1 General Principle

The two arms are driven by a single continuously rotating DC geared motor.

No servos or independently controlled motors are required.

The motor drives a crank mechanism that converts continuous rotation into alternating arm movement.

The intended mechanism consists of:
- One geared DC motor.
- One common rotating shaft.
- Two eccentric crank pins positioned 180 degrees apart.
- Two connecting rods.
- Two shoulder pivots.
- Two single-piece printed arms.

## 4.2 Arm Movement

Each arm rotates around a simple shoulder pivot.

A small mechanical lever behind each shoulder connects the arm to its respective connecting rod.

The connecting rods are driven by eccentric crank pins on the rotating shaft.

The crank pins are offset by 180 degrees so that the arms move alternately.

Expected behavior:
- Left arm moves down while right arm moves up.
- Right arm moves down while left arm moves up.
- The movement repeats continuously while the motor is running.

The hands should remain close to the laptop keyboard throughout the movement.

The movement should be small and natural-looking rather than exaggerated.

## 4.3 Mechanical Design Requirements

- The motor and crank mechanism should be hidden inside the torso.
- The mechanism should fit within the available torso volume.
- Arms should be lightweight.
- Shoulder pivots should be simple and robust.
- Connecting rods should use inexpensive metal wire or similarly simple components.
- Mechanical stops should prevent excessive arm movement.
- The mechanism should be accessible for maintenance.
- No lubrication-intensive or precision-machined components should be necessary.

The exact crank dimensions, shaft arrangement, connecting rod geometry and pivot positions will be determined during mechanical prototyping.

## 4.4 Motor Requirements

Use a small continuously rotating DC geared motor.

The motor should:
- Operate from a low-voltage DC supply.
- Provide sufficient torque for both arms.
- Rotate at a relatively low speed.
- Support speed control through PWM.
- Be small enough to fit inside the torso.

A small N20-style geared motor is a possible candidate. The final motor model and gear ratio will be selected during component evaluation.

# 5. Electronics

## 5.1 Modular Electronics Philosophy

The electronics must be assembled exclusively from ready-made development boards and breakout modules wherever possible.

The user should not need to design or assemble discrete electronic circuits.

Design requirements:
- No custom PCB.
- No individually wired discrete transistor circuits.
- No custom motor driver circuit.
- Use ready-made breakout boards for motor control.
- Use preassembled LED and buzzer modules.
- Use ready-made power regulation modules where necessary.
- Prefer JST connectors, screw terminals or equivalent plug-in connections.
- Use preassembled cables wherever possible.
- Avoid component-level soldering during final assembly.
- Keep modules removable and replaceable.

## 5.2 Controller

An ESP32 development board acts as the central controller.

It is responsible for:
- Receiving commands from the computer over USB.
- Controlling the DC motor through a motor driver breakout board.
- Controlling the RGB LED.
- Generating buzzer sounds.
- Managing the operating states.

The ESP32 should operate independently of the computer-side application once a state has been received.

## 5.3 Required Electronic Components

| Component | Quantity | Purpose |
|---|---:|---|
| ESP32 development board | 1 | Main controller |
| DC geared motor | 1 | Drives arm movement |
| DC motor driver breakout board | 1 | Motor switching and PWM control |
| WS2812-compatible RGB LED module | 1 | Status indication |
| Passive buzzer module | 1 | Audio notifications |
| Regulated power supply/module | 1 | Motor power |
| USB cable | 1 | Computer connection |
| Connecting cables | As needed | Module interconnections |
| Connectors | As needed | Modular assembly |

Additional ready-made modules for power distribution or voltage conversion may be included if required.

The exact module models will be selected during component evaluation.

## 5.4 Motor Driver

The DC motor must be controlled through a ready-made motor driver breakout board.

A small DRV8833 breakout board is a possible candidate.

The driver must:
- Support the selected motor's operating voltage.
- Support motor startup and stall current with appropriate margin.
- Accept logic-level control signals from the ESP32.
- Support PWM speed control.
- Be available as a preassembled module.
- Preferably provide convenient soldered connectors or screw terminals.

The motor and driver must be selected together after checking their electrical specifications.

Only one motor channel is required.

## 5.5 Power Architecture

The ESP32 is connected to the computer through USB.

The motor should use a separately regulated power supply if the available USB power budget is insufficient.

The motor driver handles the motor current. The ESP32 must never drive the motor directly through a GPIO pin.

The motor driver and ESP32 must share a common ground when using a separate motor supply.

The LED should receive an appropriate supply voltage and data signal.

The buzzer should be driven according to its electrical requirements.

The power architecture must prevent motor switching noise and current peaks from causing ESP32 resets or unreliable USB communication.

Power connectors should be accessible and clearly labeled.

# 6. Firmware

The ESP32 firmware should be independent of the coding agent or IDE.

It receives simple commands and translates them into physical behavior.

## 6.1 Operating States

| State | Arm behavior | LED | Sound |
|---|---|---|---|
| IDLE | Stopped | Dim blue | None |
| WORKING | Alternating typing | Pulsating blue | None |
| THINKING | Stopped | Yellow | None |
| APPROVAL | Stopped | Blinking orange | Double beep |
| COMPLETED | Stopped | Green | Success sound |
| ERROR | Stopped | Blinking red | Warning sound |

## 6.2 Typing Animation

When WORKING is activated:
- Start the motor.
- Generate alternating arm movement.
- Allow adjustment of motor speed through PWM.
- Keep the animation running independently of incoming USB commands.

When WORKING is deactivated:
- Stop the motor.
- Update the LED according to the new state.

The initial implementation does not require precise motor positioning. The arms may stop at any point in their mechanical cycle.

A future version may add a home-position sensor to park the arms consistently.

# 7. Computer Integration

## 7.1 Architecture

The computer-side software translates coding-agent events into simple commands for the ESP32.

The architecture consists of:
1. Coding agent or IDE.
2. Event integration or hook.
3. Local computer-side bridge application.
4. USB serial communication.
5. ESP32 firmware.
6. Physical movements and feedback.

The computer-side bridge should be implemented independently of the hardware firmware.

## 7.2 Supported Integrations

Initial target integrations:
- Claude Code.
- GitHub Copilot in Visual Studio Code.

Both integrations should communicate with the same bridge application.

Agent-specific event handling should remain outside the ESP32 firmware.

## 7.3 Computer-side Bridge

A lightweight Python application is proposed.

Responsibilities:
- Receive agent lifecycle events.
- Interpret event types.
- Maintain the current agent state.
- Send state changes to the ESP32.
- Manage the USB serial connection.
- Recover gracefully from connection interruptions.

The bridge should run as a persistent background process.

Agent hooks should send events to the local bridge, preferably through localhost HTTP or UDP.

## 7.4 Serial Protocol

The communication protocol should be simple and human-readable.

Example commands:

    STATE:IDLE
    STATE:WORKING
    STATE:THINKING
    STATE:APPROVAL
    STATE:COMPLETED
    STATE:ERROR

Optional configuration commands:

    SPEED:120
    LED:0,100,255
    BEEP:SUCCESS

Commands should be newline-terminated.

The final protocol will be defined during firmware development.

# 8. 3D Printing Requirements

The complete model must be printable using a Prusa MK3S+.

Design considerations:
- No support structures wherever reasonably possible.
- Minimize the number of separate printed parts.
- Use practical FDM clearances.
- Avoid fragile decorative details.
- Keep the arms lightweight.
- Provide sufficient wall thickness around motor mounts and pivots.
- Provide access to internal electronics.

PLA is the preferred material for the character and platform.

PETG may be used for mechanical components if additional flexibility or durability is needed.

Initial printing parameters:

| Parameter | Target |
|---|---|
| Layer height | 0.15–0.20 mm |
| Perimeters | 3 |
| Infill | 15–20% |
| Nozzle | 0.4 mm |
| Typical joint clearance | 0.2–0.3 mm |

Final tolerances will be determined through test prints.

# 9. Assembly and Maintenance

The assembly should be designed so that:
- The motor can be replaced without destroying the figure.
- The torso can be opened.
- The arms can be removed independently.
- The electronics can be accessed through the platform.
- Wiring is routed internally wherever possible.
- The USB connection remains accessible.
- Electronic modules can be replaced without modifying the printed structure.

The electronics should preferably be installed on a removable mounting plate or in a dedicated compartment.

The design should avoid permanent adhesive wherever screws or mechanical connections are practical.

# 10. Development Phases

## Phase 1 – Mechanical Prototype

Develop a simplified mechanical prototype containing:
- Motor.
- Crankshaft.
- Two crank pins.
- Connecting rods.
- Two arm pivots.
- Two test arms.

Goal: Validate alternating arm movement and motor torque.

## Phase 2 – Electronics Prototype

Connect:
- ESP32.
- Motor driver breakout board.
- RGB LED module.
- Buzzer module.
- Power supply.

Implement the basic state machine and serial protocol.

Goal: Verify all physical behaviors without designing custom electronic circuits.

## Phase 3 – Character CAD

Develop the final cartoon developer, platform, table and laptop.

Integrate the mechanical mechanism into the torso.

Provide mounting positions for the ready-made electronics modules.

Goal: Produce a complete printable assembly.

## Phase 4 – Firmware and Computer Integration

Develop the Python bridge and configure Claude Code and GitHub Copilot hooks.

Goal: Connect actual coding-agent activity to the physical figure.

## Phase 5 – Final Assembly and Testing

Test:
- Mechanical reliability.
- Motor temperature.
- Arm movement consistency.
- USB communication.
- State transitions.
- Long-running operation.
- Overall appearance and stability.

# 11. Open Design Decisions

The following details are intentionally not finalized:
- Exact ESP32 board model.
- Exact motor model and gear ratio.
- Exact motor driver breakout board.
- Exact power supply and power distribution modules.
- Exact crank mechanism geometry.
- Shaft and bearing requirements.
- Shoulder pivot construction.
- Connecting rod material.
- Final character appearance and facial details.
- Exact CAD dimensions.
- Serial protocol details.
- Computer-side bridge implementation.
- Final Claude Code and Copilot hook configurations.

These decisions will be made during the respective development phases.

# 12. Definition of Done

The project is considered complete when:
1. The complete assembly is no taller than 150 mm.
2. All structural components can be printed on a Prusa MK3S+.
3. The developer has two functional moving arms.
4. Both arms move alternately using one continuously rotating motor.
5. The mechanism operates reliably without servos.
6. The motor is controlled through a ready-made breakout board.
7. No custom electronic circuit design or component-level soldering is required for normal assembly.
8. The RGB LED correctly represents the agent state.
9. The buzzer provides audible notifications.
10. The ESP32 communicates with the computer over USB.
11. Claude Code and GitHub Copilot can trigger the appropriate physical states.
12. The complete assembly is compact, stable, inexpensive and serviceable.

---

**Design philosophy:** Keep the mechanical system simple, the number of printed parts low, and the electronics modular. The device should be a charming physical representation of an AI coding agent, not a complicated robotics project.
> Inspired by [Tiny Engineer](https://github.com/jamro/tiny-engineer) by jamro — all credit for the idea goes to the author. Cody is my own take on an AI coding buddy, written from scratch. Where it differs:
>
> - USB only: no Wi-Fi at all (Tiny Engineer offers Wi-Fi as an option).
> - Ready-made modules only, no custom circuit board, and fewer parts: one geared DC motor drives both arms instead of five servos, an RGB LED and a buzzer instead of a display and speaker.
> - Powered from a single USB port (≤ 500 mA).
> - Integrates Claude Code and GitHub Copilot in VS Code through a persistent host bridge.
>
> Like Tiny Engineer, the figure is 3D printed.

# Cody – Coding Agent Desk Companion

Cody is a small 3D-printed desk figure of a cartoon developer sitting at a laptop. It shows what your AI coding agent (Claude Code, GitHub Copilot in VS Code) is doing: while the agent works, the figure's arms type, driven by a single geared DC motor. An RGB LED and a buzzer signal the other states: idle, thinking, approval needed, completed, error.

The project delivers:

- printable CAD parts for an FDM printer (Prusa MK3S+);
- ESP32 firmware that drives ready-made modules (motor driver, RGB LED, buzzer) and takes commands over USB;
- a computer-side bridge with hooks for the coding agents.

## Specification

The specification in [spec/](spec/README.md) is the source of truth; start with [spec/README.md](spec/README.md). The original idea is in [spec/idea.md](spec/idea.md).

## Getting started

1. Open the repository in VS Code and choose "Reopen in Container". Every tool is installed there with a pinned version.
2. Run `ci/check` for all automated verification and the requirement report (`reports/requirements.md`), as in CI.

## Releases

Release candidates are built in GitHub Actions, verified, and then promoted to releases. See [spec/delivery.md](spec/delivery.md).

## License

[MIT](LICENSE). Third-party assets with their own licence (e.g. the generated head mesh) are marked where they are stored.

---

Created from project blueprint 0.1.0.

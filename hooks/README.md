# hooks/

Hook configuration for Claude Code and GitHub Copilot ([spec/architecture.md](../spec/architecture.md) 4.1). Requirements: AGT in [spec/requirements.md](../spec/requirements.md).

## Claude Code

`install-claude-code` installs Cody's Claude Code hooks at user level. They forward every Claude Code event to the bridge.

### What you need

| Where | Needed for | What |
|---|---|---|
| The machine where you run the installer | Installing the hooks (once) | Python 3.10 or newer (`python3`); no packages. In this repository, `uv` provides it. |
| Every environment where Claude Code runs (host and each devcontainer) | The hooks at run time | `curl` and a POSIX shell (`sh`). Without curl the hooks do nothing; Claude Code keeps working. |
| Linux host | Receiving the events | The Cody bridge running in Docker, using the compose file from the release. |
| Windows host | Receiving the events | The Cody tray app installed and running. It starts the WSL relay automatically when WSL is installed. |

The hooks use the same configuration on all platforms:

- Windows host and WSL: `127.0.0.1`.
- Docker Desktop devcontainer on Windows: `host.docker.internal`.
- Devcontainer in Docker Engine in WSL2: `172.17.0.1`, through the relay in WSL.
- Linux devcontainer: `host.docker.internal` or `172.17.0.1`; it must run on the same host as the bridge.

On Windows, the app relays traffic into the default WSL distribution. To relay into several distributions, list them in `%APPDATA%\cody\bridge.toml`:

```toml
wsl_distros = ["Ubuntu", "Debian"]
```

Set `wsl_distros = []` to disable the relay. The app applies changes after **Restart** in its tray menu. No WSL networking mode or firewall rule needs to be changed.

### Install

```bash
uv run python hooks/install-claude-code     # in this repository
python3 cody-install-claude-code-<version>  # release file, any environment with Python
```

- Run it once on the host and once for your devcontainers. Devcontainers that share the Claude Code configuration (the `claude-config` volume) need it only once; a devcontainer with its own configuration needs its own run.
- It writes to `$CLAUDE_CONFIG_DIR/settings.json` or `~/.claude/settings.json`, keeps all other settings and saves the previous file as `settings.json.bak`.
- Restart Claude Code afterwards: it reads hooks when a session starts.
- `--uninstall` removes Cody's hooks and nothing else.

### Check

With Cody running, start a Claude Code session and send a prompt. Run `cody status` on the host or in WSL: the session is listed with its state.

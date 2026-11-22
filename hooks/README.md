# hooks/

Hook configuration for Claude Code and GitHub Copilot ([spec/architecture.md](../spec/architecture.md) 4.1). Requirements: AGT in [spec/requirements.md](../spec/requirements.md).

## Claude Code

`install-claude-code` installs Cody's Claude Code hooks at user level. They forward every Claude Code event to the bridge.

### What you need

| Where | Needed for | What |
|---|---|---|
| The machine where you run the installer | Installing the hooks (once) | Python 3.10 or newer (`python3`); no packages. In this repository, `uv` provides it. |
| Every environment where Claude Code runs (host and each devcontainer) | The hooks at run time | `curl` and a POSIX shell (`sh`). Without curl the hooks do nothing; Claude Code keeps working. |
| Native Linux host | Receiving the events | Not supported between Stage 3 and Stage 6 step 6a (TD-15). |
| Windows host | Receiving the events | The Cody tray app installed and running. It starts the WSL relay automatically when WSL is installed. |

The hooks use the same configuration on supported setups:

- Windows host and WSL: `127.0.0.1`.
- Docker Desktop devcontainer on Windows: `host.docker.internal`.
- Devcontainer in Docker Engine in WSL2: `172.17.0.1`, through the relay in WSL. The hook installer’s `--port` option sets the relay listener port for this target.

Hooks default to port `8765`. To use a WSL relay listening on another port, set `--port` to the port shown in the relay log; this changes only the `172.17.0.1` target. The `127.0.0.1` and `host.docker.internal` targets stay on `8765`. The WSL relay selects the first available port in `8766`–`8770`; if all are occupied, it logs a warning and retries. Each applicable hook target is tried in order, and the hook stops at Cody's `ok` response.

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

- For Docker Engine in WSL2, pass the relay listener port, e.g. `uv run python hooks/install-claude-code --port 8766`. Omit `--port` for the default `8765`.
- Run it once on the host and once for your devcontainers. Devcontainers that share the Claude Code configuration (the `claude-config` volume) need it only once; a devcontainer with its own configuration needs its own run.
- When updating Cody to a release with a changed hook command, rerun the installer in each environment that owns Claude Code settings; the installer replaces Cody's existing commands without duplicating them.
- It writes to `$CLAUDE_CONFIG_DIR/settings.json` or `~/.claude/settings.json`, keeps all other settings and saves the previous file as `settings.json.bak`.
- Restart Claude Code afterwards: it reads hooks when a session starts.
- `--uninstall` removes Cody's hooks and nothing else.

### Check

With Cody running, start a Claude Code session and send a prompt. Run `cody status` on the host or in WSL: the session is listed with its state.

## GitHub Copilot in VS Code

The Copilot harness reads user hook files from `~/.copilot/hooks/`, or `$COPILOT_HOME/hooks/` when `COPILOT_HOME` is set. The configuration uses PascalCase-compatible event names so Copilot sends a common payload containing `hook_event_name`, `session_id`, `timestamp` and `cwd`. The bridge keys each Copilot session by `cwd`.

### Install

From the repository, copy the configuration and both platform forwarders:

```bash
hook_dir="${COPILOT_HOME:-$HOME/.copilot}/hooks"
mkdir -p "$hook_dir"
cp hooks/copilot-hooks.json "$hook_dir/cody.json"
cp hooks/cody-forward.sh "$hook_dir/cody-forward.sh"
cp hooks/cody-forward.ps1 "$hook_dir/cody-forward.ps1"
```

In PowerShell, use:

```powershell
$root = if ($env:COPILOT_HOME) { $env:COPILOT_HOME } else { Join-Path $HOME '.copilot' }
$hooks = Join-Path $root 'hooks'
New-Item -ItemType Directory -Force -Path $hooks | Out-Null
Copy-Item hooks/copilot-hooks.json (Join-Path $hooks 'cody.json')
Copy-Item hooks/cody-forward.sh (Join-Path $hooks 'cody-forward.sh')
Copy-Item hooks/cody-forward.ps1 (Join-Path $hooks 'cody-forward.ps1')
```

For a release download, copy `cody-copilot-hooks-<version>.json` to the hook folder as `cody.json`, and copy the matching `cody-copilot-forward-<version>.sh` and `.ps1` files there as `cody-forward.sh` and `cody-forward.ps1`.

The forwarders need `curl` (or `curl.exe` on Windows) wherever Copilot runs. They try `127.0.0.1`, `host.docker.internal`, then the Docker gateway, each with a 0.1 s timeout; they suppress output and always exit successfully so Cody cannot block the agent. If the WSL relay listens on a non-default port, update only the `172.17.0.1:8765` target in both scripts to the relay port. Start a new Copilot session after installing.

The VS Code 1.147.0 Copilot hook picker includes `ErrorOccurred` and does not include an approval/permission-request event. Cody maps errors to ERROR; APPROVAL remains conditional until VS Code exposes an event for it. To verify setup, start a Copilot session and use a harmless request that reads a project file; `cody status` should list the workspace folder as the Copilot session key.

Copilot also runs applicable hooks from repository `.claude/settings.json` and `.claude/settings.local.json`. When Cody's Claude-compatible hooks are configured there, the bridge recognizes Copilot's compatible payload and deduplicates it against the user-level Copilot hook. It does not read Claude Code's user-level `~/.claude/settings.json` as a Copilot hook source.

# hooks/

Hook configuration for Claude Code and GitHub Copilot ([spec/architecture.md](../spec/architecture.md) 4.1). Requirements: AGT in [spec/requirements.md](../spec/requirements.md).

## Claude Code

`install-claude-code` installs Cody's Claude Code hooks at user level. They forward every Claude Code event to the bridge.

### What you need

| Where | Needed for | What |
|---|---|---|
| The machine where you run the installer | Installing the hooks (once) | Python 3.10 or newer (`python3`); no packages. In this repository, `uv` provides it. |
| Every environment where Claude Code runs (host and each devcontainer) | The hooks at run time | `curl` and a POSIX shell (`sh`). Without curl the hooks do nothing; Claude Code keeps working. |
| The host | Receiving the events | The Cody bridge running (Docker, compose file from the release), listening on port 8765. |

A devcontainer reaches the bridge on `host.docker.internal` or `172.17.0.1`; it has to run on Docker on the same host as the bridge.

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

With the bridge running, start a Claude Code session and send a prompt, then run `cody status` on the host: the session is listed with its state.

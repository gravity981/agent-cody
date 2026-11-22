# cli/

Host command `cody` (spec/architecture.md 4.1): `cody status`, `cody state <STATE>`, `cody state clear`, `cody session clear` (removes all sessions, e.g. a stale one), `cody --version`; `cody setup` follows in stage 3 (AGT-10). A shell script using curl.

On Windows, the Cody tray app provides the bridge. The app starts a relay in the default WSL distribution, so this command works from WSL without changing its network settings. If Cody should also be reachable from other distributions, add their names to `wsl_distros` in `%APPDATA%\cody\bridge.toml` and restart the app.

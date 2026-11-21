# tools/

- `static-checks`: ruff, clang-format, shellcheck (run by `ci/check`).
- Requirement report (TRC in [spec/requirements.md](../spec/requirements.md)), Python package `cody-tools` in the uv workspace; built in step 1b.
- Developer-only browser test panel: run `uv run cody-test-panel` in the devcontainer, then use the forwarded port 8766. Stop the bridge before connecting; the panel opens USB serial directly and has exclusive ownership while connected.

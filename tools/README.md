# tools/

- `static-checks`: ruff, clang-format, shellcheck (run by `ci/check`).
- Requirement report (TRC in [spec/requirements.md](../spec/requirements.md)), Python package `cody-tools` in the uv workspace; built in step 1b.
- Developer-only hardware test panel: start the Windows Cody bridge and open Cody from the tray icon. The panel uses the bridge's device connection; it does not open USB serial itself.

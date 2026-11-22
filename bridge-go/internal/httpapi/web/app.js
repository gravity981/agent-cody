"use strict";

const byId = (id) => document.getElementById(id);
const statusMessage = byId("status-message");
const statusDetails = byId("status-details");
const relayList = byId("relay-list");
const sessionList = byId("session-list");
const setupChecklist = byId("setup-checklist");
const setupMessage = byId("setup-message");
const settingsMessage = byId("settings-message");
const restartPanel = byId("restart-panel");
const restartMessage = byId("restart-message");
const settingsForm = byId("settings-form");
const testStatusMessage = byId("test-status");
const testError = byId("test-error");
const testReplies = byId("test-replies");
const testStates = byId("test-states");
let deviceConnected = false;
let devicePort = "";
let setupStatus = null;
let flashTargetPort = "";
let testActive = false;
let testBusy = false;
let flashBusy = false;
let flashModeTouched = false;
let tuningDirty = false;
let displayedTuning = "";
let tuningQuerySent = false;

async function request(path, options = {}) {
  const response = await fetch(path, {
    cache: "no-store",
    ...options,
  });
  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try {
      const body = await response.json();
      if (body.error) message = body.error;
    } catch (error) {
      if (!(error instanceof SyntaxError)) throw error;
    }
    throw new Error(message);
  }
  return response;
}

function addDetail(label, value) {
  const term = document.createElement("dt");
  term.textContent = label;
  const description = document.createElement("dd");
  description.textContent = value;
  statusDetails.append(term, description);
}

function renderStatus(status) {
  statusDetails.replaceChildren();
  addDetail("Bridge", `${status.bridge.version} (${status.bridge.commit})`);
  const device = status.device.connected
    ? `Connected on ${status.device.port || "unknown port"}`
    : "Not connected";
  addDetail("Device", device);
  addDetail("Firmware", status.device.firmware || "Unknown");
  addDetail("Merged state", status.state);
  addDetail("Sessions", String(status.sessions.length));
  statusMessage.textContent = "Connected to the local Cody bridge.";
  const wasConnected = deviceConnected;
  deviceConnected = status.device.connected;
  devicePort = status.device.port || "";
  renderSetupChecklist();
  if (wasConnected !== deviceConnected) {
    tuningQuerySent = false;
    void refreshFlashPorts();
  }
  if (deviceConnected && !tuningQuerySent) {
    tuningQuerySent = true;
    postTest("/api/test/tuning", { action: "query" }).catch((error) => {
      tuningQuerySent = false;
      testError.textContent = `Could not query motor tuning: ${error.message}`;
    });
  }
  renderTestControls();
  renderFlashControls();

  relayList.replaceChildren();
  if (status.relays.length === 0) {
    const empty = document.createElement("p");
    empty.textContent = "No configured WSL relays are reporting yet.";
    relayList.append(empty);
  }
  for (const relay of status.relays) {
    const entry = document.createElement("div");
    entry.className = "relay";
    const name = document.createElement("strong");
    name.textContent = relay.distro;
    const target = document.createElement("p");
    target.textContent = "WSL-host hook target: 127.0.0.1:8765";
    const loopback = document.createElement("p");
    loopback.textContent = `WSL relay loopback listener: ${relay.loopback_address || "unavailable (may be normal in mirrored networking)"}`;
    const docker = document.createElement("p");
    docker.textContent = `Docker Engine gateway target: ${relay.docker_address || "not listening"}`;
    const install = document.createElement("p");
    if (relay.docker_address) {
      install.append("In this distro's Docker Engine devcontainer, from the Cody repository root, run: ");
      const command = document.createElement("code");
      const port = relay.docker_address.slice(relay.docker_address.lastIndexOf(":") + 1);
      command.textContent = `uv run python hooks/install-claude-code --port ${port}`;
      install.append(command);
    } else {
      install.textContent = "The hook installer command will appear when this distro's Docker Engine gateway relay is listening.";
    }
    entry.append(name, target, loopback, docker, install);
    relayList.append(entry);
  }

  sessionList.replaceChildren();
  if (status.sessions.length === 0) {
    const empty = document.createElement("p");
    empty.textContent = "No active sessions.";
    sessionList.append(empty);
  }
  for (const session of status.sessions) {
    const entry = document.createElement("div");
    entry.className = "session";
    const line = document.createElement("p");
    line.textContent = `${session.agent} / ${session.session} - ${session.state}`;
    const age = document.createElement("p");
    age.className = "hint";
    age.textContent = `Last event ${session.idle_s} seconds ago`;
    entry.append(line, age);
    sessionList.append(entry);
  }
}

function renderSetupChecklist() {
  if (!setupStatus) return;
  setupChecklist.replaceChildren();
  const checks = [
    [
      "Device",
      deviceConnected ? "Connected" : "Not connected. Connect the Cody device over USB.",
    ],
    [
      "Claude Code hooks",
      setupStatus.claude_code_installed
        ? `Installed in ${setupStatus.claude_settings_path}`
        : `Not installed in ${setupStatus.claude_settings_path}`,
    ],
    [
      "GitHub Copilot hooks",
      setupStatus.copilot_installed
        ? `Installed in ${setupStatus.copilot_hooks_path}`
        : `Not installed in ${setupStatus.copilot_hooks_path}`,
    ],
    [
      "curl",
      setupStatus.curl_available
        ? "Available for agent hook forwarding."
        : "Missing. Install Git for Windows or make curl.exe available on PATH.",
    ],
  ];
  for (const [label, result] of checks) {
    const item = document.createElement("li");
    item.textContent = `${label}: ${result}`;
    setupChecklist.append(item);
  }
}

async function refreshStatus() {
  try {
    const response = await request("/api/status");
    renderStatus(await response.json());
  } catch (error) {
    statusMessage.textContent = `Could not load bridge status: ${error.message}`;
    deviceConnected = false;
    devicePort = "";
    renderSetupChecklist();
    renderTestControls();
    renderFlashControls();
  }
}

async function refreshSetupStatus() {
  try {
    const response = await request("/api/setup");
    setupStatus = await response.json();
    setupMessage.textContent = "Review the setup checks, then install the hooks.";
    renderSetupChecklist();
  } catch (error) {
    setupMessage.textContent = `Could not check hook setup: ${error.message}`;
  }
}

async function installHooks() {
  byId("install-hooks").disabled = true;
  setupMessage.textContent = "Installing hooks...";
  try {
    const response = await request("/api/setup", { method: "POST" });
    setupStatus = await response.json();
    setupMessage.textContent =
      "Hooks installed. Start new Claude Code and Copilot sessions to load them.";
    renderSetupChecklist();
  } catch (error) {
    setupMessage.textContent = `Could not install hooks: ${error.message}`;
  } finally {
    byId("install-hooks").disabled = false;
  }
}

function renderTestControls() {
  byId("test-start").disabled = !deviceConnected || testActive || testBusy || flashBusy;
  byId("test-stop").disabled = !testActive || testBusy || flashBusy;
  testStatusMessage.textContent = flashBusy
    ? "Firmware flashing is in progress."
    : !deviceConnected
    ? "Device not connected. Connect a Cody device to use the test panel."
    : testActive
      ? "Test mode is active. Exit, close this page, or let its 10-second heartbeat expire to restore agent state."
      : "Device connected. Enter test mode to enable the controls.";
  for (const control of document.querySelectorAll(
    "#test-states button, #test-speed, #test-frequency, #test-kick, #test-decay, " +
    "#test-apply-motor, #test-save-tuning, #test-reset-tuning, #test-motor-start, " +
    "#test-motor-stop, #test-red, #test-green, #test-blue, #test-show-led, #test-volume, " +
    "#test-led-brightness, #test-apply-led-brightness, #test-sound, #test-apply-volume, " +
    "#test-play-sound"
  )) {
    control.disabled = !deviceConnected || !testActive || testBusy || flashBusy;
  }
}

function renderFlashControls() {
  const mode = byId("flash-mode");
  const updateOption = mode.querySelector('option[value="update"]');
  updateOption.disabled = !deviceConnected;
  mode.disabled = flashBusy;
  if (!flashModeTouched && deviceConnected) mode.value = "update";
  if (!flashBusy && !deviceConnected && mode.value === "update") mode.value = "manual";

  const manualMode = mode.value === "manual";
  byId("flash-manual-controls").hidden = !manualMode;
  byId("flash-update-target").hidden = manualMode;
  const targetPort = flashBusy ? flashTargetPort : devicePort;
  byId("flash-update-target").textContent = targetPort
    ? `Connected Cody target: ${targetPort}`
    : "Connect a Cody device to update its firmware.";
  byId("flash-port").disabled = flashBusy;
  byId("flash-refresh-ports").disabled = flashBusy;
  byId("flash-image").disabled = flashBusy;
  const image = byId("flash-image").files[0];
  byId("flash-submit").disabled = flashBusy || testBusy ||
    !image ||
    (manualMode && !byId("flash-port").value) ||
    (!manualMode && !deviceConnected);
}

async function refreshFlashPorts() {
  const select = byId("flash-port");
  const previous = select.value;
  try {
    const response = await request("/api/test/flash/ports");
    const { ports } = await response.json();
    select.replaceChildren();
    for (const port of ports) {
      const option = document.createElement("option");
      option.value = port;
      option.textContent = port;
      select.append(option);
    }
    if (ports.includes(previous)) select.value = previous;
    if (ports.length === 0) {
      byId("flash-status").textContent = "No Espressif serial ports found. Connect the board and refresh.";
    } else if (byId("flash-status").textContent.startsWith("No Espressif serial ports")) {
      byId("flash-status").textContent = "";
    }
  } catch (error) {
    byId("flash-status").textContent = `Could not list Espressif serial ports: ${error.message}`;
  }
  renderFlashControls();
}

async function flashFirmware() {
  if (flashBusy) return;
  const image = byId("flash-image").files[0];
  if (!image) {
    byId("flash-status").textContent = "Choose a firmware image first.";
    return;
  }
  const mode = byId("flash-mode").value;
  const port = mode === "update" ? devicePort : byId("flash-port").value;
  if (!port) {
    byId("flash-status").textContent = "Select an Espressif serial port.";
    return;
  }
  if (!image.name.toLowerCase().endsWith(".bin") || image.size === 0 || image.size > 4 * 1024 * 1024) {
    byId("flash-status").textContent = "Choose a non-empty .bin file no larger than 4 MiB.";
    return;
  }
  const operation = mode === "manual" ? "manual ROM bootloader flash" : "tuning-preserving update";
  const tuningWarning = mode === "manual"
    ? "\n\nManual mode cannot back up motor tuning; saved tuning and its timestamp may be lost or reset."
    : "";
  if (!window.confirm(
    `This ${operation} will replace the device firmware.${tuningWarning}\n\nImage: ${image.name}\nTarget port: ${port}\n\nContinue?`
  )) {
    return;
  }

  flashBusy = true;
  flashTargetPort = port;
  byId("flash-status").textContent = "Uploading firmware image...";
  byId("flash-output").textContent = "";
  renderFlashControls();
  renderTestControls();
  try {
    const form = new FormData();
    form.set("mode", mode);
    form.set("port", port);
    form.set("image", image, image.name);
    const response = await request("/api/test/flash", { method: "POST", body: form });
    const { id } = await response.json();
    while (true) {
      await new Promise((resolve) => setTimeout(resolve, 500));
      const statusResponse = await request(`/api/test/flash?id=${encodeURIComponent(id)}`);
      const status = await statusResponse.json();
      byId("flash-status").textContent = status.done
        ? status.success
          ? "Firmware flash completed and the device reconnected."
          : `Firmware flash failed during ${status.phase.toLowerCase()}.`
        : status.phase;
      byId("flash-output").textContent = status.output;
      if (status.done) {
        if (!status.success) {
          let message = status.error || "Firmware flashing failed";
          if (status.mode === "update" && message.toLowerCase().includes("unknown command")) {
            message += ". Select Manual ROM bootloader flash to upgrade firmware without tuning backup.";
          }
          throw new Error(message);
        }
        break;
      }
    }
  } catch (error) {
    byId("flash-status").textContent = error.message;
  } finally {
    flashBusy = false;
    await refreshStatus();
    await refreshTestStatus();
    renderFlashControls();
    renderTestControls();
  }
}

async function postTest(path, payload = {}) {
  const response = await request(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return response.json();
}

async function refreshTestStatus() {
  try {
    const response = await request("/api/test/status");
    const status = await response.json();
    testActive = status.active;
    testReplies.textContent = status.replies.join("\n");
    renderTuning(status.tuning);
    renderTestControls();
    if (testActive && !testBusy) {
      await postTest("/api/test/heartbeat");
    }
  } catch (error) {
    testError.textContent = `Could not update test-panel status: ${error.message}`;
  }
}

function renderTuning(tuning) {
  const message = byId("tuning-message");
  const details = byId("tuning-details");
  details.replaceChildren();
  if (!deviceConnected) {
    message.textContent = "Connect a Cody device to view its motor tuning.";
    displayedTuning = "";
    return;
  }
  if (tuning.error) {
    message.textContent = `Could not read motor tuning: ${tuning.error}`;
    return;
  }
  if (!tuning.known) {
    message.textContent = "Waiting for motor tuning from the device...";
    return;
  }

  const summary = [
    tuning.speed,
    tuning.frequency_hz,
    tuning.kick_start_ms,
    tuning.decay,
    tuning.timestamp,
  ];
  const signature = JSON.stringify(summary);
  if (!tuningDirty && signature !== displayedTuning) {
    byId("test-speed").value = tuning.speed;
    byId("test-frequency").value = tuning.frequency_hz;
    byId("test-kick").value = tuning.kick_start_ms;
    byId("test-decay").value = tuning.decay;
  }
  displayedTuning = signature;
  message.textContent = "Current device motor tuning";
  const lastWrite = tuning.timestamp === "UNSAVED"
    ? "Firmware defaults (never saved)"
    : new Date(tuning.timestamp).toLocaleString();
  for (const [label, value] of [
    ["Speed", String(tuning.speed)],
    ["PWM frequency", `${tuning.frequency_hz} Hz`],
    ["Startup kick", `${tuning.kick_start_ms} ms`],
    ["Decay mode", tuning.decay],
    ["Last write", lastWrite],
  ]) {
    const term = document.createElement("dt");
    term.textContent = label;
    const description = document.createElement("dd");
    description.textContent = value;
    details.append(term, description);
  }
}

async function runTestAction(action) {
  if (testBusy) return;
  testBusy = true;
  testError.textContent = "";
  renderTestControls();
  renderFlashControls();
  try {
    await action();
  } catch (error) {
    testError.textContent = error.message;
  } finally {
    await refreshTestStatus();
    testBusy = false;
    renderTestControls();
    renderFlashControls();
  }
}

function sendTestCommand(action, value) {
  return postTest("/api/test/command", { action, value: String(value) });
}

function sendLedColour() {
  return postTest("/api/test/command", {
    action: "led",
    red: Number(byId("test-red").value),
    green: Number(byId("test-green").value),
    blue: Number(byId("test-blue").value),
  });
}

async function applyLedBrightness() {
  await sendLedColour();
  await sendTestCommand("led-brightness", byId("test-led-brightness").value);
}

async function applyMotorSettings() {
  await sendTestCommand("speed", byId("test-speed").value);
  await sendTestCommand("frequency", byId("test-frequency").value);
  await sendTestCommand("kick", byId("test-kick").value);
  await sendTestCommand("decay", byId("test-decay").value);
}

function setConfig(config) {
  byId("use-default-distro").checked = config.use_default_wsl_distro;
  byId("wsl-distros").value = config.wsl_distros.join("\n");
  byId("wsl-distros").disabled = config.use_default_wsl_distro;
  byId("volume").value = config.volume;
  byId("brightness").value = config.brightness;
  byId("volume-value").textContent = `${config.volume}%`;
  byId("brightness-value").textContent = `${config.brightness}%`;
  byId("test-led-brightness").value = config.brightness;
  byId("test-led-brightness-value").textContent = `${config.brightness}%`;
  byId("session-timeout").value = config.session_timeout_min;
  byId("serial-port").value = config.serial_port;
  byId("log-level").value = config.log_level;
}

async function loadConfig() {
  try {
    const response = await request("/api/config");
    setConfig(await response.json());
  } catch (error) {
    settingsMessage.textContent = `Could not load configuration: ${error.message}`;
  }
}

byId("use-default-distro").addEventListener("change", (event) => {
  byId("wsl-distros").disabled = event.target.checked;
});
byId("install-hooks").addEventListener("click", () => void installHooks());

settingsForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  settingsMessage.textContent = "Saving...";
  const payload = {
    use_default_wsl_distro: byId("use-default-distro").checked,
    wsl_distros: byId("wsl-distros").value
      .split(/\r?\n/)
      .map((name) => name.trim())
      .filter(Boolean),
    volume: Number(byId("volume").value),
    brightness: Number(byId("brightness").value),
    session_timeout_min: Number(byId("session-timeout").value),
    serial_port: byId("serial-port").value.trim(),
    log_level: byId("log-level").value,
  };
  try {
    const response = await request("/api/config", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    const result = await response.json();
    settingsMessage.textContent = result.restart_required
      ? "Settings saved. Restart Cody to apply them."
      : "Settings saved.";
    restartPanel.hidden = !result.restart_required;
  } catch (error) {
    settingsMessage.textContent = `Could not save configuration: ${error.message}`;
    restartPanel.hidden = true;
  }
});

byId("restart-button").addEventListener("click", async () => {
  byId("restart-button").disabled = true;
  restartMessage.textContent = "Restarting Cody...";
  try {
    await request("/api/restart", { method: "POST" });
    restartMessage.textContent = "Cody restarted; reconnecting to the bridge...";
    const deadline = Date.now() + 30000;
    while (Date.now() < deadline) {
      await new Promise((resolve) => setTimeout(resolve, 750));
      try {
        await request("/api/status");
        restartMessage.textContent = "Cody is running with the saved settings.";
        await refreshStatus();
        await loadConfig();
        return;
      } catch {
        // The bridge is expected to be unavailable while its service restarts.
      }
    }
    throw new Error("Cody did not become available within 30 seconds");
  } catch (error) {
    restartMessage.textContent = `Restart failed: ${error.message}`;
  } finally {
    byId("restart-button").disabled = false;
  }
});

for (const state of ["IDLE", "WORKING", "THINKING", "APPROVAL", "COMPLETED", "ERROR"]) {
  const button = document.createElement("button");
  button.type = "button";
  button.textContent = state;
  button.addEventListener("click", () => {
    void runTestAction(() => sendTestCommand("state", state));
  });
  testStates.append(button);
}

byId("test-start").addEventListener("click", () => {
  void runTestAction(() => postTest("/api/test/start"));
});
byId("test-stop").addEventListener("click", () => {
  void runTestAction(() => postTest("/api/test/stop"));
});
byId("test-apply-motor").addEventListener("click", () => {
  void runTestAction(applyMotorSettings);
});
byId("test-motor-start").addEventListener("click", () => {
  void runTestAction(async () => {
    await applyMotorSettings();
    await sendTestCommand("motor", "START");
  });
});
byId("test-motor-stop").addEventListener("click", () => {
  void runTestAction(() => sendTestCommand("motor", "STOP"));
});
byId("test-show-led").addEventListener("click", () => {
  void runTestAction(sendLedColour);
});
byId("test-apply-led-brightness").addEventListener("click", () => {
  void runTestAction(applyLedBrightness);
});
byId("test-apply-volume").addEventListener("click", () => {
  void runTestAction(() => sendTestCommand("volume", byId("test-volume").value));
});
byId("test-play-sound").addEventListener("click", () => {
  void runTestAction(async () => {
    await sendTestCommand("volume", byId("test-volume").value);
    await sendTestCommand("beep", byId("test-sound").value);
  });
});
for (const id of ["test-speed", "test-frequency", "test-kick", "test-decay"]) {
  byId(id).addEventListener("input", () => {
    tuningDirty = true;
  });
}
byId("test-save-tuning").addEventListener("click", () => {
  void runTestAction(async () => {
    await applyMotorSettings();
    await postTest("/api/test/tuning", { action: "save" });
    tuningDirty = false;
  });
});
byId("test-reset-tuning").addEventListener("click", () => {
  void runTestAction(async () => {
    await postTest("/api/test/tuning", { action: "reset" });
    tuningDirty = false;
  });
});
byId("flash-mode").addEventListener("change", () => {
  flashModeTouched = true;
  renderFlashControls();
});
byId("flash-port").addEventListener("change", renderFlashControls);
byId("flash-image").addEventListener("change", renderFlashControls);
byId("flash-refresh-ports").addEventListener("click", () => void refreshFlashPorts());
byId("flash-submit").addEventListener("click", () => void flashFirmware());
for (const [id, outputId] of [
  ["volume", "volume-value"],
  ["brightness", "brightness-value"],
  ["test-led-brightness", "test-led-brightness-value"],
  ["test-volume", "test-volume-value"],
]) {
  byId(id).addEventListener("input", () => {
    byId(outputId).textContent = `${byId(id).value}%`;
  });
}
window.addEventListener("pagehide", () => {
  navigator.sendBeacon(
    "/api/test/stop",
    new Blob(["{}"], { type: "application/json" })
  );
});

renderTestControls();
renderFlashControls();
void refreshStatus();
void refreshSetupStatus();
void loadConfig();
void refreshTestStatus();
void refreshFlashPorts();
window.setInterval(() => void refreshStatus(), 3000);
window.setInterval(() => void refreshTestStatus(), 2000);

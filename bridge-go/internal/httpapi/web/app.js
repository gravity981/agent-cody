"use strict";

const byId = (id) => document.getElementById(id);
const statusMessage = byId("status-message");
const statusDetails = byId("status-details");
const relayList = byId("relay-list");
const sessionList = byId("session-list");
const settingsMessage = byId("settings-message");
const restartPanel = byId("restart-panel");
const restartMessage = byId("restart-message");
const settingsForm = byId("settings-form");
const testStatusMessage = byId("test-status");
const testError = byId("test-error");
const testReplies = byId("test-replies");
const testStates = byId("test-states");
let deviceConnected = false;
let testActive = false;
let testBusy = false;

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
  deviceConnected = status.device.connected;
  renderTestControls();

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

async function refreshStatus() {
  try {
    const response = await request("/api/status");
    renderStatus(await response.json());
  } catch (error) {
    statusMessage.textContent = `Could not load bridge status: ${error.message}`;
    deviceConnected = false;
    renderTestControls();
  }
}

function renderTestControls() {
  byId("test-start").disabled = !deviceConnected || testActive || testBusy;
  byId("test-stop").disabled = !testActive || testBusy;
  testStatusMessage.textContent = !deviceConnected
    ? "Device not connected. Connect a Cody device to use the test panel."
    : testActive
      ? "Test mode is active. Exit, close this page, or let its 10-second heartbeat expire to restore agent state."
      : "Device connected. Enter test mode to enable the controls.";
  for (const control of document.querySelectorAll(
    "#test-states button, #test-speed, #test-frequency, #test-kick, #test-decay, " +
      "#test-apply-motor, #test-motor-start, #test-motor-stop, #test-red, #test-green, " +
      "#test-blue, #test-show-led, #test-volume, #test-sound, #test-apply-volume, #test-play-sound"
  )) {
    control.disabled = !deviceConnected || !testActive || testBusy;
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
    renderTestControls();
    if (testActive && !testBusy) {
      await postTest("/api/test/heartbeat");
    }
  } catch (error) {
    testError.textContent = `Could not update test-panel status: ${error.message}`;
  }
}

async function runTestAction(action) {
  if (testBusy) return;
  testBusy = true;
  testError.textContent = "";
  renderTestControls();
  try {
    await action();
  } catch (error) {
    testError.textContent = error.message;
  } finally {
    await refreshTestStatus();
    testBusy = false;
    renderTestControls();
  }
}

function sendTestCommand(action, value) {
  return postTest("/api/test/command", { action, value: String(value) });
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
  byId("speed").value = config.speed;
  byId("volume").value = config.volume;
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

settingsForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  settingsMessage.textContent = "Saving...";
  const payload = {
    use_default_wsl_distro: byId("use-default-distro").checked,
    wsl_distros: byId("wsl-distros").value
      .split(/\r?\n/)
      .map((name) => name.trim())
      .filter(Boolean),
    speed: Number(byId("speed").value),
    volume: Number(byId("volume").value),
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
  void runTestAction(() =>
    postTest("/api/test/command", {
      action: "led",
      red: Number(byId("test-red").value),
      green: Number(byId("test-green").value),
      blue: Number(byId("test-blue").value),
    })
  );
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
window.addEventListener("pagehide", () => {
  navigator.sendBeacon(
    "/api/test/stop",
    new Blob(["{}"], { type: "application/json" })
  );
});

renderTestControls();
void refreshStatus();
void loadConfig();
void refreshTestStatus();
window.setInterval(() => void refreshStatus(), 3000);
window.setInterval(() => void refreshTestStatus(), 2000);

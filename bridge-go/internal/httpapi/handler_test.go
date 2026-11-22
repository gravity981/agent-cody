package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/config"
	"github.com/gravity981/agent-cody/bridge-go/internal/device"
	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

type fakeLink struct {
	status        device.Status
	states        []sessions.State
	testActive    bool
	testCommands  []string
	stopped       []sessions.State
	replies       []string
	tuning        device.TuningStatus
	tuningQueries int
}

func (l *fakeLink) Status() device.Status {
	return l.status
}

func (l *fakeLink) SetState(state sessions.State, _ time.Time) {
	l.states = append(l.states, state)
}

func (l *fakeLink) StartTest(time.Time) error {
	if !l.status.Connected {
		return device.ErrNotConnected
	}
	l.testActive = true
	return nil
}

func (l *fakeLink) SendTestCommand(command string, _ time.Time) error {
	if !l.testActive {
		return errors.New("hardware test mode is not active")
	}
	l.testCommands = append(l.testCommands, command)
	return nil
}

func (l *fakeLink) QueryTuning(time.Time) error {
	if !l.status.Connected {
		return device.ErrNotConnected
	}
	l.tuningQueries++
	return nil
}

func (l *fakeLink) StopTest(state sessions.State, _ time.Time) error {
	if l.testActive {
		l.testActive = false
		l.stopped = append(l.stopped, state)
		l.states = append(l.states, state)
	}
	return nil
}

func (l *fakeLink) Replies() []string {
	return append([]string{}, l.replies...)
}

func (l *fakeLink) Tuning() device.TuningStatus {
	return l.tuning
}

func (*fakeLink) Step(time.Time) {}

func (*fakeLink) Close(time.Time) {}

func newTestServer(t *testing.T, websiteOptions ...WebsiteOptions) (*httptest.Server, *fakeLink, string) {
	t.Helper()
	port := "/dev/ttyACM0"
	firmware := "1.2.3 abc"
	link := &fakeLink{status: device.Status{
		Connected: true, Port: &port, Firmware: &firmware,
	}}
	app := bridge.New(
		sessions.New(10*time.Minute), link,
		logging.New(io.Discard, "info"), "2.0.0", "feed123",
	)
	now := time.Unix(100, 0)
	configPath := filepath.Join(t.TempDir(), "cody", "bridge.toml")
	options := WebsiteOptions{
		ConfigPath: configPath,
		RelayStatuses: func() []RelayStatus {
			return []RelayStatus{{
				Distro:          "Ubuntu",
				LoopbackAddress: "127.0.0.1:8765",
				DockerAddress:   "172.17.0.1:8767",
			}}
		},
	}
	if len(websiteOptions) > 0 {
		options = websiteOptions[0]
		if options.ConfigPath == "" {
			options.ConfigPath = configPath
		}
	}
	server := httptest.NewServer(NewHandler(app, func() time.Time { return now }, options))
	t.Cleanup(server.Close)
	return server, link, configPath
}

func request(t *testing.T, client *http.Client, method, url string, body []byte) (*http.Response, []byte) {
	return requestWithHeaders(t, client, method, url, body, nil)
}

func requestWithHeaders(
	t *testing.T,
	client *http.Client,
	method, url string,
	body []byte,
	headers http.Header,
) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if isWebsiteRequest(req.URL.Path) {
		req.Host = SiteAddress
	}
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response, data
}

func Test_req_BRG_01_ReceivesClaudeEventsAndRejectsBadRequests(t *testing.T) {
	server, link, _ := newTestServer(t)
	client := server.Client()
	response, _ := request(t, client, http.MethodPost, server.URL+"/hooks/claude-code",
		[]byte(`{"session_id":"s1","hook_event_name":"PreToolUse"}`))
	if response.StatusCode != http.StatusOK {
		t.Fatalf("valid hook status = %d", response.StatusCode)
	}
	response, data := request(t, client, http.MethodGet, server.URL+"/status", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status endpoint = %d", response.StatusCode)
	}
	var status bridge.Status
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	if status.State != sessions.WORKING || len(status.Sessions) != 1 ||
		status.Sessions[0].Agent != "claude-code" || status.Sessions[0].Session != "s1" {
		t.Fatalf("unexpected status: %+v", status)
	}
	if len(link.states) != 1 || link.states[0] != sessions.WORKING {
		t.Fatalf("device states = %v, want [WORKING]", link.states)
	}

	for _, test := range []struct {
		path string
		body string
		code int
	}{
		{"/hooks/unknown", `{"session_id":"s2","hook_event_name":"Stop"}`, http.StatusNotFound},
		{"/hooks/claude-code", `{"hook_event_name":"Stop"}`, http.StatusBadRequest},
		{"/hooks/claude-code", `[]`, http.StatusBadRequest},
	} {
		response, _ = request(t, client, http.MethodPost, server.URL+test.path, []byte(test.body))
		if response.StatusCode != test.code {
			t.Errorf("POST %s status = %d, want %d", test.path, response.StatusCode, test.code)
		}
	}
}

func Test_req_BRG_10_StatusReportsBridgeVersionAndCommit(t *testing.T) {
	server, link, _ := newTestServer(t)
	response, data := request(t, server.Client(), http.MethodGet, server.URL+"/status", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var status bridge.Status
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	if status.Bridge.Version != "2.0.0" || status.Bridge.Commit != "feed123" {
		t.Fatalf("bridge build info = %+v", status.Bridge)
	}
	if status.Device.Firmware == nil || *status.Device.Firmware != "1.2.3 abc" {
		t.Fatalf("firmware version = %v", status.Device.Firmware)
	}

	link.status = device.Status{}
	_, data = request(t, server.Client(), http.MethodGet, server.URL+"/status", nil)
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	if status.Device.Connected || status.Device.Port != nil || status.Device.Firmware != nil {
		t.Fatalf("disconnected device status = %+v", status.Device)
	}
}

func Test_req_BRG_12_ManualStateAndSessionClearEndpoints(t *testing.T) {
	server, _, _ := newTestServer(t)
	client := server.Client()
	response, _ := request(t, client, http.MethodPost, server.URL+"/state", []byte(`{"state":"APPROVAL"}`))
	if response.StatusCode != http.StatusOK {
		t.Fatalf("manual state status = %d", response.StatusCode)
	}
	response, _ = request(t, client, http.MethodPost, server.URL+"/state", []byte(`{"state":"SLEEPING"}`))
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid state status = %d, want 400", response.StatusCode)
	}
	response, _ = request(t, client, http.MethodDelete, server.URL+"/sessions", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("clear sessions status = %d", response.StatusCode)
	}
	response, data := request(t, client, http.MethodGet, server.URL+"/status?format=text", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(data, []byte("state:    IDLE\n")) ||
		!bytes.Contains(data, []byte("bridge:   2.0.0 feed123\n")) {
		t.Fatalf("text status = %q", data)
	}
}

func Test_req_BRG_15_DeleteManualAndAllSessions(t *testing.T) {
	server, _, _ := newTestServer(t)
	client := server.Client()
	_, _ = request(t, client, http.MethodPost, server.URL+"/state", []byte(`{"state":"WORKING"}`))
	_, _ = request(t, client, http.MethodDelete, server.URL+"/state", nil)
	response, data := request(t, client, http.MethodGet, server.URL+"/status", nil)
	var status bridge.Status
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || len(status.Sessions) != 0 || status.State != sessions.IDLE {
		t.Fatalf("status after manual clear = %+v", status)
	}
}

func Test_req_BRG_21_WebsiteServesStatusAndRelayRoutingData(t *testing.T) {
	server, _, _ := newTestServer(t)
	response, page := request(t, server.Client(), http.MethodGet, server.URL+"/", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(page, []byte("Configure Claude Code hooks to use the relay")) ||
		!bytes.Contains(page, []byte("Run it in the environment that owns Claude Code's settings file")) ||
		!bytes.Contains(page, []byte("172.17.0.1:&lt;relay-port&gt;")) {
		t.Fatalf("website response status=%d body=%q", response.StatusCode, page)
	}
	if response.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("website response has no content security policy")
	}
	response, script := request(t, server.Client(), http.MethodGet, server.URL+"/app.js", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(script, []byte("uv run python hooks/install-claude-code --port ${port}")) {
		t.Fatalf("website script status=%d body=%q", response.StatusCode, script)
	}

	response, data := request(t, server.Client(), http.MethodGet, server.URL+"/api/status", nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("website status endpoint = %d: %s", response.StatusCode, data)
	}
	var status struct {
		Bridge struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
		} `json:"bridge"`
		Device struct {
			Connected bool    `json:"connected"`
			Firmware  *string `json:"firmware"`
		} `json:"device"`
		Relays []RelayStatus `json:"relays"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	if status.Bridge.Version != "2.0.0" || status.Bridge.Commit != "feed123" ||
		!status.Device.Connected || status.Device.Firmware == nil ||
		len(status.Relays) != 1 || status.Relays[0].DockerAddress != "172.17.0.1:8767" {
		t.Fatalf("website status = %+v", status)
	}
}

func Test_req_BRG_22_WebsiteValidatesAndSavesConfiguration(t *testing.T) {
	server, _, configPath := newTestServer(t)
	client := server.Client()
	payload := `{"use_default_wsl_distro":false,"wsl_distros":["Ubuntu"],"volume":0,"brightness":100,"session_timeout_min":10,"serial_port":"COM3","log_level":"debug"}`
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Origin", SiteOrigin)
	response, data := requestWithHeaders(
		t, client, http.MethodPut, server.URL+"/api/config", []byte(payload), headers,
	)
	if response.StatusCode != http.StatusOK || !bytes.Contains(data, []byte(`"restart_required":true`)) {
		t.Fatalf("save config status=%d body=%s", response.StatusCode, data)
	}
	saved, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if saved.LegacySpeed != nil || saved.Volume != 0 || saved.Brightness != 100 ||
		saved.SessionTimeoutMin != 10 ||
		saved.SerialPort == nil || *saved.SerialPort != "COM3" || saved.LogLevel != "debug" ||
		len(saved.WSLDistros) != 1 || saved.WSLDistros[0] != "Ubuntu" {
		t.Fatalf("saved config = %+v", saved)
	}

	headers.Set("Origin", "https://attacker.example")
	response, _ = requestWithHeaders(
		t, client, http.MethodPut, server.URL+"/api/config", []byte(payload), headers,
	)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin save status = %d, want 403", response.StatusCode)
	}

	headers.Set("Origin", SiteOrigin)
	response, data = requestWithHeaders(
		t, client, http.MethodPut, server.URL+"/api/config",
		[]byte(strings.Replace(payload, `"volume":0`, `"volume":101`, 1)), headers,
	)
	if response.StatusCode != http.StatusBadRequest || !bytes.Contains(data, []byte("volume")) {
		t.Fatalf("invalid config status=%d body=%s", response.StatusCode, data)
	}
	saved, err = config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Volume != 0 {
		t.Fatalf("invalid config changed saved volume to %d", saved.Volume)
	}
	response, data = requestWithHeaders(
		t, client, http.MethodPut, server.URL+"/api/config",
		[]byte(strings.Replace(payload, `"brightness":100`, `"brightness":101`, 1)), headers,
	)
	if response.StatusCode != http.StatusBadRequest || !bytes.Contains(data, []byte("brightness")) {
		t.Fatalf("invalid brightness status=%d body=%s", response.StatusCode, data)
	}
}

func Test_req_BRG_22_SessionTimeoutFieldAcceptsIntegerAndFractionalValues(t *testing.T) {
	server, _, _ := newTestServer(t)
	response, page := request(t, server.Client(), http.MethodGet, server.URL+"/", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(page, []byte(`id="session-timeout" type="number" min="0" step="any" required`)) {
		t.Fatalf("session timeout field status=%d body=%q", response.StatusCode, page)
	}
}

func Test_req_BRG_22_ConfigurationUsesPercentSlidersWithoutMotorSpeed(t *testing.T) {
	server, _, _ := newTestServer(t)
	response, page := request(t, server.Client(), http.MethodGet, server.URL+"/", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(page, []byte(`id="volume" type="range" min="0" max="100" step="1"`)) ||
		!bytes.Contains(page, []byte(`id="brightness" type="range" min="0" max="100" step="1"`)) ||
		!bytes.Contains(page, []byte("legacy Python Docker bridge is not compatible")) ||
		bytes.Contains(page, []byte(`id="speed"`)) {
		t.Fatalf("settings controls status=%d body=%s", response.StatusCode, page)
	}
	response, page = request(t, server.Client(), http.MethodGet, server.URL+"/app.js", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(page, []byte("toLocaleString()")) ||
		!bytes.Contains(page, []byte("Firmware defaults (never saved)")) {
		t.Fatalf("tuning display script status=%d body=%s", response.StatusCode, page)
	}

	response, data := request(t, server.Client(), http.MethodGet, server.URL+"/api/config", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(data, []byte(`"volume":50`)) ||
		!bytes.Contains(data, []byte(`"brightness":100`)) ||
		bytes.Contains(data, []byte(`"speed"`)) {
		t.Fatalf("default website config status=%d body=%s", response.StatusCode, data)
	}
}

func Test_req_BRG_22_WebsiteRestartRequiresSameOriginAndRestartsService(t *testing.T) {
	restarted := false
	server, _, _ := newTestServer(t, WebsiteOptions{Restart: func() { restarted = true }})
	headers := make(http.Header)
	headers.Set("Origin", "https://attacker.example")
	response, _ := requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/restart", nil, headers,
	)
	if response.StatusCode != http.StatusForbidden || restarted {
		t.Fatalf("cross-origin restart status=%d, restarted=%t", response.StatusCode, restarted)
	}
	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/state", []byte(`{"state":"ERROR"}`), headers,
	)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-origin state mutation status = %d, want 403", response.StatusCode)
	}
	headers.Set("Origin", SiteOrigin)
	response, data := requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/restart", nil, headers,
	)
	if response.StatusCode != http.StatusAccepted || !restarted ||
		!bytes.Contains(data, []byte("restarting")) {
		t.Fatalf("restart status=%d restarted=%t body=%s", response.StatusCode, restarted, data)
	}
}

func Test_req_TST_06_WebsiteUsesTheBridgeForPanelControls(t *testing.T) {
	server, link, _ := newTestServer(t)
	client := server.Client()
	response, page := request(t, client, http.MethodGet, server.URL+"/", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(page, []byte("Hardware test panel")) ||
		!bytes.Contains(page, []byte("Save tuning")) ||
		!bytes.Contains(page, []byte("Factory reset")) ||
		!bytes.Contains(page, []byte(`id="test-volume" type="range" min="0" max="100" step="1"`)) ||
		!bytes.Contains(page, []byte("Device replies")) ||
		bytes.Contains(page, []byte("Serial device")) ||
		bytes.Contains(page, []byte("/api/connect")) {
		t.Fatalf("test panel page status=%d body=%s", response.StatusCode, page)
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Origin", "https://attacker.example")
	response, _ = requestWithHeaders(
		t, client, http.MethodPost, server.URL+"/api/test/start", []byte(`{}`), headers,
	)
	if response.StatusCode != http.StatusForbidden || link.testActive {
		t.Fatalf("cross-origin start status=%d active=%t", response.StatusCode, link.testActive)
	}

	headers.Set("Origin", SiteOrigin)
	response, _ = requestWithHeaders(
		t, client, http.MethodPost, server.URL+"/api/test/start", []byte(`{}`), headers,
	)
	if response.StatusCode != http.StatusOK || !link.testActive {
		t.Fatalf("test start status=%d active=%t", response.StatusCode, link.testActive)
	}
	response, _ = requestWithHeaders(
		t, client, http.MethodPost, server.URL+"/api/test/command",
		[]byte(`{"action":"state","value":"WORKING"}`), headers,
	)
	if response.StatusCode != http.StatusAccepted ||
		!equalStrings(link.testCommands, []string{"TEST:STATE:WORKING"}) {
		t.Fatalf("test command status=%d commands=%v", response.StatusCode, link.testCommands)
	}

	link.replies = []string{"OK", "ERR test mode inactive"}
	response, data := request(t, client, http.MethodGet, server.URL+"/api/test/status", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(data, []byte(`"active":true`)) ||
		!bytes.Contains(data, []byte("ERR test mode inactive")) {
		t.Fatalf("test status=%d body=%s", response.StatusCode, data)
	}
	response, _ = requestWithHeaders(
		t, client, http.MethodPost, server.URL+"/api/test/stop", []byte(`{}`), headers,
	)
	if response.StatusCode != http.StatusOK || link.testActive ||
		len(link.stopped) != 1 || link.stopped[0] != sessions.IDLE {
		t.Fatalf("test stop status=%d active=%t restored=%v", response.StatusCode, link.testActive, link.stopped)
	}
}

func Test_req_TST_09_WebsiteQueriesSavesAndResetsDeviceTuning(t *testing.T) {
	server, link, _ := newTestServer(t)
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Origin", "https://attacker.example")
	response, _ := requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/tuning",
		[]byte(`{"action":"query"}`), headers,
	)
	if response.StatusCode != http.StatusForbidden || link.tuningQueries != 0 {
		t.Fatalf("cross-origin tuning query status=%d query count=%d", response.StatusCode, link.tuningQueries)
	}
	headers.Set("Origin", SiteOrigin)

	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/tuning",
		[]byte(`{"action":"query"}`), headers,
	)
	if response.StatusCode != http.StatusAccepted || link.tuningQueries != 1 {
		t.Fatalf("query status=%d query count=%d", response.StatusCode, link.tuningQueries)
	}
	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/tuning",
		[]byte(`{"action":"save"}`), headers,
	)
	if response.StatusCode != http.StatusConflict || len(link.testCommands) != 0 {
		t.Fatalf("save outside test mode status=%d commands=%v", response.StatusCode, link.testCommands)
	}
	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/start", []byte(`{}`), headers,
	)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("test start status=%d", response.StatusCode)
	}

	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/tuning",
		[]byte(`{"action":"save"}`), headers,
	)
	if response.StatusCode != http.StatusAccepted || link.tuningQueries != 2 ||
		!equalStrings(link.testCommands, []string{"TEST:TUNING:SAVE:1970-01-01T00:01:40Z"}) {
		t.Fatalf("save status=%d queries=%d commands=%v", response.StatusCode, link.tuningQueries, link.testCommands)
	}
	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/tuning",
		[]byte(`{"action":"reset"}`), headers,
	)
	if response.StatusCode != http.StatusAccepted || link.tuningQueries != 3 ||
		!equalStrings(link.testCommands, []string{
			"TEST:TUNING:SAVE:1970-01-01T00:01:40Z",
			"TEST:TUNING:RESET:1970-01-01T00:01:40Z",
		}) {
		t.Fatalf("reset status=%d queries=%d commands=%v", response.StatusCode, link.tuningQueries, link.testCommands)
	}

	link.tuning = device.TuningStatus{
		Known: true, Speed: 90, FrequencyHz: 24000, KickStartMs: 7,
		Decay: "SLOW", Timestamp: "2026-11-22T11:45:00Z",
	}
	response, data := request(t, server.Client(), http.MethodGet, server.URL+"/api/test/status", nil)
	if response.StatusCode != http.StatusOK ||
		!bytes.Contains(data, []byte(`"frequency_hz":24000`)) ||
		!bytes.Contains(data, []byte(`"timestamp":"2026-11-22T11:45:00Z"`)) {
		t.Fatalf("tuning status=%d body=%s", response.StatusCode, data)
	}
}

func Test_req_TST_03_req_TST_05_WebsiteValidatesPanelCommandRanges(t *testing.T) {
	server, link, _ := newTestServer(t)
	headers := make(http.Header)
	headers.Set("Content-Type", "application/json")
	headers.Set("Origin", SiteOrigin)
	response, _ := requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/start", []byte(`{}`), headers,
	)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("test start status = %d", response.StatusCode)
	}

	response, data := requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/command",
		[]byte(`{"action":"frequency","value":"30001"}`), headers,
	)
	if response.StatusCode != http.StatusBadRequest || !bytes.Contains(data, []byte("frequency")) ||
		len(link.testCommands) != 0 {
		t.Fatalf("out-of-range frequency status=%d commands=%v body=%s", response.StatusCode, link.testCommands, data)
	}
	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/command",
		[]byte(`{"action":"led","red":256,"green":0,"blue":0}`), headers,
	)
	if response.StatusCode != http.StatusBadRequest || len(link.testCommands) != 0 {
		t.Fatalf("out-of-range LED status=%d commands=%v", response.StatusCode, link.testCommands)
	}
	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/command",
		[]byte(`{"action":"frequency","value":"30000"}`), headers,
	)
	if response.StatusCode != http.StatusAccepted ||
		!equalStrings(link.testCommands, []string{"TEST:FREQ:30000"}) {
		t.Fatalf("valid frequency status=%d commands=%v", response.StatusCode, link.testCommands)
	}
	response, data = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/command",
		[]byte(`{"action":"volume","value":"101"}`), headers,
	)
	if response.StatusCode != http.StatusBadRequest || !bytes.Contains(data, []byte("volume")) ||
		len(link.testCommands) != 1 {
		t.Fatalf("out-of-range volume status=%d commands=%v body=%s", response.StatusCode, link.testCommands, data)
	}
	response, _ = requestWithHeaders(
		t, server.Client(), http.MethodPost, server.URL+"/api/test/command",
		[]byte(`{"action":"volume","value":"100"}`), headers,
	)
	if response.StatusCode != http.StatusAccepted ||
		!equalStrings(link.testCommands, []string{"TEST:FREQ:30000", "TEST:VOLUME:100"}) {
		t.Fatalf("valid volume status=%d commands=%v", response.StatusCode, link.testCommands)
	}
}

func Test_req_TST_06_TestCommandRequestMapsPanelControlsToFirmwareProtocol(t *testing.T) {
	red, green, blue := 1, 2, 3
	tests := []struct {
		name     string
		request  testCommandRequest
		expected string
	}{
		{"state preset", testCommandRequest{Action: "state", Value: "APPROVAL"}, "TEST:STATE:APPROVAL"},
		{"motor start", testCommandRequest{Action: "motor", Value: "START"}, "TEST:MOTOR:START"},
		{"speed", testCommandRequest{Action: "speed", Value: "255"}, "TEST:SPEED:255"},
		{"frequency", testCommandRequest{Action: "frequency", Value: "1000"}, "TEST:FREQ:1000"},
		{"kick", testCommandRequest{Action: "kick", Value: "100"}, "TEST:KICK:100"},
		{"decay", testCommandRequest{Action: "decay", Value: "SLOW"}, "TEST:DECAY:SLOW"},
		{"volume", testCommandRequest{Action: "volume", Value: "100"}, "TEST:VOLUME:100"},
		{"beep", testCommandRequest{Action: "beep", Value: "SUCCESS"}, "TEST:BEEP:SUCCESS"},
		{
			"LED", testCommandRequest{Action: "led", Red: &red, Green: &green, Blue: &blue},
			"TEST:LED:1,2,3",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.request.command()
			if err != nil || got != test.expected {
				t.Fatalf("command() = %q, %v; want %q", got, err, test.expected)
			}
		})
	}
	if _, err := (testCommandRequest{Action: "volume", Value: "101"}).command(); err == nil {
		t.Fatal("volume above 100 was accepted")
	}
}

func Test_req_BRG_21_WebsiteRejectsUnexpectedHost(t *testing.T) {
	server, _, _ := newTestServer(t)
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/config", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "attacker.example"
	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected Host status = %d, want 403", response.StatusCode)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

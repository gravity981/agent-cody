package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/device"
	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

type fakeLink struct {
	status device.Status
	states []sessions.State
}

func (l *fakeLink) Status() device.Status {
	return l.status
}

func (l *fakeLink) SetState(state sessions.State, _ time.Time) {
	l.states = append(l.states, state)
}

func (*fakeLink) Step(time.Time) {}

func newTestServer(t *testing.T) (*httptest.Server, *fakeLink) {
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
	server := httptest.NewServer(NewHandler(app, func() time.Time { return now }))
	t.Cleanup(server.Close)
	return server, link
}

func request(t *testing.T, client *http.Client, method, url string, body []byte) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
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
	server, link := newTestServer(t)
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
	server, link := newTestServer(t)
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
	server, _ := newTestServer(t)
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
	server, _ := newTestServer(t)
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

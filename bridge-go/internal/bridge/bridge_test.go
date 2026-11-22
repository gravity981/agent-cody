package bridge

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/device"
	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/mapping"
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
	steps         int
	closed        bool
	ports         []device.PortInfo
	suspended     bool
	restored      *device.TuningStatus
}

func (l *fakeLink) Status() device.Status {
	return l.status
}

func (l *fakeLink) SetState(state sessions.State, _ time.Time) {
	if len(l.states) == 0 || l.states[len(l.states)-1] != state {
		l.states = append(l.states, state)
	}
}

func (l *fakeLink) StartTest(time.Time) error {
	l.testActive = true
	return nil
}

func (l *fakeLink) SendTestCommand(command string, _ time.Time) error {
	l.testCommands = append(l.testCommands, command)
	return nil
}

func (l *fakeLink) QueryTuning(time.Time) error {
	l.tuningQueries++
	return nil
}

func (l *fakeLink) EspressifPorts() ([]device.PortInfo, error) {
	return append([]device.PortInfo{}, l.ports...), nil
}

func (l *fakeLink) Suspend(time.Time) {
	l.suspended = true
	l.status.Connected = false
}

func (l *fakeLink) Resume(time.Time) {
	l.suspended = false
	l.status.Connected = true
}

func (l *fakeLink) RestoreTuningAfterReconnect(tuning device.TuningStatus) error {
	l.restored = &tuning
	return nil
}

func (l *fakeLink) StopTest(state sessions.State, _ time.Time) error {
	l.testActive = false
	l.stopped = append(l.stopped, state)
	l.SetState(state, time.Time{})
	return nil
}

func (l *fakeLink) Replies() []string {
	return append([]string{}, l.replies...)
}

func (l *fakeLink) Tuning() device.TuningStatus {
	return l.tuning
}

func (l *fakeLink) Step(time.Time) {
	l.steps++
}

func (l *fakeLink) Close(time.Time) {
	l.closed = true
}

func Test_req_BRG_02_BridgeMergesSessionsAndIgnoresUnmappedEvents(t *testing.T) {
	now := time.Unix(100, 0)
	link := &fakeLink{}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "one", Name: "PreToolUse", State: sessions.WORKING,
	}, now)
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "two", Name: "PermissionRequest", State: sessions.APPROVAL,
	}, now)
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "one", Name: "unknown", Ignored: true,
	}, now.Add(time.Second))

	if got := app.Status(now).State; got != sessions.APPROVAL {
		t.Fatalf("merged state = %s, want APPROVAL", got)
	}
	if got := len(app.Status(now).Sessions); got != 2 {
		t.Fatalf("session count = %d, want 2", got)
	}
}

func Test_req_AGT_08_DuplicateCopilotSourcesCountOneWorkspaceEvent(t *testing.T) {
	start := time.Unix(100, 0)
	link := &fakeLink{}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	event := mapping.Event{
		Agent: "copilot", Session: "/workspace/project", Name: "PreToolUse",
		State: sessions.WORKING, DedupKey: "copilot-event-1",
	}
	app.Event(event, start)
	app.Event(event, start.Add(5*time.Second))

	status := app.Status(start.Add(5 * time.Second))
	if len(status.Sessions) != 1 || status.Sessions[0].Agent != "copilot" ||
		status.Sessions[0].Session != "/workspace/project" ||
		status.Sessions[0].IdleSeconds != 5 {
		t.Fatalf("duplicate event changed or duplicated the workspace session: %+v", status)
	}
}

func Test_req_BRG_03_SendsOnlyChangedMergedState(t *testing.T) {
	now := time.Unix(100, 0)
	link := &fakeLink{status: device.Status{Connected: true}}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "one", Name: "PreToolUse", State: sessions.WORKING,
	}, now)
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "two", Name: "Stop", State: sessions.COMPLETED,
	}, now)
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "one", Name: "Stop", State: sessions.COMPLETED,
	}, now)

	if got, want := link.states, []sessions.State{
		sessions.WORKING, sessions.COMPLETED,
	}; !equalStates(got, want) {
		t.Fatalf("sent states = %v, want %v", got, want)
	}
}

func Test_req_BRG_04_TickExpiresActiveSessionAndKeepsFinishedSession(t *testing.T) {
	start := time.Unix(100, 0)
	link := &fakeLink{}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "active", Name: "PreToolUse", State: sessions.WORKING,
	}, start)
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "finished", Name: "Stop", State: sessions.COMPLETED,
	}, start)

	app.Tick(start.Add(time.Minute))
	status := app.Status(start.Add(time.Minute))
	if status.State != sessions.COMPLETED || len(status.Sessions) != 1 {
		t.Fatalf("status after active timeout = %+v", status)
	}
	if link.steps != 1 {
		t.Fatalf("device step count = %d, want 1", link.steps)
	}
}

func Test_req_BRG_13_LogsStateChangeAndDelivery(t *testing.T) {
	var output bytes.Buffer
	logger := logging.New(&output, "info")
	app := New(sessions.New(time.Minute), &fakeLink{}, logger, "v", "c")
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "s1", Name: "PreToolUse", State: sessions.WORKING,
	}, time.Unix(100, 0))

	if !strings.Contains(output.String(), " INFO  state WORKING (device not connected) <- claude-code s1 PreToolUse\n") {
		t.Fatalf("state change was not logged: %q", output.String())
	}
}

func Test_req_BRG_13_DebugLogsEachReceivedEvent(t *testing.T) {
	var output bytes.Buffer
	app := New(
		sessions.New(time.Minute), &fakeLink{}, logging.New(&output, "debug"), "v", "c",
	)
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "s1", Name: "PreToolUse", State: sessions.WORKING,
	}, time.Unix(100, 0))

	if !strings.Contains(output.String(), "DEBUG event claude-code s1 PreToolUse -> WORKING\n") {
		t.Fatalf("event was not logged at debug level: %q", output.String())
	}
}

func Test_req_BRG_15_ClearRemovesAllSessionsAndSetsIdle(t *testing.T) {
	now := time.Unix(100, 0)
	link := &fakeLink{}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	app.SetManual(sessions.APPROVAL, now)
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "s1", Name: "PreToolUse", State: sessions.WORKING,
	}, now)
	app.ClearSessions(now.Add(time.Second))

	if status := app.Status(now.Add(time.Second)); status.State != sessions.IDLE || len(status.Sessions) != 0 {
		t.Fatalf("status after clearing sessions = %+v", status)
	}
}

func Test_req_TST_08_TestModeSuppressesAgentWritesAndRestoresLatestState(t *testing.T) {
	now := time.Unix(100, 0)
	link := &fakeLink{status: device.Status{Connected: true}}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	if err := app.StartTest(now); err != nil {
		t.Fatal(err)
	}
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "one", Name: "PreToolUse", State: sessions.WORKING,
	}, now.Add(time.Second))

	if got := len(link.states); got != 0 {
		t.Fatalf("normal state writes during test mode = %v", link.states)
	}
	if got := app.Status(now.Add(time.Second)).State; got != sessions.WORKING {
		t.Fatalf("merged state during test mode = %s, want WORKING", got)
	}
	if err := app.StopTest(now.Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if got, want := link.stopped, []sessions.State{sessions.WORKING}; !equalStates(got, want) {
		t.Fatalf("restored states = %v, want %v", got, want)
	}
	if app.TestStatus().Active || link.testActive {
		t.Fatal("test mode remained active after stop")
	}
}

func Test_req_TST_08_TestHeartbeatExpiryRestoresLatestMergedState(t *testing.T) {
	start := time.Unix(100, 0)
	link := &fakeLink{status: device.Status{Connected: true}}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	if err := app.StartTest(start); err != nil {
		t.Fatal(err)
	}
	app.Event(mapping.Event{
		Agent: "claude-code", Session: "one", Name: "PreToolUse", State: sessions.THINKING,
	}, start.Add(time.Second))
	if err := app.TestHeartbeat(start.Add(8 * time.Second)); err != nil {
		t.Fatal(err)
	}

	app.Tick(start.Add(18 * time.Second))
	if got, want := link.stopped, []sessions.State{sessions.THINKING}; !equalStates(got, want) {
		t.Fatalf("restored states after timeout = %v, want %v", got, want)
	}
	if app.TestStatus().Active {
		t.Fatal("test mode remained active after heartbeat expiry")
	}
}

func Test_req_TST_08_ShutdownStopsTestModeAndRestoresMergedState(t *testing.T) {
	now := time.Unix(100, 0)
	link := &fakeLink{status: device.Status{Connected: true}}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	if err := app.StartTest(now); err != nil {
		t.Fatal(err)
	}
	app.SetManual(sessions.APPROVAL, now.Add(time.Second))

	app.Close(now.Add(2 * time.Second))
	if !link.closed || link.testActive || !equalStates(link.stopped, []sessions.State{sessions.APPROVAL}) {
		t.Fatalf("bridge shutdown: closed=%t active=%t restored=%v", link.closed, link.testActive, link.stopped)
	}
}

func Test_req_TST_09_BridgeQueriesAndExposesDeviceTuning(t *testing.T) {
	now := time.Unix(100, 0)
	tuning := device.TuningStatus{
		Known: true, Speed: 90, FrequencyHz: 24000, KickStartMs: 7,
		Decay: "SLOW", Timestamp: "2026-11-22T11:45:00Z",
	}
	link := &fakeLink{tuning: tuning}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")

	if err := app.QueryTuning(now); err != nil {
		t.Fatal(err)
	}
	if link.tuningQueries != 1 {
		t.Fatalf("tuning queries = %d, want 1", link.tuningQueries)
	}
	if got := app.TestStatus().Tuning; got != tuning {
		t.Fatalf("reported tuning = %+v, want %+v", got, tuning)
	}
}

func Test_req_TST_10_FirmwareFlashReleasesDeviceAndRestoresLatestState(t *testing.T) {
	now := time.Unix(100, 0)
	portName := "COM4"
	firmware := "1.2.3 abc"
	tuning := device.TuningStatus{
		Known: true, Speed: 155, FrequencyHz: 18000, KickStartMs: 8,
		Decay: "SLOW", Timestamp: "2026-11-22T12:00:00Z",
	}
	link := &fakeLink{
		status: device.Status{Connected: true, Port: &portName, Firmware: &firmware},
		tuning: tuning,
	}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")
	if err := app.StartTest(now); err != nil {
		t.Fatal(err)
	}
	if err := app.BeginFirmwareFlash("update", portName, &tuning, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if !link.suspended || link.testActive || link.restored == nil {
		t.Fatalf("flash did not stop test, suspend link and preserve tuning: %+v", link)
	}

	app.Event(mapping.Event{
		Agent: "claude-code", Session: "new", Name: "PermissionRequest", State: sessions.APPROVAL,
	}, now.Add(2*time.Second))
	if err := app.EndFirmwareFlash(now.Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if link.suspended || !link.status.Connected {
		t.Fatalf("link did not resume after flash: %+v", link)
	}
	if got := app.Status(now.Add(3 * time.Second)).State; got != sessions.APPROVAL {
		t.Fatalf("merged state after flash = %s, want APPROVAL", got)
	}
}

func Test_req_TST_10_ManualFlashAllowsRecognizedDeviceOnAvailableEspressifPort(t *testing.T) {
	port := "COM4"
	link := &fakeLink{
		status: device.Status{Connected: true, Port: &port},
		ports:  []device.PortInfo{{Device: "COM4", VID: "303A"}},
	}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")

	if err := app.BeginFirmwareFlash("manual", "COM5", nil, time.Unix(100, 0)); err == nil {
		t.Fatal("manual flash accepted an unavailable port")
	}
	if err := app.BeginFirmwareFlash("manual", "COM4", nil, time.Unix(101, 0)); err != nil {
		t.Fatal(err)
	}
	if !link.suspended {
		t.Fatal("manual flash did not suspend the connected device")
	}
	if err := app.EndFirmwareFlash(time.Unix(102, 0)); err != nil {
		t.Fatal(err)
	}
}

func Test_req_TST_10_ManualFlashRefusesDifferentConnectedPort(t *testing.T) {
	connectedPort := "COM4"
	link := &fakeLink{
		status: device.Status{Connected: true, Port: &connectedPort},
		ports:  []device.PortInfo{{Device: "COM5", VID: "303A"}},
	}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")

	if err := app.BeginFirmwareFlash("manual", "COM5", nil, time.Unix(100, 0)); err == nil {
		t.Fatal("manual flash accepted a different port while Cody was connected")
	}
	if link.suspended {
		t.Fatal("manual flash suspended the bridge before rejecting the port")
	}
}

func Test_req_TST_10_ManualFlashAllowsUnprogrammedDevice(t *testing.T) {
	link := &fakeLink{
		ports: []device.PortInfo{{Device: "COM4", VID: "303A"}},
	}
	app := New(sessions.New(time.Minute), link, logging.New(io.Discard, "info"), "v", "c")

	if err := app.BeginFirmwareFlash("manual", "COM4", nil, time.Unix(100, 0)); err != nil {
		t.Fatal(err)
	}
	if !link.suspended {
		t.Fatal("manual flash did not suspend serial discovery")
	}
	if err := app.EndFirmwareFlash(time.Unix(101, 0)); err != nil {
		t.Fatal(err)
	}
}

func equalStates(got, want []sessions.State) bool {
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

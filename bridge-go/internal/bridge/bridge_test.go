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
	status device.Status
	states []sessions.State
	steps  int
}

func (l *fakeLink) Status() device.Status {
	return l.status
}

func (l *fakeLink) SetState(state sessions.State, _ time.Time) {
	if len(l.states) == 0 || l.states[len(l.states)-1] != state {
		l.states = append(l.states, state)
	}
}

func (l *fakeLink) Step(time.Time) {
	l.steps++
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

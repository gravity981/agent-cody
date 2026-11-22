package bridge

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/device"
	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/mapping"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

const ManualAgent = "manual"
const ManualSession = "manual"

const TestHeartbeatTimeout = 10 * time.Second

var (
	ErrTestModeActive   = errors.New("hardware test mode is already active")
	ErrTestModeInactive = errors.New("hardware test mode is not active")
)

type DeviceLink interface {
	Status() device.Status
	SetState(sessions.State, time.Time)
	StartTest(time.Time) error
	SendTestCommand(string, time.Time) error
	StopTest(sessions.State, time.Time) error
	Replies() []string
	Step(time.Time)
	Close(time.Time)
}

type Bridge struct {
	mu                 sync.Mutex
	sessions           *sessions.Sessions
	link               DeviceLink
	logger             *logging.Logger
	version            string
	commit             string
	lastState          sessions.State
	testMode           bool
	lastPanelHeartbeat time.Time
}

func New(
	activeSessions *sessions.Sessions,
	link DeviceLink,
	logger *logging.Logger,
	version, commit string,
) *Bridge {
	return &Bridge{
		sessions:  activeSessions,
		link:      link,
		logger:    logger,
		version:   version,
		commit:    commit,
		lastState: sessions.IDLE,
	}
}

func (b *Bridge) Event(event mapping.Event, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	cause := fmt.Sprintf("%s %s %s", event.Agent, event.Session, event.Name)
	stateName := "None"
	if event.End {
		stateName = mapping.End
	} else if event.State != "" {
		stateName = string(event.State)
	}
	b.logger.Debugf("event %s -> %s", cause, stateName)
	if event.Ignored {
		return
	}
	if event.End {
		b.sessions.End(event.Agent, event.Session)
	} else {
		b.sessions.Update(event.Agent, event.Session, event.State, now)
	}
	b.apply(now, cause)
}

func (b *Bridge) SetManual(state sessions.State, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessions.Update(ManualAgent, ManualSession, state, now)
	b.apply(now, "manual "+string(state))
}

func (b *Bridge) ClearManual(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessions.End(ManualAgent, ManualSession)
	b.apply(now, "manual clear")
}

func (b *Bridge) ClearSessions(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessions.Clear()
	b.apply(now, "sessions cleared")
}

func (b *Bridge) Tick(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessions.Expire(now)
	if b.testMode && now.Sub(b.lastPanelHeartbeat) >= TestHeartbeatTimeout {
		if err := b.stopTest(now, "panel heartbeat expired"); err != nil {
			b.logger.Warningf("stop hardware test after panel heartbeat expired: %v", err)
		}
	}
	b.apply(now, "session timeout")
	b.link.Step(now)
}

func (b *Bridge) apply(now time.Time, cause string) {
	b.sessions.Expire(now)
	state := b.sessions.Merged()
	deviceStatus := b.link.Status()
	if state != b.lastState {
		delivery := "device not connected"
		if b.testMode {
			delivery = "test mode; device writes suspended"
		} else if deviceStatus.Connected {
			delivery = "sent"
		}
		b.logger.Infof("state %s (%s) <- %s", state, delivery, cause)
		b.lastState = state
	}
	if b.testMode {
		return
	}
	b.link.SetState(state, now)
}

func (b *Bridge) StartTest(now time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.testMode {
		return ErrTestModeActive
	}
	if err := b.link.StartTest(now); err != nil {
		return err
	}
	b.testMode = true
	b.lastPanelHeartbeat = now
	b.logger.Infof("hardware test mode started")
	return nil
}

func (b *Bridge) TestHeartbeat(now time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.testMode {
		return ErrTestModeInactive
	}
	b.lastPanelHeartbeat = now
	return nil
}

func (b *Bridge) SendTestCommand(command string, now time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.testMode {
		return ErrTestModeInactive
	}
	return b.link.SendTestCommand(command, now)
}

func (b *Bridge) StopTest(now time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stopTest(now, "panel stopped")
}

func (b *Bridge) stopTest(now time.Time, cause string) error {
	if !b.testMode {
		return nil
	}
	b.testMode = false
	b.sessions.Expire(now)
	state := b.sessions.Merged()
	b.lastState = state
	b.logger.Infof("hardware test mode stopped (%s); restoring state %s", cause, state)
	return b.link.StopTest(state, now)
}

type TestStatus struct {
	Active  bool     `json:"active"`
	Replies []string `json:"replies"`
}

func (b *Bridge) TestStatus() TestStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	return TestStatus{Active: b.testMode, Replies: b.link.Replies()}
}

func (b *Bridge) Close(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.testMode {
		if err := b.stopTest(now, "bridge shutting down"); err != nil {
			b.logger.Warningf("stop hardware test during shutdown: %v", err)
		}
	}
	b.link.Close(now)
}

func (b *Bridge) Status(now time.Time) Status {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := Status{
		Bridge:   BuildInfo{Version: b.version, Commit: b.commit},
		Device:   b.link.Status(),
		State:    b.sessions.Merged(),
		Sessions: make([]SessionStatus, 0),
	}
	for _, session := range b.sessions.All() {
		result.Sessions = append(result.Sessions, SessionStatus{
			Agent: session.Agent, Session: session.ID, State: session.State,
			IdleSeconds: session.IdleSeconds(now),
		})
	}
	return result
}

type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

type SessionStatus struct {
	Agent       string         `json:"agent"`
	Session     string         `json:"session"`
	State       sessions.State `json:"state"`
	IdleSeconds int            `json:"idle_s"`
}

type Status struct {
	Bridge   BuildInfo       `json:"bridge"`
	Device   device.Status   `json:"device"`
	State    sessions.State  `json:"state"`
	Sessions []SessionStatus `json:"sessions"`
}

func StatusText(status Status) string {
	deviceLine := "device:   not connected"
	if status.Device.Connected && status.Device.Port != nil && status.Device.Firmware != nil {
		deviceLine = fmt.Sprintf(
			"device:   connected on %s, firmware %s",
			*status.Device.Port, *status.Device.Firmware,
		)
	}
	output := fmt.Sprintf(
		"bridge:   %s %s\n%s\nstate:    %s\nsessions: %d\n",
		status.Bridge.Version, status.Bridge.Commit, deviceLine, status.State, len(status.Sessions),
	)
	for _, session := range status.Sessions {
		output += fmt.Sprintf(
			"  %-12s %-10s %s  (last event %d s ago)\n",
			session.Agent, session.State, session.Session, session.IdleSeconds,
		)
	}
	return output
}

package bridge

import (
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

type DeviceLink interface {
	Status() device.Status
	SetState(sessions.State, time.Time)
	Step(time.Time)
}

type Bridge struct {
	mu        sync.Mutex
	sessions  *sessions.Sessions
	link      DeviceLink
	logger    *logging.Logger
	version   string
	commit    string
	lastState sessions.State
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
	b.apply(now, "session timeout")
	b.link.Step(now)
}

func (b *Bridge) apply(now time.Time, cause string) {
	b.sessions.Expire(now)
	state := b.sessions.Merged()
	deviceStatus := b.link.Status()
	if state != b.lastState {
		delivery := "device not connected"
		if deviceStatus.Connected {
			delivery = "sent"
		}
		b.logger.Infof("state %s (%s) <- %s", state, delivery, cause)
		b.lastState = state
	}
	b.link.SetState(state, now)
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

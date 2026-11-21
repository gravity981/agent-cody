package device

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

type fakePort struct {
	cody      bool
	version   string
	commands  []string
	replies   bytes.Buffer
	readError error
	closed    bool
}

func newFakePort() *fakePort {
	return &fakePort{cody: true, version: "1.2.3 abc"}
}

func (p *fakePort) Write(data []byte) (int, error) {
	if p.closed || p.readError != nil {
		return 0, errors.New("device disconnected")
	}
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte{'\n'}) {
		command := string(line)
		p.commands = append(p.commands, command)
		if !p.cody {
			continue
		}
		if command == "VERSION?" {
			_, _ = fmt.Fprintf(&p.replies, "VERSION %s\n", p.version)
		}
		_, _ = io.WriteString(&p.replies, "OK\n")
	}
	return len(data), nil
}

func (p *fakePort) Read(buffer []byte) (int, error) {
	if p.readError != nil {
		return 0, p.readError
	}
	if p.closed {
		return 0, errors.New("port closed")
	}
	if p.replies.Len() == 0 {
		return 0, nil
	}
	return p.replies.Read(buffer)
}

func (p *fakePort) Close() error {
	p.closed = true
	return nil
}

type fakeSystem struct {
	ports  map[string]*fakePort
	vids   map[string]string
	opened []string
}

func newFakeSystem() *fakeSystem {
	return &fakeSystem{
		ports: make(map[string]*fakePort),
		vids:  make(map[string]string),
	}
}

func (s *fakeSystem) plug(name string, port *fakePort, vid string) {
	s.ports[name] = port
	s.vids[name] = vid
}

func (s *fakeSystem) ListPorts() ([]PortInfo, error) {
	ports := make([]PortInfo, 0, len(s.ports))
	for name, vid := range s.vids {
		ports = append(ports, PortInfo{Device: name, VID: vid})
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i].Device < ports[j].Device })
	return ports, nil
}

func (s *fakeSystem) Open(name string) (Port, error) {
	port, ok := s.ports[name]
	if !ok {
		return nil, fmt.Errorf("no such device %s", name)
	}
	s.opened = append(s.opened, name)
	return port, nil
}

func makeLink(system System, serialPort string) *Link {
	return NewLink(system, logging.New(io.Discard, "info"), 224, 2, serialPort)
}

func connect(link *Link, start time.Time) time.Time {
	link.Step(start)
	connectedAt := start.Add(100 * time.Millisecond)
	link.Step(connectedAt)
	return connectedAt
}

func Test_req_BRG_03_SendsStateOnlyWhenItChanges(t *testing.T) {
	system := newFakeSystem()
	port := newFakePort()
	system.plug("/dev/ttyACM0", port, "303A")
	link := makeLink(system, "")
	now := connect(link, time.Unix(0, 0))
	port.commands = nil

	link.SetState(sessions.WORKING, now)
	link.SetState(sessions.WORKING, now.Add(time.Second))
	link.SetState(sessions.THINKING, now.Add(2*time.Second))

	if got, want := port.commands, []string{"STATE:WORKING", "STATE:THINKING"}; !equal(got, want) {
		t.Fatalf("commands = %v, want %v", got, want)
	}
}

func Test_req_BRG_07_SendsSettingsOnConnect(t *testing.T) {
	system := newFakeSystem()
	port := newFakePort()
	system.plug("/dev/ttyACM0", port, "303A")
	link := NewLink(system, logging.New(io.Discard, "info"), 90, 1, "")
	connect(link, time.Unix(0, 0))

	if got, want := port.commands[:4],
		[]string{"VERSION?", "SPEED:90", "VOLUME:1", "STATE:IDLE"}; !equal(got, want) {
		t.Fatalf("initial connection commands = %v, want %v", got, want)
	}
}

func Test_req_BRG_05_SendsHeartbeatsAtLeastEveryTenSeconds(t *testing.T) {
	system := newFakeSystem()
	port := newFakePort()
	system.plug("/dev/ttyACM0", port, "303A")
	link := makeLink(system, "")
	start := time.Unix(0, 0)
	now := connect(link, start)
	for ; now.Before(start.Add(21 * time.Second)); now = now.Add(100 * time.Millisecond) {
		link.Step(now)
	}

	count := 0
	for _, command := range port.commands {
		if command == "PING" {
			count++
		}
	}
	if count < 4 {
		t.Fatalf("sent %d PING commands in 20 seconds, want at least 4", count)
	}
}

func Test_req_BRG_13_DebugLogsSerialTraffic(t *testing.T) {
	system := newFakeSystem()
	port := newFakePort()
	system.plug("/dev/ttyACM0", port, "303A")
	var output bytes.Buffer
	logger := logging.New(&output, "debug")
	link := NewLink(system, logger, 224, 2, "")
	now := connect(link, time.Unix(0, 0))
	link.Step(now.Add(5 * time.Second))

	for _, line := range []string{"-> VERSION?", "<- VERSION 1.2.3 abc", "-> SPEED:224", "<- OK", "-> PING"} {
		if !strings.Contains(output.String(), line) {
			t.Errorf("missing debug log %q in %s", line, output.String())
		}
	}
	if !strings.Contains(output.String(), "connected to /dev/ttyACM0 (firmware 1.2.3 abc)") {
		t.Fatalf("connection was not logged: %s", output.String())
	}
	port.readError = errors.New("device disconnected")
	link.Step(now.Add(6 * time.Second))
	if link.Status().Connected || !strings.Contains(output.String(), "lost /dev/ttyACM0: device disconnected") {
		t.Fatalf("device loss was not handled and logged: %s", output.String())
	}
}

func Test_req_BRG_06_UsesConfiguredPortAndResendsSettingsAfterReconnect(t *testing.T) {
	system := newFakeSystem()
	system.plug("/dev/ttyACM0", newFakePort(), "303A")
	first := newFakePort()
	system.plug("/dev/ttyACM5", first, "0000")
	link := NewLink(system, logging.New(io.Discard, "info"), 90, 1, "/dev/ttyACM5")
	now := connect(link, time.Unix(0, 0))
	link.SetState(sessions.WORKING, now)
	if got := system.opened; len(got) != 1 || got[0] != "/dev/ttyACM5" {
		t.Fatalf("opened ports = %v, wanted only configured port", got)
	}

	first.readError = errors.New("device disconnected")
	link.Step(now.Add(time.Second))
	second := newFakePort()
	system.plug("/dev/ttyACM5", second, "0000")
	reconnectedAt := now.Add(2 * time.Second)
	link.Step(reconnectedAt)
	link.Step(reconnectedAt.Add(100 * time.Millisecond))

	if !link.Status().Connected {
		t.Fatal("configured port did not reconnect")
	}
	if got, want := second.commands[:4],
		[]string{"VERSION?", "SPEED:90", "VOLUME:1", "STATE:WORKING"}; !equal(got, want) {
		t.Fatalf("reconnect commands = %v, want %v", got, want)
	}
}

func Test_req_BRG_06_SkipsOtherUSBDevicesAndRejectsNonCodyEspressifPorts(t *testing.T) {
	system := newFakeSystem()
	system.plug("/dev/ttyUSB0", newFakePort(), "1A86")
	other := newFakePort()
	other.cody = false
	system.plug("/dev/ttyACM0", other, "303A")
	cody := newFakePort()
	system.plug("/dev/ttyACM1", cody, "303A")
	link := makeLink(system, "")
	start := time.Unix(0, 0)
	for now := start; now.Before(start.Add(5 * time.Second)); now = now.Add(100 * time.Millisecond) {
		link.Step(now)
	}

	if !link.Status().Connected || link.Status().Port == nil || *link.Status().Port != "/dev/ttyACM1" {
		t.Fatalf("connected to unexpected device: %+v", link.Status())
	}
	if got, want := system.opened, []string{"/dev/ttyACM0", "/dev/ttyACM1"}; !equal(got, want) {
		t.Fatalf("opened ports = %v, want %v", got, want)
	}
}

func Test_req_BRG_06_ReconnectsAfterDeviceRestart(t *testing.T) {
	system := newFakeSystem()
	port := newFakePort()
	system.plug("/dev/ttyACM0", port, "303A")
	link := NewLink(system, logging.New(io.Discard, "info"), 70, 0, "")
	now := connect(link, time.Unix(0, 0))
	port.commands = nil
	_, _ = io.WriteString(&port.replies, "READY 1.2.3\n")
	link.Step(now.Add(time.Second))

	if got, want := port.commands,
		[]string{"SPEED:70", "VOLUME:0", "STATE:IDLE"}; !equal(got, want) {
		t.Fatalf("restart commands = %v, want %v", got, want)
	}
}

func equal[T comparable](got, want []T) bool {
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

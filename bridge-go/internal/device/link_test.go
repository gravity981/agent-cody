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
	cody        bool
	version     string
	tuningReply string
	commands    []string
	replies     bytes.Buffer
	readError   error
	closed      bool
}

func newFakePort() *fakePort {
	return &fakePort{
		cody: true, version: "1.2.3 abc",
		tuningReply: "TUNING:224,10000,1,FAST,UNSAVED",
	}
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
		if command == "TUNING?" {
			_, _ = fmt.Fprintf(&p.replies, "%s\n", p.tuningReply)
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
	return NewLink(system, logging.New(io.Discard, "info"), 50, 100, serialPort)
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
	link := NewLink(system, logging.New(io.Discard, "info"), 50, 100, "")
	connect(link, time.Unix(0, 0))

	if got, want := port.commands[:4],
		[]string{"VERSION?", "VOLUME:50", "BRIGHTNESS:100", "STATE:IDLE"}; !equal(got, want) {
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
	link := NewLink(system, logger, 50, 100, "")
	now := connect(link, time.Unix(0, 0))
	link.Step(now.Add(5 * time.Second))

	for _, line := range []string{"-> VERSION?", "<- VERSION 1.2.3 abc", "-> VOLUME:50", "<- OK", "-> PING"} {
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
	link := NewLink(system, logging.New(io.Discard, "info"), 50, 100, "/dev/ttyACM5")
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
		[]string{"VERSION?", "VOLUME:50", "BRIGHTNESS:100", "STATE:WORKING"}; !equal(got, want) {
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
	link := NewLink(system, logging.New(io.Discard, "info"), 0, 100, "")
	now := connect(link, time.Unix(0, 0))
	port.commands = nil
	_, _ = io.WriteString(&port.replies, "READY 1.2.3\n")
	link.Step(now.Add(time.Second))

	if got, want := port.commands,
		[]string{"VOLUME:0", "BRIGHTNESS:100", "STATE:IDLE"}; !equal(got, want) {
		t.Fatalf("restart commands = %v, want %v", got, want)
	}
}

func Test_req_TST_05_req_TST_06_req_TST_07_DeviceLinkOwnsAndRestoresPanelSettings(t *testing.T) {
	system := newFakeSystem()
	port := newFakePort()
	system.plug("/dev/ttyACM0", port, "303A")
	link := makeLink(system, "")
	now := connect(link, time.Unix(0, 0))
	port.commands = nil

	if err := link.StartTest(now); err != nil {
		t.Fatal(err)
	}
	link.Step(now.Add(time.Millisecond))
	if err := link.SendTestCommand("TEST:STATE:WORKING", now.Add(2*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := link.SendTestCommand("TEST:VOLUME:20", now.Add(2500*time.Microsecond)); err != nil {
		t.Fatal(err)
	}
	link.Step(now.Add(3 * time.Millisecond))
	if err := link.StopTest(sessions.IDLE, now.Add(4*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	link.Step(now.Add(5 * time.Millisecond))

	if got, want := port.commands, []string{
		"TEST:START", "TEST:STATE:WORKING", "TEST:VOLUME:20", "TEST:STOP", "VOLUME:50",
		"BRIGHTNESS:100", "STATE:IDLE",
	}; !equal(got, want) {
		t.Fatalf("test commands = %v, want %v", got, want)
	}
	if got := link.Replies(); len(got) < 4 || got[len(got)-1] != "OK" {
		t.Fatalf("device replies = %v, want successful command replies", got)
	}
}

func Test_req_TST_09_QueriesAndParsesSavedTuning(t *testing.T) {
	system := newFakeSystem()
	port := newFakePort()
	port.tuningReply = "TUNING:91,24000,7,SLOW,2026-11-22T11:45:00Z"
	system.plug("/dev/ttyACM0", port, "303A")
	link := makeLink(system, "")
	now := connect(link, time.Unix(0, 0))

	if err := link.QueryTuning(now); err != nil {
		t.Fatal(err)
	}
	link.Step(now.Add(time.Millisecond))

	got := link.Tuning()
	if !got.Known || got.Speed != 91 || got.FrequencyHz != 24000 || got.KickStartMs != 7 ||
		got.Decay != "SLOW" || got.Timestamp != "2026-11-22T11:45:00Z" || got.Error != "" {
		t.Fatalf("motor tuning = %+v", got)
	}
	if !contains(port.commands, "TUNING?") {
		t.Fatalf("commands = %v, want TUNING? query", port.commands)
	}
}

func Test_req_TST_09_RejectsMalformedTuningReplies(t *testing.T) {
	for _, line := range []string{
		"TUNING:224,10000,1,FAST",
		"TUNING:256,10000,1,FAST,UNSAVED",
		"TUNING:224,999,1,FAST,UNSAVED",
		"TUNING:224,10000,101,FAST,UNSAVED",
		"TUNING:224,10000,1,MEDIUM,UNSAVED",
		"TUNING:224,10000,1,FAST,2026-11-22T12:45:00+01:00",
	} {
		if _, err := parseTuningReply(line); err == nil {
			t.Errorf("parseTuningReply(%q) succeeded, want error", line)
		}
	}
}

func Test_req_BRG_06_req_TST_07_ReconnectDuringTestReappliesPreferences(t *testing.T) {
	system := newFakeSystem()
	first := newFakePort()
	system.plug("/dev/ttyACM0", first, "303A")
	link := makeLink(system, "")
	now := connect(link, time.Unix(0, 0))
	if err := link.StartTest(now); err != nil {
		t.Fatal(err)
	}
	link.Step(now.Add(time.Millisecond))

	first.readError = errors.New("device disconnected")
	link.Step(now.Add(time.Second))
	second := newFakePort()
	system.plug("/dev/ttyACM0", second, "303A")
	reconnectedAt := now.Add(2 * time.Second)
	link.Step(reconnectedAt)
	link.Step(reconnectedAt.Add(100 * time.Millisecond))

	if got, want := second.commands,
		[]string{"VERSION?", "BRIGHTNESS:100", "TEST:START", "TEST:VOLUME:50"}; !equal(got, want) {
		t.Fatalf("reconnect commands during test = %v, want %v", got, want)
	}
}

func Test_req_TST_06_DeviceLinkReportsSerialErrors(t *testing.T) {
	system := newFakeSystem()
	port := newFakePort()
	system.plug("/dev/ttyACM0", port, "303A")
	link := makeLink(system, "")
	now := connect(link, time.Unix(0, 0))
	port.readError = errors.New("device disconnected")
	link.Step(now.Add(time.Second))

	replies := link.Replies()
	if got := replies[len(replies)-1]; got != "ERROR: device disconnected" {
		t.Fatalf("last device reply = %q, want serial error", got)
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
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

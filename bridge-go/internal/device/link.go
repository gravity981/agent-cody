package device

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

const (
	EspressifVID       = 0x303A
	HeartbeatInterval  = 5 * time.Second
	HandshakeTimeout   = 2 * time.Second
	RetryInterval      = time.Second
	maximumReplyBuffer = 4096
	maximumReplies     = 30
)

var ErrNotConnected = errors.New("device is not connected")

type Port interface {
	io.Reader
	io.Writer
	io.Closer
}

type PortInfo struct {
	Device string
	VID    string
}

type System interface {
	ListPorts() ([]PortInfo, error)
	Open(string) (Port, error)
}

type Link struct {
	system      System
	logger      *logging.Logger
	configured  string
	volume      int
	brightness  int
	state       sessions.State
	port        Port
	device      string
	firmware    string
	buffer      []byte
	handshakeAt time.Time
	nextTry     time.Time
	lastSent    time.Time
	rejected    map[string]struct{}
	testMode    bool
	replies     []string
	tuning      TuningStatus
	tuningQuery bool
}

func NewLink(system System, logger *logging.Logger, volume, brightness int, configured string) *Link {
	return &Link{
		system:     system,
		logger:     logger,
		configured: configured,
		volume:     volume,
		brightness: brightness,
		state:      sessions.IDLE,
		rejected:   make(map[string]struct{}),
	}
}

type TuningStatus struct {
	Known       bool   `json:"known"`
	Speed       int    `json:"speed"`
	FrequencyHz int    `json:"frequency_hz"`
	KickStartMs int    `json:"kick_start_ms"`
	Decay       string `json:"decay"`
	Timestamp   string `json:"timestamp"`
	Error       string `json:"error,omitempty"`
}

type Status struct {
	Connected bool    `json:"connected"`
	Port      *string `json:"port"`
	Firmware  *string `json:"firmware"`
}

func (l *Link) Status() Status {
	var port, firmware *string
	if l.device != "" {
		value := l.device
		port = &value
	}
	if l.firmware != "" {
		value := l.firmware
		firmware = &value
	}
	return Status{
		Connected: l.port != nil && l.firmware != "",
		Port:      port,
		Firmware:  firmware,
	}
}

func (l *Link) SetState(state sessions.State, now time.Time) {
	if state == l.state {
		return
	}
	l.state = state
	if l.testMode {
		return
	}
	if l.Status().Connected {
		if err := l.send("STATE:"+string(state), now); err != nil {
			l.lost(now, err)
		}
	}
}

func (l *Link) StartTest(now time.Time) error {
	if !l.Status().Connected {
		return ErrNotConnected
	}
	if l.testMode {
		return errors.New("hardware test mode is already active")
	}
	if err := l.send("TEST:START", now); err != nil {
		l.lost(now, err)
		return fmt.Errorf("start hardware test mode: %w", err)
	}
	l.testMode = true
	return nil
}

func (l *Link) SendTestCommand(command string, now time.Time) error {
	if !l.testMode {
		return errors.New("hardware test mode is not active")
	}
	if !l.Status().Connected {
		return ErrNotConnected
	}
	if err := l.send(command, now); err != nil {
		l.lost(now, err)
		return fmt.Errorf("send hardware test command: %w", err)
	}
	return nil
}

func (l *Link) StopTest(state sessions.State, now time.Time) error {
	wasActive := l.testMode
	l.testMode = false
	l.state = state
	if !wasActive {
		return nil
	}
	if !l.Status().Connected {
		return ErrNotConnected
	}
	if err := l.send("TEST:STOP", now); err != nil {
		l.lost(now, err)
		return fmt.Errorf("stop hardware test mode: %w", err)
	}
	if err := l.send(fmt.Sprintf("VOLUME:%d", l.volume), now); err != nil {
		l.lost(now, err)
		return fmt.Errorf("restore volume preference: %w", err)
	}
	if err := l.send(fmt.Sprintf("BRIGHTNESS:%d", l.brightness), now); err != nil {
		l.lost(now, err)
		return fmt.Errorf("restore brightness preference: %w", err)
	}
	if err := l.send("STATE:"+string(state), now); err != nil {
		l.lost(now, err)
		return fmt.Errorf("restore merged state: %w", err)
	}
	return nil
}

func (l *Link) Replies() []string {
	return append([]string{}, l.replies...)
}

func (l *Link) Tuning() TuningStatus {
	return l.tuning
}

func (l *Link) QueryTuning(now time.Time) error {
	if !l.Status().Connected {
		return ErrNotConnected
	}
	l.tuningQuery = true
	if err := l.send("TUNING?", now); err != nil {
		l.tuningQuery = false
		l.lost(now, err)
		return fmt.Errorf("query motor tuning: %w", err)
	}
	return nil
}

func (l *Link) Step(now time.Time) {
	if l.port == nil {
		if !now.Before(l.nextTry) {
			l.nextTry = now.Add(RetryInterval)
			l.open(now)
		}
		return
	}

	if err := l.read(now); err != nil {
		l.lost(now, err)
		return
	}
	if l.port == nil {
		return
	}
	if l.firmware == "" {
		if now.Sub(l.handshakeAt) >= HandshakeTimeout {
			l.logger.Infof("%s didn't answer VERSION?; not a Cody", l.device)
			if l.configured == "" {
				l.rejected[l.device] = struct{}{}
			}
			l.close(now)
		}
		return
	}
	if now.Sub(l.lastSent) >= HeartbeatInterval {
		if err := l.send("PING", now); err != nil {
			l.lost(now, err)
		}
	}
}

func (l *Link) Close(now time.Time) {
	l.testMode = false
	l.close(now)
}

func (l *Link) open(now time.Time) {
	candidates := l.candidates()
	for _, name := range candidates {
		port, err := l.system.Open(name)
		if err != nil {
			l.logger.Debugf("can't open %s: %v", name, err)
			continue
		}
		l.port = port
		l.device = name
		l.firmware = ""
		l.buffer = nil
		l.handshakeAt = now
		if err := l.send("VERSION?", now); err != nil {
			l.close(now)
			continue
		}
		return
	}
}

func (l *Link) candidates() []string {
	if l.configured != "" {
		return []string{l.configured}
	}
	ports, err := l.system.ListPorts()
	if err != nil {
		l.logger.Debugf("can't enumerate serial ports: %v", err)
		return nil
	}
	found := make(map[string]struct{}, len(ports))
	candidates := make([]string, 0, len(ports))
	for _, port := range ports {
		vid, err := strconv.ParseUint(port.VID, 16, 16)
		if err != nil || vid != EspressifVID {
			continue
		}
		found[port.Device] = struct{}{}
		if _, rejected := l.rejected[port.Device]; rejected {
			continue
		}
		candidates = append(candidates, port.Device)
	}
	for rejected := range l.rejected {
		if _, present := found[rejected]; !present {
			delete(l.rejected, rejected)
		}
	}
	sort.Strings(candidates)
	return candidates
}

func (l *Link) read(now time.Time) error {
	var chunk [1024]byte
	n, err := l.port.Read(chunk[:])
	if n > 0 {
		l.buffer = append(l.buffer, chunk[:n]...)
	}
	if len(l.buffer) > maximumReplyBuffer && !bytes.Contains(l.buffer, []byte{'\n'}) {
		return errors.New("device reply line is too long")
	}
	for {
		index := bytes.IndexByte(l.buffer, '\n')
		if index < 0 {
			return err
		}
		if index > maximumReplyBuffer {
			return errors.New("device reply line is too long")
		}
		line := strings.TrimSpace(string(l.buffer[:index]))
		l.buffer = l.buffer[index+1:]
		l.appendReply(line)
		l.logger.Debugf("<- %s", line)
		switch {
		case strings.HasPrefix(line, "VERSION ") && l.firmware == "":
			l.firmware = strings.TrimPrefix(line, "VERSION ")
			l.connected(now)
		case strings.HasPrefix(line, "READY") && l.firmware != "":
			l.logger.Infof("%s restarted", l.device)
			l.connected(now)
		case strings.HasPrefix(line, "TUNING:"):
			tuning, err := parseTuningReply(line)
			l.tuningQuery = false
			if err != nil {
				l.tuning.Error = err.Error()
				l.logger.Warningf("invalid motor tuning response: %v", err)
			} else {
				l.tuning = tuning
			}
		case strings.HasPrefix(line, "ERR"):
			l.logger.Warningf("device: %s", line)
			if l.tuningQuery {
				l.tuning.Error = line
				l.tuningQuery = false
			}
		}
	}
}

func (l *Link) connected(now time.Time) {
	l.logger.Infof("connected to %s (firmware %s)", l.device, l.firmware)
	commands := make([]string, 0, 4)
	if l.testMode {
		commands = append(commands,
			fmt.Sprintf("BRIGHTNESS:%d", l.brightness),
			"TEST:START",
			fmt.Sprintf("TEST:VOLUME:%d", l.volume),
		)
	} else {
		commands = append(commands,
			fmt.Sprintf("VOLUME:%d", l.volume),
			fmt.Sprintf("BRIGHTNESS:%d", l.brightness),
			"STATE:"+string(l.state),
		)
	}
	for _, command := range commands {
		if err := l.send(command, now); err != nil {
			l.lost(now, err)
			return
		}
	}
}

func (l *Link) send(command string, now time.Time) error {
	if l.port == nil {
		return errors.New("serial port is not open")
	}
	data := []byte(command + "\n")
	n, err := l.port.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	l.logger.Debugf("-> %s", command)
	l.lastSent = now
	return nil
}

func (l *Link) appendReply(reply string) {
	l.replies = append(l.replies, reply)
	if len(l.replies) > maximumReplies {
		l.replies = append([]string{}, l.replies[len(l.replies)-maximumReplies:]...)
	}
}

func (l *Link) lost(now time.Time, err error) {
	l.appendReply(fmt.Sprintf("ERROR: %s", err))
	l.logger.Infof("lost %s: %v", l.device, err)
	l.close(now)
}

func (l *Link) close(now time.Time) {
	if l.port != nil {
		if err := l.port.Close(); err != nil {
			l.logger.Warningf("close %s: %v", l.device, err)
		}
	}
	l.port = nil
	l.device = ""
	l.firmware = ""
	l.buffer = nil
	l.tuning = TuningStatus{}
	l.tuningQuery = false
	l.nextTry = now.Add(RetryInterval)
}

func parseTuningReply(line string) (TuningStatus, error) {
	values := strings.Split(strings.TrimPrefix(line, "TUNING:"), ",")
	if !strings.HasPrefix(line, "TUNING:") || len(values) != 5 {
		return TuningStatus{}, errors.New("malformed tuning response")
	}
	speed, err := strconv.Atoi(values[0])
	if err != nil || speed < 0 || speed > 255 {
		return TuningStatus{}, errors.New("invalid tuning speed")
	}
	frequency, err := strconv.Atoi(values[1])
	if err != nil || frequency < 1000 || frequency > 30000 {
		return TuningStatus{}, errors.New("invalid tuning frequency")
	}
	kick, err := strconv.Atoi(values[2])
	if err != nil || kick < 0 || kick > 100 {
		return TuningStatus{}, errors.New("invalid tuning kick")
	}
	if values[3] != "FAST" && values[3] != "SLOW" {
		return TuningStatus{}, errors.New("invalid tuning decay mode")
	}
	if values[4] != "UNSAVED" {
		timestamp, err := time.Parse(time.RFC3339Nano, values[4])
		if err != nil {
			return TuningStatus{}, errors.New("invalid tuning timestamp")
		}
		_, offset := timestamp.Zone()
		if offset != 0 {
			return TuningStatus{}, errors.New("tuning timestamp is not UTC")
		}
	}
	return TuningStatus{
		Known:       true,
		Speed:       speed,
		FrequencyHz: frequency,
		KickStartMs: kick,
		Decay:       values[3],
		Timestamp:   values[4],
	}, nil
}

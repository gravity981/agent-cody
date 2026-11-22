package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/device"
)

const (
	maximumFirmwareImageSize = 4 << 20
	maximumFlashOutputSize   = 128 << 10
	flashExecutionTimeout    = 5 * time.Minute
	tuningQueryTimeout       = 3 * time.Second
	deviceReconnectTimeout   = 30 * time.Second
)

var (
	ErrFlashBusy            = errors.New("firmware flashing is already in progress")
	ErrFlashUnavailable     = errors.New("local firmware flashing is unavailable")
	ErrInvalidFirmwareImage = errors.New("invalid firmware image")
)

type FirmwareFlashBridge interface {
	Status(time.Time) bridge.Status
	TestStatus() bridge.TestStatus
	QueryTuning(time.Time) error
	EspressifPorts() ([]device.PortInfo, error)
	BeginFirmwareFlash(string, string, *device.TuningStatus, time.Time) error
	EndFirmwareFlash(time.Time) error
}

type FirmwareFlashRunner interface {
	Available() error
	Run(context.Context, string, string, string, func(string)) error
}

type FirmwareFlashStatus struct {
	ID       string `json:"id"`
	Mode     string `json:"mode"`
	Port     string `json:"port"`
	Filename string `json:"filename"`
	Phase    string `json:"phase"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	Done     bool   `json:"done"`
	Success  bool   `json:"success"`
}

type FirmwareFlashService struct {
	mu     sync.Mutex
	ctx    context.Context
	bridge FirmwareFlashBridge
	runner FirmwareFlashRunner
	clock  func() time.Time
	job    *firmwareFlashJob
}

type firmwareFlashJob struct {
	status FirmwareFlashStatus
	image  string
}

func NewFirmwareFlashService(
	ctx context.Context,
	b FirmwareFlashBridge,
	runner FirmwareFlashRunner,
	clock func() time.Time,
) *FirmwareFlashService {
	if ctx == nil {
		ctx = context.Background()
	}
	if clock == nil {
		clock = time.Now
	}
	return &FirmwareFlashService{ctx: ctx, bridge: b, runner: runner, clock: clock}
}

func (s *FirmwareFlashService) Start(mode, port, filename string, image []byte) (string, error) {
	if mode != "first" && mode != "update" {
		return "", fmt.Errorf("%w: mode must be first or update", ErrInvalidFirmwareImage)
	}
	if strings.TrimSpace(port) == "" {
		return "", fmt.Errorf("%w: a serial port must be selected", ErrInvalidFirmwareImage)
	}
	if !strings.EqualFold(path.Ext(strings.ReplaceAll(filename, "\\", "/")), ".bin") {
		return "", fmt.Errorf("%w: image must have a .bin extension", ErrInvalidFirmwareImage)
	}
	if len(image) == 0 {
		return "", fmt.Errorf("%w: image must not be empty", ErrInvalidFirmwareImage)
	}
	if len(image) > maximumFirmwareImageSize {
		return "", fmt.Errorf("%w: image must not exceed %d MiB",
			ErrInvalidFirmwareImage, maximumFirmwareImageSize>>20,
		)
	}
	if s.runner == nil {
		return "", ErrFlashUnavailable
	}
	if err := s.runner.Available(); err != nil {
		return "", fmt.Errorf("%w: %w", ErrFlashUnavailable, err)
	}

	s.mu.Lock()
	if s.job != nil && !s.job.status.Done {
		s.mu.Unlock()
		return "", ErrFlashBusy
	}
	s.mu.Unlock()

	file, err := os.CreateTemp("", "cody-firmware-*.bin")
	if err != nil {
		return "", fmt.Errorf("create temporary firmware image: %w", err)
	}
	imagePath := file.Name()
	written, writeErr := file.Write(image)
	if writeErr == nil && written != len(image) {
		writeErr = io.ErrShortWrite
	}
	if writeErr != nil {
		_ = file.Close()
		_ = os.Remove(imagePath)
		return "", fmt.Errorf("write temporary firmware image: %w", writeErr)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(imagePath)
		return "", fmt.Errorf("close temporary firmware image: %w", err)
	}

	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		_ = os.Remove(imagePath)
		return "", fmt.Errorf("create flash job ID: %w", err)
	}
	job := &firmwareFlashJob{
		status: FirmwareFlashStatus{
			ID:       hex.EncodeToString(idBytes[:]),
			Mode:     mode,
			Port:     port,
			Filename: path.Base(strings.ReplaceAll(filename, "\\", "/")),
			Phase:    "Starting",
		},
		image: imagePath,
	}

	s.mu.Lock()
	if s.job != nil && !s.job.status.Done {
		s.mu.Unlock()
		_ = os.Remove(imagePath)
		return "", ErrFlashBusy
	}
	s.job = job
	s.mu.Unlock()

	go s.run(job)
	return job.status.ID, nil
}

func (s *FirmwareFlashService) Status(id string) (FirmwareFlashStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.job == nil || s.job.status.ID != id {
		return FirmwareFlashStatus{}, false
	}
	return s.job.status, true
}

func (s *FirmwareFlashService) run(job *firmwareFlashJob) {
	defer os.Remove(job.image)
	var tuning *device.TuningStatus
	if job.status.Mode == "update" {
		s.setPhase(job, "Reading motor tuning from device")
		if err := s.bridge.QueryTuning(s.clock()); err != nil {
			s.finish(job, err)
			return
		}
		snapshot, err := s.waitForTuning()
		if err != nil {
			s.finish(job, err)
			return
		}
		tuning = &snapshot
	}

	s.setPhase(job, "Stopping test mode and releasing the serial port")
	if err := s.bridge.BeginFirmwareFlash(job.status.Mode, job.status.Port, tuning, s.clock()); err != nil {
		s.finish(job, err)
		return
	}

	s.setPhase(job, "Flashing at 0x0")
	runContext, cancel := context.WithTimeout(s.ctx, flashExecutionTimeout)
	runErr := s.runner.Run(runContext, job.status.Mode, job.status.Port, job.image, func(text string) {
		s.appendOutput(job, text)
	})
	cancel()
	endErr := s.bridge.EndFirmwareFlash(s.clock())
	if endErr != nil {
		endErr = fmt.Errorf("resume bridge device connection: %w", endErr)
	}

	s.setPhase(job, "Waiting for Cody to reconnect")
	reconnectErr := s.waitForReconnect(job, tuning)
	if reconnectErr != nil {
		reconnectErr = fmt.Errorf("device reconnection: %w", reconnectErr)
	}
	s.finish(job, errors.Join(runErr, endErr, reconnectErr))
}

func (s *FirmwareFlashService) waitForTuning() (device.TuningStatus, error) {
	deadline := time.NewTimer(tuningQueryTimeout)
	defer deadline.Stop()
	poll := time.NewTicker(50 * time.Millisecond)
	defer poll.Stop()
	for {
		status := s.bridge.TestStatus().Tuning
		if status.Known {
			return status, nil
		}
		if status.Error != "" {
			return device.TuningStatus{}, fmt.Errorf("read motor tuning: %s", status.Error)
		}
		select {
		case <-s.ctx.Done():
			return device.TuningStatus{}, s.ctx.Err()
		case <-deadline.C:
			return device.TuningStatus{}, errors.New("timed out waiting for the motor tuning response")
		case <-poll.C:
		}
	}
}

func (s *FirmwareFlashService) waitForReconnect(
	job *firmwareFlashJob,
	tuning *device.TuningStatus,
) error {
	deadline := time.NewTimer(deviceReconnectTimeout)
	defer deadline.Stop()
	poll := time.NewTicker(100 * time.Millisecond)
	defer poll.Stop()
	for {
		status := s.bridge.Status(s.clock()).Device
		if status.Connected && status.Port != nil && *status.Port == job.status.Port {
			if tuning == nil {
				return nil
			}
			current := s.bridge.TestStatus().Tuning
			if current.Error != "" {
				return fmt.Errorf("restore motor tuning: %s", current.Error)
			}
			if sameTuning(current, *tuning) {
				return nil
			}
		}
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-deadline.C:
			return errors.New("Cody did not reconnect; reconnect the device and retry")
		case <-poll.C:
		}
	}
}

func sameTuning(left, right device.TuningStatus) bool {
	return left.Known && right.Known &&
		left.Speed == right.Speed &&
		left.FrequencyHz == right.FrequencyHz &&
		left.KickStartMs == right.KickStartMs &&
		left.Decay == right.Decay &&
		left.Timestamp == right.Timestamp
}

func (s *FirmwareFlashService) setPhase(job *firmwareFlashJob, phase string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.job == job {
		s.job.status.Phase = phase
	}
}

func (s *FirmwareFlashService) appendOutput(job *firmwareFlashJob, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.job != job {
		return
	}
	output := job.status.Output + text
	const notice = "[older output truncated]\n"
	if len(output) > maximumFlashOutputSize {
		job.status.Output = notice + output[len(output)-(maximumFlashOutputSize-len(notice)):]
	} else {
		job.status.Output = output
	}
}

func (s *FirmwareFlashService) finish(job *firmwareFlashJob, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.job != job {
		return
	}
	job.status.Done = true
	if err != nil {
		job.status.Error = err.Error()
		job.status.Phase = "Failed"
		return
	}
	job.status.Success = true
	job.status.Phase = "Complete"
}

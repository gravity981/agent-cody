//go:build windows

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/config"
	"github.com/gravity981/agent-cody/bridge-go/internal/hooksinstall"
	"github.com/gravity981/agent-cody/bridge-go/internal/httpapi"
	"github.com/gravity981/agent-cody/bridge-go/internal/relay"
	"github.com/gravity981/agent-cody/bridge-go/internal/wsl"
	"golang.org/x/sys/windows"
)

const wslRestartDelay = 5 * time.Second
const wslExitGracePeriod = 2 * time.Second

func startWSLRelays(
	ctx context.Context,
	cfg config.Config,
	logger relay.Logger,
) (<-chan struct{}, error) {
	if cfg.WSLDistros != nil && len(cfg.WSLDistros) == 0 {
		return nil, nil
	}
	wslExecutable, err := exec.LookPath("wsl.exe")
	if err != nil {
		if cfg.WSLDistros == nil && errors.Is(err, exec.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("locate wsl.exe: %w", err)
	}

	distros, err := configuredWSLDistros(ctx, wslExecutable, cfg)
	if err != nil {
		return nil, err
	}
	if len(distros) == 0 {
		logger.Infof("no default WSL distribution; relay is disabled")
		return nil, nil
	}

	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate Cody executable: %w", err)
	}
	relayPath, err := wsl.RelayBinaryPath(executable)
	if err != nil {
		return nil, fmt.Errorf("locate WSL relay binary: %w", err)
	}
	relayBinary := filepath.Join(filepath.Dir(executable), "cody-linux")
	if _, err := os.Stat(relayBinary); err != nil {
		return nil, fmt.Errorf("find WSL relay binary next to Cody: %w", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		var workers sync.WaitGroup
		for _, distro := range distros {
			workers.Add(1)
			go func(distro string) {
				defer workers.Done()
				superviseWSLRelay(ctx, wslExecutable, distro, relayPath, cfg.LogLevel, logger)
			}(distro)
		}
		workers.Wait()
	}()
	return done, nil
}

func configuredWSLDistros(
	ctx context.Context,
	wslExecutable string,
	cfg config.Config,
) ([]string, error) {
	if cfg.WSLDistros != nil {
		return wsl.RelayDistributions(cfg.WSLDistros, nil), nil
	}
	output, err := newWSLCommandContext(
		ctx,
		wslExecutable,
		"--list",
		"--verbose",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("list WSL distributions: %w", err)
	}
	return wsl.RelayDistributions(nil, output), nil
}

type wslHookInstaller struct {
	config     config.Config
	executable string
}

func newWSLHookInstaller(cfg config.Config, executable string) httpapi.WSLHookSetup {
	return &wslHookInstaller{config: cfg, executable: executable}
}

func (i *wslHookInstaller) Status(ctx context.Context) ([]httpapi.WSLHookStatus, error) {
	return i.run(ctx, "--status")
}

func (i *wslHookInstaller) Install(ctx context.Context) ([]httpapi.WSLHookStatus, error) {
	return i.run(ctx)
}

func (i *wslHookInstaller) run(
	ctx context.Context,
	args ...string,
) ([]httpapi.WSLHookStatus, error) {
	if i.config.WSLDistros != nil && len(i.config.WSLDistros) == 0 {
		return []httpapi.WSLHookStatus{}, nil
	}
	wslExecutable, err := exec.LookPath("wsl.exe")
	if err != nil {
		return nil, fmt.Errorf("locate wsl.exe: %w", err)
	}
	distros, err := configuredWSLDistros(ctx, wslExecutable, i.config)
	if err != nil {
		return nil, err
	}
	if len(distros) == 0 {
		return []httpapi.WSLHookStatus{}, nil
	}
	relayPath, err := wsl.RelayBinaryPath(i.executable)
	if err != nil {
		return nil, fmt.Errorf("locate WSL relay binary: %w", err)
	}
	relayBinary := filepath.Join(filepath.Dir(i.executable), "cody-linux")
	if _, err := os.Stat(relayBinary); err != nil {
		return nil, fmt.Errorf("find WSL relay binary next to Cody: %w", err)
	}

	statuses := make([]httpapi.WSLHookStatus, 0, len(distros))
	for _, distro := range distros {
		commandArgs := []string{"-d", distro, "--exec", relayPath, "install-hooks"}
		commandArgs = append(commandArgs, args...)
		output, err := newWSLCommandContext(ctx, wslExecutable, commandArgs...).Output()
		if err != nil {
			statuses = append(statuses, httpapi.WSLHookStatus{
				Distro: distro,
				Error:  wslCommandError(distro, err),
			})
			continue
		}
		var status hooksinstall.Status
		if err := json.Unmarshal(output, &status); err != nil {
			statuses = append(statuses, httpapi.WSLHookStatus{
				Distro: distro,
				Error:  fmt.Sprintf("decode hook status from WSL distro %q: %v", distro, err),
			})
			continue
		}
		statuses = append(statuses, httpapi.WSLHookStatus{
			Distro: distro,
			Status: status,
		})
	}
	return statuses, nil
}

func wslCommandError(distro string, err error) string {
	message := fmt.Sprintf("run hook setup in WSL distro %q: %v", distro, err)
	var commandError *exec.ExitError
	if errors.As(err, &commandError) && len(commandError.Stderr) > 0 {
		message += ": " + strings.TrimSpace(string(commandError.Stderr))
	}
	return message
}

func superviseWSLRelay(
	ctx context.Context,
	wslExecutable, distro, relayPath, logLevel string,
	logger relay.Logger,
) {
	distroLog := prefixedRelayLogger{logger: logger, distro: distro}
	for ctx.Err() == nil {
		err := runWSLRelay(ctx, wslExecutable, distro, relayPath, logLevel, distroLog)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			distroLog.Warningf("%v", err)
		}
		timer := time.NewTimer(wslRestartDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func runWSLRelay(
	ctx context.Context,
	wslExecutable, distro, relayPath, logLevel string,
	logger relay.Logger,
) error {
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	command := newWSLCommandContext(
		commandCtx,
		wslExecutable,
		"-d", distro,
		"-e", relayPath,
		"relay",
		"--log-level", logLevel,
	)
	stdin, err := command.StdinPipe()
	if err != nil {
		return fmt.Errorf("open WSL relay stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open WSL relay stdout: %w", err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return fmt.Errorf("open WSL relay stderr: %w", err)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start WSL distribution %q: %w", distro, err)
	}
	logger.Infof("started")

	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		copyRelayStderr(stderr, logger)
	}()
	relayResult := make(chan error, 1)
	go func() {
		reader, skipped, err := relay.ReadHandshake(stdout)
		if err != nil {
			relayResult <- err
			return
		}
		if skipped != 0 {
			logger.Warningf("discarded %d bytes before WSL relay handshake", skipped)
		}
		relayResult <- relay.RunServer(
			commandCtx,
			relayPipes{reader: reader, readerCloser: stdout, writer: stdin},
			relay.LoopbackAddress,
			logger,
		)
	}()

	select {
	case <-ctx.Done():
		cancel()
		<-relayResult
		<-stderrDone
		_ = command.Wait()
		return nil
	case err := <-relayResult:
		if ctx.Err() != nil {
			cancel()
			<-stderrDone
			_ = command.Wait()
			return nil
		}
		timer := time.NewTimer(wslExitGracePeriod)
		select {
		case <-stderrDone:
			timer.Stop()
		case <-timer.C:
			cancel()
			<-stderrDone
		case <-ctx.Done():
			timer.Stop()
			cancel()
			<-stderrDone
			_ = command.Wait()
			return nil
		}
		waitResult := make(chan error, 1)
		go func() {
			waitResult <- command.Wait()
		}()
		waitTimer := time.NewTimer(wslExitGracePeriod)
		var waitErr error
		select {
		case waitErr = <-waitResult:
			waitTimer.Stop()
		case <-waitTimer.C:
			cancel()
			waitErr = <-waitResult
		case <-ctx.Done():
			waitTimer.Stop()
			cancel()
			waitErr = <-waitResult
			return nil
		}
		if err != nil {
			if waitErr != nil {
				return errors.Join(
					fmt.Errorf("relay transport: %w", err),
					fmt.Errorf("WSL command: %w", waitErr),
				)
			}
			return fmt.Errorf("relay transport: %w", err)
		}
		if waitErr != nil {
			return fmt.Errorf("WSL command: %w", waitErr)
		}
		return errors.New("WSL relay process exited")
	}
}

func newWSLCommandContext(ctx context.Context, wslExecutable string, args ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, wslExecutable, args...)
	command.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
	return command
}

func copyRelayStderr(reader io.Reader, logger relay.Logger) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		logRelayLine(scanner.Text(), logger)
	}
	if err := scanner.Err(); err != nil {
		logger.Warningf("read relay stderr: %v", err)
	}
}

func logRelayLine(line string, logger relay.Logger) {
	parts := strings.SplitN(line, " ", 4)
	if len(parts) != 4 {
		logger.Warningf("%s", line)
		return
	}
	message := strings.TrimSpace(parts[3])
	switch parts[2] {
	case "INFO":
		logger.Infof("%s", message)
	case "DEBUG":
		logger.Debugf("%s", message)
	case "WARNING":
		logger.Warningf("%s", message)
	default:
		logger.Warningf("%s", line)
	}
}

type relayPipes struct {
	reader       io.Reader
	readerCloser io.Closer
	writer       io.WriteCloser
}

func (p relayPipes) Read(data []byte) (int, error)  { return p.reader.Read(data) }
func (p relayPipes) Write(data []byte) (int, error) { return p.writer.Write(data) }
func (p relayPipes) Close() error {
	return errors.Join(p.readerCloser.Close(), p.writer.Close())
}

type prefixedRelayLogger struct {
	logger relay.Logger
	distro string
}

func (l prefixedRelayLogger) Infof(format string, args ...any) {
	l.logger.Infof("relay %s: "+format, append([]any{l.distro}, args...)...)
}

func (l prefixedRelayLogger) Warningf(format string, args ...any) {
	l.logger.Warningf("relay %s: "+format, append([]any{l.distro}, args...)...)
}

func (l prefixedRelayLogger) Debugf(format string, args ...any) {
	l.logger.Debugf("relay %s: "+format, append([]any{l.distro}, args...)...)
}

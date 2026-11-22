package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

type EspflashRunner struct {
	Executable string
}

func (r EspflashRunner) Available() error {
	if runtime.GOOS != "windows" {
		return errors.New("local firmware flashing is available only in the Windows app")
	}
	if r.Executable == "" {
		return errors.New("espflash helper path is not configured")
	}
	info, err := os.Stat(r.Executable)
	if err != nil {
		return fmt.Errorf("locate espflash helper: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("espflash helper is not a regular file")
	}
	return nil
}

func (r EspflashRunner) Run(
	ctx context.Context,
	mode, port, image string,
	onOutput func(string),
) error {
	if runtime.GOOS != "windows" {
		return errors.New("local firmware flashing is available only in the Windows app")
	}
	if r.Executable == "" {
		return errors.New("espflash helper path is not configured")
	}
	if onOutput == nil {
		onOutput = func(string) {}
	}

	command := exec.CommandContext(ctx, r.Executable, espflashArgs(mode, port, image)...)
	configureFlashCommand(command)
	output := flashOutputWriter(onOutput)
	command.Stdout = output
	command.Stderr = output
	if err := command.Run(); err != nil {
		return fmt.Errorf("espflash failed: %w", err)
	}
	return nil
}

func espflashArgs(mode, port, image string) []string {
	args := []string{
		"write-bin", "--chip", "esp32c3", "--port", port,
		"--non-interactive", "--skip-update-check",
	}
	if mode == "first" {
		args = append(args, "--before", "no-reset")
	}
	return append(args, "0x0", image)
}

type flashOutputWriter func(string)

func (w flashOutputWriter) Write(data []byte) (int, error) {
	w(string(data))
	return len(data), nil
}

var _ io.Writer = flashOutputWriter(nil)

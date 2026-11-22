//go:build windows

package main

import (
	"context"
	"testing"

	"golang.org/x/sys/windows"
)

func Test_req_BRG_16_WslCommandsDoNotShowConsoleWindows(t *testing.T) {
	command := newWSLCommandContext(context.Background(), "wsl.exe", "--list")
	attributes := command.SysProcAttr
	if attributes == nil {
		t.Fatal("WSL command has no Windows process attributes")
	}
	if !attributes.HideWindow {
		t.Error("WSL command does not hide its window")
	}
	if attributes.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Error("WSL command does not set CREATE_NO_WINDOW")
	}
}

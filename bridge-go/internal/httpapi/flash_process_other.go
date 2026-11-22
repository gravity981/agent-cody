//go:build !windows

package httpapi

import "os/exec"

func configureFlashCommand(*exec.Cmd) {}

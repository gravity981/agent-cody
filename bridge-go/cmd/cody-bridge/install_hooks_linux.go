//go:build linux

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gravity981/agent-cody/bridge-go/internal/hooksinstall"
)

func runHookInstaller(args []string) int {
	if len(args) > 1 || (len(args) == 1 && args[0] != "--status") {
		fmt.Fprintln(os.Stderr, "cody-bridge install-hooks: expected only optional --status")
		return 2
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cody-bridge install-hooks: locate executable: %v\n", err)
		return 1
	}
	installer, err := hooksinstall.Default(filepath.Join(filepath.Dir(executable), "hooks"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "cody-bridge install-hooks: configure installer: %v\n", err)
		return 1
	}
	var status hooksinstall.Status
	if len(args) == 1 {
		status, err = installer.Status()
	} else {
		status, err = installer.Install()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "cody-bridge install-hooks: %v\n", err)
		return 1
	}
	if err := json.NewEncoder(os.Stdout).Encode(status); err != nil {
		fmt.Fprintf(os.Stderr, "cody-bridge install-hooks: encode status: %v\n", err)
		return 1
	}
	return 0
}

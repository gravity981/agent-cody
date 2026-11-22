//go:build !windows

package main

import (
	"github.com/gravity981/agent-cody/bridge-go/internal/config"
	"github.com/gravity981/agent-cody/bridge-go/internal/httpapi"
)

func newWSLHookInstaller(config.Config, string) httpapi.WSLHookSetup {
	return nil
}

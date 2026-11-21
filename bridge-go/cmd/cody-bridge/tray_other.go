//go:build !windows

package main

import (
	"context"

	"github.com/gravity981/agent-cody/bridge-go/internal/config"
	"github.com/gravity981/agent-cody/bridge-go/internal/relay"
)

func runWindowsTray(string) (int, bool) {
	return 0, false
}

func startWSLRelays(context.Context, config.Config, relay.Logger) (<-chan struct{}, error) {
	return nil, nil
}

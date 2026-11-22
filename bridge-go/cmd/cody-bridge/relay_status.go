package main

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/gravity981/agent-cody/bridge-go/internal/httpapi"
	"github.com/gravity981/agent-cody/bridge-go/internal/relay"
)

type relayStatusRegistry struct {
	mu     sync.RWMutex
	relays map[string]httpapi.RelayStatus
}

func newRelayStatusRegistry(distros []string) *relayStatusRegistry {
	registry := &relayStatusRegistry{
		relays: make(map[string]httpapi.RelayStatus, len(distros)),
	}
	for _, distro := range distros {
		registry.relays[distro] = httpapi.RelayStatus{Distro: distro}
	}
	return registry
}

func (r *relayStatusRegistry) Snapshot() []httpapi.RelayStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]httpapi.RelayStatus, 0, len(r.relays))
	for _, status := range r.relays {
		result = append(result, status)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Distro < result[j].Distro
	})
	return result
}

func (r *relayStatusRegistry) observe(message string) {
	prefix, event, found := strings.Cut(message, ": ")
	if !found || !strings.HasPrefix(prefix, "relay ") {
		return
	}
	distro := strings.TrimPrefix(prefix, "relay ")
	if distro == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	status := r.relays[distro]
	status.Distro = distro
	switch {
	case event == "started":
		status.LoopbackAddress = ""
		status.DockerAddress = ""
	case strings.HasPrefix(event, "relay listening on "):
		address := strings.TrimPrefix(event, "relay listening on ")
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return
		}
		switch host {
		case "127.0.0.1":
			status.LoopbackAddress = address
		case "172.17.0.1":
			status.DockerAddress = address
		}
	case strings.Contains(event, "relay transport:"),
		strings.Contains(event, "WSL command:"),
		strings.Contains(event, "start WSL distribution"),
		strings.Contains(event, "WSL relay process exited"):
		status.LoopbackAddress = ""
		status.DockerAddress = ""
	}
	r.relays[distro] = status
}

type statusTrackingRelayLogger struct {
	logger   relay.Logger
	registry *relayStatusRegistry
}

func (l statusTrackingRelayLogger) Infof(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.logger.Infof("%s", message)
	l.registry.observe(message)
}

func (l statusTrackingRelayLogger) Warningf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.logger.Warningf("%s", message)
	l.registry.observe(message)
}

func (l statusTrackingRelayLogger) Debugf(format string, args ...any) {
	l.logger.Debugf(format, args...)
}

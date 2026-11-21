//go:build linux

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/relay"
)

type relayStdio struct{}

func (relayStdio) Read(data []byte) (int, error)  { return os.Stdin.Read(data) }
func (relayStdio) Write(data []byte) (int, error) { return os.Stdout.Write(data) }
func (relayStdio) Close() error {
	return nil
}

func runRelay(args []string) int {
	flags := flag.NewFlagSet("cody-bridge relay", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	logLevel := flags.String("log-level", "info", "relay log level (info or debug)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "cody-bridge relay: unexpected arguments")
		return 2
	}
	if *logLevel != "info" && *logLevel != "debug" {
		fmt.Fprintln(os.Stderr, "cody-bridge relay: log level must be info or debug")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := logging.New(os.Stderr, *logLevel)
	if err := relay.RunClient(ctx, relayStdio{}, logger); err != nil {
		logger.Warningf("relay stopped: %v", err)
		return 1
	}
	return 0
}

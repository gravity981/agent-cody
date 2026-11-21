package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/buildinfo"
	"github.com/gravity981/agent-cody/bridge-go/internal/config"
	"github.com/gravity981/agent-cody/bridge-go/internal/device"
	"github.com/gravity981/agent-cody/bridge-go/internal/httpapi"
	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

const (
	httpAddress = "127.0.0.1:8765"
	loopPeriod  = 100 * time.Millisecond
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	flags := flag.NewFlagSet("cody-bridge", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	showVersion := flags.Bool("version", false, "print version and commit, then exit")
	configPath := flags.String("config", "", "config file (TOML)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		version, commit := buildinfo.Current()
		fmt.Printf("cody-bridge %s %s\n", version, commit)
		return 0
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "cody-bridge: unexpected arguments")
		return 2
	}

	path := *configPath
	if path == "" {
		var err error
		path, err = config.DefaultConfigPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cody-bridge: config path: %v\n", err)
			return 2
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cody-bridge: config: %v\n", err)
		return 2
	}

	logPath, err := config.DefaultLogPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cody-bridge: log path: %v\n", err)
		return 1
	}
	logger, err := logging.NewRotating(logPath, cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cody-bridge: logging: %v\n", err)
		return 1
	}
	version, commit := buildinfo.Current()
	logger.Infof("cody-bridge %s %s, config %s", version, commit, path)

	serialPort := ""
	if cfg.SerialPort != nil {
		serialPort = *cfg.SerialPort
	}
	link := device.NewLink(device.SerialSystem{}, logger, cfg.Speed, cfg.Volume, serialPort)
	activeSessions := sessions.New(time.Duration(cfg.SessionTimeoutMin * float64(time.Minute)))
	app := bridge.New(activeSessions, link, logger, version, commit)
	server := &http.Server{
		Addr:              httpAddress,
		Handler:           httpapi.NewHandler(app, time.Now),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ticker := time.NewTicker(loopPeriod)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			app.Tick(now)
		case err := <-serverErrors:
			logger.Warningf("HTTP server stopped: %v", err)
			link.Close(time.Now())
			return 1
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Warningf("HTTP server shutdown: %v", err)
			}
			cancel()
			link.Close(time.Now())
			return 0
		}
	}
}

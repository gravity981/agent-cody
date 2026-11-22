package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/buildinfo"
	"github.com/gravity981/agent-cody/bridge-go/internal/config"
	"github.com/gravity981/agent-cody/bridge-go/internal/device"
	"github.com/gravity981/agent-cody/bridge-go/internal/hooksinstall"
	"github.com/gravity981/agent-cody/bridge-go/internal/httpapi"
	"github.com/gravity981/agent-cody/bridge-go/internal/logging"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

const loopPeriod = 100 * time.Millisecond

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 && args[0] == "install-hooks" {
		return runHookInstaller(args[1:])
	}
	if len(args) > 0 && args[0] == "relay" {
		return runRelay(args[1:])
	}

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
	if code, handled := runWindowsTray(path); handled {
		return code
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := runBridge(ctx, path, nil, nil); err != nil {
		fmt.Fprintf(os.Stderr, "cody-bridge: %v\n", err)
		return 1
	}
	return 0
}

func ensureFileParentDirectory(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

func runBridge(
	ctx context.Context,
	configPath string,
	publishStatus func(bridge.Status),
	restart func(),
) (resultErr error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logPath, err := config.DefaultLogPath()
	if err != nil {
		return fmt.Errorf("log path: %w", err)
	}
	logger, err := logging.NewRotating(logPath, cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close log: %w", err))
		}
	}()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	relayStatuses := newRelayStatusRegistry(cfg.WSLDistros)
	relayLogger := statusTrackingRelayLogger{logger: logger, registry: relayStatuses}
	relayDone, err := startWSLRelays(runCtx, cfg, relayLogger)
	if err != nil {
		logger.Warningf("start WSL relay: %v", err)
		relayDone = nil
	}
	if relayDone != nil {
		defer func() {
			cancel()
			<-relayDone
		}()
	}

	version, commit := buildinfo.Current()
	logger.Infof("cody-bridge %s %s, config %s", version, commit, configPath)

	serialPort := ""
	if cfg.SerialPort != nil {
		serialPort = *cfg.SerialPort
	}
	link := device.NewLink(device.SerialSystem{}, logger, cfg.Volume, cfg.Brightness, serialPort)
	activeSessions := sessions.New(time.Duration(cfg.SessionTimeoutMin * float64(time.Minute)))
	app := bridge.New(activeSessions, link, logger, version, commit)
	var firmwareFlash *httpapi.FirmwareFlashService
	var hookInstaller *hooksinstall.Installer
	var wslHookSetup httpapi.WSLHookSetup
	if runtime.GOOS == "windows" {
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate Cody executable for firmware flashing: %w", err)
		}
		hookInstaller, err = hooksinstall.Default(filepath.Join(filepath.Dir(executable), "hooks"))
		if err != nil {
			return fmt.Errorf("configure hook installer: %w", err)
		}
		wslHookSetup = newWSLHookInstaller(cfg, executable)
		firmwareFlash = httpapi.NewFirmwareFlashService(
			runCtx, app,
			httpapi.EspflashRunner{Executable: filepath.Join(filepath.Dir(executable), "espflash.exe")},
			time.Now,
		)
	}
	server := &http.Server{
		Addr: httpapi.SiteAddress,
		Handler: httpapi.NewHandler(app, time.Now, httpapi.WebsiteOptions{
			ConfigPath:    configPath,
			RelayStatuses: relayStatuses.Snapshot,
			Restart:       restart,
			FirmwareFlash: firmwareFlash,
			HookInstaller: hookInstaller,
			WSLHookSetup:  wslHookSetup,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()
	if publishStatus != nil {
		publishStatus(app.Status(time.Now()))
	}

	ticker := time.NewTicker(loopPeriod)
	defer ticker.Stop()
	statusTicker := time.NewTicker(time.Second)
	defer statusTicker.Stop()
	for {
		select {
		case now := <-ticker.C:
			app.Tick(now)
		case now := <-statusTicker.C:
			if publishStatus != nil {
				publishStatus(app.Status(now))
			}
		case err := <-serverErrors:
			logger.Warningf("HTTP server stopped: %v", err)
			app.Close(time.Now())
			return fmt.Errorf("HTTP server stopped: %w", err)
		case <-runCtx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := server.Shutdown(shutdownCtx)
			cancel()
			app.Close(time.Now())
			if err != nil {
				logger.Warningf("HTTP server shutdown: %v", err)
				if closeErr := server.Close(); closeErr != nil {
					err = errors.Join(err, closeErr)
				}
				return fmt.Errorf("HTTP server shutdown: %w", err)
			}
			return nil
		}
	}
}

//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"fyne.io/systray"
	"github.com/gravity981/agent-cody/bridge-go/internal/autostart"
	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/buildinfo"
	"github.com/gravity981/agent-cody/bridge-go/internal/config"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
	trayview "github.com/gravity981/agent-cody/bridge-go/internal/tray"
	"golang.org/x/sys/windows"
)

func runWindowsTray(configPath string) (int, bool) {
	release, alreadyRunning, err := acquireInstance()
	if err != nil {
		showWindowsError(err)
		return 1, true
	}
	if alreadyRunning {
		if err := windows.CloseHandle(release); err != nil {
			showWindowsError(fmt.Errorf("release duplicate instance handle: %w", err))
			return 1, true
		}
		return 0, true
	}

	executable, err := os.Executable()
	if err != nil {
		showWindowsError(fmt.Errorf("locate Cody executable: %w", err))
		if closeErr := windows.CloseHandle(release); closeErr != nil {
			showWindowsError(fmt.Errorf("release instance handle: %w", closeErr))
		}
		return 1, true
	}
	autostartEnabled, autostartErr := autostart.Enabled(autostart.RegistryStore{}, executable)
	if autostartErr != nil {
		showWindowsError(autostartErr)
	}

	exit := make(chan struct{})
	loopDone := make(chan struct{})
	var exitOnce sync.Once
	version, commit := buildinfo.Current()
	systray.Run(func() {
		icon, iconErr := trayview.Icon(sessions.IDLE)
		if iconErr != nil {
			showWindowsError(iconErr)
		} else {
			systray.SetIcon(icon)
		}
		systray.SetTooltip("Cody " + version + " | starting")

		versionItem := systray.AddMenuItem("Version: "+version+" ("+commit+")", "Cody version and commit")
		versionItem.Disable()
		deviceItem := systray.AddMenuItem("Device: starting", "Serial device connection")
		deviceItem.Disable()
		stateItem := systray.AddMenuItem("State: IDLE", "Merged agent state")
		stateItem.Disable()
		sessionsItem := systray.AddMenuItem("Sessions: 0", "Active agent sessions")
		sessionsItem.Disable()
		systray.AddSeparator()
		autostartItem := systray.AddMenuItemCheckbox(
			"Start automatically",
			"Start Cody when you sign in",
			autostartEnabled,
		)
		if autostartErr != nil {
			autostartItem.Disable()
		}
		openConfigItem := systray.AddMenuItem("Open config", "Open the Cody bridge configuration")
		openLogItem := systray.AddMenuItem("Open log", "Open the Cody bridge log")
		restartItem := systray.AddMenuItem("Restart", "Restart Cody and apply configuration changes")
		quitItem := systray.AddMenuItem("Quit", "Stop Cody")

		go windowsTrayLoop(
			exit,
			loopDone,
			configPath,
			executable,
			version,
			deviceItem,
			stateItem,
			sessionsItem,
			autostartItem,
			openConfigItem,
			openLogItem,
			restartItem,
			quitItem,
			autostartEnabled,
		)
	}, func() {
		exitOnce.Do(func() { close(exit) })
	})
	<-loopDone

	if err := windows.CloseHandle(release); err != nil {
		showWindowsError(fmt.Errorf("release instance handle: %w", err))
		return 1, true
	}
	return 0, true
}

func acquireInstance() (windows.Handle, bool, error) {
	name, err := windows.UTF16PtrFromString(`Local\CodyBridge`)
	if err != nil {
		return 0, false, fmt.Errorf("encode instance name: %w", err)
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return handle, true, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("create single-instance mutex: %w", err)
	}
	return handle, false, nil
}

func showWindowsError(err error) {
	text, textErr := windows.UTF16PtrFromString(err.Error())
	title, titleErr := windows.UTF16PtrFromString("Cody")
	if textErr != nil || titleErr != nil {
		log.Printf("cody-bridge: %v (could not display error dialog)", err)
		return
	}
	if _, dialogErr := windows.MessageBox(
		0,
		text,
		title,
		windows.MB_OK|windows.MB_ICONERROR,
	); dialogErr != nil {
		log.Printf("cody-bridge: show error dialog: %v", dialogErr)
	}
}

func windowsTrayLoop(
	exit <-chan struct{},
	done chan<- struct{},
	configPath string,
	executable string,
	version string,
	deviceItem, stateItem, sessionsItem, autostartItem *systray.MenuItem,
	openConfigItem, openLogItem, restartItem, quitItem *systray.MenuItem,
	initialAutostart bool,
) {
	defer close(done)
	statusUpdates := make(chan bridge.Status, 1)
	var cancel context.CancelFunc
	var serviceResult chan error
	autostartOn := initialAutostart
	lastState := sessions.State("")
	lastDevice := ""
	lastSessionCount := -1
	lastTooltip := ""

	publishStatus := func(status bridge.Status) {
		select {
		case statusUpdates <- status:
		default:
			select {
			case <-statusUpdates:
			default:
			}
			select {
			case statusUpdates <- status:
			default:
			}
		}
	}
	startService := func() {
		ctx, stop := context.WithCancel(context.Background())
		cancel = stop
		serviceResult = make(chan error, 1)
		go func(result chan<- error) {
			result <- runBridge(ctx, configPath, publishStatus)
		}(serviceResult)
	}
	stopService := func() {
		if cancel == nil {
			return
		}
		cancel()
		if serviceResult != nil {
			if err := <-serviceResult; err != nil {
				showWindowsError(err)
			}
		}
		cancel = nil
		serviceResult = nil
	}
	startService()

	for {
		select {
		case <-exit:
			stopService()
			return
		case status := <-statusUpdates:
			if status.State != lastState {
				icon, err := trayview.Icon(status.State)
				if err != nil {
					showWindowsError(err)
				} else {
					systray.SetIcon(icon)
				}
				lastState = status.State
				stateItem.SetTitle("State: " + string(status.State))
			}
			deviceDescription := "not connected"
			if status.Device.Connected && status.Device.Port != nil {
				deviceDescription = *status.Device.Port
				if status.Device.Firmware != nil {
					deviceDescription += " (firmware " + *status.Device.Firmware + ")"
				}
			}
			if deviceDescription != lastDevice {
				deviceItem.SetTitle("Device: " + deviceDescription)
				lastDevice = deviceDescription
			}
			if len(status.Sessions) != lastSessionCount {
				sessionsItem.SetTitle(fmt.Sprintf("Sessions: %d", len(status.Sessions)))
				lastSessionCount = len(status.Sessions)
			}
			tooltip := trayview.Tooltip(status)
			if tooltip != lastTooltip {
				systray.SetTooltip(tooltip)
				lastTooltip = tooltip
			}
		case err := <-serviceResult:
			cancel = nil
			serviceResult = nil
			if err != nil {
				deviceItem.SetTitle("Device: unavailable")
				stateItem.SetTitle("State: unavailable")
				sessionsItem.SetTitle("Sessions: unavailable")
				lastState = ""
				lastDevice = "unavailable"
				lastSessionCount = -1
				systray.SetTooltip("Cody stopped: " + err.Error())
				lastTooltip = "Cody stopped: " + err.Error()
				showWindowsError(err)
			}
		case <-autostartItem.ClickedCh:
			desired := !autostartOn
			if err := autostart.SetEnabled(autostart.RegistryStore{}, executable, desired); err != nil {
				showWindowsError(err)
				continue
			}
			autostartOn = desired
			if autostartOn {
				autostartItem.Check()
			} else {
				autostartItem.Uncheck()
			}
		case <-openConfigItem.ClickedCh:
			openInNotepad(configPath)
		case <-openLogItem.ClickedCh:
			logPath, err := config.DefaultLogPath()
			if err != nil {
				showWindowsError(fmt.Errorf("get log path: %w", err))
				continue
			}
			openInNotepad(logPath)
		case <-restartItem.ClickedCh:
			stopService()
			startService()
			lastState = ""
		case <-quitItem.ClickedCh:
			systray.Quit()
		}
	}
}

func openInNotepad(path string) {
	if err := ensureFileParentDirectory(path); err != nil {
		showWindowsError(fmt.Errorf("prepare directory for %s: %w", path, err))
		return
	}
	command := exec.Command("notepad.exe", path)
	if err := command.Start(); err != nil {
		showWindowsError(fmt.Errorf("open %s: %w", path, err))
		return
	}
	go func() {
		if err := command.Wait(); err != nil {
			showWindowsError(fmt.Errorf("notepad %s: %w", path, err))
		}
	}()
}

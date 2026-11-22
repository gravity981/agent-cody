package httpapi

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/config"
	"github.com/gravity981/agent-cody/bridge-go/internal/device"
)

const websiteBodyLimit = 64 * 1024

//go:embed web/*
var websiteAssets embed.FS

type WebsiteOptions struct {
	ConfigPath    string
	RelayStatuses func() []RelayStatus
	Restart       func()
}

type RelayStatus struct {
	Distro          string `json:"distro"`
	LoopbackAddress string `json:"loopback_address,omitempty"`
	DockerAddress   string `json:"docker_address,omitempty"`
}

type websiteStatus struct {
	bridge.Status
	Relays []RelayStatus `json:"relays"`
}

type websiteConfig struct {
	UseDefaultWSLDistro bool     `json:"use_default_wsl_distro"`
	WSLDistros          []string `json:"wsl_distros"`
	Speed               int      `json:"speed"`
	Volume              int      `json:"volume"`
	SessionTimeoutMin   float64  `json:"session_timeout_min"`
	SerialPort          string   `json:"serial_port"`
	LogLevel            string   `json:"log_level"`
}

func registerWebsiteRoutes(
	mux *http.ServeMux,
	b Bridge,
	clock func() time.Time,
	options WebsiteOptions,
) {
	mux.Handle("/", newWebsiteHandler())
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		relays := []RelayStatus{}
		if options.RelayStatuses != nil {
			relays = options.RelayStatuses()
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, websiteStatus{
			Status: b.Status(clock()),
			Relays: relays,
		})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			current, err := config.Load(options.ConfigPath)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			writeJSON(w, http.StatusOK, configForWebsite(current))
		case http.MethodPut:
			if !sameOrigin(r) {
				writeAPIError(w, http.StatusForbidden, "request origin is not allowed")
				return
			}
			var requested websiteConfig
			if err := decodeWebsiteConfig(w, r, &requested); err != nil {
				writeAPIError(w, http.StatusBadRequest, err.Error())
				return
			}
			updated := requested.toConfig()
			if err := updated.Validate(); err != nil {
				writeAPIError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err := config.Save(options.ConfigPath, updated); err != nil {
				writeAPIError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"restart_required": true})
		default:
			methodNotAllowed(w, http.MethodGet+", "+http.MethodPut)
		}
	})
	mux.HandleFunc("/api/restart", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		if !sameOrigin(r) {
			writeAPIError(w, http.StatusForbidden, "request origin is not allowed")
			return
		}
		if options.Restart == nil {
			writeAPIError(w, http.StatusServiceUnavailable, "restart is unavailable")
			return
		}
		writeText(w, http.StatusAccepted, "Cody is restarting\n")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		options.Restart()
	})
	mux.HandleFunc("/api/test/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, b.TestStatus())
	})
	mux.HandleFunc("/api/test/start", func(w http.ResponseWriter, r *http.Request) {
		if !testMutationAllowed(w, r) {
			return
		}
		if err := b.StartTest(clock()); err != nil {
			writeAPIError(w, testErrorStatus(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"active": true})
	})
	mux.HandleFunc("/api/test/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if !testMutationAllowed(w, r) {
			return
		}
		if err := b.TestHeartbeat(clock()); err != nil {
			writeAPIError(w, testErrorStatus(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"active": true})
	})
	mux.HandleFunc("/api/test/stop", func(w http.ResponseWriter, r *http.Request) {
		if !testMutationAllowed(w, r) {
			return
		}
		if err := b.StopTest(clock()); err != nil {
			writeAPIError(w, testErrorStatus(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"active": false})
	})
	mux.HandleFunc("/api/test/command", func(w http.ResponseWriter, r *http.Request) {
		if !testMutationAllowed(w, r) {
			return
		}
		var requested testCommandRequest
		if err := decodeTestCommand(w, r, &requested); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		command, err := requested.command()
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := b.SendTestCommand(command, clock()); err != nil {
			writeAPIError(w, testErrorStatus(err), err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]bool{"sent": true})
	})
}

type testCommandRequest struct {
	Action string `json:"action"`
	Value  string `json:"value"`
	Red    *int   `json:"red,omitempty"`
	Green  *int   `json:"green,omitempty"`
	Blue   *int   `json:"blue,omitempty"`
}

func (c testCommandRequest) command() (string, error) {
	switch c.Action {
	case "state":
		if oneOf(c.Value, "IDLE", "WORKING", "THINKING", "APPROVAL", "COMPLETED", "ERROR") {
			return "TEST:STATE:" + c.Value, nil
		}
		return "", errors.New("state must be one of IDLE, WORKING, THINKING, APPROVAL, COMPLETED, ERROR")
	case "motor":
		if oneOf(c.Value, "START", "STOP") {
			return "TEST:MOTOR:" + c.Value, nil
		}
		return "", errors.New("motor action must be START or STOP")
	case "speed", "frequency", "kick", "volume":
		value, err := strconv.Atoi(c.Value)
		if err != nil {
			return "", fmt.Errorf("%s must be an integer", c.Action)
		}
		minimum, maximum := 0, 255
		command := "TEST:SPEED"
		switch c.Action {
		case "frequency":
			minimum, maximum, command = 1000, 30000, "TEST:FREQ"
		case "kick":
			maximum, command = 100, "TEST:KICK"
		case "volume":
			maximum, command = 3, "TEST:VOLUME"
		}
		if value < minimum || value > maximum {
			return "", fmt.Errorf("%s must be between %d and %d", c.Action, minimum, maximum)
		}
		return fmt.Sprintf("%s:%d", command, value), nil
	case "decay":
		if oneOf(c.Value, "FAST", "SLOW") {
			return "TEST:DECAY:" + c.Value, nil
		}
		return "", errors.New("decay must be FAST or SLOW")
	case "beep":
		if oneOf(c.Value, "APPROVAL", "SUCCESS", "WARNING") {
			return "TEST:BEEP:" + c.Value, nil
		}
		return "", errors.New("sound must be APPROVAL, SUCCESS or WARNING")
	case "led":
		if c.Red == nil || c.Green == nil || c.Blue == nil {
			return "", errors.New("red, green and blue values are required")
		}
		if *c.Red < 0 || *c.Red > 255 {
			return "", errors.New("red must be between 0 and 255")
		}
		if *c.Green < 0 || *c.Green > 255 {
			return "", errors.New("green must be between 0 and 255")
		}
		if *c.Blue < 0 || *c.Blue > 255 {
			return "", errors.New("blue must be between 0 and 255")
		}
		return fmt.Sprintf("TEST:LED:%d,%d,%d", *c.Red, *c.Green, *c.Blue), nil
	default:
		return "", errors.New("unknown test command")
	}
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func testMutationAllowed(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		methodNotAllowed(w, http.MethodPost)
		return false
	}
	if !sameOrigin(r) {
		writeAPIError(w, http.StatusForbidden, "request origin is not allowed")
		return false
	}
	return true
}

func testErrorStatus(err error) int {
	switch {
	case errors.Is(err, bridge.ErrTestModeActive), errors.Is(err, bridge.ErrTestModeInactive):
		return http.StatusConflict
	case errors.Is(err, device.ErrNotConnected):
		return http.StatusServiceUnavailable
	default:
		return http.StatusServiceUnavailable
	}
}

func newWebsiteHandler() http.Handler {
	assets, err := fs.Sub(websiteAssets, "web")
	if err != nil {
		panic(fmt.Errorf("open embedded website assets: %w", err))
	}
	files := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w, http.MethodGet+", "+http.MethodHead)
			return
		}
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})
}

func configForWebsite(current config.Config) websiteConfig {
	distros := append([]string{}, current.WSLDistros...)
	serialPort := ""
	if current.SerialPort != nil {
		serialPort = *current.SerialPort
	}
	return websiteConfig{
		UseDefaultWSLDistro: current.WSLDistros == nil,
		WSLDistros:          distros,
		Speed:               current.Speed,
		Volume:              current.Volume,
		SessionTimeoutMin:   current.SessionTimeoutMin,
		SerialPort:          serialPort,
		LogLevel:            current.LogLevel,
	}
}

func (c websiteConfig) toConfig() config.Config {
	var distros []string
	if !c.UseDefaultWSLDistro {
		distros = append([]string{}, c.WSLDistros...)
	}
	var serialPort *string
	if value := strings.TrimSpace(c.SerialPort); value != "" {
		serialPort = &value
	}
	return config.Config{
		SerialPort:        serialPort,
		Speed:             c.Speed,
		Volume:            c.Volume,
		SessionTimeoutMin: c.SessionTimeoutMin,
		LogLevel:          c.LogLevel,
		WSLDistros:        distros,
	}
}

func decodeWebsiteConfig(w http.ResponseWriter, r *http.Request, destination *websiteConfig) error {
	return decodeWebsiteJSON(w, r, destination, "configuration")
}

func decodeTestCommand(w http.ResponseWriter, r *http.Request, destination *testCommandRequest) error {
	return decodeWebsiteJSON(w, r, destination, "test command")
}

func decodeWebsiteJSON(w http.ResponseWriter, r *http.Request, destination any, name string) error {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return errors.New("content type must be application/json")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, websiteBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("invalid %s: %w", name, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s must contain one JSON object", name)
	}
	return nil
}

func sameOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == SiteOrigin
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

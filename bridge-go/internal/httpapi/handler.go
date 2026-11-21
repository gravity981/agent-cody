package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/mapping"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

const MaxBodyBytes = 32_000_000

const SiteAddress = "127.0.0.1:8765"
const SiteOrigin = "http://" + SiteAddress

type Bridge interface {
	Event(mapping.Event, time.Time)
	SetManual(sessions.State, time.Time)
	ClearManual(time.Time)
	ClearSessions(time.Time)
	Status(time.Time) bridge.Status
}

func NewHandler(b Bridge, clock func() time.Time, website WebsiteOptions) http.Handler {
	if clock == nil {
		clock = time.Now
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			notFound(w)
			return
		}
		status := b.Status(clock())
		if r.URL.Query().Get("format") == "text" {
			writeText(w, http.StatusOK, bridge.StatusText(status))
			return
		}
		writeJSON(w, http.StatusOK, status)
	})
	mux.HandleFunc("/hooks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/hooks/claude-code" {
			notFound(w)
			return
		}
		body, ok := readObject(w, r)
		if !ok {
			writeText(w, http.StatusBadRequest, "expected a JSON object\n")
			return
		}
		event, err := mapping.ClaudeCode(body)
		if err != nil {
			var payloadError mapping.PayloadError
			if errors.As(err, &payloadError) {
				writeText(w, http.StatusBadRequest, payloadError.Error()+"\n")
				return
			}
			writeText(w, http.StatusBadRequest, err.Error()+"\n")
			return
		}
		b.Event(event, clock())
		writeText(w, http.StatusOK, "ok\n")
	})
	mux.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			body, ok := readObject(w, r)
			if !ok {
				writeText(w, http.StatusBadRequest, "state must be one of IDLE, WORKING, THINKING, APPROVAL, COMPLETED, ERROR\n")
				return
			}
			value, _ := body["state"].(string)
			state, valid := sessions.ParseState(value)
			if !valid {
				writeText(w, http.StatusBadRequest, "state must be one of IDLE, WORKING, THINKING, APPROVAL, COMPLETED, ERROR\n")
				return
			}
			b.SetManual(state, clock())
			writeText(w, http.StatusOK, "ok\n")
		case http.MethodDelete:
			b.ClearManual(clock())
			writeText(w, http.StatusOK, "ok\n")
		default:
			notFound(w)
		}
	})
	mux.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			notFound(w)
			return
		}
		b.ClearSessions(clock())
		writeText(w, http.StatusOK, "ok\n")
	})
	registerWebsiteRoutes(mux, b, clock, website)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isWebsiteRequest(r.URL.Path) && r.Host != SiteAddress {
			writeAPIError(w, http.StatusForbidden, "request host is not allowed")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead &&
			r.Header.Get("Origin") != "" && !sameOrigin(r) {
			writeAPIError(w, http.StatusForbidden, "request origin is not allowed")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func isWebsiteRequest(path string) bool {
	return path == "/" || path == "/index.html" || path == "/app.js" ||
		path == "/style.css" || strings.HasPrefix(path, "/api/")
}

func readObject(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	if err != nil || len(data) == 0 || len(data) > MaxBodyBytes {
		return nil, false
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return nil, false
	}
	return object, true
}

func notFound(w http.ResponseWriter) {
	writeText(w, http.StatusNotFound, "not found\n")
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		writeText(w, http.StatusInternalServerError, "could not encode status\n")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}

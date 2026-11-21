package tray

import (
	"encoding/binary"
	"testing"

	"github.com/gravity981/agent-cody/bridge-go/internal/bridge"
	"github.com/gravity981/agent-cody/bridge-go/internal/device"
	"github.com/gravity981/agent-cody/bridge-go/internal/sessions"
)

func Test_req_BRG_17_TrayIconUsesTheMergedStateColor(t *testing.T) {
	tests := []struct {
		state sessions.State
		red   byte
		green byte
		blue  byte
	}{
		{sessions.IDLE, 0, 0, 255},
		{sessions.WORKING, 0, 0, 255},
		{sessions.THINKING, 255, 160, 0},
		{sessions.APPROVAL, 255, 60, 0},
		{sessions.COMPLETED, 0, 255, 0},
		{sessions.ERROR, 255, 0, 0},
	}
	for _, test := range tests {
		t.Run(string(test.state), func(t *testing.T) {
			icon, err := Icon(test.state)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateIcon(icon); err != nil {
				t.Fatal(err)
			}
			dib := icon[22:]
			center := ((iconSize-1-iconSize/2)*iconSize + iconSize/2) * 4
			pixel := dib[40+center : 40+center+4]
			if pixel[2] != test.red || pixel[1] != test.green || pixel[0] != test.blue || pixel[3] != 255 {
				t.Fatalf("center pixel BGRA = %v, want [%d %d %d 255]", pixel, test.blue, test.green, test.red)
			}
		})
	}
}

func Test_req_BRG_17_UnknownStateHasNoTrayIcon(t *testing.T) {
	if _, err := Icon("UNKNOWN"); err == nil {
		t.Fatal("Icon accepted an unknown state")
	}
}

func Test_req_BRG_16_TooltipShowsVersionDeviceStateAndSessionCount(t *testing.T) {
	port := "COM4"
	status := bridge.Status{
		Bridge:   bridge.BuildInfo{Version: "1.2.3", Commit: "abc123"},
		Device:   device.Status{Connected: true, Port: &port},
		State:    sessions.APPROVAL,
		Sessions: []bridge.SessionStatus{{Session: "one"}, {Session: "two"}},
	}
	if got, want := Tooltip(status), "Cody 1.2.3 | APPROVAL | connected: COM4 | 2 sessions"; got != want {
		t.Fatalf("Tooltip() = %q, want %q", got, want)
	}
	status.Device = device.Status{}
	if got := Tooltip(status); got != "Cody 1.2.3 | APPROVAL | not connected | 2 sessions" {
		t.Fatalf("Tooltip() without device = %q", got)
	}
}

func Test_req_BRG_17_IconDirectoryEntryPointsToCompleteDIB(t *testing.T) {
	icon, err := Icon(sessions.ERROR)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := binary.LittleEndian.Uint32(icon[18:22]), uint32(22); got != want {
		t.Fatalf("image offset = %d, want %d", got, want)
	}
	if got, want := binary.LittleEndian.Uint32(icon[14:18]), uint32(len(icon)-22); got != want {
		t.Fatalf("image size = %d, want %d", got, want)
	}
}

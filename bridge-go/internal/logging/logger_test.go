package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type closeTrackingWriter struct {
	closed bool
}

func (w *closeTrackingWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (w *closeTrackingWriter) Close() error {
	w.closed = true
	return nil
}

func Test_req_BRG_13_FormatsLogsAndHonorsDebugLevel(t *testing.T) {
	var output bytes.Buffer
	logger := New(&output, "info")
	logger.now = func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) }
	logger.Infof("state %s", "WORKING")
	logger.Warningf("device disconnected")
	logger.Debugf("not visible")
	if got, want := output.String(),
		"2026-11-21 12:00:00 INFO  state WORKING\n"+
			"2026-11-21 12:00:00 WARNING device disconnected\n"; got != want {
		t.Fatalf("log output:\n%s\nwant:\n%s", got, want)
	}

	output.Reset()
	logger = New(&output, "debug")
	logger.now = func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) }
	logger.Debugf("-> PING")
	if !strings.HasSuffix(output.String(), "DEBUG -> PING\n") {
		t.Fatalf("debug log missing: %s", output.String())
	}
}

func TestLoggerCloseClosesItsWriter(t *testing.T) {
	writer := &closeTrackingWriter{}
	if err := New(writer, "info").Close(); err != nil {
		t.Fatal(err)
	}
	if !writer.closed {
		t.Fatal("logger did not close its writer")
	}
}

func Test_req_BRG_14_RotatesToAtMostTwoTenMegabyteFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cody", "bridge.log")
	logger, err := NewRotating(path, "info")
	if err != nil {
		t.Fatal(err)
	}
	logger.Infof("%s", strings.Repeat("a", 9*1024*1024))
	logger.Infof("%s", strings.Repeat("b", 2*1024*1024))

	backups, err := filepath.Glob(filepath.Join(filepath.Dir(path), "bridge-*.log"))
	if err != nil {
		t.Fatal(err)
	}
	files := append(backups, path)
	if len(files) != MaxLogFiles {
		t.Fatalf("rotated file count = %d, want %d", len(files), MaxLogFiles)
	}
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() > MaxLogSizeMB*1024*1024 {
			t.Errorf("%s is too large: %d bytes", file, info.Size())
		}
	}
}

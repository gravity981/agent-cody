package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func Test_req_BRG_16_OpenConfigCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cody", "bridge.toml")
	if err := ensureFileParentDirectory(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("parent path %q is not a directory", filepath.Dir(path))
	}
}

func Test_req_BRG_10_VersionFlagReportsBuildInfo(t *testing.T) {
	t.Setenv("CODY_VERSION", "2.0.0")
	t.Setenv("CODY_COMMIT", "feed123")
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = oldStdout }()
	code := run([]string{"--version"})
	_ = writer.Close()
	os.Stdout = oldStdout
	output, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 || string(output) != "cody-bridge 2.0.0 feed123\n" {
		t.Fatalf("run --version = %d, %q", code, output)
	}
}

func Test_req_BRG_21_RelayStatusTracksActualListenerPorts(t *testing.T) {
	status := newRelayStatusRegistry([]string{"Ubuntu", "Debian"})
	status.observe("relay Ubuntu: relay listening on 127.0.0.1:8765")
	status.observe("relay Ubuntu: relay listening on 172.17.0.1:8767")
	got := status.Snapshot()
	if len(got) != 2 || got[0].Distro != "Debian" || got[1].Distro != "Ubuntu" ||
		got[1].LoopbackAddress != "127.0.0.1:8765" ||
		got[1].DockerAddress != "172.17.0.1:8767" {
		t.Fatalf("relay statuses = %+v", got)
	}

	status.observe("relay Ubuntu: started")
	got = status.Snapshot()
	if got[1].LoopbackAddress != "" || got[1].DockerAddress != "" {
		t.Fatalf("relay restart retained stale ports: %+v", got[1])
	}
	status.observe("relay Ubuntu: relay listening on 172.17.0.1:8768")
	status.observe("relay Ubuntu: relay transport: accept relay stream: EOF")
	got = status.Snapshot()
	if got[1].DockerAddress != "" {
		t.Fatalf("stopped relay retained Docker address: %+v", got[1])
	}
}

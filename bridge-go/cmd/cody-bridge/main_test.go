package main

import (
	"io"
	"os"
	"testing"
)

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

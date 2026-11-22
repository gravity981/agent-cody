package wsl

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func Test_req_AGT_12_RelayPathUsesInstalledWindowsDirectory(t *testing.T) {
	got, err := RelayBinaryPath(`C:\Users\Ada Lovelace\AppData\Local\Programs\Cody\cody.exe`)
	if err != nil {
		t.Fatal(err)
	}
	const want = "/mnt/c/Users/Ada Lovelace/AppData/Local/Programs/Cody/cody-linux"
	if got != want {
		t.Fatalf("RelayBinaryPath() = %q, want %q", got, want)
	}
}

func Test_req_AGT_12_DefaultDistributionDecodesWslOutput(t *testing.T) {
	for _, test := range []struct {
		name   string
		output []byte
	}{
		{
			name:   "UTF-8",
			output: []byte("  NAME              STATE           VERSION\r\n* Ubuntu-24.04     Running         2\r\n"),
		},
		{
			name:   "UTF-16LE",
			output: utf16LE("  NAME              STATE           VERSION\r\n* Debian           Stopped         2\r\n"),
		},
		{
			name:   "distribution name with spaces",
			output: []byte("  NAME              STATE           VERSION\r\n* Ubuntu Custom    Running         2\r\n"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got, want := DefaultDistribution(test.output), map[string]string{
				"UTF-8":                         "Ubuntu-24.04",
				"UTF-16LE":                      "Debian",
				"distribution name with spaces": "Ubuntu Custom",
			}[test.name]; got != want {
				t.Fatalf("DefaultDistribution() = %q, want %q", got, want)
			}
		})
	}
}

func Test_req_AGT_12_NoDefaultDistributionReturnsEmpty(t *testing.T) {
	if got := DefaultDistribution([]byte("NAME STATE VERSION\nUbuntu Running 2\n")); got != "" {
		t.Fatalf("DefaultDistribution() = %q, want empty", got)
	}
}

func utf16LE(value string) []byte {
	units := utf16.Encode([]rune(value))
	output := make([]byte, len(units)*2)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(output[i*2:], unit)
	}
	return append([]byte{0xff, 0xfe}, output...)
}

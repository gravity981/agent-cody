package httpapi

import (
	"reflect"
	"testing"
)

func Test_req_TST_10_EspflashArgsSelectFirstBootloaderModeAndAddressZero(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want []string
	}{
		{
			name: "first flash uses manual ROM bootloader entry",
			mode: "first",
			want: []string{
				"write-bin", "--chip", "esp32c3", "--port", "COM4",
				"--non-interactive", "--skip-update-check",
				"--before", "no-reset", "0x0", "firmware.bin",
			},
		},
		{
			name: "update lets espflash enter bootloader automatically",
			mode: "update",
			want: []string{
				"write-bin", "--chip", "esp32c3", "--port", "COM4",
				"--non-interactive", "--skip-update-check", "0x0", "firmware.bin",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := espflashArgs(test.mode, "COM4", "firmware.bin")
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("espflash args = %v, want %v", got, test.want)
			}
		})
	}
}

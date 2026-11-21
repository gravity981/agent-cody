package wsl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
)

func RelayBinaryPath(executable string) (string, error) {
	path := strings.ReplaceAll(executable, `\`, "/")
	if len(path) < 3 || path[1] != ':' || path[2] != '/' {
		return "", fmt.Errorf("expected an absolute drive path, got %q", executable)
	}
	drive := path[0]
	if drive < 'A' || drive > 'Z' {
		if drive < 'a' || drive > 'z' {
			return "", fmt.Errorf("invalid drive letter in %q", executable)
		}
	}
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash < 2 {
		return "", fmt.Errorf("expected an absolute drive path, got %q", executable)
	}
	directory := strings.Trim(path[2:lastSlash], "/")
	linuxPath := "/mnt/" + strings.ToLower(string(drive))
	if directory != "" {
		linuxPath += "/" + directory
	}
	return linuxPath + "/cody-linux", nil
}

func DefaultDistribution(output []byte) string {
	text := decodeOutput(output)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "*") {
			continue
		}
		nameAndColumns := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		columnBreak := strings.Index(nameAndColumns, "  ")
		if columnBreak >= 0 {
			return strings.TrimSpace(nameAndColumns[:columnBreak])
		}
		fields := strings.Fields(nameAndColumns)
		if len(fields) > 0 {
			return fields[0]
		}
	}
	return ""
}

func decodeOutput(output []byte) string {
	switch {
	case bytes.HasPrefix(output, []byte{0xff, 0xfe}):
		return decodeUTF16(output[2:], binary.LittleEndian)
	case bytes.HasPrefix(output, []byte{0xfe, 0xff}):
		return decodeUTF16(output[2:], binary.BigEndian)
	case hasUTF16NULs(output):
		return decodeUTF16(output, binary.LittleEndian)
	default:
		return string(output)
	}
}

func decodeUTF16(output []byte, order binary.ByteOrder) string {
	if len(output)%2 != 0 {
		output = output[:len(output)-1]
	}
	units := make([]uint16, len(output)/2)
	for i := range units {
		units[i] = order.Uint16(output[i*2:])
	}
	return string(utf16.Decode(units))
}

func hasUTF16NULs(output []byte) bool {
	if len(output) < 4 || len(output)%2 != 0 {
		return false
	}
	oddNULs := 0
	for i := 1; i < len(output); i += 2 {
		if output[i] == 0 {
			oddNULs++
		}
	}
	return oddNULs > len(output)/8
}

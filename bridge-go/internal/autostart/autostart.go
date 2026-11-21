package autostart

import (
	"errors"
	"fmt"
	"strings"
)

var ErrNotFound = errors.New("autostart value not found")

type Store interface {
	Read() (string, error)
	Write(string) error
	Delete() error
}

func Enabled(store Store, executable string) (bool, error) {
	value, err := store.Read()
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read autostart value: %w", err)
	}
	return equalWindowsPath(unquote(value), executable), nil
}

func SetEnabled(store Store, executable string, enabled bool) error {
	if enabled {
		if err := store.Write(`"` + executable + `"`); err != nil {
			return fmt.Errorf("enable autostart: %w", err)
		}
		return nil
	}
	if err := store.Delete(); err != nil {
		return fmt.Errorf("disable autostart: %w", err)
	}
	return nil
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}
	return value
}

func equalWindowsPath(left, right string) bool {
	return strings.EqualFold(strings.ReplaceAll(left, "/", `\`), strings.ReplaceAll(right, "/", `\`))
}

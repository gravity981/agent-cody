//go:build windows

package autostart

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	valueName  = "Cody"
)

type RegistryStore struct{}

func (RegistryStore) Read() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	defer key.Close()
	value, _, err := key.GetStringValue(valueName)
	if errors.Is(err, registry.ErrNotExist) {
		return "", ErrNotFound
	}
	return value, err
}

func (RegistryStore) Write(value string) error {
	key, _, err := registry.CreateKey(
		registry.CURRENT_USER,
		runKeyPath,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(valueName, value)
}

func (RegistryStore) Delete() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.DeleteValue(valueName)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}

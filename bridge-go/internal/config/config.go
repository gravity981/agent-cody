package config

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	SerialPort        *string `toml:"serial_port"`
	Speed             int     `toml:"speed"`
	Volume            int     `toml:"volume"`
	SessionTimeoutMin float64 `toml:"session_timeout_min"`
	LogLevel          string  `toml:"log_level"`
}

func Defaults() Config {
	return Config{
		Speed:             224,
		Volume:            2,
		SessionTimeoutMin: 10,
		LogLevel:          "info",
	}
}

func DefaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config directory: %w", err)
	}
	return filepath.Join(dir, "cody", "bridge.toml"), nil
}

func DefaultLogPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("get user cache directory: %w", err)
	}
	return filepath.Join(dir, "cody", "bridge.log"), nil
}

func Load(path string) (Config, error) {
	config := Defaults()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return config, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("config %s: %w", path, err)
	}
	return config, nil
}

func (c Config) Validate() error {
	if c.SerialPort != nil && *c.SerialPort == "" {
		return errors.New("serial_port must be a device path")
	}
	if c.Speed < 0 || c.Speed > 255 {
		return errors.New("speed must be an integer from 0 to 255")
	}
	if c.Volume < 0 || c.Volume > 3 {
		return errors.New("volume must be an integer from 0 to 3")
	}
	maximumMinutes := float64(math.MaxInt64) / float64(time.Minute)
	if math.IsNaN(c.SessionTimeoutMin) || math.IsInf(c.SessionTimeoutMin, 0) ||
		c.SessionTimeoutMin <= 0 || c.SessionTimeoutMin > maximumMinutes {
		return errors.New("session_timeout_min must be a positive number")
	}
	if c.LogLevel != "info" && c.LogLevel != "debug" {
		return errors.New(`log_level must be "info" or "debug"`)
	}
	return nil
}

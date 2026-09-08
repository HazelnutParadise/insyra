package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// defaultFetchTWIntervalMS is the minimum spacing `fetch tw` puts between
// TWSE/TPEx requests. 300ms is the backfill-safe value the datafetch docs
// recommend; the CLI is the surface most likely to run a ten-year loop.
const defaultFetchTWIntervalMS = 300

type GlobalConfig struct {
	DefaultEnv string `json:"defaultEnv"`
	LogLevel   string `json:"logLevel"`
	NoColor    bool   `json:"noColor"`
	AccelMode  string `json:"accelMode"`
	// FetchTWIntervalMS is the `fetch tw` request interval in milliseconds.
	// Zero disables throttling.
	FetchTWIntervalMS int `json:"fetchTWIntervalMS"`
}

func defaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		DefaultEnv: "default",
		LogLevel:   "info",
		NoColor:    false,
		AccelMode:  "auto",

		FetchTWIntervalMS: defaultFetchTWIntervalMS,
	}
}

func (m *Manager) GlobalConfigPath() (string, error) {
	base, err := m.BasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "config.json"), nil
}

func (m *Manager) LoadGlobalConfig() (GlobalConfig, error) {
	if err := m.EnsureBaseStructure(); err != nil {
		return GlobalConfig{}, err
	}
	path, err := m.GlobalConfigPath()
	if err != nil {
		return GlobalConfig{}, err
	}
	_, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		cfg := defaultGlobalConfig()
		if saveErr := m.SaveGlobalConfig(cfg); saveErr != nil {
			return GlobalConfig{}, saveErr
		}
		return cfg, nil
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return GlobalConfig{}, err
	}
	if len(bytes) == 0 {
		cfg := defaultGlobalConfig()
		if saveErr := m.SaveGlobalConfig(cfg); saveErr != nil {
			return GlobalConfig{}, saveErr
		}
		return cfg, nil
	}
	cfg := defaultGlobalConfig()
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return GlobalConfig{}, err
	}
	return cfg, nil
}

func (m *Manager) SaveGlobalConfig(cfg GlobalConfig) error {
	if err := m.EnsureBaseStructure(); err != nil {
		return err
	}
	path, err := m.GlobalConfigPath()
	if err != nil {
		return err
	}
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func (m *Manager) UpdateGlobalConfig(key, value string) (GlobalConfig, error) {
	cfg, err := m.LoadGlobalConfig()
	if err != nil {
		return GlobalConfig{}, err
	}
	switch key {
	case "default-env", "defaultEnv":
		name := strings.TrimSpace(value)
		if name == "" {
			return GlobalConfig{}, fmt.Errorf("invalid value for default-env: the environment name cannot be empty")
		}
		cfg.DefaultEnv = name
	case "log-level", "logLevel":
		level := strings.ToLower(strings.TrimSpace(value))
		if !slices.Contains(validLogLevels, level) {
			return GlobalConfig{}, fmt.Errorf("invalid value %q for log-level (use %s)", value, strings.Join(validLogLevels, ", "))
		}
		cfg.LogLevel = level
	case "no-color", "noColor":
		parsed, parseErr := parseConfigBool(value)
		if parseErr != nil {
			return GlobalConfig{}, fmt.Errorf("invalid value %q for no-color: %v", value, parseErr)
		}
		cfg.NoColor = parsed
	case "accel-mode", "accelMode", "accel.mode":
		mode := strings.ToLower(strings.TrimSpace(value))
		if !slices.Contains(validAccelModes, mode) {
			return GlobalConfig{}, fmt.Errorf("invalid value %q for accel-mode (use %s)", value, strings.Join(validAccelModes, ", "))
		}
		cfg.AccelMode = mode
	case "fetch.tw.interval_ms", "fetch-tw-interval-ms", "fetchTWIntervalMS":
		milliseconds, convErr := strconv.Atoi(strings.TrimSpace(value))
		if convErr != nil || milliseconds < 0 {
			return GlobalConfig{}, fmt.Errorf("invalid value %q for fetch.tw.interval_ms: expected a non-negative integer number of milliseconds", value)
		}
		cfg.FetchTWIntervalMS = milliseconds
	default:
		return GlobalConfig{}, fmt.Errorf("unknown config key %q (supported: %s)", key, strings.Join(ConfigKeys(), ", "))
	}
	if err := m.SaveGlobalConfig(cfg); err != nil {
		return GlobalConfig{}, err
	}
	return cfg, nil
}

// Package-level wrappers around the default Manager.

func GlobalConfigPath() (string, error) { return defaultManager.GlobalConfigPath() }

func LoadGlobalConfig() (GlobalConfig, error) { return defaultManager.LoadGlobalConfig() }

func SaveGlobalConfig(cfg GlobalConfig) error { return defaultManager.SaveGlobalConfig(cfg) }

func UpdateGlobalConfig(key, value string) (GlobalConfig, error) {
	return defaultManager.UpdateGlobalConfig(key, value)
}

// The accepted values for the enumerated settings. A key or a value the CLI
// does not understand is refused rather than written to disk: a typo that
// persists is worse than one that is rejected.
var (
	validLogLevels  = []string{"debug", "info", "warning", "error", "fatal"}
	validAccelModes = []string{"auto", "cpu", "gpu", "strict-gpu"}
)

// ConfigKeys lists the keys `insyra config <key> <value>` accepts.
func ConfigKeys() []string {
	return []string{"default-env", "log-level", "no-color", "accel-mode", "fetch.tw.interval_ms"}
}

// parseConfigBool accepts the spellings the rest of the CLI accepts for a
// boolean option, and refuses anything else.
func parseConfigBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	default:
		return false, fmt.Errorf("expected true or false")
	}
}

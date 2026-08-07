package env

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type GlobalConfig struct {
	DefaultEnv string `json:"defaultEnv"`
	LogLevel   string `json:"logLevel"`
	NoColor    bool   `json:"noColor"`
	AccelMode  string `json:"accelMode"`
}

func defaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		DefaultEnv: "default",
		LogLevel:   "info",
		NoColor:    false,
		AccelMode:  "auto",
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
		cfg.DefaultEnv = value
	case "log-level", "logLevel":
		cfg.LogLevel = value
	case "no-color", "noColor":
		cfg.NoColor = value == "true" || value == "1" || value == "yes"
	case "accel-mode", "accelMode", "accel.mode":
		cfg.AccelMode = value
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

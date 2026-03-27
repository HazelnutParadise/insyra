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

func GlobalConfigPath() (string, error) {
	base, err := BasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "config.json"), nil
}

func LoadGlobalConfig() (GlobalConfig, error) {
	if err := EnsureBaseStructure(); err != nil {
		return GlobalConfig{}, err
	}
	path, err := GlobalConfigPath()
	if err != nil {
		return GlobalConfig{}, err
	}
	_, statErr := os.Stat(path)
	if os.IsNotExist(statErr) {
		cfg := defaultGlobalConfig()
		if saveErr := SaveGlobalConfig(cfg); saveErr != nil {
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
		if saveErr := SaveGlobalConfig(cfg); saveErr != nil {
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

func SaveGlobalConfig(cfg GlobalConfig) error {
	if err := EnsureBaseStructure(); err != nil {
		return err
	}
	path, err := GlobalConfigPath()
	if err != nil {
		return err
	}
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func UpdateGlobalConfig(key, value string) (GlobalConfig, error) {
	cfg, err := LoadGlobalConfig()
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
	if err := SaveGlobalConfig(cfg); err != nil {
		return GlobalConfig{}, err
	}
	return cfg, nil
}

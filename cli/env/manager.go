package env

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	baseDirName = ".insyra"
	envsDirName = "envs"
)

type EnvironmentInfo struct {
	Name          string
	Path          string
	LastAccess    time.Time
	VariableCount int
}

type ExportPayload struct {
	SchemaVersion int             `json:"schemaVersion"`
	ExportedAt    string          `json:"exportedAt"`
	Environment   string          `json:"environment"`
	State         *State          `json:"state"`
	History       []string        `json:"history"`
	Config        json.RawMessage `json:"config"`
}

func BasePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, baseDirName), nil
}

func EnvsPath() (string, error) {
	base, err := BasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, envsDirName), nil
}

func ResolveEnvPath(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("environment name is required")
	}
	envsPath, err := EnvsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(envsPath, name), nil
}

func EnsureDefaultEnvironment() error {
	if err := EnsureBaseStructure(); err != nil {
		return err
	}
	if !Exists("default") {
		if err := Create("default"); err != nil {
			return err
		}
	}
	return nil
}

func EnsureBaseStructure() error {
	base, err := BasePath()
	if err != nil {
		return err
	}
	envsPath, err := EnvsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(envsPath, 0o755); err != nil {
		return err
	}
	return nil
}

func Exists(name string) bool {
	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return false
	}
	stat, err := os.Stat(envPath)
	return err == nil && stat.IsDir()
}

func Create(name string) error {
	if err := EnsureBaseStructure(); err != nil {
		return err
	}
	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(envPath); err == nil {
		return fmt.Errorf("environment already exists: %s", name)
	}
	if err := os.MkdirAll(envPath, 0o755); err != nil {
		return err
	}
	if err := writeDefaultFiles(envPath); err != nil {
		return err
	}
	return nil
}

func Open(name string) (string, error) {
	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(envPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("environment does not exist: %s", name)
		}
		return "", err
	}
	return envPath, nil
}

func Delete(name string) error {
	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(envPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment does not exist: %s", name)
		}
		return err
	}
	return os.RemoveAll(envPath)
}

func Clear(name string, keepHistory bool) error {
	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(envPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment does not exist: %s", name)
		}
		return err
	}
	if err := SaveState(name, map[string]any{}); err != nil {
		return err
	}
	if keepHistory {
		return nil
	}
	return os.WriteFile(filepath.Join(envPath, "history.txt"), []byte(""), 0o644)
}

func Rename(oldName, newName string) error {
	oldPath, err := ResolveEnvPath(oldName)
	if err != nil {
		return err
	}
	newPath, err := ResolveEnvPath(newName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment does not exist: %s", oldName)
		}
		return err
	}
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target environment already exists: %s", newName)
	}
	return os.Rename(oldPath, newPath)
}

func List() ([]EnvironmentInfo, error) {
	envsPath, err := EnvsPath()
	if err != nil {
		return nil, err
	}
	if err := EnsureBaseStructure(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(envsPath)
	if err != nil {
		return nil, err
	}
	infos := make([]EnvironmentInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		info, err := Info(name)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	return infos, nil
}

func Info(name string) (EnvironmentInfo, error) {
	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return EnvironmentInfo{}, err
	}
	state, err := LoadState(name)
	if err != nil {
		state = &State{Variables: map[string]SerializedVariable{}}
	}
	info := EnvironmentInfo{
		Name:          name,
		Path:          envPath,
		VariableCount: len(state.Variables),
	}
	if state.LastAccess != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, state.LastAccess); parseErr == nil {
			info.LastAccess = parsed
		}
	}
	return info, nil
}

func Export(name, outputPath string) error {
	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(envPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment does not exist: %s", name)
		}
		return err
	}

	state, err := LoadState(name)
	if err != nil {
		state = &State{Variables: map[string]SerializedVariable{}, LastAccess: ""}
	}

	history, err := ReadHistory(name)
	if err != nil {
		history = []string{}
	}

	configBytes, err := os.ReadFile(filepath.Join(envPath, "config.json"))
	if err != nil {
		configBytes = []byte("{}")
	}

	payload := ExportPayload{
		SchemaVersion: 1,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
		Environment:   name,
		State:         state,
		History:       history,
		Config:        json.RawMessage(configBytes),
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	parent := filepath.Dir(outputPath)
	if parent != "" && parent != "." {
		if err := os.MkdirAll(parent, 0o755); err != nil {
			return err
		}
	}

	return os.WriteFile(outputPath, bytes, 0o644)
}

func Import(inputPath, targetName string, force bool) (string, error) {
	bytes, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}

	var payload ExportPayload
	if err := json.Unmarshal(bytes, &payload); err != nil {
		return "", fmt.Errorf("invalid export payload: %w", err)
	}

	name := strings.TrimSpace(targetName)
	if name == "" {
		name = strings.TrimSpace(payload.Environment)
	}
	if name == "" {
		return "", errors.New("environment name is required")
	}

	if err := EnsureBaseStructure(); err != nil {
		return "", err
	}

	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return "", err
	}

	if !Exists(name) {
		if err := Create(name); err != nil {
			return "", err
		}
	} else if !force {
		empty, err := isEnvironmentEmpty(name)
		if err != nil {
			return "", err
		}
		if !empty {
			return "", fmt.Errorf("target environment is not empty: %s (use --force to overwrite)", name)
		}
	}

	state := payload.State
	if state == nil {
		state = &State{Variables: map[string]SerializedVariable{}, LastAccess: ""}
	}
	if state.Variables == nil {
		state.Variables = map[string]SerializedVariable{}
	}
	if state.LastAccess == "" {
		state.LastAccess = time.Now().UTC().Format(time.RFC3339)
	}

	stateBytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(envPath, "state.json"), stateBytes, 0o644); err != nil {
		return "", err
	}

	historyBytes := []byte("")
	if len(payload.History) > 0 {
		historyText := strings.Join(payload.History, "\n") + "\n"
		historyBytes = []byte(historyText)
	}
	if err := os.WriteFile(filepath.Join(envPath, "history.txt"), historyBytes, 0o644); err != nil {
		return "", err
	}

	configBytes := []byte("{}\n")
	if len(payload.Config) > 0 {
		if json.Valid(payload.Config) {
			configBytes = payload.Config
		} else {
			return "", errors.New("invalid config payload in export file")
		}
	}
	if err := os.WriteFile(filepath.Join(envPath, "config.json"), configBytes, 0o644); err != nil {
		return "", err
	}

	return name, nil
}

func isEnvironmentEmpty(name string) (bool, error) {
	state, err := LoadState(name)
	if err == nil && state != nil && len(state.Variables) > 0 {
		return false, nil
	}

	history, err := ReadHistory(name)
	if err == nil && len(history) > 0 {
		return false, nil
	}

	envPath, err := ResolveEnvPath(name)
	if err != nil {
		return false, err
	}
	configBytes, err := os.ReadFile(filepath.Join(envPath, "config.json"))
	if err != nil {
		return true, nil
	}

	trimmed := strings.TrimSpace(string(configBytes))
	if trimmed == "" || trimmed == "{}" {
		return true, nil
	}

	var configPayload any
	if err := json.Unmarshal(configBytes, &configPayload); err != nil {
		return false, fmt.Errorf("invalid existing config.json in environment %s: %w", name, err)
	}

	switch typed := configPayload.(type) {
	case map[string]any:
		return len(typed) == 0, nil
	case nil:
		return true, nil
	default:
		return false, nil
	}
}

func writeDefaultFiles(envPath string) error {
	if err := os.WriteFile(filepath.Join(envPath, "history.txt"), []byte(""), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(envPath, "state.json"), []byte("{\n  \"variables\": {},\n  \"lastAccess\": \"\"\n}\n"), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(envPath, "config.json"), []byte("{}\n"), 0o644); err != nil {
		return err
	}
	return nil
}

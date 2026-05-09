package env

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	baseDirName        = ".insyra"
	defaultEnvsDirName = "envs"
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

// Manager owns a single environment-storage root and exposes all
// env-related operations (CRUD, state, history, config) as methods.
//
// Embedders that want per-workspace isolation should construct one Manager
// per workspace via NewManager. The CLI binary and any code calling the
// package-level wrapper functions share a single process-wide Default()
// Manager rooted at <UserHomeDir>/.insyra/envs (overridable via SetBasePath
// and SetEnvsDirName).
type Manager struct {
	mu          sync.RWMutex
	basePath    string // empty = use UserHomeDir/.insyra
	envsDirName string // empty = "envs"
}

// NewManager constructs a Manager rooted at basePath. envsDirName overrides
// the per-environment subfolder name; pass "" for the default "envs".
//
// Examples:
//
//	env.NewManager("", "")                       // ~/.insyra/envs/<name>/
//	env.NewManager("/ws/.idensyra", "")          // /ws/.idensyra/envs/<name>/
//	env.NewManager("/ws/.idensyra", "insights")  // /ws/.idensyra/insights/<name>/
//
// Both fields are fixed at construction; create a new Manager to point
// somewhere else.
func NewManager(basePath, envsDirName string) *Manager {
	return &Manager{
		basePath:    strings.TrimSpace(basePath),
		envsDirName: strings.TrimSpace(envsDirName),
	}
}

var defaultManager = &Manager{}

// Default returns the shared process-wide Manager used by the package-level
// wrapper functions (env.BasePath, env.Create, …) and by SetBasePath.
func Default() *Manager { return defaultManager }

// SetBasePath overrides the default Manager's root. Pass "" to restore the
// default <UserHomeDir>/.insyra. This only affects the package-level
// wrappers and any caller that explicitly uses Default(); per-session
// Managers created via NewManager are unaffected.
func SetBasePath(path string) { defaultManager.SetBasePath(path) }

// SetBasePath updates this Manager's root. Pass "" to fall back to the
// default <UserHomeDir>/.insyra. Safe to call before opening or creating
// any environment; not safe to change while an environment is in use.
func (m *Manager) SetBasePath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.basePath = strings.TrimSpace(path)
}

// SetEnvsDirName updates this Manager's per-environment subfolder name.
// Pass "" to fall back to the default "envs". Same usage caveats as
// SetBasePath — not safe to change while an environment is in use.
func (m *Manager) SetEnvsDirName(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.envsDirName = strings.TrimSpace(name)
}

func (m *Manager) BasePath() (string, error) {
	m.mu.RLock()
	bp := m.basePath
	m.mu.RUnlock()
	if bp != "" {
		return bp, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, baseDirName), nil
}

func (m *Manager) EnvsDirName() string {
	m.mu.RLock()
	dir := m.envsDirName
	m.mu.RUnlock()
	if dir == "" {
		return defaultEnvsDirName
	}
	return dir
}

func (m *Manager) EnvsPath() (string, error) {
	base, err := m.BasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, m.EnvsDirName()), nil
}

func (m *Manager) ResolveEnvPath(name string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", errors.New("environment name is required")
	}
	envsPath, err := m.EnvsPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(envsPath, name), nil
}

func (m *Manager) EnsureDefaultEnvironment() error {
	if err := m.EnsureBaseStructure(); err != nil {
		return err
	}
	if !m.Exists("default") {
		if err := m.Create("default"); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) EnsureBaseStructure() error {
	base, err := m.BasePath()
	if err != nil {
		return err
	}
	envsPath, err := m.EnvsPath()
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

func (m *Manager) Exists(name string) bool {
	envPath, err := m.ResolveEnvPath(name)
	if err != nil {
		return false
	}
	stat, err := os.Stat(envPath)
	return err == nil && stat.IsDir()
}

func (m *Manager) Create(name string) error {
	if err := m.EnsureBaseStructure(); err != nil {
		return err
	}
	envPath, err := m.ResolveEnvPath(name)
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

func (m *Manager) Open(name string) (string, error) {
	envPath, err := m.ResolveEnvPath(name)
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

func (m *Manager) Delete(name string) error {
	envPath, err := m.ResolveEnvPath(name)
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

func (m *Manager) Clear(name string, keepHistory bool) error {
	envPath, err := m.ResolveEnvPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(envPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment does not exist: %s", name)
		}
		return err
	}
	if err := m.SaveState(name, map[string]any{}); err != nil {
		return err
	}
	if keepHistory {
		return nil
	}
	return os.WriteFile(filepath.Join(envPath, "history.txt"), []byte(""), 0o644)
}

func (m *Manager) Rename(oldName, newName string) error {
	oldPath, err := m.ResolveEnvPath(oldName)
	if err != nil {
		return err
	}
	newPath, err := m.ResolveEnvPath(newName)
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

func (m *Manager) List() ([]EnvironmentInfo, error) {
	envsPath, err := m.EnvsPath()
	if err != nil {
		return nil, err
	}
	if err := m.EnsureBaseStructure(); err != nil {
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
		info, err := m.Info(name)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	return infos, nil
}

func (m *Manager) Info(name string) (EnvironmentInfo, error) {
	envPath, err := m.ResolveEnvPath(name)
	if err != nil {
		return EnvironmentInfo{}, err
	}
	state, err := m.LoadState(name)
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

func (m *Manager) Export(name, outputPath string) error {
	envPath, err := m.ResolveEnvPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(envPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment does not exist: %s", name)
		}
		return err
	}

	state, err := m.LoadState(name)
	if err != nil {
		state = &State{Variables: map[string]SerializedVariable{}, LastAccess: ""}
	}

	history, err := m.ReadHistory(name)
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

func (m *Manager) Import(inputPath, targetName string, force bool) (string, error) {
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

	if err := m.EnsureBaseStructure(); err != nil {
		return "", err
	}

	envPath, err := m.ResolveEnvPath(name)
	if err != nil {
		return "", err
	}

	if !m.Exists(name) {
		if err := m.Create(name); err != nil {
			return "", err
		}
	} else if !force {
		empty, err := m.isEnvironmentEmpty(name)
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

func (m *Manager) isEnvironmentEmpty(name string) (bool, error) {
	state, err := m.LoadState(name)
	if err == nil && state != nil && len(state.Variables) > 0 {
		return false, nil
	}

	history, err := m.ReadHistory(name)
	if err == nil && len(history) > 0 {
		return false, nil
	}

	envPath, err := m.ResolveEnvPath(name)
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

// Package-level wrappers around the default Manager. These exist for
// backward compatibility with code that doesn't yet thread a Manager
// through. New code should prefer NewManager + method calls.

func BasePath() (string, error)               { return defaultManager.BasePath() }
func EnvsPath() (string, error)                { return defaultManager.EnvsPath() }
func ResolveEnvPath(name string) (string, error) {
	return defaultManager.ResolveEnvPath(name)
}
func EnsureDefaultEnvironment() error           { return defaultManager.EnsureDefaultEnvironment() }
func EnsureBaseStructure() error                { return defaultManager.EnsureBaseStructure() }
func Exists(name string) bool                   { return defaultManager.Exists(name) }
func Create(name string) error                  { return defaultManager.Create(name) }
func Open(name string) (string, error)          { return defaultManager.Open(name) }
func Delete(name string) error                  { return defaultManager.Delete(name) }
func Clear(name string, keepHistory bool) error { return defaultManager.Clear(name, keepHistory) }
func Rename(oldName, newName string) error      { return defaultManager.Rename(oldName, newName) }
func List() ([]EnvironmentInfo, error)          { return defaultManager.List() }
func Info(name string) (EnvironmentInfo, error) { return defaultManager.Info(name) }
func Export(name, outputPath string) error      { return defaultManager.Export(name, outputPath) }
func Import(inputPath, targetName string, force bool) (string, error) {
	return defaultManager.Import(inputPath, targetName, force)
}

package env

import (
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

func Clear(name string) error {
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

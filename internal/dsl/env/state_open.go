package env

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// OpenVariables restores envName's variables for a command that is about to run
// against them. An environment with no state.json yet starts empty. A state.json
// that exists but cannot be read is an error naming the file, so the command
// does not run and the save that follows it cannot write an empty variable map
// over the variables the file still holds.
func (m *Manager) OpenVariables(envName string) (map[string]any, error) {
	vars, err := m.RestoreVariables(envName)
	if err == nil {
		return vars, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	path := "state.json"
	if envPath, pathErr := m.ResolveEnvPath(envName); pathErr == nil {
		path = filepath.Join(envPath, "state.json")
	}
	return nil, fmt.Errorf("environment %q: %s cannot be read: %w; nothing was run and nothing was changed. Repair or restore the file, or delete it to start the environment empty", envName, path, err)
}

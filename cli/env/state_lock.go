package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	json "github.com/goccy/go-json"
)

const (
	stateLockName    = "state.lock"
	stateLockTimeout = 10 * time.Second
	// stateLockStale is how old a lock file may get before it is taken to
	// belong to a process that died while holding it.
	stateLockStale = 30 * time.Second
)

// lockState takes envPath's state lock and returns the function that releases
// it. The lock only covers one read-merge-write of state.json, never a whole
// command or REPL session, so waiting for it is short.
func lockState(envPath string) (func(), error) {
	path := filepath.Join(envPath, stateLockName)
	deadline := time.Now().Add(stateLockTimeout)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = f.WriteString(strconv.Itoa(os.Getpid()))
			_ = f.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		// Windows refuses with "access denied", not "exists", to create a file
		// whose deletion by another holder has not finished. That is a lock
		// someone is letting go of, so wait for it like a held one.
		held := errors.Is(err, os.ErrExist) || (runtime.GOOS == "windows" && errors.Is(err, os.ErrPermission))
		if !held {
			return nil, err
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > stateLockStale {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("another insyra process is saving this environment (%s is held: %v); try again", path, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// baselineFor returns the encoded variables this Manager last read or wrote for
// envPath, if it has done either.
func (m *Manager) baselineFor(envPath string) (map[string]string, bool) {
	m.baseMu.Lock()
	defer m.baseMu.Unlock()
	base, ok := m.baselines[envPath]
	return base, ok
}

func (m *Manager) setBaseline(envPath string, stored map[string]json.RawMessage) {
	base := make(map[string]string, len(stored))
	for k, v := range stored {
		base[k] = string(v)
	}
	m.baseMu.Lock()
	defer m.baseMu.Unlock()
	if m.baselines == nil {
		m.baselines = map[string]map[string]string{}
	}
	m.baselines[envPath] = base
}

// setBaselineFromVars records how each restored variable encodes, which is what
// a later save compares against to tell the variables this process changed.
func (m *Manager) setBaselineFromVars(envPath string, vars map[string]any) {
	stored := make(map[string]json.RawMessage, len(vars))
	for key, value := range vars {
		if raw, err := marshalVariable(value); err == nil {
			stored[key] = raw
		}
	}
	m.setBaseline(envPath, stored)
}

func unsavedNames(unsaved []UnsavedVariable) map[string]bool {
	names := make(map[string]bool, len(unsaved))
	for _, u := range unsaved {
		names[u.Name] = true
	}
	return names
}

// currentStored reads the variables state.json holds now, as stored. A file
// that is missing or unreadable holds none.
func currentStored(path string) map[string]json.RawMessage {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Variables map[string]json.RawMessage `json:"variables"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}
	return doc.Variables
}

// mergeStored lays this process's changes over the file's current variables.
// mine holds every variable this process has; base is what it read or wrote
// last. A variable equal to base is unchanged here, so the file's version, which
// another process may have changed or removed, stands. A variable that differs
// or is new is this process's. A variable in base that mine no longer has was
// removed here. A variable only the file has was created elsewhere and is kept.
func mergeStored(mine map[string]json.RawMessage, unsaved map[string]bool, base map[string]string, file map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(mine)+len(file))
	for key, raw := range file {
		if _, had := base[key]; had {
			if _, still := mine[key]; !still && !unsaved[key] {
				continue // removed by this process
			}
			if unsaved[key] {
				continue
			}
		}
		out[key] = raw
	}
	for key, raw := range mine {
		if prev, had := base[key]; had && prev == string(raw) {
			continue // unchanged here: the file's version stands, or another process removed it
		}
		out[key] = raw
	}
	return out
}

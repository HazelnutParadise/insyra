package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
)

type SerializedVariable struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	Data any    `json:"data"`
}

type State struct {
	Variables  map[string]SerializedVariable `json:"variables"`
	LastAccess string                        `json:"lastAccess"`
}

// UnsavedVariable is a variable SaveVariables left out of state.json.
type UnsavedVariable struct {
	Name   string
	Type   string // the Go type the variable held, as %T prints it
	Reason string // why the environment cannot store it
}

// SaveVariables writes vars as envName's state.json and returns the variables
// it left out because the environment cannot store them, sorted by name. The
// error means the file was not written.
func (m *Manager) SaveVariables(envName string, vars map[string]any) ([]UnsavedVariable, error) {
	envPath, err := m.ResolveEnvPath(envName)
	if err != nil {
		return nil, err
	}
	// Every variable is encoded and written out on its own, so one the
	// environment cannot hold leaves the rest of the state alone and is
	// reported instead of being dropped without a word.
	stored := make(map[string]json.RawMessage, len(vars))
	var unsaved []UnsavedVariable
	for key, value := range vars {
		raw, err := marshalVariable(value)
		if err != nil {
			unsaved = append(unsaved, UnsavedVariable{
				Name:   key,
				Type:   fmt.Sprintf("%T", value),
				Reason: err.Error(),
			})
			continue
		}
		stored[key] = raw
	}
	document := struct {
		Variables  map[string]json.RawMessage `json:"variables"`
		LastAccess string                     `json:"lastAccess"`
	}{
		Variables:  stored,
		LastAccess: time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	// Atomic write: write to a temp file then rename over state.json (rename is
	// atomic on the same filesystem), so an interruption mid-write cannot leave a
	// truncated/corrupted state.json that would wipe the user's variables.
	finalPath := filepath.Join(envPath, "state.json")
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return nil, err
	}
	if len(unsaved) == 0 {
		return nil, nil
	}
	sort.Slice(unsaved, func(i, j int) bool { return unsaved[i].Name < unsaved[j].Name })
	return unsaved, nil
}

// SaveState writes vars as envName's state.json. A variable the environment
// cannot store is left out; SaveVariables reports which. The error means the
// file was not written.
func (m *Manager) SaveState(envName string, vars map[string]any) error {
	_, err := m.SaveVariables(envName, vars)
	return err
}

// marshalVariable encodes value and writes the encoded form out, so that a
// value the environment cannot encode and an encoded value the encoder cannot
// write are the same failure to SaveState.
func marshalVariable(value any) (json.RawMessage, error) {
	encoded, err := encodeVariable(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(encoded)
}

// LoadState reads envName's state.json. Top-level scalars come back typed:
// one saved by this release at the Go type it was saved with, one written by
// an earlier release as int64 when it is an integer literal and float64
// otherwise. Every other variable keeps the form it has in the file;
// RestoreVariables turns those into Go values.
func (m *Manager) LoadState(envName string) (*State, error) {
	state, err := m.readState(envName)
	if err != nil {
		return nil, err
	}
	for key, sv := range state.Variables {
		switch sv.Type {
		case kindScalar:
			if value, err := decodeStoredScalar(sv); err == nil {
				sv.Data = value
			}
		case "DataTable", "DataList", kindTable, kindList, kindSlice, kindScaler, kindHClust:
			// keep stored form as-is
		default:
			// legacy "Raw" and unknown kinds: apply legacy decoding
			sv.Data = decodeEnvValue(sv.Data)
		}
		state.Variables[key] = sv
	}
	return state, nil
}

// readState reads envName's state.json as stored: numbers stay json.Number and nothing is converted.
func (m *Manager) readState(envName string) (*State, error) {
	envPath, err := m.ResolveEnvPath(envName)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(envPath, "state.json"))
	if err != nil {
		return nil, err
	}
	var state State
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&state); err != nil {
		return nil, err
	}
	if state.Variables == nil {
		state.Variables = map[string]SerializedVariable{}
	}
	return &state, nil
}

// RestoreVariables reads envName's variables back at the Go types they were
// saved with. A variable in the layout earlier releases wrote is read by the
// rules those releases used. A variable this build cannot decode is kept as an
// unreadableVariable so the next save writes it back with its original type
// and name intact.
func (m *Manager) RestoreVariables(envName string) (map[string]any, error) {
	state, err := m.readState(envName)
	if err != nil {
		return nil, err
	}
	vars := make(map[string]any, len(state.Variables))
	for key, serialized := range state.Variables {
		value, ok, err := decodeVariable(serialized)
		if !ok {
			if isLegacyKind(serialized.Type) {
				vars[key] = deserializeLegacyVariable(serialized)
			} else {
				// A kind this build does not know, such as one a newer
				// release writes, is kept as it was stored.
				vars[key] = unreadableVariable{stored: serialized}
			}
			continue
		}
		if err != nil {
			// The stored form becomes an unreadableVariable so the next save
			// writes it back as it was, preserving its type and name.
			vars[key] = unreadableVariable{stored: serialized}
			continue
		}
		vars[key] = value
	}
	return vars, nil
}

// specialFloatKey is how a file written by an earlier release spells a float64
// JSON cannot represent: {"$float": "NaN" | "+Inf" | "-Inf"}. Nothing writes
// that object any more; a current file stores the same three names as plain
// strings under the cell's own type tag.
const specialFloatKey = "$float"

// decodeEnvValue reads one cell of a file written by an earlier release: a
// $float marker object becomes the float64 it stands for, and a number is typed
// by coerceEnvNumber. It reads the writing that release did, and nothing writes
// that shape any more.
func decodeEnvValue(v any) any {
	if m, ok := v.(map[string]any); ok && len(m) == 1 {
		if s, ok := m[specialFloatKey].(string); ok {
			switch s {
			case "NaN":
				return math.NaN()
			case "+Inf":
				return math.Inf(1)
			case "-Inf":
				return math.Inf(-1)
			}
		}
	}
	return coerceEnvNumber(v)
}

// isLegacyKind reports whether a stored type is one an earlier release wrote,
// which the legacy reader below understands.
func isLegacyKind(kind string) bool {
	switch kind {
	case "DataTable", "DataList", "Raw":
		return true
	}
	return false
}

// deserializeLegacyVariable reads a variable in the layout earlier releases
// wrote: a "DataTable" (either as one JSON document string or as columns of
// $float markers), a "DataList", or a "Raw" scalar. Nothing writes that layout
// any more, so this is the only reader of it.
func deserializeLegacyVariable(serialized SerializedVariable) any {
	switch serialized.Type {
	case "DataTable":
		switch data := serialized.Data.(type) {
		case string:
			// Legacy layout: the whole table as a JSON document.
			table, err := insyra.ReadJSON(data)
			if err != nil || table == nil {
				return serialized.Data
			}
			if serialized.Name != "" {
				table.SetName(serialized.Name)
			}
			return table
		case map[string]any:
			table := insyra.NewDataTable()
			cols, _ := data["columns"].([]any)
			for _, c := range cols {
				cm, ok := c.(map[string]any)
				if !ok {
					continue
				}
				raw, _ := cm["data"].([]any)
				cells := make([]any, len(raw))
				for i, e := range raw {
					cells[i] = decodeEnvValue(e)
				}
				dl := insyra.NewDataList(cells...)
				if name, ok := cm["name"].(string); ok {
					dl.SetName(name)
				}
				table.AppendCols(dl)
			}
			if rn, ok := data["rowNames"].([]any); ok && len(rn) > 0 {
				names := make([]string, len(rn))
				for i, e := range rn {
					names[i], _ = e.(string)
				}
				table.SetRowNames(names)
			}
			if serialized.Name != "" {
				table.SetName(serialized.Name)
			}
			return table
		}
	case "DataList":
		if arr, ok := serialized.Data.([]any); ok {
			converted := make([]any, len(arr))
			for i, e := range arr {
				converted[i] = decodeEnvValue(e)
			}
			dl := insyra.NewDataList(converted...)
			if serialized.Name != "" {
				dl.SetName(serialized.Name)
			}
			return dl
		}
	}
	return decodeEnvValue(serialized.Data)
}

// coerceEnvNumber types a json.Number (produced by UseNumber decoding) as int64
// when it is an integer literal (preserving values beyond 2^53) and float64
// otherwise; non-number values pass through unchanged.
func coerceEnvNumber(v any) any {
	if n, ok := v.(json.Number); ok {
		if i, err := n.Int64(); err == nil {
			return i
		}
		if f, err := n.Float64(); err == nil {
			return f
		}
		return n.String()
	}
	return v
}

func (m *Manager) AppendHistory(envName, command string) error {
	envPath, err := m.ResolveEnvPath(envName)
	if err != nil {
		return err
	}
	file := filepath.Join(envPath, "history.txt")
	handle, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		_ = handle.Close()
	}()
	_, err = fmt.Fprintf(handle, "%s\n", command)
	return err
}

func (m *Manager) ReadHistory(envName string) ([]string, error) {
	envPath, err := m.ResolveEnvPath(envName)
	if err != nil {
		return nil, err
	}
	bytes, err := os.ReadFile(filepath.Join(envPath, "history.txt"))
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return []string{}, nil
	}
	lines := []string{}
	current := ""
	for _, ch := range string(bytes) {
		if ch == '\n' {
			if current != "" {
				lines = append(lines, current)
			}
			current = ""
			continue
		}
		current += string(ch)
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines, nil
}

// Package-level wrappers around the default Manager.

func SaveState(envName string, vars map[string]any) error {
	return defaultManager.SaveState(envName, vars)
}

func SaveVariables(envName string, vars map[string]any) ([]UnsavedVariable, error) {
	return defaultManager.SaveVariables(envName, vars)
}

func LoadState(envName string) (*State, error) {
	return defaultManager.LoadState(envName)
}

func RestoreVariables(envName string) (map[string]any, error) {
	return defaultManager.RestoreVariables(envName)
}

func AppendHistory(envName, command string) error {
	return defaultManager.AppendHistory(envName, command)
}

func ReadHistory(envName string) ([]string, error) {
	return defaultManager.ReadHistory(envName)
}

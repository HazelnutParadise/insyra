package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func SaveState(envName string, vars map[string]any) error {
	envPath, err := ResolveEnvPath(envName)
	if err != nil {
		return err
	}
	state := State{
		Variables:  map[string]SerializedVariable{},
		LastAccess: time.Now().UTC().Format(time.RFC3339),
	}
	for key, value := range vars {
		state.Variables[key] = serializeVariable(value)
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(envPath, "state.json"), payload, 0o644)
}

func LoadState(envName string) (*State, error) {
	envPath, err := ResolveEnvPath(envName)
	if err != nil {
		return nil, err
	}
	bytes, err := os.ReadFile(filepath.Join(envPath, "state.json"))
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(bytes, &state); err != nil {
		return nil, err
	}
	if state.Variables == nil {
		state.Variables = map[string]SerializedVariable{}
	}
	return &state, nil
}

func RestoreVariables(envName string) (map[string]any, error) {
	state, err := LoadState(envName)
	if err != nil {
		return nil, err
	}
	vars := make(map[string]any, len(state.Variables))
	for key, serialized := range state.Variables {
		vars[key] = deserializeVariable(serialized)
	}
	return vars, nil
}

func serializeVariable(value any) SerializedVariable {
	switch typed := value.(type) {
	case *insyra.DataTable:
		return SerializedVariable{Type: "DataTable", Name: typed.GetName(), Data: typed.ToJSON_String(true)}
	case *insyra.DataList:
		return SerializedVariable{Type: "DataList", Name: typed.GetName(), Data: typed.Data()}
	default:
		return SerializedVariable{Type: "Raw", Data: typed}
	}
}

func deserializeVariable(serialized SerializedVariable) any {
	switch serialized.Type {
	case "DataTable":
		if text, ok := serialized.Data.(string); ok {
			table, err := insyra.ReadJSON(text)
			if err != nil {
				return serialized.Data
			}
			if table != nil && serialized.Name != "" {
				table.SetName(serialized.Name)
			}
			if table != nil {
				return table
			}
		}
	case "DataList":
		if arr, ok := serialized.Data.([]any); ok {
			dl := insyra.NewDataList(arr...)
			if serialized.Name != "" {
				dl.SetName(serialized.Name)
			}
			return dl
		}
		if arr, ok := serialized.Data.([]interface{}); ok {
			converted := make([]any, len(arr))
			copy(converted, arr)
			dl := insyra.NewDataList(converted...)
			if serialized.Name != "" {
				dl.SetName(serialized.Name)
			}
			return dl
		}
	}
	return serialized.Data
}

func AppendHistory(envName, command string) error {
	envPath, err := ResolveEnvPath(envName)
	if err != nil {
		return err
	}
	file := filepath.Join(envPath, "history.txt")
	handle, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = handle.Close()
	}()
	_, err = fmt.Fprintf(handle, "%s\n", command)
	return err
}

func ReadHistory(envName string) ([]string, error) {
	envPath, err := ResolveEnvPath(envName)
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

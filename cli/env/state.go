package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
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

func (m *Manager) SaveState(envName string, vars map[string]any) error {
	envPath, err := m.ResolveEnvPath(envName)
	if err != nil {
		return err
	}
	state := State{
		Variables:  map[string]SerializedVariable{},
		LastAccess: time.Now().UTC().Format(time.RFC3339),
	}
	for key, value := range vars {
		// Scalers hold fitted state in unexported fields and cannot be
		// round-tripped through JSON; they are session-only.
		if _, ok := value.(insyra.Scaler); ok {
			continue
		}
		state.Variables[key] = serializeVariable(value)
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	// Atomic write: write to a temp file then rename over state.json (rename is
	// atomic on the same filesystem), so an interruption mid-write cannot leave a
	// truncated/corrupted state.json that would wipe the user's variables.
	finalPath := filepath.Join(envPath, "state.json")
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, finalPath)
}

func (m *Manager) LoadState(envName string) (*State, error) {
	envPath, err := m.ResolveEnvPath(envName)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(envPath, "state.json"))
	if err != nil {
		return nil, err
	}
	var state State
	// Decode numbers as json.Number so integer variables keep their int64 type
	// (and large integers keep full precision) instead of collapsing to float64;
	// they are typed below and, for DataList elements, when the list is restored.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&state); err != nil {
		return nil, err
	}
	if state.Variables == nil {
		state.Variables = map[string]SerializedVariable{}
	}
	// Top-level scalars get the same numeric typing DataList elements get, so a
	// saved float64 reloads as float64 and an int64 as int64 instead of staying
	// a json.Number that no type assertion in a command will match.
	for key, serialized := range state.Variables {
		if serialized.Type == "DataList" || serialized.Type == "DataTable" {
			continue
		}
		serialized.Data = decodeEnvValue(serialized.Data)
		state.Variables[key] = serialized
	}
	return &state, nil
}

func (m *Manager) RestoreVariables(envName string) (map[string]any, error) {
	state, err := m.LoadState(envName)
	if err != nil {
		return nil, err
	}
	vars := make(map[string]any, len(state.Variables))
	for key, serialized := range state.Variables {
		vars[key] = deserializeVariable(serialized)
	}
	return vars, nil
}

// specialFloatKey marks a JSON object that stands in for a float64 JSON
// cannot represent: {"$float": "NaN" | "+Inf" | "-Inf"}. CSV files with
// blank cells load as NaN, so without this a table with one missing value
// could not be saved at all.
const specialFloatKey = "$float"

// encodeSpecialFloats replaces NaN and infinities (at any depth of []any)
// with their marker objects so encoding/json accepts the value.
func encodeSpecialFloats(v any) any {
	switch t := v.(type) {
	case float64:
		return encodeSpecialFloat64(t)
	case float32:
		return encodeSpecialFloat64(float64(t))
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = encodeSpecialFloats(e)
		}
		return out
	default:
		return v
	}
}

func encodeSpecialFloat64(f float64) any {
	switch {
	case math.IsNaN(f):
		return map[string]any{specialFloatKey: "NaN"}
	case math.IsInf(f, 1):
		return map[string]any{specialFloatKey: "+Inf"}
	case math.IsInf(f, -1):
		return map[string]any{specialFloatKey: "-Inf"}
	default:
		return f
	}
}

// decodeEnvValue is the inverse of encodeSpecialFloats plus the json.Number
// typing of coerceEnvNumber, applied to one cell.
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

// serializedTable is the on-disk shape of a DataTable variable: columns in
// order with their names, plus row names when any are set. Older state
// files hold the table as a JSON string (ToJSON_String) and are still read.
type serializedTable struct {
	Columns  []serializedColumn `json:"columns"`
	RowNames []string           `json:"rowNames,omitempty"`
}

type serializedColumn struct {
	Name string `json:"name"`
	Data []any  `json:"data"`
}

func serializeVariable(value any) SerializedVariable {
	switch typed := value.(type) {
	case *insyra.DataTable:
		if typed == nil {
			return SerializedVariable{Type: "Raw", Data: nil}
		}
		st := serializedTable{}
		for i := 0; i < typed.NumCols(); i++ {
			col := typed.GetColByNumber(i)
			st.Columns = append(st.Columns, serializedColumn{
				Name: col.GetName(),
				Data: encodeSpecialFloats(col.Data()).([]any),
			})
		}
		for _, rn := range typed.RowNames() {
			if rn != "" {
				st.RowNames = typed.RowNames()
				break
			}
		}
		return SerializedVariable{Type: "DataTable", Name: typed.GetName(), Data: st}
	case *insyra.DataList:
		if typed == nil {
			return SerializedVariable{Type: "Raw", Data: nil}
		}
		return SerializedVariable{Type: "DataList", Name: typed.GetName(), Data: encodeSpecialFloats(typed.Data())}
	default:
		return SerializedVariable{Type: "Raw", Data: encodeSpecialFloats(typed)}
	}
}

func deserializeVariable(serialized SerializedVariable) any {
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

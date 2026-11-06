package env

import (
	"bytes"
	"encoding/json"
	"time"
)

// stateFile is state.json as stored, and the "state" object of an export
// file. lastAccess is an RFC3339 string, or "" for an environment whose last
// access is unknown, which is the form every release reads and writes.
type stateFile[V any] struct {
	Variables  map[string]V    `json:"variables"`
	LastAccess json.RawMessage `json:"lastAccess,omitempty"`
}

// exportFile is the file Export writes and Import reads.
type exportFile struct {
	SchemaVersion int                            `json:"schemaVersion"`
	ExportedAt    string                         `json:"exportedAt"`
	Environment   string                         `json:"environment"`
	State         *stateFile[SerializedVariable] `json:"state"`
	History       []string                       `json:"history"`
	Config        json.RawMessage                `json:"config"`
}

// decodeJSON decodes data into v with numbers kept as json.Number, so an
// integer beyond 2^53 is written back out as the same literal.
func decodeJSON(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return dec.Decode(v)
}

// stateFromFile turns a stored state into a State.
func stateFromFile(file *stateFile[SerializedVariable]) *State {
	if file == nil {
		return &State{Variables: map[string]SerializedVariable{}}
	}
	variables := file.Variables
	if variables == nil {
		variables = map[string]SerializedVariable{}
	}
	return &State{Variables: variables, LastAccess: decodeLastAccess(file.LastAccess)}
}

// stateToFile turns a State into its stored form.
func stateToFile(state *State) (*stateFile[SerializedVariable], error) {
	lastAccess, err := encodeLastAccess(state.LastAccess)
	if err != nil {
		return nil, err
	}
	return &stateFile[SerializedVariable]{Variables: state.Variables, LastAccess: lastAccess}, nil
}

// decodeLastAccess reads a stored timestamp. The time is only metadata, so
// anything that is not an RFC3339 string JSON can write back, including a
// missing field, null or a value of another JSON type, is an unknown time.
func decodeLastAccess(raw json.RawMessage) time.Time {
	var text string
	if len(raw) == 0 || json.Unmarshal(raw, &text) != nil || text == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return time.Time{}
	}
	// Parse accepts some offsets (such as +24:00) that JSON cannot write.
	if _, err := parsed.MarshalJSON(); err != nil {
		return time.Time{}
	}
	return parsed
}

// encodeLastAccess writes t as stored: "" for an unknown time, otherwise its
// RFC3339 text. A time JSON cannot represent is an error.
func encodeLastAccess(t time.Time) (json.RawMessage, error) {
	if t.IsZero() {
		return json.RawMessage(`""`), nil
	}
	return t.MarshalJSON()
}

// MarshalJSON writes p in the layout Export writes.
func (p ExportPayload) MarshalJSON() ([]byte, error) {
	file := exportFile{
		SchemaVersion: p.SchemaVersion,
		ExportedAt:    p.ExportedAt,
		Environment:   p.Environment,
		History:       p.History,
		Config:        p.Config,
	}
	if p.State != nil {
		state, err := stateToFile(p.State)
		if err != nil {
			return nil, err
		}
		file.State = state
	}
	return json.Marshal(file)
}

// UnmarshalJSON reads a file Export wrote, by the rules Import reads it with.
func (p *ExportPayload) UnmarshalJSON(data []byte) error {
	var file exportFile
	if err := decodeJSON(data, &file); err != nil {
		return err
	}
	*p = ExportPayload{
		SchemaVersion: file.SchemaVersion,
		ExportedAt:    file.ExportedAt,
		Environment:   file.Environment,
		History:       file.History,
		Config:        file.Config,
	}
	if file.State != nil {
		p.State = stateFromFile(file.State)
	}
	return nil
}

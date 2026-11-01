package env

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateLastAccessLoadsLegacyFiles(t *testing.T) {
	for _, tc := range []struct{ name, field, want string }{
		{"empty", `,"lastAccess":""`, ""},
		{"dated", `,"lastAccess":"2026-10-01T12:34:56Z"`, "2026-10-01T12:34:56Z"},
		{"fraction-offset", `,"lastAccess":"2026-10-01T12:34:56.123456789+08:00"`, "2026-10-01T12:34:56.123456789+08:00"},
		{"missing", "", ""},
		{"null", `,"lastAccess":null`, ""},
		{"invalid-text", `,"lastAccess":"not a date"`, ""},
		{"invalid-zone", `,"lastAccess":"2026-10-01T12:34:56+24:00"`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newStateTestManager(t)
			path, err := mgr.ResolveEnvPath("default")
			if err != nil {
				t.Fatal(err)
			}
			raw := []byte(`{"variables":{"large":{"type":"Raw","data":9007199254740993},"future":{"type":"future","data":{"n":9007199254740993}}}` + tc.field + `}`)
			if err := os.WriteFile(filepath.Join(path, "state.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			state, err := mgr.LoadState("default")
			if err != nil {
				t.Fatal(err)
			}
			got, ok := any(state.LastAccess).(time.Time)
			if !ok {
				t.Fatalf("LastAccess = %T, want time.Time", state.LastAccess)
			}
			var want time.Time
			if tc.want != "" {
				want, err = time.Parse(time.RFC3339, tc.want)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !got.Equal(want) {
				t.Fatalf("LastAccess = %v, want %v", got, want)
			}
			info, err := mgr.Info("default")
			if err != nil {
				t.Fatal(err)
			}
			if info.VariableCount != 2 || !info.LastAccess.Equal(want) {
				t.Fatalf("Info = %+v", info)
			}
			vars, err := mgr.RestoreVariables("default")
			if err != nil {
				t.Fatal(err)
			}
			if vars["large"] != int64(9007199254740993) {
				t.Fatalf("large = %#v", vars["large"])
			}
			if _, ok := vars["future"].(unreadableVariable); !ok {
				t.Fatalf("future = %T", vars["future"])
			}
			export := filepath.Join(t.TempDir(), "export.json")
			if err := mgr.Export("default", export); err != nil {
				t.Fatal(err)
			}
			if _, err := mgr.Import(export, "copy", false); err != nil {
				t.Fatal(err)
			}
			imported, err := mgr.LoadState("copy")
			if err != nil {
				t.Fatal(err)
			}
			importedTime := any(imported.LastAccess).(time.Time)
			if !want.IsZero() && !importedTime.Equal(want) {
				t.Fatalf("import time = %v, want %v", importedTime, want)
			}
			restored, err := mgr.RestoreVariables("copy")
			if err != nil {
				t.Fatal(err)
			}
			if restored["large"] != int64(9007199254740993) {
				t.Fatalf("imported large = %#v", restored["large"])
			}
			if err := mgr.SaveState("copy", restored); err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(readStateFile(t, mgr, "copy"), []byte(`9007199254740993`)) {
				t.Fatal("large integer lost")
			}
			future := restored["future"].(unreadableVariable)
			n := future.stored.Data.(map[string]any)["n"]
			if n != json.Number("9007199254740993") {
				t.Fatalf("future n = %#v", n)
			}
		})
	}
}

func TestStateLastAccessOldWriterFixture(t *testing.T) {
	raw, err := os.ReadFile("testdata/state-0.3.6.json")
	if err != nil {
		t.Fatal(err)
	}
	mgr := newStateTestManager(t)
	path, err := mgr.ResolveEnvPath("default")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "state.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	state, err := mgr.LoadState("default")
	if err != nil {
		t.Fatal(err)
	}
	if stamp, ok := any(state.LastAccess).(time.Time); !ok || stamp.IsZero() {
		t.Fatalf("LastAccess = %#v", state.LastAccess)
	}
	vars, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 2 || vars["large"] != int64(9007199254740993) || vars["text"] != "old writer" {
		t.Fatalf("vars = %#v", vars)
	}
}

func TestStateLastAccessWritesLegacyShape(t *testing.T) {
	for _, stamp := range []string{"", "2026-10-01T12:34:56Z", "2026-10-01T12:34:56.123456789+08:00"} {
		var state State
		if err := json.Unmarshal([]byte(`{"variables":{},"lastAccess":"`+stamp+`"}`), &state); err != nil {
			t.Fatal(err)
		}
		if _, ok := any(state.LastAccess).(time.Time); !ok {
			t.Fatalf("LastAccess = %T", state.LastAccess)
		}
		raw, err := json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		// The old State definition must still decode the new output.
		var old struct {
			Variables  map[string]SerializedVariable `json:"variables"`
			LastAccess string                        `json:"lastAccess"`
		}
		if err := json.Unmarshal(raw, &old); err != nil {
			t.Fatal(err)
		}
		if old.LastAccess != stamp {
			t.Fatalf("timestamp = %q, want %q", old.LastAccess, stamp)
		}
	}
}

func TestStateLastAccessRejectsExistingStructuralErrors(t *testing.T) {
	for _, value := range []string{`123`, `true`, `{}`, `[]`} {
		mgr := newStateTestManager(t)
		path, err := mgr.ResolveEnvPath("default")
		if err != nil {
			t.Fatal(err)
		}
		raw := []byte(`{"variables":{"x":{"type":"Raw","data":1}},"lastAccess":` + value + `}`)
		if err := os.WriteFile(filepath.Join(path, "state.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := mgr.LoadState("default"); err == nil {
			t.Fatalf("accepted %s", value)
		}
		if _, err := mgr.RestoreVariables("default"); err == nil {
			t.Fatalf("restored %s", value)
		}
		if !bytes.Equal(readStateFile(t, mgr, "default"), raw) {
			t.Fatal("read changed file")
		}
	}
}

func TestStateLastAccessRefusesUnrepresentableTime(t *testing.T) {
	for _, stamp := range []time.Time{
		time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.FixedZone("invalid", 24*60*60)),
	} {
		state := State{LastAccess: stamp}
		if _, err := json.Marshal(state); err == nil {
			t.Fatalf("marshaled invalid RFC3339 time %v", stamp)
		}
	}
}

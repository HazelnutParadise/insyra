package env

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// CLI-2: NaN and infinities survive a save/restore round trip for both
// DataList and DataTable variables, and a table never degrades to a string.
func TestSaveStateRoundTripsNaN(t *testing.T) {
	mgr := NewManager(t.TempDir(), "envs")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	dl := insyra.NewDataList(1.0, math.NaN(), math.Inf(1), math.Inf(-1), "s", nil).SetName("L")
	dt := insyra.NewDataTable(
		insyra.NewDataList(1.0, math.NaN(), 3.0).SetName("v"),
		insyra.NewDataList("a", "b", "c").SetName("k"),
	).SetName("T")
	if err := mgr.SaveState("default", map[string]any{"dl": dl, "dt": dt}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	vars, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	gotDL, ok := vars["dl"].(*insyra.DataList)
	if !ok {
		t.Fatalf("dl restored as %T", vars["dl"])
	}
	d := gotDL.Data()
	if num(d[0]) != 1 || !isNaN(d[1]) || d[2] != math.Inf(1) || d[3] != math.Inf(-1) || d[4] != "s" || d[5] != nil {
		t.Fatalf("dl restored as %v", d)
	}
	if gotDL.GetName() != "L" {
		t.Fatalf("dl name = %q", gotDL.GetName())
	}
	gotDT, ok := vars["dt"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("dt restored as %T", vars["dt"])
	}
	if gotDT.GetName() != "T" {
		t.Fatalf("dt name = %q", gotDT.GetName())
	}
	v := gotDT.GetColByName("v").Data()
	if num(v[0]) != 1 || !isNaN(v[1]) || num(v[2]) != 3 {
		t.Fatalf("dt column v restored as %v", v)
	}
	if k := gotDT.GetColByName("k").Data(); k[1] != "b" {
		t.Fatalf("dt column k restored as %v", k)
	}
}

// num reads a restored number regardless of whether the round trip typed
// it int64 or float64 (whole floats come back as int64 today).
func num(v any) float64 {
	switch t := v.(type) {
	case int64:
		return float64(t)
	case float64:
		return t
	}
	return math.NaN()
}

func isNaN(v any) bool {
	f, ok := v.(float64)
	return ok && math.IsNaN(f)
}

// A table or list without NaN or ±Inf is written exactly as before: a
// DataTable as its ToJSON_String document, a DataList as its cells, and
// float32 cells keep their float32 encoding.
func TestSaveStateKeepsLegacyLayoutWithoutSpecialFloats(t *testing.T) {
	mgr := NewManager(t.TempDir(), "envs")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	dl := insyra.NewDataList(1, 2.5, float32(0.1), int64(1)<<60, "s", nil, true).SetName("L")
	dt := insyra.NewDataTable(
		insyra.NewDataList(1.0, float32(0.1), 3).SetName("v"),
		insyra.NewDataList("a", "b", nil).SetName("k"),
	).SetName("T")
	raw := []any{1.5, float32(0.1), "x"}
	vars := map[string]any{"dl": dl, "dt": dt, "raw": raw, "f": float32(0.1)}
	if err := mgr.SaveState("default", vars); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	envPath, err := mgr.ResolveEnvPath("default")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(envPath, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Variables map[string]json.RawMessage `json:"variables"`
	}
	if err := json.Unmarshal(b, &file); err != nil {
		t.Fatal(err)
	}
	legacy := map[string]SerializedVariable{
		"dl":  {Type: "DataList", Name: "L", Data: dl.Data()},
		"dt":  {Type: "DataTable", Name: "T", Data: dt.ToJSON_String(true)},
		"raw": {Type: "Raw", Data: raw},
		"f":   {Type: "Raw", Data: float32(0.1)},
	}
	for key, want := range legacy {
		wantBytes, err := json.Marshal(want)
		if err != nil {
			t.Fatal(err)
		}
		var got bytes.Buffer
		if err := json.Compact(&got, file.Variables[key]); err != nil {
			t.Fatal(err)
		}
		if got.String() != string(wantBytes) {
			t.Errorf("%s written as %s, want the legacy %s", key, got.String(), wantBytes)
		}
	}
}

// A state.json written by an earlier release (DataTable as a JSON string)
// still restores.
func TestRestoreVariablesReadsLegacyDataTable(t *testing.T) {
	mgr := NewManager(t.TempDir(), "envs")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	envPath, err := mgr.ResolveEnvPath("default")
	if err != nil {
		t.Fatal(err)
	}
	old := `{
  "variables": {
    "dt": {
      "type": "DataTable",
      "name": "T",
      "data": "[\n  {\n    \"k\": \"a\",\n    \"v\": 1\n  },\n  {\n    \"k\": \"b\",\n    \"v\": 2.5\n  }\n]"
    },
    "dl": {
      "type": "DataList",
      "name": "L",
      "data": [1, 2.5, "s", null]
    }
  },
  "lastAccess": "2026-01-01T00:00:00Z"
}
`
	if err := os.WriteFile(filepath.Join(envPath, "state.json"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	vars, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	dt, ok := vars["dt"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("dt restored as %T", vars["dt"])
	}
	if dt.GetName() != "T" || dt.NumRows() != 2 {
		t.Fatalf("dt restored as %q with %d rows", dt.GetName(), dt.NumRows())
	}
	if v := dt.GetColByName("v").Data(); num(v[0]) != 1 || num(v[1]) != 2.5 {
		t.Fatalf("dt column v restored as %v", v)
	}
	if k := dt.GetColByName("k").Data(); k[0] != "a" || k[1] != "b" {
		t.Fatalf("dt column k restored as %v", k)
	}
	dl, ok := vars["dl"].(*insyra.DataList)
	if !ok {
		t.Fatalf("dl restored as %T", vars["dl"])
	}
	if d := dl.Data(); num(d[0]) != 1 || num(d[1]) != 2.5 || d[2] != "s" || d[3] != nil {
		t.Fatalf("dl restored as %v", d)
	}
}

// A typed nil DataList or DataTable held in a variable no longer crashes
// SaveState.
func TestSaveStateTypedNil(t *testing.T) {
	mgr := NewManager(t.TempDir(), "envs")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatal(err)
	}
	var dl *insyra.DataList
	var dt *insyra.DataTable
	if err := mgr.SaveState("default", map[string]any{"dl": dl, "dt": dt}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
}

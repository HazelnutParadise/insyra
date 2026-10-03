package env

import (
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

// num reads a restored number regardless of whether it came back typed: the
// legacy layout an earlier release wrote has no cell type, so a whole float
// there comes back as int64.
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

// A state.json written by an earlier release still restores: a DataTable as
// columns of $float marker objects, a DataList of markers, and Raw scalars that
// are a marker, a whole number or a fractional one.
func TestRestoreVariablesReadsLegacyNaNLayout(t *testing.T) {
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
      "data": {
        "columns": [
          {"name": "v", "data": [1, {"$float": "NaN"}, 3]},
          {"name": "k", "data": ["a", "b", "c"]}
        ],
        "rowNames": ["x", "y", "z"]
      }
    },
    "dl": {
      "type": "DataList",
      "data": [1.5, {"$float": "+Inf"}, null]
    },
    "negInf": {"type": "Raw", "data": {"$float": "-Inf"}},
    "whole": {"type": "Raw", "data": 7},
    "frac": {"type": "Raw", "data": 1.25}
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
	if dt.GetName() != "T" {
		t.Errorf("dt name = %q want %q", dt.GetName(), "T")
	}
	if v := dt.GetColByName("v").Data(); len(v) != 3 || !isNaN(v[1]) {
		t.Errorf("dt column v restored as %v, want a NaN at row 1", v)
	}
	if k := dt.GetColByName("k").Data(); k[1] != "b" {
		t.Errorf("dt column k restored as %v", k)
	}
	names := dt.RowNames()
	if len(names) != 3 || names[0] != "x" || names[1] != "y" || names[2] != "z" {
		t.Errorf("dt row names restored as %v want x, y, z", names)
	}

	dl, ok := vars["dl"].(*insyra.DataList)
	if !ok {
		t.Fatalf("dl restored as %T", vars["dl"])
	}
	if d := dl.Data(); len(d) != 3 || d[1] != math.Inf(1) || d[2] != nil {
		t.Errorf("dl restored as %v, want +Inf at row 1 and nil at row 2", d)
	}

	if got, ok := vars["negInf"].(float64); !ok || got != math.Inf(-1) {
		t.Errorf("negInf = %#v want float64(-Inf)", vars["negInf"])
	}
	if got, ok := vars["whole"].(int64); !ok || got != 7 {
		t.Errorf("whole = %#v want int64(7)", vars["whole"])
	}
	if got, ok := vars["frac"].(float64); !ok || got != 1.25 {
		t.Errorf("frac = %#v want float64(1.25)", vars["frac"])
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

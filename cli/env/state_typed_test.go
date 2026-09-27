package env

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// readStateFile returns state.json of an environment, as the bytes on disk.
func readStateFile(t *testing.T, mgr *Manager, envName string) []byte {
	t.Helper()
	envPath, err := mgr.ResolveEnvPath(envName)
	if err != nil {
		t.Fatalf("ResolveEnvPath(%q): %v", envName, err)
	}
	raw, err := os.ReadFile(filepath.Join(envPath, "state.json"))
	if err != nil {
		t.Fatalf("reading state.json: %v", err)
	}
	return raw
}

// writtenType reports the type tag an environment wrote for one variable.
func writtenType(t *testing.T, raw []byte, key string) string {
	t.Helper()
	var file struct {
		Variables map[string]json.RawMessage `json:"variables"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decoding state.json: %v", err)
	}
	var variable SerializedVariable
	if err := json.Unmarshal(file.Variables[key], &variable); err != nil {
		t.Fatalf("decoding variable %q: %v", key, err)
	}
	return variable.Type
}

// SaveState writes the kind each variable was encoded under, so a file can be
// read without guessing what a variable holds.
func TestSaveStateWritesTypedLayout(t *testing.T) {
	mgr := newStateTestManager(t)
	vars := map[string]any{
		"dt": insyra.NewDataTable(insyra.NewDataList(1, 2)),
		"dl": insyra.NewDataList(1, 2),
		"s":  1.25,
	}
	if err := mgr.SaveState("default", vars); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	raw := readStateFile(t, mgr, "default")
	for key, want := range map[string]string{"dt": "table", "dl": "list", "s": "scalar"} {
		if got := writtenType(t, raw, key); got != want {
			t.Errorf("%s written with type %q, want %q", key, got, want)
		}
	}
}

// A table comes back in the column order it was saved in, with every cell at
// the Go type it held, and with its row names and name. A second Manager over
// the same directory reads only the file, so nothing in memory can answer for it.
func TestSaveStateRoundTripsColumnOrderAndTypes(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir, "")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatalf("EnsureDefaultEnvironment: %v", err)
	}
	when := time.Date(2026, 3, 4, 5, 6, 7, 8, time.UTC)
	dt := insyra.NewDataTable(
		insyra.NewDataList(3.0, 1.5, math.NaN()).SetName("zeta"),
		insyra.NewDataList("x", nil, "z").SetName("alpha"),
		insyra.NewDataList(when, when.Add(time.Hour), when.Add(2*time.Hour)).SetName("when"),
	).SetName("T")
	dt.SetRowNames([]string{"a", "b", "c"})
	if err := mgr.SaveState("default", map[string]any{"dt": dt}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	vars, err := NewManager(dir, "").RestoreVariables("default")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}
	restored, ok := vars["dt"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("dt restored as %T want *insyra.DataTable", vars["dt"])
	}
	if restored.GetName() != "T" {
		t.Errorf("dt name = %q want %q", restored.GetName(), "T")
	}
	if names := restored.RowNames(); !reflect.DeepEqual(names, []string{"a", "b", "c"}) {
		t.Errorf("dt row names = %v want a, b, c", names)
	}
	if restored.NumCols() != 3 {
		t.Fatalf("dt has %d columns, want 3", restored.NumCols())
	}
	for i, want := range []string{"zeta", "alpha", "when"} {
		if got := restored.GetColByNumber(i).GetName(); got != want {
			t.Errorf("dt column %d = %q want %q", i, got, want)
		}
	}

	zeta := restored.GetColByNumber(0).Data()
	if got, ok := zeta[0].(float64); !ok || got != 3.0 {
		t.Errorf("zeta[0] = %#v want float64(3)", zeta[0])
	}
	if got, ok := zeta[1].(float64); !ok || got != 1.5 {
		t.Errorf("zeta[1] = %#v want float64(1.5)", zeta[1])
	}
	if f, ok := zeta[2].(float64); !ok || !math.IsNaN(f) {
		t.Errorf("zeta[2] = %#v want NaN", zeta[2])
	}
	alpha := restored.GetColByNumber(1).Data()
	if alpha[0] != "x" || alpha[1] != nil || alpha[2] != "z" {
		t.Errorf("alpha restored as %v want x, nil, z", alpha)
	}
	for i, cell := range restored.GetColByNumber(2).Data() {
		got, ok := cell.(time.Time)
		if !ok {
			t.Fatalf("when[%d] = %#v want time.Time", i, cell)
		}
		if want := when.Add(time.Duration(i) * time.Hour); !got.Equal(want) {
			t.Errorf("when[%d] = %v want %v", i, got, want)
		}
	}
}

// A variable the environment cannot hold is left out of the file and named in
// the returned list, while every other variable is written and restored.
func TestSaveVariablesReportsUnsavedVariables(t *testing.T) {
	mgr := newStateTestManager(t)
	bad := insyra.NewDataTable(insyra.NewDataList(nil, struct{ X int }{1}).SetName("payload"))
	vars := map[string]any{
		"r":   &stats.LinearRegressionResult{},
		"bad": bad,
		"x":   1.5,
	}

	unsaved, err := mgr.SaveVariables("default", vars)
	if err != nil {
		t.Fatalf("SaveVariables: %v", err)
	}
	if len(unsaved) != 2 {
		t.Fatalf("reported %d variables, want 2: %+v", len(unsaved), unsaved)
	}
	if unsaved[0].Name != "bad" || unsaved[1].Name != "r" {
		t.Errorf("reported %q then %q, want bad then r", unsaved[0].Name, unsaved[1].Name)
	}
	if got := unsaved[0].Type; got != "*insyra.DataTable" {
		t.Errorf("bad type = %q want %q", got, "*insyra.DataTable")
	}
	if got := unsaved[1].Type; got != "*stats.LinearRegressionResult" {
		t.Errorf("r type = %q want %q", got, "*stats.LinearRegressionResult")
	}
	for _, variable := range unsaved {
		if variable.Reason == "" {
			t.Errorf("%s reported with no reason", variable.Name)
		}
	}
	if reason := unsaved[0].Reason; !strings.Contains(reason, `column "payload"`) {
		t.Errorf("bad reason = %q, want it to name the column", reason)
	}

	restored, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}
	if len(restored) != 1 {
		t.Errorf("restored %d variables (%v), want only x", len(restored), restored)
	}
	if got, ok := restored["x"].(float64); !ok || got != 1.5 {
		t.Errorf("x = %#v want float64(1.5)", restored["x"])
	}
}

// SaveState writes the file even when a variable cannot be stored, and says
// nothing about it: the error is only for a file that was not written.
func TestSaveStateIgnoresUnsavedVariables(t *testing.T) {
	mgr := newStateTestManager(t)
	bad := insyra.NewDataTable(insyra.NewDataList(nil, struct{ X int }{1}).SetName("payload"))
	vars := map[string]any{
		"r":   &stats.LinearRegressionResult{},
		"bad": bad,
		"x":   1.5,
	}

	if err := mgr.SaveState("default", vars); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	restored, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}
	if len(restored) != 1 {
		t.Errorf("restored %d variables (%v), want only x", len(restored), restored)
	}
	if got, ok := restored["x"].(float64); !ok || got != 1.5 {
		t.Errorf("x = %#v want float64(1.5)", restored["x"])
	}
}

// A scaler holds its fitted state in unexported fields, so it used to be
// dropped on every save. It is stored now, and comes back with the parameters
// it was fitted with.
func TestSaveStateStoresScaler(t *testing.T) {
	mgr := newStateTestManager(t)
	dt := insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 6.0).SetName("a"))
	scaler := insyra.NewStandardScaler()
	if err := scaler.Fit(dt, "a"); err != nil {
		t.Fatalf("Fit: %v", err)
	}
	if err := mgr.SaveState("default", map[string]any{"sc": scaler}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	vars, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}
	restored, ok := vars["sc"].(*insyra.StandardScaler)
	if !ok {
		t.Fatalf("sc restored as %T want *insyra.StandardScaler", vars["sc"])
	}
	if got, want := restored.Params(), scaler.Params(); !reflect.DeepEqual(got, want) {
		t.Errorf("restored params = %s, want %s", fmt.Sprint(got), fmt.Sprint(want))
	}
}

// An environment holding a NaN scalar can be exported: nothing in the export
// path writes a float64 the encoder refuses.
func TestExportWithNaNScalar(t *testing.T) {
	mgr := newStateTestManager(t)
	if err := mgr.SaveState("default", map[string]any{"x": math.NaN()}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	out := filepath.Join(t.TempDir(), "export.json")
	if err := mgr.Export("default", out); err != nil {
		t.Fatalf("Export: %v", err)
	}
}

// An integer beyond 2^53 keeps its value through an export and an import.
func TestExportImportKeepsLargeInt(t *testing.T) {
	const big = int64(9007199254740993) // 2^53 + 1, unrepresentable as float64
	mgr := newStateTestManager(t)
	dt := insyra.NewDataTable(insyra.NewDataList(big, 2).SetName("id"))
	if err := mgr.SaveState("default", map[string]any{"n": big, "dt": dt}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	out := filepath.Join(t.TempDir(), "export.json")
	if err := mgr.Export("default", out); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if _, err := mgr.Import(out, "copy", false); err != nil {
		t.Fatalf("Import: %v", err)
	}

	vars, err := mgr.RestoreVariables("copy")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}
	if got, ok := vars["n"].(int64); !ok || got != big {
		t.Errorf("n = %#v want int64(%d)", vars["n"], big)
	}
	table, ok := vars["dt"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("dt restored as %T want *insyra.DataTable", vars["dt"])
	}
	if got, ok := table.GetColByName("id").Data()[0].(int64); !ok || got != big {
		t.Errorf("dt id[0] = %#v want int64(%d)", table.GetColByName("id").Data()[0], big)
	}
}

// LoadState types legacy Raw scalars: integer literal -> int64, fractional -> float64,
// special float marker -> the float64 it represents.
func TestLoadStateTypesLegacyScalars(t *testing.T) {
	mgr := newStateTestManager(t)
	envPath, err := mgr.ResolveEnvPath("default")
	if err != nil {
		t.Fatal(err)
	}
	stored := `{
  "variables": {
    "whole": {"type": "Raw", "data": 7},
    "frac": {"type": "Raw", "data": 1.25},
    "negInf": {"type": "Raw", "data": {"$float": "-Inf"}}
  },
  "lastAccess": "2026-01-01T00:00:00Z"
}
`
	if err := os.WriteFile(filepath.Join(envPath, "state.json"), []byte(stored), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := mgr.LoadState("default")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	if got, ok := state.Variables["whole"].Data.(int64); !ok || got != 7 {
		t.Errorf("whole = %#v want int64(7)", state.Variables["whole"].Data)
	}
	if got, ok := state.Variables["frac"].Data.(float64); !ok || got != 1.25 {
		t.Errorf("frac = %#v want float64(1.25)", state.Variables["frac"].Data)
	}
	if got, ok := state.Variables["negInf"].Data.(float64); !ok || got != math.Inf(-1) {
		t.Errorf("negInf = %#v want float64(-Inf)", state.Variables["negInf"].Data)
	}
}

// A variable the decoder cannot read is kept as an unreadableVariable rather
// than dropped, so nothing in the file is lost to a reader that cannot parse it.
// The next save writes it back with its original type and name intact.
func TestRestoreKeepsUndecodableVariable(t *testing.T) {
	mgr := newStateTestManager(t)
	envPath, err := mgr.ResolveEnvPath("default")
	if err != nil {
		t.Fatal(err)
	}
	stored := `{
  "variables": {
    "x": {"type": "table", "name": "T", "data": "garbage"},
    "y": {"type": "scalar", "data": {"type": "float64", "value": 2.5}}
  },
  "lastAccess": "2026-01-01T00:00:00Z"
}
`
	if err := os.WriteFile(filepath.Join(envPath, "state.json"), []byte(stored), 0o644); err != nil {
		t.Fatal(err)
	}
	vars, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}
	if got, ok := vars["y"].(float64); !ok || got != 2.5 {
		t.Errorf("y = %#v want float64(2.5)", vars["y"])
	}
	// x should be an unreadableVariable
	if _, ok := vars["x"].(unreadableVariable); !ok {
		t.Errorf("x = %T want unreadableVariable", vars["x"])
	}

	// SaveState the same map and verify x is written back with type and name
	if err := mgr.SaveState("default", vars); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	raw := readStateFile(t, mgr, "default")
	var file struct {
		Variables map[string]json.RawMessage `json:"variables"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("decoding state.json: %v", err)
	}
	var xVar SerializedVariable
	if err := json.Unmarshal(file.Variables["x"], &xVar); err != nil {
		t.Fatalf("decoding x: %v", err)
	}
	if xVar.Type != "table" {
		t.Errorf("x type = %q want %q", xVar.Type, "table")
	}
	if xVar.Name != "T" {
		t.Errorf("x name = %q want %q", xVar.Name, "T")
	}
	if xVar.Data != "garbage" {
		t.Errorf("x data = %#v want %q", xVar.Data, "garbage")
	}
}

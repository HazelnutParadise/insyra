package env

import (
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func newStateTestManager(t *testing.T) *Manager {
	t.Helper()
	mgr := NewManager(t.TempDir(), "")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatalf("EnsureDefaultEnvironment: %v", err)
	}
	return mgr
}

func saveAndRestore(t *testing.T, vars map[string]any) map[string]any {
	t.Helper()
	mgr := newStateTestManager(t)
	if err := mgr.SaveState("default", vars); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	restored, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}
	return restored
}

func TestState_FloatScalarRoundTrip(t *testing.T) {
	restored := saveAndRestore(t, map[string]any{"s": 1.25})
	value, ok := restored["s"].(float64)
	if !ok {
		t.Fatalf("s = %T want float64", restored["s"])
	}
	if value != 1.25 {
		t.Errorf("s = %v want 1.25", value)
	}
}

func TestState_IntScalarRoundTrip(t *testing.T) {
	restored := saveAndRestore(t, map[string]any{"n": int64(7)})
	value, ok := restored["n"].(int64)
	if !ok {
		t.Fatalf("n = %T want int64", restored["n"])
	}
	if value != 7 {
		t.Errorf("n = %v want 7", value)
	}
}

func TestState_LargeIntKeepsPrecision(t *testing.T) {
	const big = int64(9007199254740993) // 2^53 + 1, unrepresentable as float64
	restored := saveAndRestore(t, map[string]any{"n": big})
	if got, ok := restored["n"].(int64); !ok || got != big {
		t.Fatalf("n = %#v want int64(%d)", restored["n"], big)
	}
}

func TestState_StringAndBoolUnchanged(t *testing.T) {
	restored := saveAndRestore(t, map[string]any{"s": "hello", "b": true})
	if got, ok := restored["s"].(string); !ok || got != "hello" {
		t.Errorf("s = %#v want \"hello\"", restored["s"])
	}
	if got, ok := restored["b"].(bool); !ok || got != true {
		t.Errorf("b = %#v want true", restored["b"])
	}
}

// Every cell of a list comes back at the Go type it was saved with, so a
// command that type-asserts a cell after a restore sees what it wrote.
func TestState_DataListAndDataTableKeepCellTypes(t *testing.T) {
	dl := insyra.NewDataList(1, 2.5, "three")
	dl.SetName("dl")
	dt := insyra.NewDataTable(insyra.NewDataList(1, 2))

	restored := saveAndRestore(t, map[string]any{"dl": dl, "dt": dt})
	list, ok := restored["dl"].(*insyra.DataList)
	if !ok {
		t.Fatalf("dl = %T want *insyra.DataList", restored["dl"])
	}
	if got, ok := list.Get(0).(int); !ok || got != 1 {
		t.Errorf("dl[0] = %#v want int(1)", list.Get(0))
	}
	if got, ok := list.Get(1).(float64); !ok || got != 2.5 {
		t.Errorf("dl[1] = %#v want float64(2.5)", list.Get(1))
	}
	if got, ok := list.Get(2).(string); !ok || got != "three" {
		t.Errorf("dl[2] = %#v want string(\"three\")", list.Get(2))
	}
	if _, ok := restored["dt"].(*insyra.DataTable); !ok {
		t.Fatalf("dt = %T want *insyra.DataTable", restored["dt"])
	}
}

// LoadState types top-level scalars: one saved by this release comes back at
// the Go type it was saved with; one written by an earlier release comes back
// as int64 when it is an integer literal and float64 otherwise. Every other
// variable keeps the form it has in the file; RestoreVariables turns those
// into Go values.
func TestLoadState_TypesScalars(t *testing.T) {
	mgr := newStateTestManager(t)
	dt := insyra.NewDataTable(insyra.NewDataList(1, 2))
	if err := mgr.SaveState("default", map[string]any{"s": 1.25, "i": 7, "t": dt}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	state, err := mgr.LoadState("default")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	// s: float64 scalar
	sVar := state.Variables["s"]
	if sVar.Type != "scalar" {
		t.Errorf("s type = %q want %q", sVar.Type, "scalar")
	}
	if got, ok := sVar.Data.(float64); !ok || got != 1.25 {
		t.Errorf("s data = %#v want float64(1.25)", sVar.Data)
	}

	// i: int scalar (saved as int by Go)
	iVar := state.Variables["i"]
	if iVar.Type != "scalar" {
		t.Errorf("i type = %q want %q", iVar.Type, "scalar")
	}
	if got, ok := iVar.Data.(int); !ok || got != 7 {
		t.Errorf("i data = %#v want int(7)", iVar.Data)
	}

	// t: table variable keeps stored form
	tVar := state.Variables["t"]
	if tVar.Type != "table" {
		t.Errorf("t type = %q want %q", tVar.Type, "table")
	}
	if _, ok := tVar.Data.(map[string]any); !ok {
		t.Errorf("t data = %T want map[string]any", tVar.Data)
	}
}

package env

import (
	"encoding/json"
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

func TestState_DataListAndDataTableUnaffected(t *testing.T) {
	dl := insyra.NewDataList(1, 2.5, "three")
	dl.SetName("dl")
	dt := insyra.NewDataTable(insyra.NewDataList(1, 2))

	restored := saveAndRestore(t, map[string]any{"dl": dl, "dt": dt})
	list, ok := restored["dl"].(*insyra.DataList)
	if !ok {
		t.Fatalf("dl = %T want *insyra.DataList", restored["dl"])
	}
	if got := list.Get(0); got != int64(1) {
		t.Errorf("dl[0] = %#v want int64(1)", got)
	}
	if got := list.Get(1); got != 2.5 {
		t.Errorf("dl[1] = %#v want 2.5", got)
	}
	if _, ok := restored["dt"].(*insyra.DataTable); !ok {
		t.Fatalf("dt = %T want *insyra.DataTable", restored["dt"])
	}
}

// LoadState is where the coercion happens, so the raw State must already carry
// typed scalars for any caller that reads it directly.
func TestLoadState_CoercesScalarsBeforeReturning(t *testing.T) {
	mgr := newStateTestManager(t)
	if err := mgr.SaveState("default", map[string]any{"s": 1.25, "n": int64(7)}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	state, err := mgr.LoadState("default")
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if _, isNumber := state.Variables["s"].Data.(json.Number); isNumber {
		t.Error("s should not come back as json.Number")
	}
	if got, ok := state.Variables["s"].Data.(float64); !ok || got != 1.25 {
		t.Errorf("s = %#v want float64(1.25)", state.Variables["s"].Data)
	}
	if got, ok := state.Variables["n"].Data.(int64); !ok || got != 7 {
		t.Errorf("n = %#v want int64(7)", state.Variables["n"].Data)
	}
}

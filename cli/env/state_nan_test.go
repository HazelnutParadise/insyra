package env

import (
	"math"
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

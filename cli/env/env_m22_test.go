package env

import (
	"path/filepath"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// M22: env save/restore must preserve int64 — including large-integer precision
// beyond 2^53 — for DataList and DataTable variables, instead of collapsing them
// to float64 through JSON.
func TestSaveRestore_PreservesInt64(t *testing.T) {
	mgr := NewManager(filepath.Join(t.TempDir(), "ws"), "")
	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		t.Fatalf("EnsureDefaultEnvironment: %v", err)
	}

	dl := insyra.NewDataList(int64(9007199254740993), int64(2), 3.5).SetName("dl")
	dt := insyra.NewDataTable(
		insyra.NewDataList(int64(9007199254740993), int64(20)).SetName("id"))

	if err := mgr.SaveState("default", map[string]any{"dl": dl, "dt": dt}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	vars, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatalf("RestoreVariables: %v", err)
	}

	rdl, ok := vars["dl"].(*insyra.DataList)
	if !ok {
		t.Fatalf("dl restored as %T", vars["dl"])
	}
	if got := rdl.Data()[0]; got != int64(9007199254740993) {
		t.Fatalf("dl[0] = %v (%T), want int64(9007199254740993)", got, got)
	}
	if got := rdl.Data()[2]; got != 3.5 {
		t.Fatalf("dl[2] = %v (%T), want float64(3.5)", got, got)
	}

	rdt, ok := vars["dt"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("dt restored as %T", vars["dt"])
	}
	if got := rdt.GetColByName("id").Data()[0]; got != int64(9007199254740993) {
		t.Fatalf("dt id[0] = %v (%T), want int64(9007199254740993)", got, got)
	}
}

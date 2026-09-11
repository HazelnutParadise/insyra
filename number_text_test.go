package insyra

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A revenue in the millions is an ordinary amount in New Taiwan dollars, and
// Go's %v wrote it as "1.5e+06". ToJSON never did: it goes through
// encoding/json, whose rule every text output now follows.
func TestNumberTextInCCL(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	for _, tc := range []struct {
		expr string
		want []any
	}{
		{"'營收：' & A * B", []any{"營收：1500000", "營收：1500000", "營收：1900"}},
		{"TOSTR(A * B)", []any{"1500000", "1500000", "1900"}},
		{"LEN(A * B)", []any{7.0, 7.0, 4.0}},
		{"LEN(1000000)", []any{7.0, 7.0, 7.0}},     // a literal is a float64
		{"'x' & 0.00001", []any{"x0.00001", "x0.00001", "x0.00001"}},
		{"'x' & 0.0000001", []any{"x1e-07", "x1e-07", "x1e-07"}}, // unchanged
	} {
		dt := NewDataTable(
			NewDataList(1500, 25000, 380).SetName("price"),
			NewDataList(1000, 60, 5).SetName("qty"),
		)
		dt.AddColUsingCCL("r", tc.expr)
		if err := dt.PopErr(); err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		if got := dt.GetColByName("r").Data(); !slices.Equal(got, tc.want) {
			t.Errorf("%s = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// The file changes; the numbers do not. Every value read back must be the
// same float64, bit for bit.
func TestNumberTextInCSVRoundTrips(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	values := []any{1500000.0, 0.00001, 12345.678, 1e-7, 1e21, -2500000.0}
	dt := NewDataTable(NewDataList(values...).SetName("v"))
	path := filepath.Join(t.TempDir(), "n.csv")
	if err := dt.ToCSV(path, false, true, false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const want = "v\n1500000\n0.00001\n12345.678\n1e-07\n1e+21\n-2500000\n"
	if string(raw) != want {
		t.Errorf("CSV =\n%s\nwant\n%s", raw, want)
	}

	back, err := ReadCSV_File(path, false, true)
	if err != nil {
		t.Fatal(err)
	}
	got := back.GetColByName("v").Data()
	for i, orig := range values {
		f, ok := got[i].(float64)
		if !ok {
			t.Fatalf("row %d read back as %T, want float64", i, got[i])
		}
		if math.Float64bits(f) != math.Float64bits(orig.(float64)) {
			t.Errorf("row %d: read back %v, wrote %v", i, f, orig)
		}
	}
}

// ToStringSlice's whole job is turning values into text.
func TestNumberTextInToStringSlice(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	got := NewDataList(1500000.0, 0.00001, 42, "x", 1e-7).ToStringSlice()
	want := []string{"1500000", "0.00001", "42", "x", "1e-07"}
	if !slices.Equal(got, want) {
		t.Errorf("ToStringSlice = %v, want %v", got, want)
	}
}

// Column names generated from values follow the same rule.
func TestNumberTextInGeneratedColumnNames(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	dt := NewDataTable(NewDataList(1500000.0, 2500000.0, 1500000.0).SetName("price"))
	out, _, err := dt.OneHotEncode(OneHotOptions{Columns: []string{"price"}})
	if err != nil {
		t.Fatal(err)
	}
	names := out.ColNames()
	for _, want := range []string{"price_1500000", "price_2500000"} {
		if !slices.Contains(names, want) {
			t.Errorf("one-hot columns = %v, want %q among them", names, want)
		}
	}

	pv := NewDataTable(
		NewDataList("a", "b").SetName("id"),
		NewDataList(1500000.0, 2500000.0).SetName("k"),
		NewDataList(1, 2).SetName("v"),
	)
	pivoted, err := pv.Pivot(PivotConfig{Index: []string{"id"}, Columns: "k", Values: "v"})
	if err != nil {
		t.Fatal(err)
	}
	pnames := pivoted.ColNames()
	for _, want := range []string{"1500000", "2500000"} {
		if !slices.Contains(pnames, want) {
			t.Errorf("pivot columns = %v, want %q among them", pnames, want)
		}
	}
}

// nil is not a number and keeps its documented name.
func TestNilCategoryNameUnchanged(t *testing.T) {
	if got := oneHotCategoryColumnName("c", "", nil); got != "c_<nil>" {
		t.Errorf("one-hot nil name = %q, want %q", got, "c_<nil>")
	}
	if got := pivotColLabel(nil); got != "<nil>" {
		t.Errorf("pivot nil label = %q, want %q", got, "<nil>")
	}
}

package py

import (
	"context"
	"math"
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// TestReturnSendsMissingValuesAsNil runs the real environment, so it builds
// it on first use and runs only with INSYRA_PY_E2E=1. polars and pandas sent
// a DataFrame through NumPy, which has no missing integer: a polars null came
// back as NaN, a polars list column and a pandas pd.NA made insyra.Return
// raise, and a struct column came back as floats (#415).
func TestReturnSendsMissingValuesAsNil(t *testing.T) {
	if os.Getenv("INSYRA_PY_E2E") != "1" {
		t.Skip("set INSYRA_PY_E2E=1 to run Python in the real environment")
	}
	ctx := context.Background()

	plNulls, err := Run[*insyra.DataTable](ctx, `insyra.Return(pl.DataFrame({"id": [7, None], "x": [1.5, float("nan")]}))`)
	if err != nil {
		t.Fatal(err)
	}
	if id, null := plNulls.GetElementByNumberIndex(0, 0), plNulls.GetElementByNumberIndex(1, 0); id != any(7.0) || null != nil {
		t.Errorf("a polars integer column with a null came back as %#v and %#v", id, null)
	}
	if x := plNulls.GetElementByNumberIndex(1, 1); !math.IsNaN(asFloat(x)) {
		t.Errorf("a polars NaN came back as %#v, want NaN", x)
	}

	plNested, err := Run[*insyra.DataTable](ctx, `insyra.Return(pl.DataFrame({"l": [[1, 2], None], "s": [{"x": 1}, None]}))`)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := plNested.GetElementByNumberIndex(0, 0).([]any)
	record, _ := plNested.GetElementByNumberIndex(0, 1).(map[string]any)
	if len(list) != 2 || list[0] != any(1.0) || list[1] != any(2.0) || record["x"] != any(1.0) || plNested.GetElementByNumberIndex(1, 0) != nil || plNested.GetElementByNumberIndex(1, 1) != nil {
		t.Errorf("polars list and struct columns came back as %#v and %#v", plNested.GetElementByNumberIndex(0, 0), plNested.GetElementByNumberIndex(0, 1))
	}

	pdNulls, err := Run[*insyra.DataTable](ctx, `insyra.Return(pd.DataFrame({"id": pd.Series([7, None], dtype="Int64"), "b": pd.Series([True, None], dtype="boolean"), "s": pd.Series(["a", None], dtype="string"), "f": [0.5, float("nan")]}))`)
	if err != nil {
		t.Fatal(err)
	}
	if id, b, s := pdNulls.GetElementByNumberIndex(0, 0), pdNulls.GetElementByNumberIndex(0, 1), pdNulls.GetElementByNumberIndex(0, 2); id != any(7.0) || b != any(true) || s != any("a") {
		t.Errorf("pandas nullable columns came back as %#v, %#v and %#v", id, b, s)
	}
	for c := range 3 {
		if v := pdNulls.GetElementByNumberIndex(1, c); v != nil {
			t.Errorf("pd.NA in column %d came back as %#v, want nil", c, v)
		}
	}
	if f := pdNulls.GetElementByNumberIndex(1, 3); !math.IsNaN(asFloat(f)) {
		t.Errorf("the NaN of a NumPy-backed pandas column came back as %#v, want NaN", f)
	}

	pdSeries, err := Run[*insyra.DataList](ctx, `insyra.Return(pd.Series([7, None], dtype="Int64"))`)
	if err != nil {
		t.Fatal(err)
	}
	if pdSeries.Len() != 2 || pdSeries.Get(0) != any(7.0) || pdSeries.Get(1) != nil {
		t.Errorf("a pandas Int64 Series came back as %v", pdSeries.Data())
	}
}

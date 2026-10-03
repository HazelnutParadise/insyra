package py

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// The review of py-ipc-server-errors measured a DataFrame with a missing value
// coming back as nil with no error.
func TestANaNInAResultComesBackAsNaN(t *testing.T) {
	useFakePython(t, `pyjson:{"_insyra_type": "datatable", "data": [[1.0, NaN], [Infinity, -Infinity]], "columns": ["a", "b"], "index": ["0", "1"]}`)
	dt, err := Run[*insyra.DataTable](context.Background(), "insyra.Return(df)")
	if err != nil {
		t.Fatal(err)
	}
	if dt == nil {
		t.Fatal("the table came back nil")
	}
	if !math.IsNaN(asFloat(dt.GetElementByNumberIndex(0, 1))) || !math.IsInf(asFloat(dt.GetElementByNumberIndex(1, 0)), 1) || !math.IsInf(asFloat(dt.GetElementByNumberIndex(1, 1)), -1) {
		t.Errorf("the cells are %v", dt.To2DSlice())
	}

	useFakePython(t, `pyjson:{"x": [1.5, NaN]}`)
	got, err := Run[map[string][]float64](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	if x := got["x"]; len(x) != 2 || x[0] != 1.5 || !math.IsNaN(x[1]) {
		t.Errorf("got %v", got)
	}
}

// go-json refuses an integer too large for a float64 too, and that result also
// came back as nil.
func TestAResultGoCannotReadIsAnError(t *testing.T) {
	useFakePython(t, "pyjson:"+strings.Repeat("9", 400))
	got, err := Run[any](context.Background(), "insyra.Return(10**400)")
	if err == nil || !strings.Contains(err.Error(), "float64") {
		t.Errorf("got %v, %v; want an error saying the number does not fit in a float64", got, err)
	}
}

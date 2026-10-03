package py

import (
	"context"
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// bigID is 2^53 + 1, which a float64 rounds to 2^53.
const bigID = 9007199254740993

// The review of py-nested-table-results measured 2^53 + 1 coming back as 2^53:
// every number was decoded into a float64, and an any kept that float64.
func TestALargeIntegerBindsExactly(t *testing.T) {
	var n int64
	if err := bindPyResult(&n, int64(bigID)); err != nil || n != bigID {
		t.Errorf("int64: got %d, %v", n, err)
	}
	var u uint64
	if err := bindPyResult(&u, uint64(math.MaxUint64)); err != nil || u != math.MaxUint64 {
		t.Errorf("uint64: got %d, %v", u, err)
	}
	var v any
	if err := bindPyResult(&v, int64(bigID)); err != nil || v != any(int64(bigID)) {
		t.Errorf("any: got %#v, %v", v, err)
	}
	var m map[string]any
	if err := bindPyResult(&m, map[string]any{"id": int64(bigID), "n": 3.0}); err != nil || m["id"] != any(int64(bigID)) || m["n"] != any(3.0) {
		t.Errorf("map[string]any: got %#v, %v", m, err)
	}
	type record struct {
		ID   int64
		Any  any
		List []any
	}
	var r record
	if err := bindPyResult(&r, map[string]any{"ID": int64(bigID), "Any": int64(bigID), "List": []any{int64(bigID), 1.5}}); err != nil {
		t.Fatal(err)
	}
	if r.ID != bigID || r.Any != any(int64(bigID)) || len(r.List) != 2 || r.List[0] != any(int64(bigID)) || r.List[1] != any(1.5) {
		t.Errorf("struct: got %#v", r)
	}
	var f float64
	if err := bindPyResult(&f, int64(bigID)); err != nil || f != float64(bigID) {
		t.Errorf("float64: got %v, %v", f, err)
	}
}

func TestALargeIntegerInATableKeepsEveryDigit(t *testing.T) {
	payload := map[string]any{"_insyra_type": "datatable", "data": []any{[]any{int64(bigID)}, []any{int64(math.MaxInt64)}}, "columns": []any{"id"}}
	var dt *insyra.DataTable
	if err := bindPyResult(&dt, payload); err != nil {
		t.Fatal(err)
	}
	if a, b := dt.GetElementByNumberIndex(0, 0), dt.GetElementByNumberIndex(1, 0); a != any(int64(bigID)) || b != any(int64(math.MaxInt64)) {
		t.Errorf("the cells are %#v and %#v", a, b)
	}
}

func TestALargeIntegerArrivesThroughTheRunner(t *testing.T) {
	useFakePython(t, `pyjson:{"id": 9007199254740993, "n": 3, "x": 0.5}`)
	got, err := Run[map[string]any](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	if got["id"] != any(int64(bigID)) || got["n"] != any(3.0) || got["x"] != any(0.5) {
		t.Errorf("got %#v", got)
	}

	useFakePython(t, `pyjson:9007199254740993`)
	if n, err := Run[int64](context.Background(), "insyra.Return(x)"); err != nil || n != bigID {
		t.Errorf("int64: got %d, %v", n, err)
	}
}

package ccl

import (
	"reflect"
	"strings"
	"testing"
)

// rowSliceCtx is a two-column table: A = [10 20 30], B = [1 0 3].
func rowSliceCtx(t *testing.T) *MapContext {
	t.Helper()
	return mapCtx(t, map[string][]any{
		"a": {10.0, 20.0, 30.0},
		"b": {1.0, 0.0, 3.0},
	})
}

// A range and '@' stand for a row, so an expression that is one of them holds
// that row's values. `A:B` used to fill every cell with an internal
// ColumnRange struct, and `LAG(@, 1)` with the whole flattened table.
func TestRowShapedValues(t *testing.T) {
	ctx := rowSliceCtx(t)

	got, err := evalCol(t, ctx, "A:B")
	if err != nil {
		t.Fatalf("A:B: %v", err)
	}
	want := [][]any{{10.0, 1.0}, {20.0, 0.0}, {30.0, 3.0}}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Fatalf("A:B = %v, want %v", got, want)
		}
	}

	viaAt, err := evalCol(t, ctx, "LAG(@, 1)")
	if err != nil {
		t.Fatalf("LAG(@, 1): %v", err)
	}
	viaExplicit, err := evalCol(t, ctx, "LAG(@.#, 1)")
	if err != nil {
		t.Fatalf("LAG(@.#, 1): %v", err)
	}
	if !reflect.DeepEqual(viaAt, viaExplicit) {
		t.Errorf("LAG(@, 1) = %v, want the same as LAG(@.#, 1) = %v", viaAt, viaExplicit)
	}
	if viaAt[0] != nil {
		t.Errorf("LAG(@, 1): first row = %v, want nil", viaAt[0])
	}

	shifted, err := evalCol(t, ctx, "LAG(A:B, 1)")
	if err != nil {
		t.Fatalf("LAG(A:B, 1): %v", err)
	}
	if shifted[0] != nil || !reflect.DeepEqual(shifted[1], []any{10.0, 1.0}) {
		t.Errorf("LAG(A:B, 1) = %v, want nil then [10 1]", shifted)
	}

	led, err := evalCol(t, ctx, "LEAD(A:B, 1)")
	if err != nil {
		t.Fatalf("LEAD(A:B, 1): %v", err)
	}
	if !reflect.DeepEqual(led[0], []any{20.0, 0.0}) || led[2] != nil {
		t.Errorf("LEAD(A:B, 1) = %v, want [20 0] first and nil last", led)
	}
}

// A row range names rows but not of what, and a sequence function that does
// arithmetic cannot take a row.
func TestRowShapedValuesThatStayErrors(t *testing.T) {
	ctx := rowSliceCtx(t)
	for _, expr := range []string{"1:2", "CUMSUM(@)", "DIFF(@)", "ROLLING_SUM(A:B, 2)"} {
		if got, err := evalCol(t, ctx, expr); err == nil {
			t.Errorf("%s = %v; want an error", expr, got)
		}
	}
	if _, err := evalCol(t, ctx, "1:2"); err == nil || !strings.Contains(err.Error(), "attach it to a column") {
		t.Errorf("a bare row range should say how to write it: %v", err)
	}
}

// The readings that consume a range keep the values they always had: a range
// inside an aggregate is every cell in those columns, and a row range attached
// to a column is that slice of it, evaluated once.
func TestRangeConsumersUnchanged(t *testing.T) {
	ctx := rowSliceCtx(t)
	for _, tc := range []struct {
		expr string
		want float64
	}{
		{"SUM(A:B)", 64},
		{"SUM(A.(0:1))", 30},
		{"AVG(A:B)", 64.0 / 6},
		{"MAX(A:B)", 30},
	} {
		got, err := evalCol(t, ctx, tc.expr)
		if err != nil {
			t.Errorf("%s: %v", tc.expr, err)
			continue
		}
		for _, v := range got {
			if v != tc.want {
				t.Errorf("%s = %v, want %v in every row", tc.expr, got, tc.want)
				break
			}
		}
	}
}

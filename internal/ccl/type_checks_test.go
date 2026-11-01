package ccl

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestTypeChecks(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {1.5},
		"B": {"word"},
		"C": {true},
		"D": {time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
		"E": {nil},
		"F": {""},
		"G": {math.NaN()},
		"H": {"12"},
		"I": {"2024-01-02"},
		"J": {"#N/A"},
		"K": {math.Inf(1)},
		"L": {int64(7)},
		"M": {time.Hour},
		"N": {" "},
	})
	type row struct{ blank, num, text, boolean, date bool }
	for _, c := range []struct {
		col  string
		want row
	}{
		{"A", row{num: true}},
		{"B", row{text: true}},
		{"C", row{boolean: true}},
		{"D", row{date: true}},
		{"E", row{blank: true}},
		{"F", row{blank: true, text: true}},
		{"G", row{}},
		{"H", row{text: true}},
		{"I", row{text: true, date: true}},
		{"J", row{text: true}},
		{"K", row{num: true}},
		{"L", row{num: true}},
		{"M", row{}},
		{"N", row{text: true}},
	} {
		for name, want := range map[string]bool{
			"ISBLANK": c.want.blank, "ISNUM": c.want.num, "ISTEXT": c.want.text,
			"ISBOOL": c.want.boolean, "ISDATE": c.want.date,
		} {
			expr := name + "(" + c.col + ")"
			got, err := evalOne(t, ctx, expr)
			if err != nil {
				t.Errorf("%s: %v", expr, err)
				continue
			}
			if got != want {
				t.Errorf("%s = %v, want %v", expr, got, want)
			}
		}
	}
	for _, name := range []string{"ISBLANK", "ISNUM", "ISTEXT", "ISBOOL", "ISDATE", "ISERROR"} {
		for _, args := range []string{"", "A, B"} {
			expr := name + "(" + args + ")"
			got, err := evalOne(t, ctx, expr)
			if err == nil || !strings.Contains(err.Error(), "requires 1 argument") {
				t.Errorf("%s = %#v, %v; want an argument error", expr, got, err)
			}
		}
	}
}

func TestIsError(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {6.0}, "B": {0.0}, "C": {"word"}, "D": {nil}, "E": {math.NaN()}})
	for _, c := range []struct {
		expr string
		want bool
	}{
		{"ISERROR(A / B)", true},
		{"ISERROR(C + 1)", true},
		{"ISERROR(SQRT(-1))", true},
		{"ISERROR(A / 2)", false},
		{"ISERROR(D)", false},
		{"ISERROR(E)", false},
		{"IF(ISERROR(A / B), TRUE, FALSE)", true},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %v, want %v", c.expr, got, c.want)
		}
	}
	for _, expr := range []string{"ISERROR(NOSUCHFN(A))", "ISERROR(['nope'])"} {
		if got, err := evalOne(t, ctx, expr); err == nil {
			t.Errorf("%s = %v, want the mistake in the formula reported", expr, got)
		}
	}
}

// A function the caller registered under these names is called, as it was before
// CCL had them.
func TestRegisteredIfErrorAndIsErrorAreCalled(t *testing.T) {
	RegisterFunction("IFERROR", func(args ...any) (any, error) { return "mine", nil })
	RegisterFunction("ISERROR", func(args ...any) (any, error) { return "mine", nil })
	t.Cleanup(func() {
		registryMu.Lock()
		delete(defaultFunctions, "IFERROR")
		delete(defaultFunctions, "ISERROR")
		delete(userFunctions, "IFERROR")
		delete(userFunctions, "ISERROR")
		registryMu.Unlock()
	})
	ctx := mapCtx(t, map[string][]any{"A": {1.0}})
	for _, expr := range []string{"IFERROR(A, 0)", "ISERROR(A)"} {
		got, err := evalOne(t, ctx, expr)
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if got != "mine" {
			t.Errorf("%s = %#v, want the registered function's answer", expr, got)
		}
	}
}

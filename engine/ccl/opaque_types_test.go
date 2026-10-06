package ccl_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/HazelnutParadise/insyra/engine/ccl"
)

// #259: Context is how a program applies CCL to its own data, so its method set
// is a promise. A new capability comes as an optional interface, the way
// GlobalRowContext does; changing this list breaks every implementation outside
// the module.
func TestContextMethodSetIsFixed(t *testing.T) {
	want := []string{
		"GetAllData", "GetCell", "GetCellByName", "GetCol", "GetColByName",
		"GetColCount", "GetColData", "GetColDataByName", "GetColIndexByName",
		"GetCurrentRow", "GetRowAt", "GetRowCount", "GetRowIndex",
		"GetRowIndexByName", "SetRowIndex",
	}
	ctxType := reflect.TypeOf((*ccl.Context)(nil)).Elem()
	got := make([]string, 0, ctxType.NumMethod())
	for i := 0; i < ctxType.NumMethod(); i++ {
		got = append(got, ctxType.Method(i).Name)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("Context's methods are %v, want %v", got, want)
	}
}

// #259: CCLNode only holds what the compiler produced, so passing a string or
// a number to Evaluate does not compile; the zero value is refused with an
// error rather than a panic.
func TestCCLNodeIsOpaque(t *testing.T) {
	nodeType := reflect.TypeOf(ccl.CCLNode{})
	if nodeType.Kind() != reflect.Struct {
		t.Fatalf("CCLNode is a %s, want an opaque struct", nodeType.Kind())
	}
	for i := 0; i < nodeType.NumField(); i++ {
		if nodeType.Field(i).IsExported() {
			t.Errorf("CCLNode exports the field %s", nodeType.Field(i).Name)
		}
	}

	ctx := mapContext(t)
	if _, err := ccl.Evaluate(ccl.CCLNode{}, ctx); err == nil {
		t.Error("Evaluate accepted the zero CCLNode")
	}
	if _, err := ccl.EvaluateStatement(ccl.CCLNode{}, ctx); err == nil {
		t.Error("EvaluateStatement accepted the zero CCLNode")
	}
	node, err := ccl.CompileExpression("A + B")
	if err != nil {
		t.Fatal(err)
	}
	if node == (ccl.CCLNode{}) {
		t.Error("CompileExpression returned the zero CCLNode")
	}
}

// #259: MapContext is built by NewMapContext and moved with SetRowIndex, which
// check what they are given; fields a caller could set behind them made an
// out-of-range row or an added column read as nil with no error.
func TestMapContextExportsNoFields(t *testing.T) {
	mapType := reflect.TypeOf(ccl.MapContext{})
	for i := 0; i < mapType.NumField(); i++ {
		if mapType.Field(i).IsExported() {
			t.Errorf("MapContext exports the field %s", mapType.Field(i).Name)
		}
	}

	ctx, err := ccl.NewMapContext(map[string][]any{"A": {1, 2, 3}, "B": {10, 20, 30}})
	if err != nil {
		t.Fatal(err)
	}
	var _ ccl.Context = ctx
	if ctx.GetRowCount() != 3 || ctx.GetColCount() != 2 {
		t.Errorf("rows %d, columns %d; want 3 and 2", ctx.GetRowCount(), ctx.GetColCount())
	}
	if err := ctx.SetRowIndex(99); err == nil {
		t.Error("SetRowIndex(99) on three rows succeeded")
	}
	if err := ctx.SetRowIndex(2); err != nil {
		t.Fatal(err)
	}
	node, err := ccl.CompileExpression("A + B")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := ccl.Evaluate(node, ctx); err != nil || got != int64(33) {
		t.Errorf("A + B on row 2: %v, %v; want 33", got, err)
	}
	if got, err := ctx.GetColByName("B"); err != nil || got != 30 {
		t.Errorf("GetColByName(B) on row 2: %v, %v; want 30", got, err)
	}
}

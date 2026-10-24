package py

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// JSON cannot carry a NaN or an infinity, so binding a result holding one
// through JSON failed. Each kind of type that can hold one is checked here.
func TestANaNBindsIntoFloatsAndAny(t *testing.T) {
	var f float64
	if err := bindPyResult(&f, math.NaN()); err != nil || !math.IsNaN(f) {
		t.Errorf("float64: got %v, %v", f, err)
	}
	var p *float64
	if err := bindPyResult(&p, math.Inf(1)); err != nil || p == nil || !math.IsInf(*p, 1) {
		t.Errorf("*float64: got %v, %v", p, err)
	}
	var v any
	if err := bindPyResult(&v, map[string]any{"x": []any{1.0, math.NaN()}}); err != nil {
		t.Fatal(err)
	}
	m, _ := v.(map[string]any)
	x, _ := m["x"].([]any)
	if len(x) != 2 || x[0] != 1.0 || !math.IsNaN(asFloat(x[1])) {
		t.Errorf("any: got %#v", v)
	}
	var s []float64
	if err := bindPyResult(&s, []any{math.Inf(1), 2.0, math.Inf(-1)}); err != nil || len(s) != 3 || !math.IsInf(s[0], 1) || s[1] != 2 || !math.IsInf(s[2], -1) {
		t.Errorf("[]float64: got %v, %v", s, err)
	}
	var arr [2]float32
	if err := bindPyResult(&arr, []any{math.NaN(), math.Inf(1)}); err != nil || !math.IsNaN(float64(arr[0])) || !math.IsInf(float64(arr[1]), 1) {
		t.Errorf("[2]float32: got %v, %v", arr, err)
	}
	var byName map[string]float64
	if err := bindPyResult(&byName, map[string]any{"a": math.NaN(), "b": 1.0}); err != nil || !math.IsNaN(byName["a"]) || byName["b"] != 1 {
		t.Errorf("map[string]float64: got %v, %v", byName, err)
	}
}

func TestANaNBindsIntoAStructField(t *testing.T) {
	type point struct {
		X    float64
		Y    []float64 `json:"y"`
		Name string
		N    int
	}
	var got point
	err := bindPyResult(&got, map[string]any{"X": math.NaN(), "y": []any{math.Inf(-1), 0.5}, "Name": "a", "N": 3.0})
	if err != nil {
		t.Fatal(err)
	}
	if !math.IsNaN(got.X) || len(got.Y) != 2 || !math.IsInf(got.Y[0], -1) || got.Y[1] != 0.5 || got.Name != "a" || got.N != 3 {
		t.Errorf("got %+v", got)
	}

	type withTable struct {
		Table *insyra.DataTable `json:"table"`
		Score float64           `json:"score"`
	}
	var w withTable
	payload := map[string]any{"_insyra_type": "datatable", "data": []any{[]any{1.0, math.NaN()}}, "columns": []any{"a", "b"}}
	if err := bindPyResult(&w, map[string]any{"table": payload, "score": math.Inf(1)}); err != nil {
		t.Fatal(err)
	}
	if !math.IsInf(w.Score, 1) || w.Table == nil {
		t.Fatalf("got %+v", w)
	}
	if cell := w.Table.GetElementByNumberIndex(0, 1); !math.IsNaN(asFloat(cell)) {
		t.Errorf("the table's NaN cell is %v", cell)
	}
}

// A type that cannot hold a NaN gets an error naming it, not a zero.
func TestANaNForATypeThatCannotHoldOneIsAnError(t *testing.T) {
	var n int
	if err := bindPyResult(&n, math.NaN()); err == nil || !strings.Contains(err.Error(), "NaN") {
		t.Errorf("int: got %v, %v", n, err)
	}
	type counted struct{ N int }
	var c counted
	if err := bindPyResult(&c, map[string]any{"N": math.Inf(1)}); err == nil || !strings.Contains(err.Error(), "Inf") {
		t.Errorf("struct with an int: got %+v, %v", c, err)
	}
	var s string
	if err := bindPyResult(&s, math.NaN()); err == nil {
		t.Errorf("string: got %q, nil", s)
	}
	var self selfDecoding
	if err := bindPyResult(&self, map[string]any{"x": math.NaN()}); err == nil {
		t.Error("a type with its own UnmarshalJSON got a NaN without an error")
	}
}

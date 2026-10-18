package insyra_test

import (
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestToFloat64(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
	}{
		{42, 42},
		{3.14, 3.14},
		{"not a number", 0},
	}

	for _, test := range tests {
		result := insyra.ToFloat64(test.input)
		if result != test.expected {
			t.Errorf("ToFloat64(%v) = %v; expected %v", test.input, result, test.expected)
		}
	}
}

func TestToFloat64Safe(t *testing.T) {
	tests := []struct {
		input    any
		expected float64
		expectOk bool
	}{
		{42, 42, false},
		{3.14, 3.14, false},
		{"not a number", 0, true},
	}

	for _, test := range tests {
		result, ok := insyra.ToFloat64Safe(test.input)
		if result != test.expected || ok != !test.expectOk {
			t.Errorf("ToFloat64Safe(%v) = %v, %v; expected %v, %v",
				test.input, result, ok, test.expected, test.expectOk)
		}
	}
}

func TestSliceToF64(t *testing.T) {
	tests := []struct {
		input    []any
		expected []float64
	}{
		{[]any{42, 3.14, "not a number"}, []float64{42, 3.14, 0}},
	}

	for _, test := range tests {
		result := insyra.SliceToF64(test.input)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("SliceToF64(%v) = %v; expected %v", test.input, result, test.expected)
		}
	}
}

func TestProcessData(t *testing.T) {
	arr := [2]int{4, 5}
	slice := []float64{1.5, 2.5}
	cases := []struct {
		name  string
		input any
		want  []any
	}{
		{"DataList", insyra.NewDataList(1, 2, 3), []any{1, 2, 3}},
		{"slice", []float64{1.5, 2.5}, []any{1.5, 2.5}},
		{"array", arr, []any{4, 5}},
		{"pointer to slice", &slice, []any{1.5, 2.5}},
		{"pointer to array", &arr, []any{4, 5}},
		{"empty slice", []int{}, []any{}},
	}
	for _, c := range cases {
		got, err := insyra.ProcessData(c.input)
		if err != nil {
			t.Errorf("%s: ProcessData returned error %v", c.name, err)
			continue
		}
		if got == nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: ProcessData = %#v, want %#v", c.name, got, c.want)
		}
	}
}

// ProcessData reports what it cannot read as an error. It used to return
// (nil, 0), which a caller could not tell from an empty slice, and it crashed
// on a nil *DataList.
func TestProcessDataRefusesWhatItCannotRead(t *testing.T) {
	var nilList *insyra.DataList
	var nilSlice *[]int
	for _, input := range []any{nil, 42, "text", map[string]int{"a": 1}, nilList, nilSlice} {
		got, err := insyra.ProcessData(input)
		if err == nil {
			t.Errorf("ProcessData(%#v) = %#v with no error", input, got)
		}
		if got != nil {
			t.Errorf("ProcessData(%#v) returned data %#v alongside its error", input, got)
		}
	}
}

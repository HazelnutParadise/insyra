package insyra_test

import (
	"math/big"
	"reflect"
	"testing"
	"time"

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

// SqrtRat never panics: a negative or nil input gives nil. SqrtRat(-1) used to
// panic inside math/big.
func TestSqrtRatNegativeReturnsNil(t *testing.T) {
	if got := insyra.SqrtRat(big.NewRat(-1, 1)); got != nil {
		t.Errorf("SqrtRat(-1) = %v, want nil", got)
	}
	if got := insyra.SqrtRat(nil); got != nil {
		t.Errorf("SqrtRat(nil) = %v, want nil", got)
	}
	if got := insyra.SqrtRat(big.NewRat(9, 4)); got == nil || got.Cmp(big.NewRat(3, 2)) != 0 {
		t.Errorf("SqrtRat(9/4) = %v, want 3/2", got)
	}
	if got := insyra.SqrtRat(new(big.Rat)); got == nil || got.Sign() != 0 {
		t.Errorf("SqrtRat(0) = %v, want 0", got)
	}
}

// PowRat raises to a negative exponent through the reciprocal. PowRat(b, -2)
// used to return 1, because the loop over the exponent ran zero times.
func TestPowRatNegativeExponent(t *testing.T) {
	cases := []struct {
		base     *big.Rat
		exponent int
		want     *big.Rat
	}{
		{big.NewRat(2, 3), -2, big.NewRat(9, 4)},
		{big.NewRat(-2, 1), -3, big.NewRat(-1, 8)},
		{big.NewRat(-2, 1), 3, big.NewRat(-8, 1)},
		{big.NewRat(3, 5), 2, big.NewRat(9, 25)},
		{big.NewRat(7, 1), 0, big.NewRat(1, 1)},
		{new(big.Rat), 0, big.NewRat(1, 1)},
		{new(big.Rat), 3, new(big.Rat)},
	}
	for _, c := range cases {
		got := insyra.PowRat(c.base, c.exponent)
		if got == nil || got.Cmp(c.want) != 0 {
			t.Errorf("PowRat(%v, %d) = %v, want %v", c.base, c.exponent, got, c.want)
		}
	}
	if got := insyra.PowRat(new(big.Rat), -1); got != nil {
		t.Errorf("PowRat(0, -1) = %v, want nil: zero has no reciprocal", got)
	}
	if got := insyra.PowRat(nil, 2); got != nil {
		t.Errorf("PowRat(nil, 2) = %v, want nil", got)
	}
	base := big.NewRat(2, 1)
	insyra.PowRat(base, 5).SetInt64(0)
	if base.Cmp(big.NewRat(2, 1)) != 0 {
		t.Errorf("changing PowRat's result changed its base to %v", base)
	}
}

func TestSortTimes(t *testing.T) {
	tests := []struct {
		input    []time.Time
		expected []time.Time
	}{
		{
			input:    []time.Time{time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC), time.Date(2020, 1, 2, 15, 4, 5, 0, time.UTC)},
			expected: []time.Time{time.Date(2020, 1, 2, 15, 4, 5, 0, time.UTC), time.Date(2021, 1, 2, 15, 4, 5, 0, time.UTC)},
		},
	}

	for _, test := range tests {
		original := make([]time.Time, len(test.input))
		copy(original, test.input)
		insyra.SortTimes(test.input)
		if !reflect.DeepEqual(test.input, test.expected) {
			t.Errorf("SortTimes(%v) = %v; expected %v", original, test.input, test.expected)
		}
	}
}

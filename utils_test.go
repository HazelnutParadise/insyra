package insyra_test

import (
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

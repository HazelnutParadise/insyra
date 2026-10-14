package insyra

import (
	"slices"
	"testing"
)

// The chainable-never-nil rule has two halves: a method that changes the list
// returns the list, unchanged, when it fails; a method that builds a new list
// returns an empty one carrying the error. These five used to break it
// (#218's classification, the AGENTS.md follow-up of 2026-09-28).

func TestNormalizeFailureReturnsTheList(t *testing.T) {
	cases := map[string]*DataList{
		"text cell":     NewDataList(1.0, 2.0, "x"),
		"no numbers":    NewDataList(nil, nil),
		"all text cell": NewDataList("a"),
	}
	for name, dl := range cases {
		t.Run(name, func(t *testing.T) {
			before := dl.Data()
			got := dl.Normalize()
			if got != dl {
				t.Fatalf("a failed Normalize returned another list (%v); it must return its receiver", got.Data())
			}
			if !slices.Equal(dl.Data(), before) {
				t.Fatalf("a failed Normalize changed the list to %v", dl.Data())
			}
			if dl.Err() == nil {
				t.Fatalf("a failed Normalize recorded no error")
			}
		})
	}
}

func TestNewListTransformsCarryTheirFailure(t *testing.T) {
	seed := SamplingOptions{UseSeed: true, Seed: 1}
	cases := []struct {
		name string
		call func(*DataList) *DataList
	}{
		{"Sample with two options", func(dl *DataList) *DataList { return dl.Sample(1, false, seed, seed) }},
		{"Sample n <= 0", func(dl *DataList) *DataList { return dl.Sample(0, false) }},
		{"Sample n > Len without replacement", func(dl *DataList) *DataList { return dl.Sample(9, false) }},
		{"SampleFrac with two options", func(dl *DataList) *DataList { return dl.SampleFrac(0.5, false, seed, seed) }},
		{"SampleFrac frac out of range", func(dl *DataList) *DataList { return dl.SampleFrac(2, false) }},
		{"Rank with two flags", func(dl *DataList) *DataList { return dl.Rank(true, false) }},
		{"Shift with two fills", func(dl *DataList) *DataList { return dl.Shift(1, 0, 0) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dl := NewDataList(3, 1, 2)
			got := tc.call(dl)
			if got == dl {
				t.Fatalf("the failed call returned its receiver %v as if it were the result", got.Data())
			}
			if got.Len() != 0 {
				t.Fatalf("the failed call returned %v; want an empty list", got.Data())
			}
			if got.Err() == nil {
				t.Fatalf("the result of the failed call carries no error")
			}
			if dl.Err() == nil {
				t.Fatalf("the failed call recorded no error on the receiver")
			}
			if !slices.Equal(dl.Data(), []any{3, 1, 2}) {
				t.Fatalf("the failed call changed the receiver to %v", dl.Data())
			}
		})
	}
}

func TestEmptyInputToSampleIsStillNotAnError(t *testing.T) {
	empty := NewDataList()
	if got := empty.Sample(1, false); got.Len() != 0 || got.Err() != nil {
		t.Fatalf("Sample of an empty list = %v, err %v; want empty, no error", got.Data(), got.Err())
	}
	if got := empty.SampleFrac(0.5, false); got.Len() != 0 || got.Err() != nil {
		t.Fatalf("SampleFrac of an empty list = %v, err %v; want empty, no error", got.Data(), got.Err())
	}
	if empty.Err() != nil {
		t.Fatalf("sampling an empty list recorded an error: %v", empty.Err())
	}
}

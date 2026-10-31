package ta

import (
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestRegimeDurations(t *testing.T) {
	// Rows 0..19: bull 2->5, bear 5->9, bull 9->12, then still falling from 12.
	regimes := []Regime{
		{Kind: RegimeBull, Start: 2, End: 5, Length: 3},
		{Kind: RegimeBear, Start: 5, End: 9, Length: 4},
		{Kind: RegimeBull, Start: 9, End: 12, Length: 3},
	}
	d, o, err := RegimeDurations(regimes, RegimeBull, 20, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Data(), []any{3, 3}) || !reflect.DeepEqual(o.Data(), []any{true, true}) {
		t.Errorf("bull: %v %v", d.Data(), o.Data())
	}
	d, o, err = RegimeDurations(regimes, RegimeBear, 20, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Data(), []any{4, 7}) || !reflect.DeepEqual(o.Data(), []any{true, false}) {
		t.Errorf("bear with the open stretch: %v %v", d.Data(), o.Data())
	}

	// Two segments, rows 0..9 and 10..19: each segment's open stretch ends at
	// its own last row, and one ending on that row is left out.
	seg := []Regime{
		{Kind: RegimeBull, Start: 1, End: 4, Length: 3},
		{Kind: RegimeBear, Start: 11, End: 14, Length: 3},
		{Kind: RegimeBull, Start: 14, End: 19, Length: 5},
	}
	d, o, err = RegimeDurations(seg, RegimeBear, 20, []int{10})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Data(), []any{5, 3}) || !reflect.DeepEqual(o.Data(), []any{false, true}) {
		t.Errorf("segments, bear: %v %v", d.Data(), o.Data())
	}
	d, _, err = RegimeDurations(seg, RegimeBull, 20, []int{10})
	if err != nil || !reflect.DeepEqual(d.Data(), []any{3, 5}) {
		t.Errorf("segments, bull: %v %v", d.Data(), err)
	}

	for name, f := range map[string]func() error{
		"bad kind":      func() error { _, _, e := RegimeDurations(regimes, RegimeKind(7), 20, nil); return e },
		"no rows":       func() error { _, _, e := RegimeDurations(regimes, RegimeBull, 0, nil); return e },
		"past the rows": func() error { _, _, e := RegimeDurations(regimes, RegimeBull, 10, nil); return e },
		"bad start":     func() error { _, _, e := RegimeDurations(regimes, RegimeBull, 20, []int{25}); return e },
	} {
		if f() == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestRegimeDurationsFromZigZag(t *testing.T) {
	prices := []any{100, 104, 110, 118, 112, 101, 95, 99, 108, 117, 126, 120, 111}
	swings, err := ZigZag(insyra.NewDataList(prices...), 0.05)
	if err != nil {
		t.Fatal(err)
	}
	regimes, err := Regimes(swings)
	if err != nil {
		t.Fatal(err)
	}
	d, o, err := RegimeDurations(regimes, RegimeBear, len(prices), nil)
	if err != nil {
		t.Fatal(err)
	}
	if d.Len() == 0 || d.Len() != o.Len() {
		t.Errorf("durations %v observed %v from regimes %+v", d.Data(), o.Data(), regimes)
	}
}

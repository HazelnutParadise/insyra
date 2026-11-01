package ta

import (
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func pricesOf(v ...float64) *insyra.DataList {
	a := make([]any, len(v))
	for i, x := range v {
		a[i] = x
	}
	return insyra.NewDataList(a...)
}

func TestRegimeDurations(t *testing.T) {
	// 20 rows: bull 2->5, bear 5->9, bull 9->12, then falling to a low at
	// row 16 and rebounding a little.
	p := pricesOf(10, 9, 8, 9, 10, 12, 11, 10, 9, 7, 8, 10, 13, 12, 11, 10, 9, 9.5, 9.8, 9.6)
	regimes := []Regime{
		{Kind: RegimeBull, Start: 2, End: 5, Length: 3},
		{Kind: RegimeBear, Start: 5, End: 9, Length: 4},
		{Kind: RegimeBull, Start: 9, End: 12, Length: 3},
	}
	d, o, err := RegimeDurations(regimes, RegimeBull, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Data(), []any{3, 3}) || !reflect.DeepEqual(o.Data(), []any{true, true}) {
		t.Errorf("bull: %v %v", d.Data(), o.Data())
	}
	d, o, err = RegimeDurations(regimes, RegimeBear, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The open bear is censored at its lowest price, row 16, not the last row.
	if !reflect.DeepEqual(d.Data(), []any{4, 4}) || !reflect.DeepEqual(o.Data(), []any{true, false}) {
		t.Errorf("bear with the open stretch: %v %v", d.Data(), o.Data())
	}

	// Two segments, rows 0..9 and 10..19: each segment's open stretch is
	// searched within its own rows, and one with no further extreme is left
	// out.
	q := pricesOf(5, 4, 5, 6, 8, 7, 6, 5, 6, 7, 9, 10, 12, 11, 8, 9, 10, 11, 12, 13)
	seg := []Regime{
		{Kind: RegimeBull, Start: 1, End: 4, Length: 3},
		{Kind: RegimeBear, Start: 12, End: 14, Length: 2},
		{Kind: RegimeBull, Start: 14, End: 19, Length: 5},
	}
	d, o, err = RegimeDurations(seg, RegimeBear, q, []int{10})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.Data(), []any{3, 2}) || !reflect.DeepEqual(o.Data(), []any{false, true}) {
		t.Errorf("segments, bear: %v %v", d.Data(), o.Data())
	}
	d, _, err = RegimeDurations(seg, RegimeBull, q, []int{10})
	if err != nil || !reflect.DeepEqual(d.Data(), []any{3, 5}) {
		t.Errorf("segments, bull: %v %v", d.Data(), err)
	}

	for name, f := range map[string]func() error{
		"bad kind":      func() error { _, _, e := RegimeDurations(regimes, RegimeKind(7), p, nil); return e },
		"no prices":     func() error { _, _, e := RegimeDurations(regimes, RegimeBull, pricesOf(), nil); return e },
		"past the rows": func() error { _, _, e := RegimeDurations(regimes, RegimeBull, pricesOf(1, 2, 3), nil); return e },
		"bad start":     func() error { _, _, e := RegimeDurations(regimes, RegimeBull, p, []int{25}); return e },
		"only the bulls": func() error {
			_, _, e := RegimeDurations([]Regime{regimes[0], regimes[2]}, RegimeBull, p, nil)
			return e
		},
		"out of order": func() error {
			_, _, e := RegimeDurations([]Regime{regimes[1], regimes[0]}, RegimeBull, p, nil)
			return e
		},
		"bad length": func() error {
			_, _, e := RegimeDurations([]Regime{{Kind: RegimeBull, Start: 2, End: 5, Length: 4}}, RegimeBull, p, nil)
			return e
		},
	} {
		if f() == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestRegimeDurationsFromZigZag(t *testing.T) {
	prices := pricesOf(100, 104, 110, 118, 112, 101, 95, 99, 108, 117, 126, 120, 111)
	swings, err := ZigZag(prices, 0.05)
	if err != nil {
		t.Fatal(err)
	}
	regimes, err := Regimes(swings)
	if err != nil {
		t.Fatal(err)
	}
	d, o, err := RegimeDurations(regimes, RegimeBear, prices, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d.Len() == 0 || d.Len() != o.Len() {
		t.Errorf("durations %v observed %v from regimes %+v", d.Data(), o.Data(), regimes)
	}
}

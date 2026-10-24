package ta

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func expectPositions(t *testing.T, name string, got *insyra.DataList, want []float64) {
	t.Helper()
	g := floats(t, got)
	if len(g) != len(want) {
		t.Fatalf("%s: got %v, want %v", name, g, want)
	}
	for i := range want {
		if g[i] != want[i] {
			t.Fatalf("%s: row index %d: got %v, want %v (all %v)", name, i, g[i], want[i], g)
		}
	}
}

// 100, 94, 100, 110, 103, 100 with a 5% filter:
// row 1: 94 is 6% below the high of 100, so the high is confirmed and the rule sells short;
// row 2: 100 is 6.4% above the low of 94, so the low is confirmed and the rule buys;
// row 3: 110 keeps the long; row 4: 103 is 6.4% below 110, so it sells short again; row 5 holds.
var filterPrices = []float64{100, 94, 100, 110, 103, 100}

func TestFilterRuleFollowsConfirmedSwings(t *testing.T) {
	got, err := FilterRule(prices(filterPrices...), 0.05)
	if err != nil {
		t.Fatal(err)
	}
	expectPositions(t, "long-short", got, []float64{0, -1, 1, 1, -1, -1})
}

func TestFilterRuleLongOnlyGoesFlatInsteadOfShort(t *testing.T) {
	got, err := FilterRule(prices(filterPrices...), 0.05, FilterRuleOptions{LongOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	expectPositions(t, "long-only", got, []float64{0, 0, 1, 1, 0, 0})
}

// A break at row 3 drops everything carried over it: the position is flat on the
// first row of the new segment and the rule starts again from there.
func TestFilterRuleResetsAtASegmentStart(t *testing.T) {
	got, err := FilterRule(prices(filterPrices...), 0.05, FilterRuleOptions{SegmentStarts: []int{3}})
	if err != nil {
		t.Fatal(err)
	}
	expectPositions(t, "segmented", got, []float64{0, -1, 1, 0, -1, -1})
}

// The position at a row uses no later price: cutting the series short gives the
// same positions for the rows that remain.
func TestFilterRuleIsCausal(t *testing.T) {
	full, err := FilterRule(prices(filterPrices...), 0.05)
	if err != nil {
		t.Fatal(err)
	}
	cut, err := FilterRule(prices(filterPrices[:4]...), 0.05)
	if err != nil {
		t.Fatal(err)
	}
	f, c := floats(t, full), floats(t, cut)
	for i := range c {
		if f[i] != c[i] {
			t.Errorf("row %d: %v on the full series, %v on the cut one", i, f[i], c[i])
		}
	}
}

func TestFilterRuleRefusesBadInput(t *testing.T) {
	for name, call := range map[string]func() error{
		"threshold 0":   func() error { _, e := FilterRule(prices(filterPrices...), 0); return e },
		"threshold 1":   func() error { _, e := FilterRule(prices(filterPrices...), 1); return e },
		"threshold NaN": func() error { _, e := FilterRule(prices(filterPrices...), math.NaN()); return e },
		"price zero":    func() error { _, e := FilterRule(prices(100, 0, 100), 0.05); return e },
		"nil":           func() error { _, e := FilterRule(nil, 0.05); return e },
		"two options": func() error {
			_, e := FilterRule(prices(filterPrices...), 0.05, FilterRuleOptions{}, FilterRuleOptions{})
			return e
		},
		"segment outside": func() error {
			_, e := FilterRule(prices(filterPrices...), 0.05, FilterRuleOptions{SegmentStarts: []int{99}})
			return e
		},
	} {
		if call() == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// Many breaks must not make the rule quadratic: 20,000 rows in 10,000 two-row
// segments finish at once, and each segment starts flat.
func TestFilterRuleWithManySegments(t *testing.T) {
	const segments = 10000
	vals := make([]float64, 0, 2*segments)
	starts := make([]int, 0, segments)
	for i := 0; i < segments; i++ {
		starts = append(starts, 2*i)
		vals = append(vals, 100, 90)
	}
	got, err := FilterRule(prices(vals...), 0.05, FilterRuleOptions{SegmentStarts: starts})
	if err != nil {
		t.Fatal(err)
	}
	g := floats(t, got)
	for i := 0; i < segments; i++ {
		if g[2*i] != 0 || g[2*i+1] != -1 {
			t.Fatalf("segment %d: positions %v, %v, want 0, -1", i, g[2*i], g[2*i+1])
		}
	}
}

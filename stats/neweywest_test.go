package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

var neweyWestSeries = []any{0.012, -0.004, 0.021, 0.008, -0.015, 0.003, 0.017, 0.009, -0.006, 0.011, 0.004, -0.002, 0.019, -0.008, 0.006, 0.013, -0.011, 0.007, 0.002, 0.015}

func nearlyEqual(a, b, rel float64) bool {
	return math.Abs(a-b) <= rel*math.Max(1, math.Abs(b))
}

// Expected values are the same Bartlett-weighted long-run variance written out
// in R (mean(x), gamma_j = sum((x_t-m)(x_{t-j}-m))/n, weights 1-j/(L+1)); the
// R sandwich package was not available to compare against.
func TestNeweyWestMeanTestMatchesReference(t *testing.T) {
	cases := []struct {
		lag          int
		se, tStat, p float64
	}{
		{0, 0.00221638782707359, 2.2784821042208, 0.0344356195310107},
		{3, 0.0011580020509481, 4.36095946105224, 0.000336300832089364},
		{5, 0.000964197637071017, 5.23751542820681, 4.6900593794514e-05},
	}
	for _, c := range cases {
		got, err := NeweyWestMeanTest(insyra.NewDataList(neweyWestSeries...), c.lag)
		if err != nil {
			t.Fatal(err)
		}
		if !nearlyEqual(got.Mean, 0.00505, 1e-12) || !nearlyEqual(got.StdErr, c.se, 1e-10) ||
			!nearlyEqual(got.Statistic, c.tStat, 1e-10) || !nearlyEqual(got.PValue, c.p, 1e-8) ||
			got.N != 20 || got.Lag != c.lag || *got.DF != 19 {
			t.Errorf("lag %d: got %+v", c.lag, got)
		}
	}
}

func TestNeweyWestLagZeroIsTheOneSampleTTestWithNDivisor(t *testing.T) {
	list := insyra.NewDataList(neweyWestSeries...)
	nw, err := NeweyWestMeanTest(list, 0)
	if err != nil {
		t.Fatal(err)
	}
	tt, err := SingleSampleTTest(list, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := tt.Statistic * math.Sqrt(20.0/19.0)
	if !nearlyEqual(nw.Statistic, want, 1e-12) {
		t.Errorf("t = %v, want %v", nw.Statistic, want)
	}
}

func TestNeweyWestAutomaticLag(t *testing.T) {
	if got := NeweyWestLag(20); got != 2 {
		t.Errorf("NeweyWestLag(20) = %d, want 2 (R: floor(4*(20/100)^(2/9)))", got)
	}
	got, err := NeweyWestMeanTest(insyra.NewDataList(neweyWestSeries...), -1)
	if err != nil || got.Lag != 2 {
		t.Errorf("automatic lag: %+v, %v", got, err)
	}
}

func TestNeweyWestRefusesWhatItCannotTest(t *testing.T) {
	if _, err := NeweyWestMeanTest(insyra.NewDataList(1.0), 0); err == nil {
		t.Error("one observation should be an error")
	}
	if _, err := NeweyWestMeanTest(insyra.NewDataList(1.0, 2.0, 3.0), 3); err == nil {
		t.Error("lag equal to n should be an error")
	}
	if _, err := NeweyWestMeanTest(insyra.NewDataList(1.0, math.NaN(), 3.0), 1); err == nil {
		t.Error("NaN should be an error")
	}
	var nilList *insyra.DataList
	if _, err := NeweyWestMeanTest(nilList, 1); err == nil {
		t.Error("nil list should be an error")
	}
}

func TestNeweyWestConstantSeries(t *testing.T) {
	got, err := NeweyWestMeanTest(insyra.NewDataList(2.0, 2.0, 2.0, 2.0), 1)
	if err != nil || !math.IsInf(got.Statistic, 1) || got.PValue != 0 {
		t.Errorf("got %+v, %v", got, err)
	}
	zero, err := NeweyWestMeanTest(insyra.NewDataList(0.0, 0.0, 0.0), 1)
	if err != nil || !math.IsNaN(zero.Statistic) {
		t.Errorf("got %+v, %v", zero, err)
	}
}

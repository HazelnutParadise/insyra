package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// The series and the autocorrelations come from R:
// x <- sin(1:30)*cos((1:30)^2/7)+(1:30)%%3; acf(x, lag.max = 4, plot = FALSE)
var acfRSeries = []float64{
	1.83289913904393, 2.76483662938638, 0.0396880522745193, 1.49612601234937, 2.87169490190588,
	-0.116599053667903, 1.49530367784744, 1.04969881031128, 0.224419739560878, 1.08051601204788,
	1.99304543320307, 0.0807554490772304, 1.23058210853274, 1.04643554819475, 0.485906632343552,
	0.876560319093997, 2.86776357870378, 0.502275517225149, 1.03923256383015, 2.75646879680801,
	0.824855933994463, 0.991152115430381, 1.1664411718297, -0.745197737710051, 0.967297159791227,
	1.47862107587395, -0.852544656579262, 1.12352466968351, 1.52004371431599, 0.961134222447336,
}

func acfList(x []float64) insyra.IDataList {
	vals := make([]any, len(x))
	for i, v := range x {
		vals[i] = v
	}
	return insyra.NewDataList(vals...)
}

func TestAutocorrelationMatchesR(t *testing.T) {
	want := []float64{-0.245712279507009, -0.228375969899597, 0.574745827287615, -0.249339608113798}
	got, err := Autocorrelation(acfList(acfRSeries), 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("%d lags, want 4", len(got))
	}
	n := float64(len(acfRSeries))
	for i, w := range want {
		if got[i].Lag != i+1 || math.Abs(got[i].Rho-w) > 1e-12 {
			t.Errorf("lag %d: rho %v, R says %v", i+1, got[i].Rho, w)
		}
		if math.Abs(got[i].T-math.Sqrt(n)*w) > 1e-12 {
			t.Errorf("lag %d: T %v, want sqrt(n)*rho", i+1, got[i].T)
		}
		if p := zPValue(got[i].T, TwoSided); math.Abs(got[i].P-p) > 1e-15 {
			t.Errorf("lag %d: P %v, want %v", i+1, got[i].P, p)
		}
	}
}

func TestAutocorrelationRobustMatchesTheDefinition(t *testing.T) {
	x := acfRSeries
	n := len(x)
	var mu float64
	for _, v := range x {
		mu += v
	}
	mu /= float64(n)
	var ss float64
	for _, v := range x {
		ss += (v - mu) * (v - mu)
	}
	got, err := Autocorrelation(acfList(x), 3)
	if err != nil {
		t.Fatal(err)
	}
	for j := 1; j <= 3; j++ {
		// delta = n * sum_{t=j+1..n} (x_t-mu)^2 (x_{t-j}-mu)^2 / ss^2   (Lo and MacKinlay 1988)
		var num float64
		for k := j; k < n; k++ {
			a, b := x[k]-mu, x[k-j]-mu
			num += a * a * b * b
		}
		delta := float64(n) * num / (ss * ss)
		want := math.Sqrt(float64(n)) * got[j-1].Rho / math.Sqrt(delta)
		if math.Abs(got[j-1].TRobust-want) > 1e-12 {
			t.Errorf("lag %d: TRobust %v, want %v", j, got[j-1].TRobust, want)
		}
		if p := zPValue(want, TwoSided); math.Abs(got[j-1].PRobust-p) > 1e-15 {
			t.Errorf("lag %d: PRobust %v, want %v", j, got[j-1].PRobust, p)
		}
	}
}

func TestAutocorrelationRefusesBadInput(t *testing.T) {
	ok := acfList(acfRSeries)
	for name, call := range map[string]func() error{
		"maxLag 0":        func() error { _, err := Autocorrelation(ok, 0); return err },
		"maxLag n":        func() error { _, err := Autocorrelation(ok, len(acfRSeries)); return err },
		"constant series": func() error { _, err := Autocorrelation(insyra.NewDataList(2.0, 2.0, 2.0, 2.0), 1); return err },
		"text cell":       func() error { _, err := Autocorrelation(insyra.NewDataList(1.0, "x", 3.0, 4.0), 1); return err },
		"nil list":        func() error { _, err := Autocorrelation(nil, 1); return err },
		"empty list":      func() error { _, err := Autocorrelation(insyra.NewDataList(), 1); return err },
	} {
		if call() == nil {
			t.Errorf("%s: no error", name)
		}
	}
	// Not constant, but every lagged product is zero: TRobust would be 0/0.
	if _, err := Autocorrelation(insyra.NewDataList(-1.0, 0.0, 1.0), 1); err == nil {
		t.Error("zero robust variance: no error")
	}
	if _, err := Autocorrelation(ok, len(acfRSeries)-1); err != nil {
		t.Errorf("maxLag n-1 should work: %v", err)
	}
}

func TestAutocorrelationRefusesAConstantSeriesOfDecimals(t *testing.T) {
	// The mean of 100 copies of 0.1 is not exactly 0.1, so squared deviations
	// alone would call this series non-constant and report Rho 0.99.
	vals := make([]any, 100)
	for i := range vals {
		vals[i] = 0.1
	}
	if _, err := Autocorrelation(insyra.NewDataList(vals...), 1); err == nil {
		t.Fatal("a constant series of 0.1 was not refused")
	}
}

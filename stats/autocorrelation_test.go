package stats

import (
	"math"
	"os"
	"strconv"
	"strings"
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

func TestAutocorrelationDoesNotDependOnTheScale(t *testing.T) {
	base := []float64{1, 2, 3, 4, 2, 5, 1, 6}
	want, err := Autocorrelation(acfList(base), 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, scale := range []float64{1e80, 1e-80, 1e150} {
		scaled := make([]float64, len(base))
		for i, v := range base {
			scaled[i] = v * scale
		}
		got, err := Autocorrelation(acfList(scaled), 2)
		if err != nil {
			t.Fatalf("scale %g: %v", scale, err)
		}
		for j := range want {
			for label, pair := range map[string][2]float64{"Rho": {got[j].Rho, want[j].Rho}, "T": {got[j].T, want[j].T}, "TRobust": {got[j].TRobust, want[j].TRobust}, "PRobust": {got[j].PRobust, want[j].PRobust}, "Q": {got[j].Q, want[j].Q}, "QRobust": {got[j].QRobust, want[j].QRobust}} {
				if math.IsNaN(pair[0]) || math.Abs(pair[0]-pair[1]) > 1e-9*math.Max(1, math.Abs(pair[1])) {
					t.Errorf("scale %g lag %d: %s = %v, want %v", scale, j+1, label, pair[0], pair[1])
				}
			}
		}
	}
}

const (
	acfPortmanteauR  = "testdata/autocorrelation_reference.txt"
	acfPortmanteauPy = "testdata/autocorrelation_reference_py.txt"
)

// acfReference reads the key=value lines a reference script printed.
func acfReference(t *testing.T, path string) map[string]float64 {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	out := map[string]float64{}
	for _, line := range strings.Split(string(raw), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			t.Fatalf("%s: %q: %v", path, line, err)
		}
		out[strings.TrimSpace(k)] = f
	}
	return out
}

func acfRef(t *testing.T, ref map[string]float64, key string) float64 {
	t.Helper()
	v, ok := ref[key]
	if !ok {
		t.Fatalf("reference has no %s", key)
	}
	return v
}

func acfClusteredSeries(t *testing.T) []float64 {
	t.Helper()
	ref := acfReference(t, acfPortmanteauR)
	x := make([]float64, 200)
	for i := range x {
		x[i] = acfRef(t, ref, "clustered.x["+strconv.Itoa(i+1)+"]")
	}
	return x
}

func TestAutocorrelationPortmanteauMatchesRAndStatsmodels(t *testing.T) {
	refR, refPy := acfReference(t, acfPortmanteauR), acfReference(t, acfPortmanteauPy)
	for _, c := range []struct {
		prefix string
		x      []float64
		maxLag int
	}{
		{"short", acfRSeries, 4},
		{"clustered", acfClusteredSeries(t), 10},
	} {
		got, err := Autocorrelation(acfList(c.x), c.maxLag)
		if err != nil {
			t.Fatalf("%s: %v", c.prefix, err)
		}
		for m := 1; m <= c.maxLag; m++ {
			r := got[m-1]
			key := func(f string) string { return c.prefix + "." + f + "[" + strconv.Itoa(m) + "]" }
			for name, ref := range map[string]map[string]float64{"R": refR, "statsmodels": refPy} {
				if w := acfRef(t, ref, key("Q")); math.Abs(r.Q-w) > 1e-10*math.Max(1, w) {
					t.Errorf("%s lag %d: Q %v, %s says %v", c.prefix, m, r.Q, name, w)
				}
				if w := acfRef(t, ref, key("PQ")); math.Abs(r.PQ-w) > 1e-12 {
					t.Errorf("%s lag %d: PQ %v, %s says %v", c.prefix, m, r.PQ, name, w)
				}
			}
			if w := acfRef(t, refR, key("QRobust")); math.Abs(r.QRobust-w) > 1e-10*math.Max(1, w) {
				t.Errorf("%s lag %d: QRobust %v, R says %v", c.prefix, m, r.QRobust, w)
			}
			if w := acfRef(t, refR, key("PQRobust")); math.Abs(r.PQRobust-w) > 1e-12 {
				t.Errorf("%s lag %d: PQRobust %v, R says %v", c.prefix, m, r.PQRobust, w)
			}
		}
	}
}

func TestAutocorrelationQRobustIsTheSumOfSquaredTRobust(t *testing.T) {
	got, err := Autocorrelation(acfList(acfClusteredSeries(t)), 10)
	if err != nil {
		t.Fatal(err)
	}
	var sum float64
	for m, r := range got {
		sum += r.TRobust * r.TRobust
		if math.Abs(r.QRobust-sum) > 1e-12*math.Max(1, sum) {
			t.Errorf("lag %d: QRobust %v, sum of TRobust^2 %v", m+1, r.QRobust, sum)
		}
		if p := chiSquaredPValue(sum, float64(m+1)); math.Abs(r.PQRobust-p) > 1e-15 {
			t.Errorf("lag %d: PQRobust %v, want %v", m+1, r.PQRobust, p)
		}
	}
}

func TestAutocorrelationShowsThePortmanteauColumns(t *testing.T) {
	got, err := Autocorrelation(acfList(acfRSeries), 2)
	if err != nil {
		t.Fatal(err)
	}
	header := strings.Fields(strings.Split(got.String(), "\n")[1])
	want := []string{"Lag", "Rho", "T", "P", "TRobust", "PRobust", "Q", "PQ", "QRobust", "PQRobust"}
	if strings.Join(header, " ") != strings.Join(want, " ") {
		t.Errorf("header %v, want %v", header, want)
	}
}

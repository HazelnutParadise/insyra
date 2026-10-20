package stats_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// robustRegressionData is a seeded regression whose errors are both
// heteroskedastic (their scale grows with |x1|) and AR(1), the case robust
// and HAC standard errors exist for.
func robustRegressionData(n int) (y []float64, xs [][]float64) {
	r := rand.New(rand.NewPCG(478, 1))
	x1 := make([]float64, n)
	x2 := make([]float64, n)
	y = make([]float64, n)
	prevX2, prevE := 0.0, 0.0
	for i := range n {
		x1[i] = r.NormFloat64()
		x2[i] = 0.6*prevX2 + r.NormFloat64()
		prevX2 = x2[i]
		e := 0.5*prevE + (0.5+math.Abs(x1[i]))*r.NormFloat64()
		prevE = e
		y[i] = 0.3 + 1.2*x1[i] - 0.7*x2[i] + e
	}
	return y, [][]float64{x1, x2}
}

func floatList(values []float64) *insyra.DataList {
	raw := make([]any, len(values))
	for i, v := range values {
		raw[i] = v
	}
	return insyra.NewDataList(raw...)
}

func robustFit(t *testing.T, y []float64, xs [][]float64, opts stats.LinearRegressionOptions) *stats.LinearRegressionResult {
	t.Helper()
	lists := make([]insyra.IDataList, len(xs))
	for i, x := range xs {
		lists[i] = floatList(x)
	}
	fit, err := stats.LinearRegressionWithOptions(opts, floatList(y), lists...)
	if err != nil {
		t.Fatal(err)
	}
	return fit
}

func TestCrossLangLinearRegressionRobustCovariance(t *testing.T) {
	requirePythonTools(t)
	y, xs := robustRegressionData(120)
	cases := []struct {
		opts stats.LinearRegressionOptions
		name string
		lag  int
	}{
		{stats.LinearRegressionOptions{CovType: stats.CovHC0}, "HC0", 0},
		{stats.LinearRegressionOptions{CovType: stats.CovHC1}, "HC1", 0},
		{stats.LinearRegressionOptions{CovType: stats.CovHC2}, "HC2", 0},
		{stats.LinearRegressionOptions{CovType: stats.CovHC3}, "HC3", 0},
		{stats.LinearRegressionOptions{CovType: stats.CovHAC, Lag: 0}, "HAC", 0},
		{stats.LinearRegressionOptions{CovType: stats.CovHAC, Lag: 4}, "HAC", 4},
		{stats.LinearRegressionOptions{CovType: stats.CovHAC, Lag: 12}, "HAC", 12},
		{stats.LinearRegressionOptions{CovType: stats.CovHAC, Lag: -1}, "HAC", stats.NeweyWestLag(120)},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v/lag%d", c.opts.CovType, c.opts.Lag), func(t *testing.T) {
			fit := robustFit(t, y, xs, c.opts)
			if fit.CovType != c.opts.CovType || fit.Lag != c.lag {
				t.Fatalf("recorded CovType %v lag %d, want %v lag %d", fit.CovType, fit.Lag, c.opts.CovType, c.lag)
			}
			ref := runPythonBaseline(t, "ols_robust", map[string]any{
				"y": y, "xs": xs, "cov_type": c.name, "lag": c.lag,
			})
			for field, got := range map[string][]float64{
				"coef": fit.Coefficients, "se": fit.StandardErrors, "t": fit.TValues, "p": fit.PValues,
			} {
				want := ref[field].([]any)
				for i, w := range want {
					wv := w.(float64)
					if math.Abs(got[i]-wv) > 1e-9*math.Max(1, math.Abs(wv)) {
						t.Errorf("%s[%d]: insyra %.15g, statsmodels %.15g", field, i, got[i], wv)
					}
				}
			}
		})
	}
}

func TestLinearRegressionHACLagZeroIsHC0(t *testing.T) {
	y, xs := robustRegressionData(80)
	hac := robustFit(t, y, xs, stats.LinearRegressionOptions{CovType: stats.CovHAC, Lag: 0})
	hc0 := robustFit(t, y, xs, stats.LinearRegressionOptions{CovType: stats.CovHC0})
	if !reflect.DeepEqual(hac.StandardErrors, hc0.StandardErrors) ||
		!reflect.DeepEqual(hac.TValues, hc0.TValues) || !reflect.DeepEqual(hac.PValues, hc0.PValues) {
		t.Errorf("HAC lag 0 %v differs from HC0 %v", hac.StandardErrors, hc0.StandardErrors)
	}
}

func TestLinearRegressionZeroOptionsIsUnchanged(t *testing.T) {
	y, xs := robustRegressionData(60)
	plain, err := stats.LinearRegression(floatList(y), floatList(xs[0]), floatList(xs[1]))
	if err != nil {
		t.Fatal(err)
	}
	zero := robustFit(t, y, xs, stats.LinearRegressionOptions{})
	// %#v prints every float in a form that parses back to the same bits, and
	// prints NaN as NaN, which reflect.DeepEqual would call unequal to itself.
	if fmt.Sprintf("%#v", plain) != fmt.Sprintf("%#v", zero) {
		t.Errorf("zero options differ from LinearRegression:\n%+v\n%+v", plain, zero)
	}
	if plain.CovType != stats.CovOLS || plain.Lag != 0 {
		t.Errorf("CovType %v lag %d, want CovOLS lag 0", plain.CovType, plain.Lag)
	}
	// The classical standard errors are what they were before options existed:
	// sqrt(SSE/(n-k) · (XᵀX)⁻¹[i][i]), written out here for the slope of a
	// simple regression, sqrt(SSE/(n-2) / Sxx).
	simple, err := stats.LinearRegression(floatList(y), floatList(xs[0]))
	if err != nil {
		t.Fatal(err)
	}
	mean := 0.0
	for _, v := range xs[0] {
		mean += v
	}
	mean /= float64(len(xs[0]))
	sxx, sse := 0.0, 0.0
	for i, v := range xs[0] {
		sxx += (v - mean) * (v - mean)
		sse += simple.Residuals[i] * simple.Residuals[i]
	}
	want := math.Sqrt(sse / float64(len(y)-2) / sxx)
	if math.Abs(simple.StandardError-want) > 1e-12*want {
		t.Errorf("slope SE %v, want %v", simple.StandardError, want)
	}
}

func TestLinearRegressionCovarianceDoesNotMoveTheFit(t *testing.T) {
	y, xs := robustRegressionData(60)
	ols := robustFit(t, y, xs, stats.LinearRegressionOptions{})
	hc3 := robustFit(t, y, xs, stats.LinearRegressionOptions{CovType: stats.CovHC3})
	if !reflect.DeepEqual(ols.Coefficients, hc3.Coefficients) || !reflect.DeepEqual(ols.Residuals, hc3.Residuals) ||
		ols.RSquared != hc3.RSquared || ols.AdjustedRSquared != hc3.AdjustedRSquared {
		t.Error("the covariance type changed the fit")
	}
	if reflect.DeepEqual(ols.StandardErrors, hc3.StandardErrors) {
		t.Error("HC3 returned the classical standard errors")
	}
	// The simple-regression convenience fields follow the chosen covariance.
	simple := robustFit(t, y, xs[:1], stats.LinearRegressionOptions{CovType: stats.CovHC1})
	if simple.StandardError != simple.StandardErrors[1] || simple.ConfidenceIntervalSlope != simple.ConfidenceIntervals[1] {
		t.Error("simple-regression fields do not follow the robust covariance")
	}
}

func TestLinearRegressionRefusesBadCovarianceSettings(t *testing.T) {
	y, xs := robustRegressionData(20)
	lists := []insyra.IDataList{floatList(xs[0]), floatList(xs[1])}
	_, err := stats.LinearRegressionWithOptions(stats.LinearRegressionOptions{CovType: stats.CovType(99)}, floatList(y), lists...)
	if err == nil || !strings.Contains(err.Error(), "CovType(99)") {
		t.Errorf("unknown CovType: %v", err)
	}
	if _, err := stats.LinearRegressionWithOptions(stats.LinearRegressionOptions{CovType: stats.CovHAC, Lag: 20}, floatList(y), lists...); err == nil {
		t.Error("a lag equal to n should be refused")
	}
	fit, err := stats.LinearRegressionWithOptions(stats.LinearRegressionOptions{CovType: stats.CovHC0, Lag: 5}, floatList(y), lists...)
	if err != nil || fit.Lag != 0 {
		t.Errorf("a lag outside CovHAC should be ignored and recorded as 0: %v, %v", fit, err)
	}
}

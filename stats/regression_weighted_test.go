package stats_test

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func weightedFixture() (y, w, x1, x2 *insyra.DataList, yRaw, wRaw, aRaw, bRaw []float64) {
	const n = 45
	ys := make([]any, n)
	ws := make([]any, n)
	a := make([]any, n)
	b := make([]any, n)
	yRaw = make([]float64, n)
	wRaw = make([]float64, n)
	aRaw = make([]float64, n)
	bRaw = make([]float64, n)
	for i := 0; i < n; i++ {
		v1 := float64(i % 8)
		v2 := float64((i * 5) % 9)
		weight := 0.5 + float64(i%4)
		yv := 4 + 1.5*v1 - 2*v2 + 0.3*math.Sin(float64(i)*1.9)
		a[i], b[i], ws[i], ys[i] = v1, v2, weight, yv
		aRaw[i], bRaw[i], wRaw[i], yRaw[i] = v1, v2, weight, yv
	}
	return insyra.NewDataList(ys...), insyra.NewDataList(ws...),
		insyra.NewDataList(a...), insyra.NewDataList(b...),
		yRaw, wRaw, aRaw, bRaw
}

// Uniform weights change nothing: the weighted solution must be OLS exactly.
func TestUniformWeightsReproduceOLS(t *testing.T) {
	y, _, x1, x2, _, _, _, _ := weightedFixture()
	uniform := make([]any, y.Len())
	for i := range uniform {
		uniform[i] = 2.5 // any constant, not only 1
	}
	weighted, err := stats.WeightedLinearRegression(y, insyra.NewDataList(uniform...), x1, x2)
	if err != nil {
		t.Fatal(err)
	}
	ols, err := stats.LinearRegression(y, x1, x2)
	if err != nil {
		t.Fatal(err)
	}
	for j := range ols.Coefficients {
		if math.Abs(weighted.Coefficients[j]-ols.Coefficients[j]) > 1e-9 {
			t.Fatalf("coefficient %d: weighted %v vs OLS %v under uniform weights", j, weighted.Coefficients[j], ols.Coefficients[j])
		}
	}
}

// Weights must actually matter: upweighting a subset moves the fit toward it.
func TestWeightsChangeTheFit(t *testing.T) {
	y, w, x1, x2, _, _, _, _ := weightedFixture()
	weighted, err := stats.WeightedLinearRegression(y, w, x1, x2)
	if err != nil {
		t.Fatal(err)
	}
	ols, err := stats.LinearRegression(y, x1, x2)
	if err != nil {
		t.Fatal(err)
	}
	same := true
	for j := range ols.Coefficients {
		if math.Abs(weighted.Coefficients[j]-ols.Coefficients[j]) > 1e-12 {
			same = false
		}
	}
	if same {
		t.Fatal("non-uniform weights produced the OLS fit; the weights are not being read")
	}
}

func TestWeightedRefusals(t *testing.T) {
	y, _, x1, x2, _, _, _, _ := weightedFixture()
	bad := func(value any) *insyra.DataList {
		ws := make([]any, y.Len())
		for i := range ws {
			ws[i] = 1.0
		}
		ws[3] = value
		return insyra.NewDataList(ws...)
	}
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"zero", 0.0},
		{"negative", -0.5},
		{"missing", nil},
		{"text", "abc"},
		{"infinite", math.Inf(1)},
	} {
		_, err := stats.WeightedLinearRegression(y, bad(tc.value), x1, x2)
		if err == nil {
			t.Fatalf("a %s weight was accepted", tc.name)
		}
		// The row is what makes the refusal actionable.
		if !strings.Contains(err.Error(), "4") && !strings.Contains(err.Error(), "row") {
			t.Fatalf("%s: error %q does not locate the offending weight", tc.name, err)
		}
	}
	if _, err := stats.WeightedLinearRegression(y, nil, x1, x2); err == nil {
		t.Fatal("nil weights were accepted")
	}
}

func TestCrossLangWeightedLeastSquares(t *testing.T) {
	requirePythonTools(t)
	y, w, x1, x2, yRaw, wRaw, aRaw, bRaw := weightedFixture()

	fit, err := stats.WeightedLinearRegression(y, w, x1, x2)
	if err != nil {
		t.Fatal(err)
	}
	newXs := [][]float64{{1, 4, 7}, {2, 0, 8}}
	baseline := runPythonBaseline(t, "wls", map[string]any{
		"y": yRaw, "xs": [][]float64{aRaw, bRaw}, "weights": wRaw, "new_xs": newXs,
	})

	compare := func(label string, got, want float64, tolerance float64) {
		t.Helper()
		if math.Abs(got-want) > tolerance {
			t.Fatalf("%s: insyra %v vs statsmodels %v", label, got, want)
		}
	}
	for j, ref := range baseline["coefficients"].([]any) {
		compare("coefficient", fit.Coefficients[j], ref.(float64), 1e-8)
	}
	// The inference is the point of WLS over ridge: SE, t and p must match the
	// reference, not merely the point estimates.
	for j, ref := range baseline["standard_errors"].([]any) {
		compare("standard error", fit.StandardErrors[j], ref.(float64), 1e-8)
	}
	for j, ref := range baseline["t_values"].([]any) {
		compare("t value", fit.TValues[j], ref.(float64), 1e-6)
	}
	for j, ref := range baseline["p_values"].([]any) {
		compare("p value", fit.PValues[j], ref.(float64), 1e-8)
	}
	compare("weighted R²", fit.RSquared, baseline["r_squared"].(float64), 1e-10)

	predicted, err := fit.Predict(stats.PredictResponse,
		insyra.NewDataList(1.0, 4.0, 7.0), insyra.NewDataList(2.0, 0.0, 8.0))
	if err != nil {
		t.Fatal(err)
	}
	for i, ref := range baseline["predictions"].([]any) {
		got, _ := insyra.ToFloat64Safe(predicted.Get(i))
		compare("prediction", got, ref.(float64), 1e-8)
	}
}

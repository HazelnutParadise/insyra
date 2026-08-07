package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Ridge is exact, so it must match scikit-learn to solver precision. Lasso is
// iterative; both sides run to a tight shared tolerance and the unique optimum
// is what they agree on.
func TestCrossLangRegularizedRegression(t *testing.T) {
	requirePythonTools(t)

	const n = 50
	ys := make([]any, n)
	a := make([]any, n)
	b := make([]any, n)
	c := make([]any, n)
	yRaw := make([]float64, n)
	aRaw := make([]float64, n)
	bRaw := make([]float64, n)
	cRaw := make([]float64, n)
	for i := 0; i < n; i++ {
		v1 := float64(i % 7)
		v2 := float64((i * 3) % 11)
		v3 := math.Sin(float64(i) * 0.9) // near-noise: the predictor lasso should drop
		yv := 3 + 2*v1 - 1.5*v2 + 0.08*math.Cos(float64(i)*1.3)
		a[i], b[i], c[i], ys[i] = v1, v2, v3, yv
		aRaw[i], bRaw[i], cRaw[i], yRaw[i] = v1, v2, v3, yv
	}
	y := insyra.NewDataList(ys...)
	x1 := insyra.NewDataList(a...)
	x2 := insyra.NewDataList(b...)
	x3 := insyra.NewDataList(c...)
	newXs := [][]float64{{0.5, 3.5, 6}, {1, 5, 9}, {0.2, -0.4, 0.9}}

	compare := func(t *testing.T, label string, got, want float64, tolerance float64) {
		t.Helper()
		if math.Abs(got-want) > tolerance {
			t.Fatalf("%s: insyra %v vs sklearn %v (|diff| %.3g > %.3g)", label, got, want, math.Abs(got-want), tolerance)
		}
	}
	predictionLists := func() []insyra.IDataList {
		out := make([]insyra.IDataList, len(newXs))
		for j, values := range newXs {
			anyValues := make([]any, len(values))
			for i, v := range values {
				anyValues[i] = v
			}
			out[j] = insyra.NewDataList(anyValues...)
		}
		return out
	}

	t.Run("ridge", func(t *testing.T) {
		const alpha = 0.7
		fit, err := stats.RidgeRegression(y, alpha, x1, x2, x3)
		if err != nil {
			t.Fatal(err)
		}
		baseline := runPythonBaseline(t, "ridge", map[string]any{
			"y": yRaw, "xs": [][]float64{aRaw, bRaw, cRaw}, "alpha": alpha, "new_xs": newXs,
		})
		compare(t, "intercept", fit.Coefficients[0], baseline["intercept"].(float64), 1e-8)
		refCoefficients := baseline["coefficients"].([]any)
		for j, ref := range refCoefficients {
			compare(t, "coefficient", fit.Coefficients[j+1], ref.(float64), 1e-8)
		}
		predicted, err := fit.Predict(stats.PredictResponse, predictionLists()...)
		if err != nil {
			t.Fatal(err)
		}
		for i, ref := range baseline["predictions"].([]any) {
			got, _ := insyra.ToFloat64Safe(predicted.Get(i))
			compare(t, "prediction", got, ref.(float64), 1e-8)
		}
	})

	t.Run("lasso", func(t *testing.T) {
		const alpha = 0.1
		const tolerance = 1e-10
		fit, err := stats.LassoRegression(y, alpha, []insyra.IDataList{x1, x2, x3}, stats.LassoOptions{
			Tolerance:     tolerance,
			MaxIterations: 100000,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !fit.Converged {
			t.Fatalf("did not converge in %d iterations", fit.Iterations)
		}
		baseline := runPythonBaseline(t, "lasso", map[string]any{
			"y": yRaw, "xs": [][]float64{aRaw, bRaw, cRaw}, "alpha": alpha,
			"tol": tolerance, "max_iter": 100000, "new_xs": newXs,
		})
		compare(t, "intercept", fit.Coefficients[0], baseline["intercept"].(float64), 1e-6)
		refCoefficients := baseline["coefficients"].([]any)
		zeroAgreement := 0
		for j, ref := range refCoefficients {
			compare(t, "coefficient", fit.Coefficients[j+1], ref.(float64), 1e-6)
			// Exact zeros must agree as zeros, not merely as small numbers —
			// sparsity is the deliverable being verified.
			if ref.(float64) == 0 {
				zeroAgreement++
				if fit.Coefficients[j+1] != 0 {
					t.Fatalf("coefficient %d: sklearn selected it out (0) but insyra kept %v", j, fit.Coefficients[j+1])
				}
			}
		}
		predicted, err := fit.Predict(stats.PredictResponse, predictionLists()...)
		if err != nil {
			t.Fatal(err)
		}
		for i, ref := range baseline["predictions"].([]any) {
			got, _ := insyra.ToFloat64Safe(predicted.Get(i))
			compare(t, "prediction", got, ref.(float64), 1e-6)
		}
	})
}

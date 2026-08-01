package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func regularizedFixture() (y *insyra.DataList, x1, x2 *insyra.DataList) {
	// Deterministic, mildly noisy y = 3 + 2·x1 − 1.5·x2 + noise.
	const n = 40
	ys := make([]any, n)
	a := make([]any, n)
	b := make([]any, n)
	for i := 0; i < n; i++ {
		v1 := float64(i % 7)
		v2 := float64((i * 3) % 11)
		noise := 0.05 * math.Sin(float64(i))
		a[i] = v1
		b[i] = v2
		ys[i] = 3 + 2*v1 - 1.5*v2 + noise
	}
	return insyra.NewDataList(ys...), insyra.NewDataList(a...), insyra.NewDataList(b...)
}

// α = 0 removes the penalty, so ridge must reproduce OLS to numerical noise.
func TestRidgeWithZeroPenaltyEqualsOLS(t *testing.T) {
	y, x1, x2 := regularizedFixture()
	ols, err := stats.LinearRegression(y, x1, x2)
	if err != nil {
		t.Fatal(err)
	}
	ridge, err := stats.RidgeRegression(y, 0, x1, x2)
	if err != nil {
		t.Fatal(err)
	}
	if len(ridge.Coefficients) != len(ols.Coefficients) {
		t.Fatalf("coefficient counts differ: %d vs %d", len(ridge.Coefficients), len(ols.Coefficients))
	}
	for j := range ols.Coefficients {
		if math.Abs(ridge.Coefficients[j]-ols.Coefficients[j]) > 1e-8 {
			t.Fatalf("coefficient %d: ridge %v vs OLS %v", j, ridge.Coefficients[j], ols.Coefficients[j])
		}
	}
}

// Collinearity is the ordinary reason to reach for ridge: the unpenalized
// normal equations are singular, the penalized ones are not.
func TestRidgeHandlesCollinearityOLSCannot(t *testing.T) {
	const n = 30
	ys := make([]any, n)
	a := make([]any, n)
	b := make([]any, n) // exactly 2·a — perfectly collinear
	for i := 0; i < n; i++ {
		v := float64(i%9) + 1
		a[i] = v
		b[i] = 2 * v
		ys[i] = 1 + v + 0.01*math.Cos(float64(i))
	}
	y := insyra.NewDataList(ys...)
	x1 := insyra.NewDataList(a...)
	x2 := insyra.NewDataList(b...)

	if _, err := stats.LinearRegression(y, x1, x2); err == nil {
		t.Fatal("OLS accepted perfectly collinear predictors; the fixture is not exercising singularity")
	}
	ridge, err := stats.RidgeRegression(y, 1.0, x1, x2)
	if err != nil {
		t.Fatalf("ridge failed on the data it exists for: %v", err)
	}
	for j, c := range ridge.Coefficients {
		if math.IsNaN(c) || math.IsInf(c, 0) {
			t.Fatalf("coefficient %d is not finite: %v", j, c)
		}
	}
}

// Sparsity is lasso's point, and it must be exact zero, not merely small.
func TestLassoDrivesCoefficientsToExactZero(t *testing.T) {
	y, x1, x2 := regularizedFixture()
	fit, err := stats.LassoRegression(y, 50.0, []insyra.IDataList{x1, x2})
	if err != nil {
		t.Fatal(err)
	}
	if !fit.Converged {
		t.Fatalf("did not converge in %d iterations", fit.Iterations)
	}
	for j := 1; j < len(fit.Coefficients); j++ {
		if fit.Coefficients[j] != 0 {
			t.Fatalf("coefficient %d = %v under a crushing penalty, want exactly 0", j, fit.Coefficients[j])
		}
	}
	// With every slope at zero the intercept is the mean of y.
	mean := 0.0
	for i := 0; i < y.Len(); i++ {
		v, _ := insyra.ToFloat64Safe(y.Get(i))
		mean += v
	}
	mean /= float64(y.Len())
	if math.Abs(fit.Coefficients[0]-mean) > 1e-9 {
		t.Fatalf("intercept %v, want the mean %v", fit.Coefficients[0], mean)
	}

	// A small penalty must keep real predictors.
	light, err := stats.LassoRegression(y, 0.01, []insyra.IDataList{x1, x2})
	if err != nil {
		t.Fatal(err)
	}
	if light.Coefficients[1] == 0 || light.Coefficients[2] == 0 {
		t.Fatalf("a light penalty removed a real predictor: %v", light.Coefficients)
	}
}

func TestLassoReportsNonConvergenceInsteadOfLying(t *testing.T) {
	y, x1, x2 := regularizedFixture()
	fit, err := stats.LassoRegression(y, 0.01, []insyra.IDataList{x1, x2}, stats.LassoOptions{
		MaxIterations: 1,
		Tolerance:     1e-14,
	})
	if err != nil {
		t.Fatalf("a starved iteration cap must still return the best estimate: %v", err)
	}
	if fit.Converged {
		t.Fatal("one iteration at tolerance 1e-14 reported convergence")
	}
	if fit.Iterations != 1 {
		t.Fatalf("iterations = %d, want 1", fit.Iterations)
	}
}

func TestRegularizedRefusals(t *testing.T) {
	y, x1, x2 := regularizedFixture()
	for _, alpha := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, err := stats.RidgeRegression(y, alpha, x1, x2); err == nil {
			t.Fatalf("ridge accepted penalty %v", alpha)
		}
		if _, err := stats.LassoRegression(y, alpha, []insyra.IDataList{x1, x2}); err == nil {
			t.Fatalf("lasso accepted penalty %v", alpha)
		}
	}

	// Unreadable input is refused by the shared loader, with the row named.
	dirty := insyra.NewDataList(1.0, 2.0, nil, 4.0, 5.0, 6.0, 7.0, 8.0)
	target := insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0)
	if _, err := stats.RidgeRegression(target, 1.0, dirty); err == nil {
		t.Fatal("ridge accepted a predictor holding nil")
	}
	if _, err := stats.LassoRegression(target, 1.0, []insyra.IDataList{dirty}); err == nil {
		t.Fatal("lasso accepted a predictor holding nil")
	}
}

func TestRegularizedPredictMatchesManualComputation(t *testing.T) {
	y, x1, x2 := regularizedFixture()
	ridge, err := stats.RidgeRegression(y, 0.5, x1, x2)
	if err != nil {
		t.Fatal(err)
	}
	newX1 := insyra.NewDataList(1.0, 4.0)
	newX2 := insyra.NewDataList(2.0, 7.0)
	predicted, err := ridge.Predict(stats.PredictResponse, newX1, newX2)
	if err != nil {
		t.Fatal(err)
	}
	for i, xs := range [][2]float64{{1, 2}, {4, 7}} {
		want := ridge.Coefficients[0] + ridge.Coefficients[1]*xs[0] + ridge.Coefficients[2]*xs[1]
		got, _ := insyra.ToFloat64Safe(predicted.Get(i))
		if math.Abs(got-want) > 1e-12 {
			t.Fatalf("prediction %d = %v, want %v", i, got, want)
		}
	}

	// Wrong predictor count is refused.
	if _, err := ridge.Predict(stats.PredictResponse, newX1); err == nil {
		t.Fatal("a one-predictor request against a two-predictor fit was accepted")
	}
}

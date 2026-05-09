package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestCorrelationTwoPointInferenceIsUndefined(t *testing.T) {
	got, err := stats.Correlation(
		insyra.NewDataList([]float64{1, 2}),
		insyra.NewDataList([]float64{2, 4}),
		stats.PearsonCorrelation,
	)
	if err != nil {
		t.Fatalf("Correlation returned error: %v", err)
	}
	if got.DF == nil || *got.DF != 0 {
		t.Fatalf("DF = %v, want 0", got.DF)
	}
	if !math.IsNaN(got.PValue) {
		t.Fatalf("PValue = %v, want NaN for df=0", got.PValue)
	}
	if got.CI == nil || !math.IsNaN(got.CI[0]) || !math.IsNaN(got.CI[1]) {
		t.Fatalf("CI = %v, want [NaN NaN]", got.CI)
	}
}

func TestLinearRegressionDoesNotUseZeroFallbackForUndefinedInference(t *testing.T) {
	got, err := stats.LinearRegression(
		insyra.NewDataList([]float64{2, 4, 6, 8}),
		insyra.NewDataList([]float64{1, 2, 3, 4}),
	)
	if err != nil {
		t.Fatalf("LinearRegression returned error: %v", err)
	}
	if got.StandardErrorIntercept != 0 {
		t.Fatalf("intercept SE = %v, want exact zero for perfect fit", got.StandardErrorIntercept)
	}
	if !math.IsNaN(got.TValueIntercept) || !math.IsNaN(got.PValueIntercept) {
		t.Fatalf("zero intercept with zero SE should have undefined t/p, got t=%v p=%v", got.TValueIntercept, got.PValueIntercept)
	}
	if got.StandardError != 0 || !math.IsInf(got.TValue, 1) || got.PValue != 0 {
		t.Fatalf("nonzero slope with zero SE should produce Inf/0, got se=%v t=%v p=%v", got.StandardError, got.TValue, got.PValue)
	}
}

func TestMultipleLinearRegressionSimpleOnlyFieldsAreNaN(t *testing.T) {
	got, err := stats.LinearRegression(
		insyra.NewDataList([]float64{1, 2, 4, 7, 11}),
		insyra.NewDataList([]float64{1, 2, 3, 4, 5}),
		insyra.NewDataList([]float64{2, 1, 3, 5, 8}),
	)
	if err != nil {
		t.Fatalf("LinearRegression returned error: %v", err)
	}
	if !math.IsNaN(got.Slope) || !math.IsNaN(got.StandardError) || !math.IsNaN(got.TValue) || !math.IsNaN(got.PValue) {
		t.Fatalf("simple-regression scalar fields should be NaN for multiple regression, got slope=%v se=%v t=%v p=%v", got.Slope, got.StandardError, got.TValue, got.PValue)
	}
	if got.Intercept != got.Coefficients[0] {
		t.Fatalf("Intercept = %v, want coefficient[0] %v", got.Intercept, got.Coefficients[0])
	}
}

func TestPCARejectsZeroVarianceColumn(t *testing.T) {
	dt := insyra.NewDataTable()
	dt.AppendCols(insyra.NewDataList([]float64{1, 2, 3}).SetName("x"))
	dt.AppendCols(insyra.NewDataList([]float64{5, 5, 5}).SetName("constant"))

	if _, err := stats.PCA(dt, 1); err == nil {
		t.Fatalf("expected PCA to reject zero-variance column")
	}
}

func TestMomentShapeMethodsRejectUndefinedConstantCases(t *testing.T) {
	if got, err := stats.Skewness([]float64{3, 3, 3}); err == nil || !math.IsNaN(got) {
		t.Fatalf("Skewness constant data = %v, %v; want NaN error", got, err)
	}

	groups := []insyra.IDataList{
		insyra.NewDataList([]float64{1, 1, 1}),
		insyra.NewDataList([]float64{2, 3, 4}),
	}
	if _, err := stats.BartlettTest(groups); err == nil {
		t.Fatalf("expected BartlettTest to reject zero-variance group")
	}
}

func TestChiSquareRejectsUndefinedInputs(t *testing.T) {
	if _, err := stats.ChiSquareGoodnessOfFit(insyra.NewDataList(), nil, false); err == nil {
		t.Fatalf("expected goodness-of-fit to reject empty input")
	}
	if _, err := stats.ChiSquareGoodnessOfFit(insyra.NewDataList("a", "b"), []float64{0.2, 0.2}, false); err == nil {
		t.Fatalf("expected goodness-of-fit to reject probabilities that do not sum to 1")
	}
	if _, err := stats.ChiSquareIndependenceTest(insyra.NewDataList("a", "a"), insyra.NewDataList("x", "y")); err == nil {
		t.Fatalf("expected independence test to reject one row category")
	}
}

func TestZTestRejectsInvalidAlternative(t *testing.T) {
	if _, err := stats.SingleSampleZTest(insyra.NewDataList([]float64{1, 2, 3}), 0, 1, stats.AlternativeHypothesis("bad"), 0.95); err == nil {
		t.Fatalf("expected single-sample z-test to reject invalid alternative")
	}
	if _, err := stats.TwoSampleZTest(
		insyra.NewDataList([]float64{1, 2, 3}),
		insyra.NewDataList([]float64{2, 3, 4}),
		1,
		1,
		stats.AlternativeHypothesis("bad"),
		0.95,
	); err == nil {
		t.Fatalf("expected two-sample z-test to reject invalid alternative")
	}
}

func TestHypothesisTestsRejectInvalidConfidenceLevel(t *testing.T) {
	if _, err := stats.SingleSampleTTest(insyra.NewDataList([]float64{1, 2, 3}), 0, 1.2); err == nil {
		t.Fatalf("expected t-test to reject invalid confidence level")
	}
	if _, err := stats.TwoSampleTTest(
		insyra.NewDataList([]float64{1, 2, 3}),
		insyra.NewDataList([]float64{2, 3, 4}),
		false,
		0,
	); err == nil {
		t.Fatalf("expected two-sample t-test to reject invalid confidence level")
	}
	if _, err := stats.SingleSampleZTest(insyra.NewDataList([]float64{1, 2, 3}), 0, 1, stats.TwoSided, 1); err == nil {
		t.Fatalf("expected z-test to reject invalid confidence level")
	}
}

package ml_test

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

func TestBuiltInMetricsDeclareTheDirectionTheyActuallyHave(t *testing.T) {
	for _, tc := range []struct {
		metric ml.Metric
		want   ml.MetricDirection
	}{
		{ml.AccuracyMetric{}, ml.HigherIsBetter},
		{ml.R2Metric{}, ml.HigherIsBetter},
		{ml.ROCAUCMetric{}, ml.HigherIsBetter},
		{ml.RMSEMetric{}, ml.LowerIsBetter},
		{ml.MAEMetric{}, ml.LowerIsBetter},
		{ml.LogLossMetric{}, ml.LowerIsBetter},
		{ml.ConfusionMatrixMetric{}, ml.NoDirection},
	} {
		if got := tc.metric.Direction(); got != tc.want {
			t.Errorf("%s declares %v, want %v", tc.metric.Name(), got, tc.want)
		}
	}
}

// The case that silently picks the wrong model: two results whose means differ,
// where the answer flips with the metric's direction.
func TestBetterFollowsTheDeclaredDirection(t *testing.T) {
	loss := &ml.CrossValidationResult{Metric: "rmse", Direction: ml.LowerIsBetter, Mean: 0.4}
	worseLoss := &ml.CrossValidationResult{Metric: "rmse", Direction: ml.LowerIsBetter, Mean: 0.9}
	if better, err := ml.Better(loss, worseLoss); err != nil || !better {
		t.Fatalf("smaller loss not ranked better: better=%v err=%v", better, err)
	}
	if better, err := ml.Better(worseLoss, loss); err != nil || better {
		t.Fatalf("larger loss ranked better: better=%v err=%v", better, err)
	}

	gain := &ml.CrossValidationResult{Metric: "r2", Direction: ml.HigherIsBetter, Mean: 0.9}
	worseGain := &ml.CrossValidationResult{Metric: "r2", Direction: ml.HigherIsBetter, Mean: 0.4}
	if better, err := ml.Better(gain, worseGain); err != nil || !better {
		t.Fatalf("larger gain not ranked better: better=%v err=%v", better, err)
	}

	// Identical means with opposite directions must not both rank the same
	// way, or the direction is being ignored.
	loserAsGain := &ml.CrossValidationResult{Metric: "rmse", Direction: ml.HigherIsBetter, Mean: 0.4}
	otherAsGain := &ml.CrossValidationResult{Metric: "rmse", Direction: ml.HigherIsBetter, Mean: 0.9}
	asLoss, err := ml.Better(loss, worseLoss)
	if err != nil {
		t.Fatal(err)
	}
	asGain, err := ml.Better(loserAsGain, otherAsGain)
	if err != nil {
		t.Fatal(err)
	}
	if asLoss == asGain {
		t.Fatalf("the same means ranked identically under opposite directions; direction is not being read")
	}
}

func TestBetterRefusesWhatItCannotRank(t *testing.T) {
	rmse := &ml.CrossValidationResult{Metric: "rmse", Direction: ml.LowerIsBetter, Mean: 0.4}
	r2 := &ml.CrossValidationResult{Metric: "r2", Direction: ml.HigherIsBetter, Mean: 0.9}
	if _, err := ml.Better(rmse, r2); err == nil {
		t.Fatal("results from different metrics were compared")
	} else if !strings.Contains(err.Error(), "rmse") || !strings.Contains(err.Error(), "r2") {
		t.Fatalf("error %q does not name both metrics", err)
	}

	matrix := &ml.CrossValidationResult{Metric: "confusion_matrix", Direction: ml.NoDirection, Mean: math.NaN()}
	other := &ml.CrossValidationResult{Metric: "confusion_matrix", Direction: ml.NoDirection, Mean: math.NaN()}
	if _, err := ml.Better(matrix, other); err == nil {
		t.Fatal("directionless results were ranked")
	}
	if _, err := ml.Better(nil, rmse); err == nil {
		t.Fatal("a nil result was compared")
	}
}

func TestCrossValidationResultCarriesTheDirection(t *testing.T) {
	x, y := directionRegressionData()
	result, err := ml.CrossValidate(x, y, ml.Estimator{
		Name: "linear",
		Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitLinearRegression(x, y)
		},
	}, 4, ml.RMSEMetric{})
	if err != nil {
		t.Fatalf("cross-validate: %v", err)
	}
	if result.Direction != ml.LowerIsBetter {
		t.Fatalf("result carries %v, want %v", result.Direction, ml.LowerIsBetter)
	}
}

// A metric that returns a rankable number while declaring nothing about which
// way is better must be refused, not defaulted.
type directionlessScalarMetric struct{}

func (directionlessScalarMetric) Name() string                  { return "silent" }
func (directionlessScalarMetric) Kind() ml.MetricKind           { return ml.RegressionMetric }
func (directionlessScalarMetric) Direction() ml.MetricDirection { return ml.NoDirection }
func (directionlessScalarMetric) Evaluate(_ *insyra.DataList, _ ml.Prediction) (ml.MetricResult, error) {
	return ml.MetricResult{Name: "silent", Score: 0.5}, nil
}

func TestScalarScoreWithoutADirectionIsRefused(t *testing.T) {
	x, y := directionRegressionData()
	estimator := ml.Estimator{
		Name: "linear",
		Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitLinearRegression(x, y)
		},
	}
	if _, err := ml.CrossValidate(x, y, estimator, 4, directionlessScalarMetric{}); err == nil {
		t.Fatal("a scalar score with no direction was accepted")
	}

	model, err := ml.FitLinearRegression(x, y)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ml.Score(model, x, y, directionlessScalarMetric{}); err == nil {
		t.Fatal("Score accepted a scalar score with no direction")
	}
}

func directionRegressionData() (*insyra.DataTable, *insyra.DataList) {
	const n = 40
	xs := make([]any, n)
	ys := make([]any, n)
	for i := range xs {
		xs[i] = float64(i%9) + 1
		ys[i] = 2.5*(float64(i%9)+1) + 4 + 0.1*float64(i%3)
	}
	return insyra.NewDataTable(insyra.NewDataList(xs...).SetName("x")),
		insyra.NewDataList(ys...).SetName("y")
}

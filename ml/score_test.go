package ml_test

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

// Score must produce the metric's own value for that data, not a number of its
// own. Computing the same quantity by hand is what makes that checkable.
func TestScoreMatchesTheMetricComputedDirectly(t *testing.T) {
	x, y := directionRegressionData()
	model, err := ml.FitLinearRegression(x, y)
	if err != nil {
		t.Fatal(err)
	}
	predicted, err := model.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	want, err := ml.RMSE(y, predicted)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ml.Score(model, x, y, ml.RMSEMetric{})
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	if got.Score != want {
		t.Fatalf("Score returned %.17g, want %.17g", got.Score, want)
	}
	if got.Name != "rmse" {
		t.Fatalf("Score named the result %q, want %q", got.Name, "rmse")
	}
}

// Score takes a fitted model rather than an estimator, so refitting is
// impossible by construction. What is worth pinning is that it uses the model
// it was given unchanged: scoring a subset must equal the metric computed by
// hand from that same model's predictions over the subset, not from a model
// refitted on it.
func TestScoreUsesTheModelItWasGiven(t *testing.T) {
	x, y := directionRegressionData()
	model, err := ml.FitLinearRegression(x, y)
	if err != nil {
		t.Fatal(err)
	}

	const head = 12
	subsetX := insyra.NewDataTable(insyra.NewDataList(x.GetColByNumber(0).Data()[:head]...).SetName("x"))
	subsetY := insyra.NewDataList(y.Data()[:head]...)

	predicted, err := model.Predict(subsetX)
	if err != nil {
		t.Fatal(err)
	}
	want, err := ml.RMSE(subsetY, predicted)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ml.Score(model, subsetX, subsetY, ml.RMSEMetric{})
	if err != nil {
		t.Fatalf("score over a subset: %v", err)
	}
	if got.Score != want {
		t.Fatalf("Score returned %.17g over the subset, want %.17g from the same model", got.Score, want)
	}

	// A model refitted on those twelve rows would fit them more closely, so a
	// score equal to the refitted one would be the tell.
	refitted, err := ml.FitLinearRegression(subsetX, subsetY)
	if err != nil {
		t.Fatal(err)
	}
	refittedPredictions, err := refitted.Predict(subsetX)
	if err != nil {
		t.Fatal(err)
	}
	refittedRMSE, err := ml.RMSE(subsetY, refittedPredictions)
	if err != nil {
		t.Fatal(err)
	}
	if got.Score == refittedRMSE {
		t.Fatal("the subset score equals a model refitted on the subset")
	}
}

func TestScoreRefusesAMetricTheModelCannotServe(t *testing.T) {
	x, y := directionRegressionData()
	model, err := ml.FitLinearRegression(x, y)
	if err != nil {
		t.Fatal(err)
	}
	// A classification metric over a regression model is refused before any
	// prediction is made, on the same terms cross-validation refuses it.
	if _, err := ml.Score(model, x, y, ml.AccuracyMetric{}); err == nil {
		t.Fatal("a classification metric scored a regression model")
	}

	labels, features := scoreClassificationData()
	logistic, err := ml.FitLogisticRegression(features, labels)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ml.Score(logistic, features, labels, ml.RMSEMetric{}); err == nil {
		t.Fatal("a regression metric scored a classifier")
	}
}

// A metric that wants labels, over a model that reports probabilities, must get
// labels — derived the same way cross-validation derives them.
func TestScoreServesALabelMetricFromAProbabilityModel(t *testing.T) {
	labels, features := scoreClassificationData()
	model, err := ml.FitLogisticRegression(features, labels)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := ml.Score(model, features, labels, ml.AccuracyMetric{})
	if err != nil {
		t.Fatalf("score: %v", err)
	}

	// One fold covering every row, so cross-validation's held-out set is the
	// whole dataset and its per-fold score is comparable to Score's.
	viaCV, err := ml.CrossValidate(features, labels, ml.Estimator{
		Name: "logistic",
		Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitLogisticRegression(x, y)
		},
	}, 2, ml.AccuracyMetric{}, insyra.SamplingOptions{UseSeed: true, Seed: 7})
	if err != nil {
		t.Fatalf("cross-validate: %v", err)
	}
	if math.IsNaN(direct.Score) || math.IsNaN(viaCV.Mean) {
		t.Fatal("accuracy came back undefined from a model that reports probabilities")
	}
	if direct.Score < 0 || direct.Score > 1 {
		t.Fatalf("accuracy %v is outside [0,1]; labels were not derived", direct.Score)
	}
}

func TestScoreValidatesItsArguments(t *testing.T) {
	x, y := directionRegressionData()
	model, err := ml.FitLinearRegression(x, y)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ml.Score(nil, x, y, ml.RMSEMetric{}); err == nil {
		t.Fatal("a nil model was scored")
	}
	if _, err := ml.Score(model, x, nil, ml.RMSEMetric{}); err == nil {
		t.Fatal("a nil target was scored")
	}
	if _, err := ml.Score(model, x, y, nil); err == nil {
		t.Fatal("a nil metric was accepted")
	}
	short := insyra.NewDataList(y.Data()[:5]...)
	_, err = ml.Score(model, x, short, ml.RMSEMetric{})
	if err == nil {
		t.Fatal("mismatched lengths were accepted")
	}
	if !strings.Contains(err.Error(), "5") {
		t.Fatalf("error %q does not name both counts", err)
	}
}

func scoreClassificationData() (*insyra.DataList, *insyra.DataTable) {
	const n = 60
	xs := make([]any, n)
	labels := make([]any, n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			xs[i] = 1.0 + 0.1*float64(i%5)
			labels[i] = "churned"
		} else {
			xs[i] = 6.0 + 0.1*float64(i%5)
			labels[i] = "retained"
		}
	}
	return insyra.NewDataList(labels...).SetName("y"),
		insyra.NewDataTable(insyra.NewDataList(xs...).SetName("x"))
}

package ml_test

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

func TestKFoldPartitionsRowsAndIsReproducible(t *testing.T) {
	dt := insyra.NewDataTable(
		insyra.NewDataList(0, 1, 2, 3, 4, 5, 6, 7, 8, 9).SetName("id"),
	)
	options := insyra.SamplingOptions{UseSeed: true, Seed: 42}
	foldsA, err := ml.KFold(dt, 3, options)
	if err != nil {
		t.Fatal(err)
	}
	foldsB, err := ml.KFold(dt, 3, options)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int]bool{}
	for i, fold := range foldsA {
		if fold.NumRows() == 0 {
			t.Fatalf("fold %d is empty", i)
		}
		if fold.NumRows() != foldsB[i].NumRows() {
			t.Fatalf("seeded fold %d size changed", i)
		}
		for row := 0; row < fold.NumRows(); row++ {
			id := fold.GetElementByNumberIndex(row, 0).(int)
			if seen[id] {
				t.Fatalf("row %d appeared more than once", id)
			}
			seen[id] = true
			if id != foldsB[i].GetElementByNumberIndex(row, 0).(int) {
				t.Fatalf("seeded fold %d changed", i)
			}
		}
	}
	if len(seen) != dt.NumRows() {
		t.Fatalf("partition contains %d rows, want %d", len(seen), dt.NumRows())
	}
}

func TestStratifiedKFoldPreservesClassProportions(t *testing.T) {
	dt := insyra.NewDataTable(
		insyra.NewDataList(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11).SetName("id"),
	)
	labels := insyra.NewDataList("a", "a", "a", "a", "a", "a", "b", "b", "b", "b", "c", "c")
	folds, err := ml.StratifiedKFold(dt, labels, 2, insyra.SamplingOptions{UseSeed: true, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	for i, fold := range folds {
		counts := map[string]int{}
		for row := 0; row < fold.NumRows(); row++ {
			id := fold.GetElementByNumberIndex(row, 0).(int)
			counts[labels.Get(id).(string)]++
		}
		if counts["a"] != 3 || counts["b"] != 2 || counts["c"] != 1 {
			t.Fatalf("fold %d class counts = %v, want a=3,b=2,c=1", i, counts)
		}
	}
}

func TestStratifiedKFoldNamesTooSmallClass(t *testing.T) {
	dt := insyra.NewDataTable(insyra.NewDataList(1, 2, 3, 4).SetName("x"))
	labels := insyra.NewDataList("rare", "common", "common", "common")
	_, err := ml.StratifiedKFold(dt, labels, 2)
	if err == nil || !strings.Contains(err.Error(), "rare") {
		t.Fatalf("error = %v, want the undersized class name", err)
	}
}

func TestMetricsAgainstWorkedExamples(t *testing.T) {
	labels := insyra.NewDataList("a", "b", "a", "b")
	predictions := insyra.NewDataList("a", "a", "a", "b")
	accuracy, err := ml.Accuracy(labels, predictions)
	if err != nil || accuracy != 0.75 {
		t.Fatalf("accuracy = %v, err=%v, want 0.75", accuracy, err)
	}

	probabilities := insyra.NewDataTable(
		insyra.NewDataList(0.8, 0.2).SetName("a"),
		insyra.NewDataList(0.2, 0.8).SetName("b"),
	)
	logLoss, err := ml.LogLoss(insyra.NewDataList("a", "b"), probabilities)
	if err != nil || math.Abs(logLoss+math.Log(0.8)) > 1e-12 {
		t.Fatalf("log loss = %v, err=%v, want %v", logLoss, err, -math.Log(0.8))
	}

	aucProbabilities := insyra.NewDataTable(
		insyra.NewDataList(0.9, 0.6, 0.65, 0.2).SetName("negative"),
		insyra.NewDataList(0.1, 0.4, 0.35, 0.8).SetName("positive"),
	)
	auc, err := ml.ROCAUC(insyra.NewDataList("negative", "negative", "positive", "positive"), aucProbabilities)
	if err != nil || math.Abs(auc-0.75) > 1e-12 {
		t.Fatalf("ROC AUC = %v, err=%v, want 0.75", auc, err)
	}

	matrix, err := ml.ConfusionMatrix(labels, predictions)
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix.Labels) != 2 || matrix.Counts[0][0] != 2 || matrix.Counts[0][1] != 0 || matrix.Counts[1][0] != 1 || matrix.Counts[1][1] != 1 {
		t.Fatalf("confusion matrix = %#v", matrix)
	}

	actual := insyra.NewDataList(1, 2, 3)
	continuous := insyra.NewDataList(1, 4, 2)
	rmse, err := ml.RMSE(actual, continuous)
	if err != nil || math.Abs(rmse-math.Sqrt(5.0/3.0)) > 1e-12 {
		t.Fatalf("RMSE = %v, err=%v", rmse, err)
	}
	mae, err := ml.MAE(actual, continuous)
	if err != nil || math.Abs(mae-1) > 1e-12 {
		t.Fatalf("MAE = %v, err=%v, want 1", mae, err)
	}
	r2, err := ml.R2(actual, continuous)
	if err != nil || math.Abs(r2+1.5) > 1e-12 {
		t.Fatalf("R2 = %v, err=%v", r2, err)
	}
}

func TestCrossValidateRefitsPreprocessingAndReturnsFoldScores(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(0, 1, 2, 3, 100, 101).SetName("x"))
	y := insyra.NewDataList(0, 1, 2, 3, 100, 101)
	fitCount := 0
	trainingMeans := make([]float64, 0, 3)

	result, err := ml.CrossValidate(x, y, ml.Estimator{Name: "scaled", Fit: func(trainX *insyra.DataTable, _ *insyra.DataList) (ml.Model, error) {
		fitCount++
		scaler := insyra.NewStandardScaler()
		scaled, err := scaler.FitTransform(trainX, "x")
		if err != nil {
			return nil, err
		}
		values := scaled.GetColByName("x").ToF64Slice()
		mean := 0.0
		for _, value := range values {
			mean += value
		}
		trainingMeans = append(trainingMeans, mean/float64(len(values)))
		return &scaledModel{scaler: scaler}, nil
	}}, 3, ml.RMSEMetric{}, insyra.SamplingOptions{UseSeed: true, Seed: 11})
	if err != nil {
		t.Fatal(err)
	}
	if fitCount != 3 || len(trainingMeans) != 3 || len(result.Scores) != 3 || result.Metric != "rmse" {
		t.Fatalf("fitCount=%d means=%v scores=%v metric=%q", fitCount, trainingMeans, result.Scores, result.Metric)
	}
	for i, mean := range trainingMeans {
		if math.Abs(mean) > 1e-12 {
			t.Fatalf("fold %d preprocessing was not fitted on its training rows: scaled mean=%v", i+1, mean)
		}
	}
}

func TestCrossValidateReportsFailingFold(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(1, 2, 3, 4, 5, 6).SetName("x"))
	y := insyra.NewDataList(1, 2, 3, 4, 5, 6)
	fitCount := 0
	_, err := ml.CrossValidate(x, y, ml.Estimator{Name: "fails", Fit: func(*insyra.DataTable, *insyra.DataList) (ml.Model, error) {
		fitCount++
		if fitCount == 2 {
			return nil, errors.New("synthetic fit failure")
		}
		return identityModel{}, nil
	}}, 3, ml.RMSEMetric{}, insyra.SamplingOptions{UseSeed: true, Seed: 1})
	if err == nil || !strings.Contains(err.Error(), "fold 2") {
		t.Fatalf("error = %v, want fold 2", err)
	}
}

func TestCrossValidateRefusesMismatchedMetric(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(1, 2, 3, 4, 5, 6).SetName("x"))
	y := insyra.NewDataList(0, 0, 1, 1, 0, 1)
	_, err := ml.CrossValidate(x, y, ml.Estimator{Name: "linear", Fit: ml.FitLinearRegression}, 3, ml.AccuracyMetric{}, insyra.SamplingOptions{UseSeed: true, Seed: 1})
	if err == nil || !strings.Contains(err.Error(), "classification model") {
		t.Fatalf("error = %v, want classification mismatch", err)
	}
}

func TestCrossValidateUsesSuppliedMetricNameOnEveryFold(t *testing.T) {
	x := insyra.NewDataTable(insyra.NewDataList(1, 2, 3, 4).SetName("x"))
	y := insyra.NewDataList(1, 2, 3, 4)
	result, err := ml.CrossValidate(x, y, ml.Estimator{Name: "identity", Fit: func(*insyra.DataTable, *insyra.DataList) (ml.Model, error) {
		return identityModel{}, nil
	}}, 2, renamedMetric{}, insyra.SamplingOptions{UseSeed: true, Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	if result.Metric != "declared" || len(result.FoldResults) != 2 {
		t.Fatalf("result = %#v", result)
	}
	for _, fold := range result.FoldResults {
		if fold.Name != "declared" {
			t.Fatalf("fold metric name = %q, want declared", fold.Name)
		}
	}
}

type identityModel struct{}

func (identityModel) Features() []string { return []string{"x"} }
func (identityModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	return dt.GetColByName("x"), nil
}

type scaledModel struct{ scaler *insyra.StandardScaler }

func (m *scaledModel) Features() []string { return []string{"x"} }
func (m *scaledModel) Predict(dt *insyra.DataTable) (*insyra.DataList, error) {
	scaled, err := m.scaler.Transform(dt)
	if err != nil {
		return nil, err
	}
	return scaled.GetColByName("x"), nil
}

type renamedMetric struct{}

func (renamedMetric) Name() string                  { return "declared" }
func (renamedMetric) Kind() ml.MetricKind           { return ml.RegressionMetric }
func (renamedMetric) Direction() ml.MetricDirection { return ml.HigherIsBetter }
func (renamedMetric) Evaluate(*insyra.DataList, ml.Prediction) (ml.MetricResult, error) {
	return ml.MetricResult{Name: "wrong", Score: 1}, nil
}

// TestROCAUCRefusesLabelsOutsideTheClassSet pins a disagreement between two
// metrics over the same malformed input. ROC AUC used to treat any label that
// was not the positive class as negative, so a label belonging to neither
// reported confident discrimination — AUC 1 with a nil error — while log loss
// refused the identical input. A metric that cannot understand its input must
// say so rather than score it.
func TestROCAUCRefusesLabelsOutsideTheClassSet(t *testing.T) {
	classes := insyra.NewDataList("neg", "pos")
	probabilities := insyra.NewDataTable(
		insyra.NewDataList(0.9, 0.1, 0.5).SetName("neg"),
		insyra.NewDataList(0.1, 0.9, 0.5).SetName("pos"),
	)
	labels := insyra.NewDataList("neg", "pos", "TOTALLY_UNKNOWN")

	if _, err := ml.ROCAUC(labels, probabilities, classes); err == nil {
		t.Fatal("ROC AUC accepted a label belonging to neither class")
	}
	if _, err := ml.LogLoss(labels, probabilities, classes); err == nil {
		t.Fatal("log loss accepted a label belonging to neither class")
	}

	// The well-formed case still scores.
	good := insyra.NewDataList("neg", "pos", "pos")
	if _, err := ml.ROCAUC(good, probabilities, classes); err != nil {
		t.Fatalf("ROC AUC refused a well-formed input: %v", err)
	}
}

// externalProbabilityMetric is defined here rather than inside the package, so
// that it can only use what a third party can use. Before the markers were
// exported it could not have declared its requirement at all, and would have
// been handed Predict output while silently expecting probabilities.
type externalProbabilityMetric struct{ seen ml.Prediction }

func (*externalProbabilityMetric) Name() string                  { return "external_proba" }
func (*externalProbabilityMetric) Kind() ml.MetricKind           { return ml.ClassificationMetric }
func (*externalProbabilityMetric) Direction() ml.MetricDirection { return ml.HigherIsBetter }
func (*externalProbabilityMetric) NeedsProbabilities() bool      { return true }
func (m *externalProbabilityMetric) Evaluate(yTrue *insyra.DataList, p ml.Prediction) (ml.MetricResult, error) {
	m.seen = p
	return ml.MetricResult{Name: "external_proba", Score: 1}, nil
}

// externalPlainMetric declares nothing, and must keep receiving what it always did.
type externalPlainMetric struct{ seen ml.Prediction }

func (*externalPlainMetric) Name() string                  { return "external_plain" }
func (*externalPlainMetric) Kind() ml.MetricKind           { return ml.ClassificationMetric }
func (*externalPlainMetric) Direction() ml.MetricDirection { return ml.HigherIsBetter }
func (m *externalPlainMetric) Evaluate(yTrue *insyra.DataList, p ml.Prediction) (ml.MetricResult, error) {
	m.seen = p
	return ml.MetricResult{Name: "external_plain", Score: 1}, nil
}

// externalDeclinedMetric implements the interface and answers false, which must
// be treated the same as not implementing it — the declaration is read, not
// merely the presence of the method.
type externalDeclinedMetric struct{ seen ml.Prediction }

func (*externalDeclinedMetric) Name() string                  { return "external_declined" }
func (*externalDeclinedMetric) Kind() ml.MetricKind           { return ml.ClassificationMetric }
func (*externalDeclinedMetric) Direction() ml.MetricDirection { return ml.HigherIsBetter }
func (*externalDeclinedMetric) NeedsProbabilities() bool      { return false }
func (m *externalDeclinedMetric) Evaluate(yTrue *insyra.DataList, p ml.Prediction) (ml.MetricResult, error) {
	m.seen = p
	return ml.MetricResult{Name: "external_declined", Score: 1}, nil
}

func labelledFixture() (*insyra.DataTable, *insyra.DataList, ml.Estimator) {
	n := 40
	a := make([]any, n)
	lab := make([]any, n)
	for i := range a {
		a[i] = float64(i)
		if i < n/2 {
			lab[i] = "no"
		} else {
			lab[i] = "yes"
		}
	}
	x := insyra.NewDataTable(insyra.NewDataList(a...).SetName("a"))
	y := insyra.NewDataList(lab...)
	est := ml.Estimator{Name: "logit", Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
		return ml.FitLogisticRegression(x, y)
	}}
	return x, y, est
}

// TestExternalMetricCanRequestProbabilities is the point of exporting the
// markers. A metric outside the package declares it needs probabilities and
// receives them; the extension point extends.
func TestExternalMetricCanRequestProbabilities(t *testing.T) {
	x, y, est := labelledFixture()
	metric := &externalProbabilityMetric{}
	if _, err := ml.CrossValidate(x, y, est, 2, metric); err != nil {
		t.Fatalf("cross-validate: %v", err)
	}
	if metric.seen.Probabilities == nil {
		t.Fatal("an external metric declaring it needs probabilities did not receive them")
	}
	if metric.seen.Classes == nil {
		t.Fatal("probabilities arrived without the classes naming their columns")
	}
}

// A metric that declares nothing keeps receiving predictions, as before.
func TestExternalMetricDeclaringNothingGetsPredictions(t *testing.T) {
	x, y, est := labelledFixture()
	metric := &externalPlainMetric{}
	if _, err := ml.CrossValidate(x, y, est, 2, metric); err != nil {
		t.Fatalf("cross-validate: %v", err)
	}
	if metric.seen.Values == nil {
		t.Fatal("a metric declaring nothing should still receive the model's predictions")
	}
	if metric.seen.Probabilities != nil {
		t.Fatal("a metric declaring nothing should not receive probabilities")
	}
}

// Answering false must mean the same as not asking.
func TestExternalMetricDecliningIsTreatedAsNotAsking(t *testing.T) {
	x, y, est := labelledFixture()
	metric := &externalDeclinedMetric{}
	if _, err := ml.CrossValidate(x, y, est, 2, metric); err != nil {
		t.Fatalf("cross-validate: %v", err)
	}
	if metric.seen.Probabilities != nil {
		t.Fatal("a metric answering false was handed probabilities anyway; the declaration is not being read")
	}
	if metric.seen.Values == nil {
		t.Fatal("a metric answering false should receive predictions")
	}
}

// A model that cannot produce probabilities is refused, not scored on values of
// a different kind.
func TestProbabilityMetricRefusesAModelWithoutProbabilities(t *testing.T) {
	n := 30
	a := make([]any, n)
	yv := make([]any, n)
	for i := range a {
		a[i] = float64(i)
		yv[i] = float64(i) * 2
	}
	x := insyra.NewDataTable(insyra.NewDataList(a...).SetName("a"))
	y := insyra.NewDataList(yv...)
	est := ml.Estimator{Name: "linear", Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
		return ml.FitLinearRegression(x, y)
	}}
	if _, err := ml.CrossValidate(x, y, est, 2, &externalProbabilityMetric{}); err == nil {
		t.Fatal("a probability metric was accepted against a model that produces none")
	}
}

// TestRegressionMetricRefusesAClusterer pins the reason Clusterer exists.
// KMeansModel.Predict returns cluster ids and KMeansModel has no Classes(), so
// nothing used to refuse an RMSE over them — the result was a correct number
// about nothing.
func TestRegressionMetricRefusesAClusterer(t *testing.T) {
	n := 24
	a := make([]any, n)
	b := make([]any, n)
	y := make([]any, n)
	for i := range a {
		// Two separated blobs, so every cross-validation fold can fit two
		// clusters without emptying one.
		if i < n/2 {
			a[i], b[i] = float64(i%4)*0.3, float64(i/4)*0.3
		} else {
			a[i], b[i] = 30+float64(i%4)*0.3, 30+float64(i/4)*0.3
		}
		y[i] = float64(i % 2)
	}
	x := insyra.NewDataTable(
		insyra.NewDataList(a...).SetName("a"),
		insyra.NewDataList(b...).SetName("b"),
	)
	labels := insyra.NewDataList(y...)

	est := ml.Estimator{Name: "kmeans", Fit: func(x *insyra.DataTable, _ *insyra.DataList) (ml.Model, error) {
		return ml.FitKMeans(x, 2)
	}}
	if _, err := ml.CrossValidate(x, labels, est, 3, ml.RMSEMetric{}); err == nil {
		t.Fatal("a regression metric scored a clustering model's cluster ids")
	}
}

// A clusterer reports the number of groups it converged on.
func TestClustererReportsItsClusterCount(t *testing.T) {
	// Three well-separated blobs, so the fit converges on three clusters rather
	// than emptying one — a degenerate fixture makes this test flaky, not wrong.
	centres := [][2]float64{{0, 0}, {20, 20}, {40, 0}}
	var a, b []any
	for _, c := range centres {
		for k := 0; k < 6; k++ {
			a = append(a, c[0]+float64(k%3)*0.4)
			b = append(b, c[1]+float64(k/3)*0.4)
		}
	}
	x := insyra.NewDataTable(
		insyra.NewDataList(a...).SetName("a"),
		insyra.NewDataList(b...).SetName("b"),
	)
	model, err := ml.FitKMeans(x, 3)
	if err != nil {
		t.Fatalf("fit: %v", err)
	}
	clusterer, ok := any(model).(ml.Clusterer)
	if !ok {
		t.Fatal("a fitted KMeans model does not declare itself a Clusterer")
	}
	if got := clusterer.Clusters(); got != 3 {
		t.Fatalf("Clusters() = %d, want 3", got)
	}
}

// A model declaring neither is scored exactly as before.
func TestModelDeclaringNeitherIsStillScored(t *testing.T) {
	n := 30
	a := make([]any, n)
	yv := make([]any, n)
	for i := range a {
		a[i] = float64(i)
		yv[i] = float64(i)*2 + 1
	}
	x := insyra.NewDataTable(insyra.NewDataList(a...).SetName("a"))
	y := insyra.NewDataList(yv...)
	est := ml.Estimator{Name: "linear", Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
		return ml.FitLinearRegression(x, y)
	}}
	if _, err := ml.CrossValidate(x, y, est, 3, ml.RMSEMetric{}); err != nil {
		t.Fatalf("a plain regression model should still be scored: %v", err)
	}
}

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

func (renamedMetric) Name() string        { return "declared" }
func (renamedMetric) Kind() ml.MetricKind { return ml.RegressionMetric }
func (renamedMetric) Evaluate(*insyra.DataList, ml.Prediction) (ml.MetricResult, error) {
	return ml.MetricResult{Name: "wrong", Score: 1}, nil
}

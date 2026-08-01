package ml_test

import (
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/reftest"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/ml/mltest"
)

func boostingClassificationData() (*insyra.DataTable, *insyra.DataList) {
	const n = 100
	f1 := make([]any, n)
	f2 := make([]any, n)
	labels := make([]any, n)
	for i := 0; i < n; i++ {
		v1 := math.Sin(float64(i)*0.7)*3 + float64(i%5)
		v2 := math.Cos(float64(i)*1.1)*2 - float64(i%3)
		f1[i] = v1
		f2[i] = v2
		if v1-v2 > 1.5 {
			labels[i] = "keep"
		} else {
			labels[i] = "drop"
		}
	}
	return insyra.NewDataTable(
			insyra.NewDataList(f1...).SetName("f1"),
			insyra.NewDataList(f2...).SetName("f2"),
		),
		insyra.NewDataList(labels...).SetName("y")
}

func TestGradientBoostingPassesConformance(t *testing.T) {
	x, y := boostingClassificationData()
	classifier, err := ml.FitGradientBoostingClassifier(x, y, ml.GradientBoostingOptions{Stages: 30})
	if err != nil {
		t.Fatal(err)
	}
	mltest.RunConformance(t, classifier, x, y)

	rx, ry := forestRegressionData()
	regressor, err := ml.FitGradientBoostingRegressor(rx, ry, ml.GradientBoostingOptions{Stages: 30})
	if err != nil {
		t.Fatal(err)
	}
	mltest.RunConformance(t, regressor, rx, nil)
}

// Boosting is deterministic — no bootstrap, no feature subsampling — so two
// fits must agree exactly with no seed involved.
func TestGradientBoostingIsDeterministic(t *testing.T) {
	x, y := forestRegressionData()
	first, err := ml.FitGradientBoostingRegressor(x, y)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ml.FitGradientBoostingRegressor(x, y)
	if err != nil {
		t.Fatal(err)
	}
	firstPredictions, err := first.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	secondPredictions, err := second.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstPredictions.Data(), secondPredictions.Data()) {
		t.Fatal("two identical fits disagree")
	}
}

// Each stage fits what the previous stages left unexplained, so more stages
// must not fit the training data worse — the property that makes it boosting
// rather than an ensemble of restarts.
func TestGradientBoostingStagesImproveTrainingFit(t *testing.T) {
	x, y := forestRegressionData()
	shallow, err := ml.FitGradientBoostingRegressor(x, y, ml.GradientBoostingOptions{Stages: 5})
	if err != nil {
		t.Fatal(err)
	}
	deep, err := ml.FitGradientBoostingRegressor(x, y, ml.GradientBoostingOptions{Stages: 200})
	if err != nil {
		t.Fatal(err)
	}
	shallowScore, err := ml.Score(shallow, x, y, ml.RMSEMetric{})
	if err != nil {
		t.Fatal(err)
	}
	deepScore, err := ml.Score(deep, x, y, ml.RMSEMetric{})
	if err != nil {
		t.Fatal(err)
	}
	if deepScore.Score > shallowScore.Score {
		t.Fatalf("200 stages fit training data worse than 5 (RMSE %v vs %v)", deepScore.Score, shallowScore.Score)
	}
	if deepR2, err := ml.Score(deep, x, y, ml.R2Metric{}); err != nil || deepR2.Score < 0.95 {
		t.Fatalf("boosted R² = %v (err %v) on a nearly deterministic surface", deepR2.Score, err)
	}
}

func TestGradientBoostingClassifierSeparatesTheClasses(t *testing.T) {
	x, y := boostingClassificationData()
	model, err := ml.FitGradientBoostingClassifier(x, y, ml.GradientBoostingOptions{Stages: 100})
	if err != nil {
		t.Fatal(err)
	}
	score, err := ml.Score(model, x, y, ml.AccuracyMetric{})
	if err != nil {
		t.Fatal(err)
	}
	if score.Score < 0.95 {
		t.Fatalf("training accuracy %v on a separable boundary", score.Score)
	}
	probabilities, err := model.PredictProba(x)
	if err != nil {
		t.Fatal(err)
	}
	for row := 0; row < probabilities.NumRows(); row++ {
		total := 0.0
		for col := 0; col < probabilities.NumCols(); col++ {
			value, _ := insyra.ToFloat64Safe(probabilities.GetElementByNumberIndex(row, col))
			total += value
		}
		if math.Abs(total-1) > 1e-9 {
			t.Fatalf("row %d probabilities sum to %v", row, total)
		}
	}
}

func TestGradientBoostingRefusals(t *testing.T) {
	x, y := boostingClassificationData()
	if _, err := ml.FitGradientBoostingClassifier(x, y, ml.GradientBoostingOptions{Stages: -1}); err == nil {
		t.Fatal("a negative stage count was accepted")
	}
	if _, err := ml.FitGradientBoostingClassifier(x, y, ml.GradientBoostingOptions{LearningRate: -0.1}); err == nil {
		t.Fatal("a negative learning rate was accepted")
	}

	// Three classes: refused with the limit named, not approximated.
	tx, ty := forestClassificationData()
	_, err := ml.FitGradientBoostingClassifier(tx, ty)
	if err == nil {
		t.Fatal("a three-class target was accepted by a binary-only booster")
	}
	if !strings.Contains(err.Error(), "binary") {
		t.Fatalf("error %q does not name the binary limit", err)
	}
}

func TestGradientBoostingAccuracyAgainstScikitLearn(t *testing.T) {
	const verification = "the gradient-boosting comparison against scikit-learn"
	if os.Getenv("INSYRA_RUN_ML_SKLEARN") != "1" && !reftest.Strict() {
		t.Skipf("set INSYRA_RUN_ML_SKLEARN=1 or %s=1 to run %s", reftest.StrictEnv, verification)
	}
	xTrain := [][]float64{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {4, 4}, {4, 5}, {5, 4}, {5, 5}}
	yTrain := []any{0, 0, 0, 0, 1, 1, 1, 1}
	xTest := [][]float64{{0.5, 0.5}, {4.5, 4.5}, {0.2, 0.8}, {4.8, 4.2}}
	yTest := []any{0, 1, 0, 1}

	train := tableFromRows(xTrain, []string{"x1", "x2"})
	test := tableFromRows(xTest, []string{"x1", "x2"})
	model, err := ml.FitGradientBoostingClassifier(train, insyra.NewDataList(yTrain...), ml.GradientBoostingOptions{Stages: 50})
	if err != nil {
		t.Fatal(err)
	}
	predicted, err := model.Predict(test)
	if err != nil {
		t.Fatal(err)
	}
	goAccuracy := accuracy(predicted.Data(), yTest)

	script := "import json; from sklearn.ensemble import GradientBoostingClassifier; X=[[0,0],[0,1],[1,0],[1,1],[4,4],[4,5],[5,4],[5,5]]; y=[0,0,0,0,1,1,1,1]; xt=[[0.5,0.5],[4.5,4.5],[0.2,0.8],[4.8,4.2]]; yt=[0,1,0,1]; m=GradientBoostingClassifier(n_estimators=50, random_state=0).fit(X,y); print(json.dumps(sum(a==b for a,b in zip(m.predict(xt),yt))/len(yt)))"
	pythonCommand := "python"
	if _, err := exec.LookPath(pythonCommand); err != nil {
		pythonCommand = "python3"
	}
	output, err := exec.Command(pythonCommand, "-c", script).Output()
	if err != nil {
		reftest.Missing(t, "python with scikit-learn", verification, err)
	}
	var sklearnAccuracy float64
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &sklearnAccuracy); err != nil {
		t.Fatal(err)
	}
	if goAccuracy != sklearnAccuracy {
		t.Fatalf("accuracy %v vs scikit-learn %v on cleanly separable data", goAccuracy, sklearnAccuracy)
	}
}

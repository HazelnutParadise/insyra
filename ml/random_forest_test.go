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

// Two informative features and one pure-noise feature over three classes.
func forestClassificationData() (*insyra.DataTable, *insyra.DataList) {
	const n = 120
	f1 := make([]any, n)
	f2 := make([]any, n)
	noise := make([]any, n)
	labels := make([]any, n)
	for i := 0; i < n; i++ {
		class := i % 3
		f1[i] = float64(class*4) + math.Sin(float64(i)*1.7)
		f2[i] = float64(class*-3) + math.Cos(float64(i)*2.3)
		noise[i] = math.Sin(float64(i) * 12.9)
		labels[i] = []string{"ant", "bee", "cat"}[class]
	}
	return insyra.NewDataTable(
			insyra.NewDataList(f1...).SetName("f1"),
			insyra.NewDataList(f2...).SetName("f2"),
			insyra.NewDataList(noise...).SetName("noise"),
		),
		insyra.NewDataList(labels...).SetName("y")
}

func forestRegressionData() (*insyra.DataTable, *insyra.DataList) {
	const n = 120
	f1 := make([]any, n)
	f2 := make([]any, n)
	y := make([]any, n)
	for i := 0; i < n; i++ {
		v1 := float64(i % 11)
		v2 := float64((i * 7) % 13)
		f1[i] = v1
		f2[i] = v2
		y[i] = 2*v1 - v2 + 0.1*math.Sin(float64(i))
	}
	return insyra.NewDataTable(
			insyra.NewDataList(f1...).SetName("f1"),
			insyra.NewDataList(f2...).SetName("f2"),
		),
		insyra.NewDataList(y...).SetName("y")
}

func TestRandomForestPassesConformance(t *testing.T) {
	x, y := forestClassificationData()
	seed := int64(7)
	classifier, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{Trees: 25, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	mltest.RunConformance(t, classifier, x, y)

	rx, ry := forestRegressionData()
	regressor, err := ml.FitRandomForestRegressor(rx, ry, ml.RandomForestOptions{Trees: 25, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	mltest.RunConformance(t, regressor, rx, nil)
}

// Trees fit in parallel, so determinism must come from the seed alone, not
// from scheduling. Two fits with the same seed must agree exactly; the drawn
// seed reported on the model must reproduce an unseeded fit.
func TestRandomForestSeedDeterminesTheForest(t *testing.T) {
	x, y := forestClassificationData()
	seed := int64(42)
	first, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{Trees: 30, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{Trees: 30, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	firstProba, err := first.PredictProba(x)
	if err != nil {
		t.Fatal(err)
	}
	secondProba, err := second.PredictProba(x)
	if err != nil {
		t.Fatal(err)
	}
	for col := 0; col < firstProba.NumCols(); col++ {
		if !reflect.DeepEqual(firstProba.GetColByNumber(col).Data(), secondProba.GetColByNumber(col).Data()) {
			t.Fatalf("column %d differs between two fits with the same seed; parallelism leaked into the result", col)
		}
	}
	if !reflect.DeepEqual(first.FeatureImportances(), second.FeatureImportances()) {
		t.Fatal("importances differ between two fits with the same seed")
	}

	unseeded, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{Trees: 10})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{Trees: 10, Seed: &unseeded.Seed})
	if err != nil {
		t.Fatal(err)
	}
	unseededPredictions, err := unseeded.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	replayedPredictions, err := replayed.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(unseededPredictions.Data(), replayedPredictions.Data()) {
		t.Fatal("the reported seed did not reproduce the unseeded forest")
	}
}

func TestRandomForestProbabilitiesAreAverages(t *testing.T) {
	x, y := forestClassificationData()
	seed := int64(3)
	forest, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{Trees: 20, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	probabilities, err := forest.PredictProba(x)
	if err != nil {
		t.Fatal(err)
	}
	for row := 0; row < probabilities.NumRows(); row++ {
		total := 0.0
		for col := 0; col < probabilities.NumCols(); col++ {
			value, _ := insyra.ToFloat64Safe(probabilities.GetElementByNumberIndex(row, col))
			if value < 0 || value > 1 {
				t.Fatalf("row %d col %d probability %v outside [0,1]", row, col, value)
			}
			total += value
		}
		if math.Abs(total-1) > 1e-9 {
			t.Fatalf("row %d probabilities sum to %v", row, total)
		}
	}
}

// The informative features must outrank pure noise — the property importances
// exist to report.
func TestRandomForestImportancesRankSignalAboveNoise(t *testing.T) {
	x, y := forestClassificationData()
	seed := int64(5)
	forest, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{Trees: 40, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	importances := forest.FeatureImportances()
	if len(importances) != 3 {
		t.Fatalf("%d importances for 3 features", len(importances))
	}
	total := 0.0
	for _, value := range importances {
		total += value
	}
	if math.Abs(total-1) > 1e-9 {
		t.Fatalf("importances sum to %v, want 1", total)
	}
	// features: f1, f2, noise — in fitted order.
	if importances[2] >= importances[0] || importances[2] >= importances[1] {
		t.Fatalf("noise (%v) outranked signal (%v, %v)", importances[2], importances[0], importances[1])
	}
}

func TestRandomForestRegressorFitsTheSurface(t *testing.T) {
	x, y := forestRegressionData()
	seed := int64(11)
	forest, err := ml.FitRandomForestRegressor(x, y, ml.RandomForestOptions{Trees: 50, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	score, err := ml.Score(forest, x, y, ml.R2Metric{})
	if err != nil {
		t.Fatal(err)
	}
	if score.Score < 0.9 {
		t.Fatalf("forest R² = %v on a nearly deterministic surface", score.Score)
	}
}

func TestRandomForestRefusals(t *testing.T) {
	x, y := forestClassificationData()
	if _, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{Trees: -3}); err == nil {
		t.Fatal("a negative tree count was accepted")
	}
	if _, err := ml.FitRandomForestClassifier(x, y, ml.RandomForestOptions{MaxFeatures: -1}); err == nil {
		t.Fatal("a negative feature budget was accepted")
	}
}

// On cleanly separable data both forests should classify the held-out points
// perfectly; agreement at accuracy level is what is checkable across two
// implementations whose bootstrap draws cannot align.
func TestRandomForestAccuracyAgainstScikitLearn(t *testing.T) {
	const verification = "the random-forest comparison against scikit-learn"
	if os.Getenv("INSYRA_RUN_ML_SKLEARN") != "1" && !reftest.Strict() {
		t.Skipf("set INSYRA_RUN_ML_SKLEARN=1 or %s=1 to run %s", reftest.StrictEnv, verification)
	}
	xTrain := [][]float64{{0, 0}, {0, 1}, {1, 0}, {1, 1}, {4, 4}, {4, 5}, {5, 4}, {5, 5}}
	yTrain := []any{0, 0, 0, 0, 1, 1, 1, 1}
	xTest := [][]float64{{0.5, 0.5}, {4.5, 4.5}, {0.2, 0.8}, {4.8, 4.2}}
	yTest := []any{0, 1, 0, 1}

	train := tableFromRows(xTrain, []string{"x1", "x2"})
	test := tableFromRows(xTest, []string{"x1", "x2"})
	seed := int64(1)
	forest, err := ml.FitRandomForestClassifier(train, insyra.NewDataList(yTrain...), ml.RandomForestOptions{Trees: 50, Seed: &seed})
	if err != nil {
		t.Fatal(err)
	}
	predicted, err := forest.Predict(test)
	if err != nil {
		t.Fatal(err)
	}
	goAccuracy := accuracy(predicted.Data(), yTest)

	script := "import json; from sklearn.ensemble import RandomForestClassifier; X=[[0,0],[0,1],[1,0],[1,1],[4,4],[4,5],[5,4],[5,5]]; y=[0,0,0,0,1,1,1,1]; xt=[[0.5,0.5],[4.5,4.5],[0.2,0.8],[4.8,4.2]]; yt=[0,1,0,1]; m=RandomForestClassifier(n_estimators=50, random_state=0).fit(X,y); print(json.dumps(sum(a==b for a,b in zip(m.predict(xt),yt))/len(yt)))"
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

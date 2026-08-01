package ml_test

import (
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/internal/reftest"
	"github.com/HazelnutParadise/insyra/ml"
)

// The case that shows the option does something: the true boundary sits at
// 3.5, deep inside the first quantile bin of 200 distinct values, so the
// histogram search cannot place a split there and the exact search can.
func TestExactSplitsFindBoundariesQuantileEdgesMiss(t *testing.T) {
	const n = 200
	xs := make([]any, n)
	ys := make([]any, n)
	for i := 0; i < n; i++ {
		xs[i] = float64(i)
		if float64(i) < 3.5 {
			ys[i] = "low"
		} else {
			ys[i] = "high"
		}
	}
	x := insyra.NewDataTable(insyra.NewDataList(xs...).SetName("x"))
	y := insyra.NewDataList(ys...)

	exact, err := ml.FitDecisionTreeClassifier(x, y, ml.DecisionTreeOptions{ExactSplits: true, MaxDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	histogram, err := ml.FitDecisionTreeClassifier(x, y, ml.DecisionTreeOptions{MaxDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	exactScore, err := ml.Score(exact, x, y, ml.AccuracyMetric{})
	if err != nil {
		t.Fatal(err)
	}
	histogramScore, err := ml.Score(histogram, x, y, ml.AccuracyMetric{})
	if err != nil {
		t.Fatal(err)
	}
	if exactScore.Score != 1 {
		t.Fatalf("exact splitting missed a clean boundary: accuracy %v", exactScore.Score)
	}
	if histogramScore.Score == 1 {
		t.Fatal("the histogram search also found the boundary; this fixture no longer distinguishes the two searches")
	}
}

func TestExactSplitsRefusesABinCap(t *testing.T) {
	x, y := forestClassificationData()
	_, err := ml.FitDecisionTreeClassifier(x, y, ml.DecisionTreeOptions{ExactSplits: true, MaxBins: 64})
	if err == nil {
		t.Fatal("ExactSplits combined with MaxBins was accepted")
	}
	if !strings.Contains(err.Error(), "ExactSplits") {
		t.Fatalf("error %q does not name the conflict", err)
	}
}

func TestEnsemblesInheritExactSplits(t *testing.T) {
	x, y := forestRegressionData()
	seed := int64(4)
	if _, err := ml.FitRandomForestRegressor(x, y, ml.RandomForestOptions{
		Trees: 5, Seed: &seed, Tree: ml.DecisionTreeOptions{ExactSplits: true, MaxDepth: 4},
	}); err != nil {
		t.Fatalf("forest with exact splits: %v", err)
	}
	if _, err := ml.FitGradientBoostingRegressor(x, y, ml.GradientBoostingOptions{
		Stages: 5, Tree: ml.DecisionTreeOptions{ExactSplits: true, MaxDepth: 3},
	}); err != nil {
		t.Fatalf("boosting with exact splits: %v", err)
	}
}

// Prediction for prediction against scikit-learn, not merely the same
// accuracy: same data, same depth, exact splitting on both sides — the same
// tree must come out.
func TestExactTreeMatchesScikitLearnPredictions(t *testing.T) {
	const verification = "the exact-tree comparison against scikit-learn"
	if os.Getenv("INSYRA_RUN_ML_SKLEARN") != "1" && !reftest.Strict() {
		t.Skipf("set INSYRA_RUN_ML_SKLEARN=1 or %s=1 to run %s", reftest.StrictEnv, verification)
	}

	// Two features, irregular values, labels from nested thresholds with
	// distinct split gains so tie-breaking never decides the tree.
	const n = 60
	f1 := make([]float64, n)
	f2 := make([]float64, n)
	labels := make([]int, n)
	targets := make([]float64, n)
	for i := 0; i < n; i++ {
		f1[i] = math.Sin(float64(i)*1.3)*10 + float64(i%7)
		f2[i] = math.Cos(float64(i)*0.7)*5 + float64(i%4)
		if f1[i] > 2.5 {
			if f2[i] > 1.0 {
				labels[i] = 2
			} else {
				labels[i] = 1
			}
		} else {
			labels[i] = 0
		}
		// Smooth, so every candidate split has a generically distinct
		// variance gain — a piecewise-constant target ties many candidates
		// and a tie is resolved by convention, not by mathematics, so the
		// two implementations may legitimately differ there.
		targets[i] = math.Sin(f1[i]*0.6)*3 + f2[i]*0.7 - math.Cos(f2[i]*0.4)
	}
	probe1 := make([]float64, 0, 15*15)
	probe2 := make([]float64, 0, 15*15)
	for a := 0; a < 15; a++ {
		for b := 0; b < 15; b++ {
			probe1 = append(probe1, -12+float64(a)*2.0)
			probe2 = append(probe2, -8+float64(b)*1.5)
		}
	}

	toAny := func(values []float64) []any {
		out := make([]any, len(values))
		for i, v := range values {
			out[i] = v
		}
		return out
	}
	train := insyra.NewDataTable(
		insyra.NewDataList(toAny(f1)...).SetName("f1"),
		insyra.NewDataList(toAny(f2)...).SetName("f2"),
	)
	probe := insyra.NewDataTable(
		insyra.NewDataList(toAny(probe1)...).SetName("f1"),
		insyra.NewDataList(toAny(probe2)...).SetName("f2"),
	)
	labelValues := make([]any, n)
	targetValues := make([]any, n)
	for i := 0; i < n; i++ {
		labelValues[i] = labels[i]
		targetValues[i] = targets[i]
	}

	classifier, err := ml.FitDecisionTreeClassifier(train, insyra.NewDataList(labelValues...), ml.DecisionTreeOptions{ExactSplits: true, MaxDepth: 4})
	if err != nil {
		t.Fatal(err)
	}
	// Depth 3 with a 5-sample leaf floor, matched on both sides. Deeper trees
	// reach nodes of two or three rows, where several splits remove all
	// variance and tie exactly; scikit-learn breaks such ties by a per-node
	// random feature order, which no deterministic implementation can match.
	// The comparison stays where the mathematics, not the coin, decides.
	regressor, err := ml.FitDecisionTreeRegressor(train, insyra.NewDataList(targetValues...), ml.DecisionTreeOptions{ExactSplits: true, MaxDepth: 3, MinSamplesLeaf: 5})
	if err != nil {
		t.Fatal(err)
	}
	classifierPredictions, err := classifier.Predict(probe)
	if err != nil {
		t.Fatal(err)
	}
	regressorPredictions, err := regressor.Predict(probe)
	if err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{
		"f1": f1, "f2": f2, "labels": labels, "targets": targets,
		"probe1": probe1, "probe2": probe2,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	script := `
import json, sys
import numpy as np
from sklearn.tree import DecisionTreeClassifier, DecisionTreeRegressor
data = json.loads(sys.argv[1])
X = np.column_stack([data["f1"], data["f2"]]).astype(np.float32)
P = np.column_stack([data["probe1"], data["probe2"]]).astype(np.float32)
c = DecisionTreeClassifier(max_depth=4, random_state=0).fit(X, data["labels"])
r = DecisionTreeRegressor(max_depth=3, min_samples_leaf=5, random_state=0).fit(X, data["targets"])
print(json.dumps({
    "labels": [int(v) for v in c.predict(P)],
    "values": [float(v) for v in r.predict(P)],
}))
`
	pythonCommand := "python"
	if _, err := exec.LookPath(pythonCommand); err != nil {
		pythonCommand = "python3"
	}
	output, err := exec.Command(pythonCommand, "-c", script, string(encoded)).Output()
	if err != nil {
		reftest.Missing(t, "python with scikit-learn", verification, err)
	}
	var reference struct {
		Labels []int     `json:"labels"`
		Values []float64 `json:"values"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &reference); err != nil {
		t.Fatal(err)
	}

	mismatchedLabels := 0
	for i := range reference.Labels {
		got, _ := insyra.ToFloat64Safe(classifierPredictions.Get(i))
		if int(got) != reference.Labels[i] {
			mismatchedLabels++
		}
	}
	if mismatchedLabels != 0 {
		t.Fatalf("%d of %d probe labels differ from scikit-learn's exact tree", mismatchedLabels, len(reference.Labels))
	}
	for i := range reference.Values {
		got, _ := insyra.ToFloat64Safe(regressorPredictions.Get(i))
		if math.Abs(got-reference.Values[i]) > 1e-6*math.Max(math.Abs(reference.Values[i]), 1) {
			t.Fatalf("probe %d: regression %v vs scikit-learn %v", i, got, reference.Values[i])
		}
	}
}

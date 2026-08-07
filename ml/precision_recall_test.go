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

func almost(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("%s = %.17g, want %.17g", name, got, want)
	}
}

// Every averaging mode against numbers small enough to work by hand.
//
//	yTrue = a a a b b c    yPred = a b a b c c
//
//	class  tp  predicted  support  precision  recall  f1
//	a      2   2          3        1          2/3     4/5
//	b      1   2          2        1/2        1/2     1/2
//	c      1   2          1        1/2        1       2/3
func TestPrecisionRecallF1AgainstWorkedExample(t *testing.T) {
	yTrue := insyra.NewDataList("a", "a", "a", "b", "b", "c")
	yPred := insyra.NewDataList("a", "b", "a", "b", "c", "c")
	prediction := ml.Prediction{Values: yPred}

	evaluate := func(metric ml.Metric) float64 {
		t.Helper()
		result, err := metric.Evaluate(yTrue, prediction)
		if err != nil {
			t.Fatalf("%s: %v", metric.Name(), err)
		}
		return result.Score
	}

	// Macro: unweighted means of the per-class columns.
	almost(t, "macro precision", evaluate(ml.PrecisionMetric{}), (1+0.5+0.5)/3)
	almost(t, "macro recall", evaluate(ml.RecallMetric{}), (2.0/3+0.5+1)/3)
	almost(t, "macro f1", evaluate(ml.F1Metric{}), (4.0/5+0.5+2.0/3)/3)

	// Micro: 4 exact matches out of 6 — identical for all three, and equal to
	// accuracy by definition.
	for _, metric := range []ml.Metric{
		ml.PrecisionMetric{Average: ml.MicroAverage},
		ml.RecallMetric{Average: ml.MicroAverage},
		ml.F1Metric{Average: ml.MicroAverage},
	} {
		almost(t, "micro "+metric.Name(), evaluate(metric), 4.0/6)
	}

	// Weighted: per-class scores in proportion to support 3, 2, 1.
	almost(t, "weighted precision", evaluate(ml.PrecisionMetric{Average: ml.WeightedAverage}), (3*1+2*0.5+1*0.5)/6)
	almost(t, "weighted recall", evaluate(ml.RecallMetric{Average: ml.WeightedAverage}), (3*(2.0/3)+2*0.5+1*1)/6)
	almost(t, "weighted f1", evaluate(ml.F1Metric{Average: ml.WeightedAverage}), (3*(4.0/5)+2*0.5+1*(2.0/3))/6)

	// The direct helpers are the macro default.
	direct, err := ml.Precision(yTrue, yPred)
	if err != nil {
		t.Fatal(err)
	}
	almost(t, "Precision helper", direct, (1+0.5+0.5)/3)
}

// The property whose absence made the ROC AUC positive-class option pointless
// is present here: the two choices give different numbers, which is exactly why
// binary averaging refuses to choose one itself.
//
//	yTrue = p p p n n    yPred = p n p p n
func TestBinaryAveragingDependsOnThePositiveClass(t *testing.T) {
	yTrue := insyra.NewDataList("p", "p", "p", "n", "n")
	yPred := insyra.NewDataList("p", "n", "p", "p", "n")
	prediction := ml.Prediction{Values: yPred}

	asPositive := func(class string) float64 {
		t.Helper()
		result, err := (ml.PrecisionMetric{Average: ml.BinaryAverage, PositiveClass: class}).Evaluate(yTrue, prediction)
		if err != nil {
			t.Fatal(err)
		}
		return result.Score
	}
	p := asPositive("p")
	n := asPositive("n")
	almost(t, "precision of p", p, 2.0/3)
	almost(t, "precision of n", n, 1.0/2)
	if p == n {
		t.Fatal("the two positive-class choices gave the same precision; the asymmetry this API is built on is missing")
	}
}

func TestPrecisionRecallF1Refusals(t *testing.T) {
	yTrue := insyra.NewDataList("p", "p", "n")
	prediction := ml.Prediction{Values: insyra.NewDataList("p", "n", "n")}

	// Binary without a positive class: nothing is chosen on the caller's behalf.
	if _, err := (ml.F1Metric{Average: ml.BinaryAverage}).Evaluate(yTrue, prediction); err == nil {
		t.Fatal("binary averaging ran without a positive class")
	}

	// A positive class that a non-binary average would silently ignore.
	if _, err := (ml.F1Metric{PositiveClass: "p"}).Evaluate(yTrue, prediction); err == nil {
		t.Fatal("a positive class was accepted under macro averaging, where it does nothing")
	}

	// A positive class nobody observed, naming what was observed.
	_, err := (ml.F1Metric{Average: ml.BinaryAverage, PositiveClass: "q"}).Evaluate(yTrue, prediction)
	if err == nil {
		t.Fatal("an unobserved positive class was accepted")
	}
	for _, want := range []string{"q", "p", "n"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err, want)
		}
	}

	// Binary averaging over three observed labels, as scikit-learn refuses.
	multiTrue := insyra.NewDataList("a", "b", "c")
	multiPrediction := ml.Prediction{Values: insyra.NewDataList("a", "b", "c")}
	if _, err := (ml.F1Metric{Average: ml.BinaryAverage, PositiveClass: "a"}).Evaluate(multiTrue, multiPrediction); err == nil {
		t.Fatal("binary averaging ran over three observed labels")
	}
}

// A class that is never predicted contributes zero precision rather than
// failing the evaluation — scikit-learn's zero_division=0.
//
//	yTrue = a a b    yPred = a a a
func TestNeverPredictedClassContributesZero(t *testing.T) {
	yTrue := insyra.NewDataList("a", "a", "b")
	prediction := ml.Prediction{Values: insyra.NewDataList("a", "a", "a")}

	result, err := (ml.PrecisionMetric{}).Evaluate(yTrue, prediction)
	if err != nil {
		t.Fatalf("a never-predicted class failed the evaluation: %v", err)
	}
	almost(t, "macro precision with a silent class", result.Score, (2.0/3+0)/2)

	f1, err := (ml.F1Metric{}).Evaluate(yTrue, prediction)
	if err != nil {
		t.Fatal(err)
	}
	almost(t, "macro f1 with a silent class", f1.Score, (4.0/5+0)/2)
}

func TestPrecisionRecallF1ThroughTheHarness(t *testing.T) {
	labels, features := scoreClassificationData()
	metric := ml.F1Metric{Average: ml.BinaryAverage, PositiveClass: "churned"}

	model, err := ml.FitLogisticRegression(features, labels)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := ml.Score(model, features, labels, metric)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if direct.Score < 0 || direct.Score > 1 || math.IsNaN(direct.Score) {
		t.Fatalf("f1 = %v is not a score", direct.Score)
	}

	result, err := ml.CrossValidate(features, labels, ml.Estimator{
		Name: "logistic",
		Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitLogisticRegression(x, y)
		},
	}, 3, metric, insyra.SamplingOptions{UseSeed: true, Seed: 5})
	if err != nil {
		t.Fatalf("cross-validate: %v", err)
	}
	if result.Direction != ml.HigherIsBetter {
		t.Fatalf("result carries %v, want %v", result.Direction, ml.HigherIsBetter)
	}
}

// The same numbers out of scikit-learn. Runs where the reference is installed;
// reports through the shared gate where it is not.
func TestPrecisionRecallF1AgainstScikitLearn(t *testing.T) {
	const verification = "the precision/recall/F1 comparison against scikit-learn"
	if os.Getenv("INSYRA_RUN_ML_SKLEARN") != "1" && !reftest.Strict() {
		t.Skipf("set INSYRA_RUN_ML_SKLEARN=1 or %s=1 to run %s", reftest.StrictEnv, verification)
	}

	script := `
import json
from sklearn.metrics import precision_recall_fscore_support
y_true = ["a","a","a","b","b","c"]
y_pred = ["a","b","a","b","c","c"]
out = {}
for average in ("macro","micro","weighted"):
    p, r, f, _ = precision_recall_fscore_support(y_true, y_pred, average=average, zero_division=0)
    out[average] = [p, r, f]
bt = ["p","p","p","n","n"]
bp = ["p","n","p","p","n"]
p, r, f, _ = precision_recall_fscore_support(bt, bp, average="binary", pos_label="p", zero_division=0)
out["binary"] = [p, r, f]
print(json.dumps(out))
`
	pythonCommand := "python"
	if _, err := exec.LookPath(pythonCommand); err != nil {
		pythonCommand = "python3"
	}
	output, err := exec.Command(pythonCommand, "-c", script).Output()
	if err != nil {
		reftest.Missing(t, "python with scikit-learn", verification, err)
	}
	var reference map[string][3]float64
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &reference); err != nil {
		t.Fatal(err)
	}

	yTrue := insyra.NewDataList("a", "a", "a", "b", "b", "c")
	prediction := ml.Prediction{Values: insyra.NewDataList("a", "b", "a", "b", "c", "c")}
	for name, average := range map[string]ml.ClassAverage{
		"macro": ml.MacroAverage, "micro": ml.MicroAverage, "weighted": ml.WeightedAverage,
	} {
		p, err := (ml.PrecisionMetric{Average: average}).Evaluate(yTrue, prediction)
		if err != nil {
			t.Fatal(err)
		}
		r, err := (ml.RecallMetric{Average: average}).Evaluate(yTrue, prediction)
		if err != nil {
			t.Fatal(err)
		}
		f, err := (ml.F1Metric{Average: average}).Evaluate(yTrue, prediction)
		if err != nil {
			t.Fatal(err)
		}
		almost(t, name+" precision vs sklearn", p.Score, reference[name][0])
		almost(t, name+" recall vs sklearn", r.Score, reference[name][1])
		almost(t, name+" f1 vs sklearn", f.Score, reference[name][2])
	}

	binaryTrue := insyra.NewDataList("p", "p", "p", "n", "n")
	binaryPrediction := ml.Prediction{Values: insyra.NewDataList("p", "n", "p", "p", "n")}
	metric := ml.PrecisionMetric{Average: ml.BinaryAverage, PositiveClass: "p"}
	p, err := metric.Evaluate(binaryTrue, binaryPrediction)
	if err != nil {
		t.Fatal(err)
	}
	r, err := (ml.RecallMetric{Average: ml.BinaryAverage, PositiveClass: "p"}).Evaluate(binaryTrue, binaryPrediction)
	if err != nil {
		t.Fatal(err)
	}
	f, err := (ml.F1Metric{Average: ml.BinaryAverage, PositiveClass: "p"}).Evaluate(binaryTrue, binaryPrediction)
	if err != nil {
		t.Fatal(err)
	}
	almost(t, "binary precision vs sklearn", p.Score, reference["binary"][0])
	almost(t, "binary recall vs sklearn", r.Score, reference["binary"][1])
	almost(t, "binary f1 vs sklearn", f.Score, reference["binary"][2])
}

package stats

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestLogisticRegressionWithStringClasses(t *testing.T) {
	y := insyra.NewDataList("no", "no", "yes", "no", "yes", "yes", "no", "yes", "yes", "no")
	x := insyra.NewDataList(-2.1, -1.5, -0.4, 0.1, 0.6, 1.2, 1.7, 2.1, 2.8, 3.3)
	got, err := LogisticRegressionWithOptions(LogisticRegressionOptions{
		PositiveClass: "yes",
		MaxIter:       100,
		Tolerance:     1e-10,
	}, y, x)
	if err != nil {
		t.Fatalf("LogisticRegressionWithOptions error: %v", err)
	}
	if got.PositiveClass != "yes" || len(got.ClassLabels) != 2 || got.ClassLabels[1] != "yes" {
		t.Fatalf("unexpected class labels: positive=%v labels=%v", got.PositiveClass, got.ClassLabels)
	}
	if len(got.Coefficients) != 2 || len(got.FittedProbabilities) != y.Len() {
		t.Fatalf("unexpected result sizes")
	}
	if math.Abs(got.OddsRatios[1]-math.Exp(got.Coefficients[1])) > 1e-12 {
		t.Fatalf("odds ratio does not match exp(coef)")
	}
}

func TestLogisticRegressionSeparationError(t *testing.T) {
	y := insyra.NewDataList(0, 0, 0, 0, 1, 1, 1, 1)
	x := insyra.NewDataList(-4, -3, -2, -1, 1, 2, 3, 4)
	_, err := LogisticRegressionWithOptions(LogisticRegressionOptions{
		MaxIter:          100,
		Tolerance:        1e-10,
		SeparationPolicy: SepError,
	}, y, x)
	if err == nil || !strings.Contains(err.Error(), "separation") {
		t.Fatalf("expected separation error, got %v", err)
	}
}

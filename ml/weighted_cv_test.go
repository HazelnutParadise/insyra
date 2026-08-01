package ml_test

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
)

// Every row's weight is a function of its feature value, so an estimator can
// check row-by-row that it was handed the right weights — the misalignment
// this channel exists to prevent is exactly what a closure over the full
// weight list produces once folds shuffle rows.
func TestWeightedCrossValidationAlignsWeightsWithRows(t *testing.T) {
	const n = 40
	xs := make([]any, n)
	ys := make([]any, n)
	ws := make([]any, n)
	for i := 0; i < n; i++ {
		xs[i] = float64(i)
		ys[i] = 2*float64(i) + 1
		ws[i] = 1000 + float64(i) // weight derivable from the feature value
	}
	x := insyra.NewDataTable(insyra.NewDataList(xs...).SetName("x"))
	y := insyra.NewDataList(ys...).SetName("y")
	weights := insyra.NewDataList(ws...).SetName("w")

	folds := 0
	estimator := ml.Estimator{
		Name: "probe",
		FitWeighted: func(trainX *insyra.DataTable, trainY *insyra.DataList, trainW *insyra.DataList) (ml.Model, error) {
			folds++
			if trainW.Len() != trainX.NumRows() {
				t.Fatalf("fold %d: %d weights for %d rows", folds, trainW.Len(), trainX.NumRows())
			}
			column := trainX.GetColByNumber(0)
			for row := 0; row < trainX.NumRows(); row++ {
				feature, _ := insyra.ToFloat64Safe(column.Get(row))
				weight, _ := insyra.ToFloat64Safe(trainW.Get(row))
				if weight != 1000+feature {
					t.Fatalf("fold %d row %d: feature %v carries weight %v, want %v — weights misaligned with rows",
						folds, row, feature, weight, 1000+feature)
				}
			}
			return ml.FitLinearRegression(trainX, trainY)
		},
	}
	result, err := ml.CrossValidateWeighted(x, y, weights, estimator, 4, ml.RMSEMetric{},
		insyra.SamplingOptions{UseSeed: true, Seed: 17})
	if err != nil {
		t.Fatalf("weighted cross-validation: %v", err)
	}
	if folds != 4 {
		t.Fatalf("the weighted fitter ran %d times over 4 folds", folds)
	}
	if math.IsNaN(result.Mean) {
		t.Fatal("no mean came back")
	}
}

// WLS end-to-end: the weighted estimator this channel was built for.
func TestWeightedCrossValidationRunsWLS(t *testing.T) {
	x, y := regularizedTable()
	ws := make([]any, y.Len())
	for i := range ws {
		ws[i] = 0.5 + float64(i%4)
	}
	weights := insyra.NewDataList(ws...)
	result, err := ml.CrossValidateWeighted(x, y, weights, ml.Estimator{
		Name: "wls",
		FitWeighted: func(x *insyra.DataTable, y *insyra.DataList, w *insyra.DataList) (ml.Model, error) {
			return ml.FitWeightedLinearRegression(x, y, w)
		},
	}, 4, ml.R2Metric{}, insyra.SamplingOptions{UseSeed: true, Seed: 23})
	if err != nil {
		t.Fatalf("weighted cross-validation of WLS: %v", err)
	}
	if result.Mean < 0.9 {
		t.Fatalf("WLS R² mean = %v on nearly noiseless linear data", result.Mean)
	}
	if result.Direction != ml.HigherIsBetter {
		t.Fatalf("result carries %v", result.Direction)
	}
}

func TestWeightedCrossValidationRefusals(t *testing.T) {
	x, y := regularizedTable()
	good := make([]any, y.Len())
	for i := range good {
		good[i] = 1.0
	}
	weighted := ml.Estimator{
		Name: "wls",
		FitWeighted: func(x *insyra.DataTable, y *insyra.DataList, w *insyra.DataList) (ml.Model, error) {
			return ml.FitWeightedLinearRegression(x, y, w)
		},
	}

	// An estimator without the weighted fitter: refused, not silently unweighted.
	unweighted := ml.Estimator{Name: "plain", Fit: ml.FitLinearRegression}
	_, err := ml.CrossValidateWeighted(x, y, insyra.NewDataList(good...), unweighted, 4, ml.RMSEMetric{})
	if err == nil {
		t.Fatal("an estimator without FitWeighted was fitted")
	}
	if !strings.Contains(err.Error(), "FitWeighted") {
		t.Fatalf("error %q does not say what is missing", err)
	}

	if _, err := ml.CrossValidateWeighted(x, y, nil, weighted, 4, ml.RMSEMetric{}); err == nil {
		t.Fatal("nil weights were accepted")
	}
	short := insyra.NewDataList(1.0, 2.0)
	if _, err := ml.CrossValidateWeighted(x, y, short, weighted, 4, ml.RMSEMetric{}); err == nil {
		t.Fatal("a short weight list was accepted")
	}
	for _, bad := range []any{0.0, -1.0, nil, "abc", math.Inf(1)} {
		ws := append([]any(nil), good...)
		ws[5] = bad
		if _, err := ml.CrossValidateWeighted(x, y, insyra.NewDataList(ws...), weighted, 4, ml.RMSEMetric{}); err == nil {
			t.Fatalf("weight %v was accepted", bad)
		}
	}
}

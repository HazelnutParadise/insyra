package ml_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/ml"
	"github.com/HazelnutParadise/insyra/ml/mltest"
	"github.com/HazelnutParadise/insyra/stats"
)

func regularizedTable() (*insyra.DataTable, *insyra.DataList) {
	const n = 40
	a := make([]any, n)
	b := make([]any, n)
	y := make([]any, n)
	for i := 0; i < n; i++ {
		v1 := float64(i % 7)
		v2 := float64((i * 3) % 11)
		a[i] = v1
		b[i] = v2
		y[i] = 3 + 2*v1 - 1.5*v2 + 0.05*math.Sin(float64(i))
	}
	return insyra.NewDataTable(
			insyra.NewDataList(a...).SetName("x1"),
			insyra.NewDataList(b...).SetName("x2"),
		),
		insyra.NewDataList(y...).SetName("y")
}

func TestRegularizedModelsPassConformance(t *testing.T) {
	x, y := regularizedTable()
	ridge, err := ml.FitRidgeRegression(x, y, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	mltest.RunConformance(t, ridge, x, nil)

	lasso, err := ml.FitLassoRegression(x, y, 0.1)
	if err != nil {
		t.Fatal(err)
	}
	mltest.RunConformance(t, lasso, x, nil)
}

// The wrapper adds name-based binding and nothing else: its predictions must
// equal the stats result's own.
func TestRegularizedWrapperMatchesStatsDirectly(t *testing.T) {
	x, y := regularizedTable()
	model, err := ml.FitRidgeRegression(x, y, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := stats.RidgeRegression(y, 0.5, x.GetColByName("x1"), x.GetColByName("x2"))
	if err != nil {
		t.Fatal(err)
	}
	wantList, err := direct.Predict(stats.PredictResponse, x.GetColByName("x1"), x.GetColByName("x2"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := model.Predict(x)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < got.Len(); i++ {
		gotValue, _ := insyra.ToFloat64Safe(got.Get(i))
		wantValue, _ := insyra.ToFloat64Safe(wantList.Get(i))
		if gotValue != wantValue {
			t.Fatalf("row %d: wrapper %v vs stats %v", i, gotValue, wantValue)
		}
	}
}

func TestRegularizedModelsWorkInTheHarness(t *testing.T) {
	x, y := regularizedTable()
	result, err := ml.CrossValidate(x, y, ml.Estimator{
		Name: "ridge",
		Fit: func(x *insyra.DataTable, y *insyra.DataList) (ml.Model, error) {
			return ml.FitRidgeRegression(x, y, 1.0)
		},
	}, 4, ml.RMSEMetric{}, insyra.SamplingOptions{UseSeed: true, Seed: 9})
	if err != nil {
		t.Fatalf("cross-validate ridge: %v", err)
	}
	if math.IsNaN(result.Mean) || result.Mean < 0 {
		t.Fatalf("ridge RMSE mean = %v", result.Mean)
	}

	lasso, err := ml.FitLassoRegression(x, y, 0.05)
	if err != nil {
		t.Fatal(err)
	}
	score, err := ml.Score(lasso, x, y, ml.R2Metric{})
	if err != nil {
		t.Fatalf("score lasso: %v", err)
	}
	if score.Score < 0.9 {
		t.Fatalf("lasso R² = %v on nearly noiseless linear data; the fit is wrong", score.Score)
	}
}

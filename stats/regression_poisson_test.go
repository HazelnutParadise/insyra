package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestPoissonRegressionWithOffset(t *testing.T) {
	y := insyra.NewDataList(1, 2, 3, 4, 6, 8, 9, 12)
	x := insyra.NewDataList(0.1, 0.4, 0.8, 1.2, 1.7, 2.0, 2.4, 2.9)
	offset := insyra.NewDataList(
		math.Log(1.0), math.Log(1.1), math.Log(0.9), math.Log(1.3),
		math.Log(1.2), math.Log(1.5), math.Log(1.4), math.Log(1.8),
	)
	got, err := PoissonRegressionWithOptions(PoissonRegressionOptions{
		MaxIter:         100,
		Tolerance:       1e-10,
		Offset:          offset,
		DispersionCheck: true,
	}, y, x)
	if err != nil {
		t.Fatalf("PoissonRegressionWithOptions error: %v", err)
	}
	if len(got.Coefficients) != 2 || len(got.FittedRates) != y.Len() {
		t.Fatalf("unexpected result sizes")
	}
	if got.PearsonChi2 <= 0 || math.IsNaN(got.DispersionStatistic) {
		t.Fatalf("unexpected dispersion stats: pearson=%v dispersion=%v", got.PearsonChi2, got.DispersionStatistic)
	}
	if math.Abs(got.IncidenceRateRatios[1]-math.Exp(got.Coefficients[1])) > 1e-12 {
		t.Fatalf("IRR does not match exp(coef)")
	}
}

func TestPoissonPredictOffsetHandling(t *testing.T) {
	y := insyra.NewDataList(1, 2, 3, 4, 6, 8, 9, 12)
	x := insyra.NewDataList(0.1, 0.4, 0.8, 1.2, 1.7, 2.0, 2.4, 2.9)
	offset := insyra.NewDataList(
		math.Log(1.0), math.Log(1.1), math.Log(0.9), math.Log(1.3),
		math.Log(1.2), math.Log(1.5), math.Log(1.4), math.Log(1.8),
	)
	res, err := PoissonRegressionWithOptions(PoissonRegressionOptions{MaxIter: 100, Tolerance: 1e-10, Offset: offset}, y, x)
	if err != nil {
		t.Fatalf("fit: %v", err)
	}

	// Predict without an offset must refuse: silently dropping the fitted offset
	// would produce wrong rate predictions on new data.
	if _, err := res.Predict(PredictLinear, insyra.NewDataList(0.5, 1.0)); err == nil {
		t.Fatalf("Predict on an offset-fitted model should error")
	}

	newX := insyra.NewDataList(0.5, 1.0)
	newOffset := insyra.NewDataList(math.Log(2.0), math.Log(3.0))
	lin, err := res.PredictWithOffset(PredictLinear, newOffset, newX)
	if err != nil {
		t.Fatalf("PredictWithOffset: %v", err)
	}
	b := res.Coefficients
	want := []float64{
		b[0] + b[1]*0.5 + math.Log(2.0),
		b[0] + b[1]*1.0 + math.Log(3.0),
	}
	got := lin.ToF64Slice()
	if len(got) != len(want) {
		t.Fatalf("expected %d predictions, got %d", len(want), len(got))
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("linear[%d] = %v, want %v (offset must be added to eta)", i, got[i], want[i])
		}
	}

	// Offset length must match the predictor rows.
	if _, err := res.PredictWithOffset(PredictLinear, insyra.NewDataList(math.Log(2.0)), newX); err == nil {
		t.Fatalf("offset length mismatch should error")
	}
}

func TestPoissonRegressionDispersionFlag(t *testing.T) {
	y := insyra.NewDataList(0, 0, 12, 0, 18, 1, 25, 0, 30, 2)
	x := insyra.NewDataList(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	got, err := PoissonRegressionWithOptions(PoissonRegressionOptions{
		MaxIter:         100,
		Tolerance:       1e-10,
		DispersionCheck: true,
	}, y, x)
	if err != nil {
		t.Fatalf("PoissonRegressionWithOptions error: %v", err)
	}
	if got.OverDispersed != (got.DispersionStatistic > 1.5) {
		t.Fatalf("over-dispersion flag mismatch: flag=%v dispersion=%v", got.OverDispersed, got.DispersionStatistic)
	}
}

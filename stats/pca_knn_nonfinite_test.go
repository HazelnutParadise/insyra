package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestPCARefusesNonFiniteCells(t *testing.T) {
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		dt := insyra.NewDataTable(
			insyra.NewDataList(1.0, 2.0, 3.0, bad),
			insyra.NewDataList(2.0, 1.0, 4.0, 3.0),
		)
		if _, err := PCA(dt); err == nil {
			t.Fatalf("PCA with %v in the data returned no error", bad)
		}
	}
}

func TestKNNRegressRefusesNonFiniteTargets(t *testing.T) {
	train := insyra.NewDataTable(insyra.NewDataList(1.0, 2.0, 3.0, 4.0))
	test := insyra.NewDataTable(insyra.NewDataList(2.5))
	for _, bad := range []float64{math.NaN(), math.Inf(1)} {
		targets := insyra.NewDataList(1.0, 2.0, 3.0, bad)
		if _, err := KNNRegress(train, targets, test, 2); err == nil {
			t.Fatalf("KNNRegress with target %v returned no error", bad)
		}
	}
}

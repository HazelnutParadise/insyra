package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/stat/distuv"
)

func TestCorrelationCIUsesFisherTransform(t *testing.T) {
	x := insyra.NewDataList([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	y := insyra.NewDataList([]float64{1, 2, 3, 5, 4, 6, 8, 7, 10, 9})

	pearson, err := stats.Correlation(x, y, stats.PearsonCorrelation)
	if err != nil || pearson == nil || pearson.CI == nil {
		t.Fatalf("expected Pearson correlation with CI, got %+v", pearson)
	}

	pLower, pUpper := fisherCI(pearson.Statistic, 10, 0.95)
	if !floatAlmostEqual(pearson.CI[0], pLower, 1e-12) || !floatAlmostEqual(pearson.CI[1], pUpper, 1e-12) {
		t.Fatalf("Pearson CI mismatch: got [%v %v], want [%v %v]", pearson.CI[0], pearson.CI[1], pLower, pUpper)
	}

	spearman, err := stats.Correlation(x, y, stats.SpearmanCorrelation)
	if err != nil || spearman == nil || spearman.CI == nil {
		t.Fatalf("expected Spearman correlation with CI, got %+v", spearman)
	}

	sLower, sUpper := fisherCI(spearman.Statistic, 10, 0.95)
	if !floatAlmostEqual(spearman.CI[0], sLower, 1e-12) || !floatAlmostEqual(spearman.CI[1], sUpper, 1e-12) {
		t.Fatalf("Spearman Fisher CI mismatch: got [%v %v], want [%v %v]", spearman.CI[0], spearman.CI[1], sLower, sUpper)
	}

	legacyLower, legacyUpper := legacyLinearCI(spearman.Statistic, 10, 0.95)
	if math.Abs(spearman.CI[0]-legacyLower) < 1e-6 && math.Abs(spearman.CI[1]-legacyUpper) < 1e-6 {
		t.Fatalf("Spearman CI unexpectedly matches legacy linear approximation [%v %v]", legacyLower, legacyUpper)
	}
}

func fisherCI(r float64, n int, confidenceLevel float64) (float64, float64) {
	z := 0.5 * math.Log((1+r)/(1-r))
	se := 1 / math.Sqrt(float64(n-3))
	zCrit := distuv.Normal{Mu: 0, Sigma: 1}.Quantile(1 - (1-confidenceLevel)/2)
	zLower := z - zCrit*se
	zUpper := z + zCrit*se
	return fisherInv(zLower), fisherInv(zUpper)
}

func fisherInv(z float64) float64 {
	exp2z := math.Exp(2 * z)
	return (exp2z - 1) / (exp2z + 1)
}

func legacyLinearCI(r float64, n int, confidenceLevel float64) (float64, float64) {
	se := 1 / math.Sqrt(float64(n-3))
	zCrit := distuv.Normal{Mu: 0, Sigma: 1}.Quantile(1 - (1-confidenceLevel)/2)
	return r - zCrit*se, r + zCrit*se
}

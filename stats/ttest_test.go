package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

type tTestCase struct {
	data        []float64
	mu          float64
	expectT     float64
	expectP     float64
	expectDF    float64
	expectCI    [2]float64
	expectCohen float64
}

func floatEquals(a, b, epsilon float64) bool {
	return math.Abs(a-b) <= epsilon
}

func TestSingleSampleTTest(t *testing.T) {
	tests := []tTestCase{
		{
			data:        []float64{52.12, 58.36, 57.49, 51.31, 61.25, 42.88, 46.89, 59.45, 56.20, 44.39, 48.15, 50.99, 56.84, 47.90, 49.78},
			mu:          50,
			expectT:     1.5367,
			expectP:     0.1467,
			expectDF:    14,
			expectCI:    [2]float64{49.1030, 55.4304},
			expectCohen: 0.3968,
		},
		{
			data:        []float64{49.55, 42.01, 50.22, 52.88, 42.75, 47.36, 54.17, 41.29, 45.63, 47.48, 55.81, 44.10, 43.75, 46.52, 50.94, 53.60, 43.07, 48.77, 46.82, 50.90},
			mu:          50,
			expectT:     -2.1928,
			expectP:     0.0410,
			expectDF:    19,
			expectCI:    [2]float64{45.8584, 49.9036},
			expectCohen: 0.4903,
		},
	}

	for i, test := range tests {
		dl := insyra.NewDataList(test.data)
		result := stats.SingleSampleTTest(dl, test.mu, 0.95)

		if !floatEquals(result.Statistic, test.expectT, 0.01) {
			t.Errorf("case %d: T mismatch, got %f, want %f", i, result.Statistic, test.expectT)
		}
		if !floatEquals(result.PValue, test.expectP, 0.01) {
			t.Errorf("case %d: P mismatch, got %f, want %f", i, result.PValue, test.expectP)
		}
		if !floatEquals(*result.DF, test.expectDF, 0.01) {
			t.Errorf("case %d: DF mismatch, got %f, want %f", i, *result.DF, test.expectDF)
		}
		if !floatEquals(result.CI[0], test.expectCI[0], 0.05) || !floatEquals(result.CI[1], test.expectCI[1], 0.05) {
			t.Errorf("case %d: CI mismatch, got [%f, %f], want [%f, %f]", i, result.CI[0], result.CI[1], test.expectCI[0], test.expectCI[1])
		}
		if len(result.EffectSizes) > 0 && !floatEquals(result.EffectSizes[0].Value, test.expectCohen, 0.01) {
			t.Errorf("case %d: Effect size mismatch, got %f, want %f", i, result.EffectSizes[0].Value, test.expectCohen)
		}
	}
}

func TestTwoSampleTTest(t *testing.T) {
	data1 := insyra.NewDataList([]float64{55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7, 52.4, 56.1})
	data2 := insyra.NewDataList([]float64{46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9, 48.3, 45.4})
	result := stats.TwoSampleTTest(data1, data2, false, 0.95)

	expectT := 5.1337
	expectP := 0.000162
	expectDF := 13.7291
	expectCI := [2]float64{4.9713, 12.1287}
	expectCohen := 2.2959

	if !floatEquals(result.Statistic, expectT, 0.01) {
		t.Errorf("Two-sample T mismatch, got %f, want %f", result.Statistic, expectT)
	}
	if !floatEquals(result.PValue, expectP, 0.01) {
		t.Errorf("Two-sample P mismatch, got %f, want %f", result.PValue, expectP)
	}
	if !floatEquals(*result.DF, expectDF, 0.1) {
		t.Errorf("Two-sample DF mismatch, got %f, want %f", *result.DF, expectDF)
	}
	if !floatEquals(result.CI[0], expectCI[0], 0.05) || !floatEquals(result.CI[1], expectCI[1], 0.05) {
		t.Errorf("Two-sample CI mismatch, got [%f, %f], want [%f, %f]", result.CI[0], result.CI[1], expectCI[0], expectCI[1])
	}
	if len(result.EffectSizes) > 0 && !floatEquals(result.EffectSizes[0].Value, expectCohen, 0.01) {
		t.Errorf("Two-sample Effect size mismatch, got %f, want %f", result.EffectSizes[0].Value, expectCohen)
	}
}

func TestPairedTTest(t *testing.T) {
	before := insyra.NewDataList([]float64{55.0, 48.5, 51.6, 53.2, 50.4, 52.1, 54.0, 56.3})
	after := insyra.NewDataList([]float64{52.2, 45.7, 49.8, 50.9, 47.6, 50.3, 52.1, 53.9})
	result := stats.PairedTTest(before, after, 0.95)

	expectT := 14.6264
	expectP := 0.000001668
	expectDF := 7.0
	expectCI := [2]float64{1.9491, 2.7009}
	expectCohen := 5.1712

	if !floatEquals(result.Statistic, expectT, 0.01) {
		t.Errorf("Paired T mismatch, got %f, want %f", result.Statistic, expectT)
	}
	if !floatEquals(result.PValue, expectP, 0.01) {
		t.Errorf("Paired P mismatch, got %f, want %f", result.PValue, expectP)
	}
	if !floatEquals(*result.DF, expectDF, 0.01) {
		t.Errorf("Paired DF mismatch, got %f, want %f", *result.DF, expectDF)
	}
	if !floatEquals(result.CI[0], expectCI[0], 0.05) || !floatEquals(result.CI[1], expectCI[1], 0.05) {
		t.Errorf("Paired CI mismatch, got [%f, %f], want [%f, %f]", result.CI[0], result.CI[1], expectCI[0], expectCI[1])
	}
	if len(result.EffectSizes) > 0 && !floatEquals(result.EffectSizes[0].Value, expectCohen, 0.01) {
		t.Errorf("Paired Effect size mismatch, got %f, want %f", result.EffectSizes[0].Value, expectCohen)
	}
}

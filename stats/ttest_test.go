package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

type tTestCase struct {
	name        string
	data        []float64
	mu          float64
	expectT     float64
	expectP     float64
	expectDF    float64
	expectCI    [2]float64
	expectCohen float64
}

func floatEquals(a, b, epsilon float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsInf(a, 0) && math.IsInf(b, 0) && ((a > 0 && b > 0) || (a < 0 && b < 0)) {
		return true
	}
	return math.Abs(a-b) <= epsilon
}

func TestSingleSampleTTest_R(t *testing.T) {
	tests := []tTestCase{
		{
			name:        "test1",
			data:        []float64{52.12, 58.36, 57.49, 51.31, 61.25, 42.88, 46.89, 59.45, 56.20, 44.39, 48.15, 50.99, 56.84, 47.90, 49.78},
			mu:          50,
			expectT:     1.5367,
			expectP:     0.1467,
			expectDF:    14,
			expectCI:    [2]float64{49.1030, 55.4304},
			expectCohen: 0.3968,
		},
		{
			name:        "test2",
			data:        []float64{49.55, 42.01, 50.22, 52.88, 42.75, 47.36, 54.17, 41.29, 45.63, 47.48, 55.81, 44.10, 43.75, 46.52, 50.94, 53.60, 43.07, 48.77, 46.82, 50.90},
			mu:          50,
			expectT:     -2.1928,
			expectP:     0.0410,
			expectDF:    19,
			expectCI:    [2]float64{45.8584, 49.9036},
			expectCohen: -0.4903214,
		},
		{
			name:        "small_data_large_diff",
			data:        []float64{12, 14, 15, 11, 13},
			mu:          10,
			expectT:     4.2426,
			expectP:     0.0132356,
			expectDF:    4.0,
			expectCI:    [2]float64{11.0368, 14.9632},
			expectCohen: 1.8974,
		},
		{
			name: "large_data_small_diff",
			data: []float64{9.8, 10.1, 9.9, 10.2, 9.7, 10.0, 10.3, 9.6, 10.4, 10.0,
				9.9, 10.1, 9.8, 10.2, 10.0, 9.7, 10.3, 9.6, 10.4, 9.9},
			mu:          10,
			expectT:     -0.0886,
			expectP:     0.9303070,
			expectDF:    19,
			expectCI:    [2]float64{9.8769, 10.1131},
			expectCohen: -0.0198,
		},
		{
			name:        "data_equals_mu_constant",
			data:        []float64{50, 50, 50, 50, 50},
			mu:          50,
			expectT:     math.NaN(),
			expectP:     math.NaN(),
			expectDF:    4.0,
			expectCI:    [2]float64{50, 50},
			expectCohen: 0.0,
		},
		{
			name:        "constant_data_not_equal_mu",
			data:        []float64{10, 10, 10},
			mu:          5,
			expectT:     math.Inf(1),
			expectP:     0.0000000,
			expectDF:    2.0,
			expectCI:    [2]float64{10, 10},
			expectCohen: math.Inf(1),
		},
		{
			name:        "data_with_negative_values",
			data:        []float64{-5, -2, -6, -3, -4},
			mu:          0,
			expectT:     -5.6569,
			expectP:     0.0048127,
			expectDF:    4.0,
			expectCI:    [2]float64{-5.9632, -2.0368},
			expectCohen: -2.5298,
		},
		{
			name:        "data_very_close_to_mu",
			data:        []float64{10.01, 9.99, 10.02, 9.98, 10.00},
			mu:          10,
			expectT:     0.0,
			expectP:     1.0,
			expectDF:    4.0,
			expectCI:    [2]float64{9.9804, 10.0196},
			expectCohen: 0.0,
		},
	}

	for _, test := range tests {
		dl := insyra.NewDataList(test.data)
		result := stats.SingleSampleTTest(dl, test.mu, 0.95)

		if !floatEquals(result.Statistic, test.expectT, 0.01) {
			t.Errorf("case %s: T mismatch, got %f, want %f", test.name, result.Statistic, test.expectT)
		}
		if !floatEquals(result.PValue, test.expectP, 0.01) {
			t.Errorf("case %s: P mismatch, got %f, want %f", test.name, result.PValue, test.expectP)
		}
		if !floatEquals(*result.DF, test.expectDF, 0.01) {
			t.Errorf("case %s: DF mismatch, got %f, want %f", test.name, *result.DF, test.expectDF)
		}
		if !floatEquals(result.CI[0], test.expectCI[0], 0.05) || !floatEquals(result.CI[1], test.expectCI[1], 0.05) {
			t.Errorf("case %s: CI mismatch, got [%f, %f], want [%f, %f]", test.name, result.CI[0], result.CI[1], test.expectCI[0], test.expectCI[1])
		}
		if len(result.EffectSizes) > 0 && !floatEquals(result.EffectSizes[0].Value, test.expectCohen, 0.01) {
			t.Errorf("case %s: Effect size mismatch, got %f, want %f", test.name, result.EffectSizes[0].Value, test.expectCohen)
		}
	}
}

func TestTwoSampleTTest_R(t *testing.T) {
	tests := []struct {
		name     string
		data1    []float64
		data2    []float64
		equalVar bool
		expectT  float64
		expectP  float64
		expectDF float64
		expectCI [2]float64
		expectD  float64
	}{
		{
			name:     "equal_var_true_strong_diff",
			data1:    []float64{55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7, 52.4, 56.1},
			data2:    []float64{46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9, 48.3, 45.4},
			equalVar: true,
			expectT:  5.1337,
			expectP:  0.00007,
			expectDF: 18.0,
			expectCI: [2]float64{5.0510, 12.0490},
			expectD:  2.2959,
		},
		{
			name:     "equal_var_false_strong_diff",
			data1:    []float64{55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7, 52.4, 56.1},
			data2:    []float64{46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9, 48.3, 45.4},
			equalVar: false,
			expectT:  5.1337,
			expectP:  0.000162,
			expectDF: 13.7291,
			expectCI: [2]float64{4.9713, 12.1287},
			expectD:  2.2959,
		},
		{
			name:     "equal_var_true_small_diff",
			data1:    []float64{50.2, 48.5, 52.1, 51.3, 50.5, 49.9, 48.7, 50.8},
			data2:    []float64{47.9, 48.1, 49.5, 50.2, 48.8, 49.7, 48.9, 49.1},
			equalVar: true,
			expectT:  2.3881,
			expectP:  0.031579,
			expectDF: 14.0,
			expectCI: [2]float64{0.1248, 2.3252},
			expectD:  1.1941,
		},
		{
			name:     "equal_var_false_small_diff",
			data1:    []float64{60.0, 59.5, 58.8, 60.2, 59.9},
			data2:    []float64{55.1, 52.3, 58.4, 53.6, 54.2, 53.1},
			equalVar: false,
			expectT:  5.7181,
			expectP:  0.001411,
			expectDF: 5.7781,
			expectCI: [2]float64{2.9710, 7.4890},
			expectD:  3.3217,
		},
		{
			name:     "diff_sample_size_equal_var_small_diff",
			data1:    []float64{10.1, 10.3, 10.0, 9.9, 10.2},
			data2:    []float64{9.8, 9.7, 9.9, 10.0, 9.6, 9.5, 9.4},
			equalVar: true,
			expectT:  3.5044,
			expectP:  0.0056847,
			expectDF: 10.0,
			expectCI: [2]float64{0.1457, 0.6543},
			expectD:  2.052,
		},
		{
			name:     "diff_sample_size_unequal_var_large_diff",
			data1:    []float64{100, 105, 98, 110, 102, 103},
			data2:    []float64{80, 85, 78, 90, 82},
			equalVar: false,
			expectT:  7.3855,
			expectP:  0.0000682,
			expectDF: 8.1967,
			expectCI: [2]float64{13.7813, 26.2187},
			expectD:  4.4947,
		}, {
			name:     "data_near_zero_equal_var",
			data1:    []float64{0.01, 0.02, 0.015, 0.025},
			data2:    []float64{-0.01, -0.005, -0.012, -0.008},
			equalVar: true,
			expectT:  7.3817,
			expectP:  0.0003170,
			expectDF: 6.0,
			expectCI: [2]float64{0.0175, 0.035},
			expectD:  5.2196,
		},
		{
			name:     "data_with_negative_values_unequal_var",
			data1:    []float64{-5, -4, -6, -3, -5},
			data2:    []float64{1, 2, 0, 3, 1, 2},
			equalVar: false,
			expectT:  -9.1615,
			expectP:  0.0000125,
			expectDF: 8.3203,
			expectCI: [2]float64{-7.6252, -4.5748},
			expectD:  -5.5685,
		},
		{
			name:     "data_with_duplicates_equal_var",
			data1:    []float64{10, 12, 10, 11, 12, 10},
			data2:    []float64{8, 9, 8, 7, 9, 8},
			equalVar: true,
			expectT:  5.275,
			expectP:  0.0003602,
			expectDF: 10.0,
			expectCI: [2]float64{1.5403, 3.793},
			expectD:  3.0455,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data1 := insyra.NewDataList(tt.data1)
			data2 := insyra.NewDataList(tt.data2)
			result := stats.TwoSampleTTest(data1, data2, tt.equalVar, 0.95)

			if !floatEquals(result.Statistic, tt.expectT, 0.01) {
				t.Errorf("[%s] T value mismatch: got %f, want %f", tt.name, result.Statistic, tt.expectT)
			}
			if !floatEquals(result.PValue, tt.expectP, 0.001) {
				t.Errorf("[%s] P value mismatch: got %f, want %f", tt.name, result.PValue, tt.expectP)
			}
			if !floatEquals(*result.DF, tt.expectDF, 0.1) {
				t.Errorf("[%s] DF mismatch: got %f, want %f", tt.name, *result.DF, tt.expectDF)
			}
			if !floatEquals(result.CI[0], tt.expectCI[0], 0.05) || !floatEquals(result.CI[1], tt.expectCI[1], 0.05) {
				t.Errorf("[%s] CI mismatch: got [%f, %f], want [%f, %f]",
					tt.name, result.CI[0], result.CI[1], tt.expectCI[0], tt.expectCI[1])
			}
			if len(result.EffectSizes) > 0 && !floatEquals(result.EffectSizes[0].Value, tt.expectD, 0.01) {
				t.Errorf("[%s] Cohen's d mismatch: got %f, want %f", tt.name, result.EffectSizes[0].Value, tt.expectD)
			}
		})
	}
}

func TestPairedTTest_R(t *testing.T) {
	before := insyra.NewDataList([]float64{55.0, 48.5, 51.6, 53.2, 50.4, 52.1, 54.0, 56.3})
	after := insyra.NewDataList([]float64{52.2, 45.7, 49.8, 50.9, 47.6, 50.3, 52.1, 53.9})
	result := stats.PairedTTest(before, after, 0.95)

	expectT := 14.6264
	expectP := 0.000001668
	expectDF := 7.0
	expectCI := [2]float64{1.9491, 2.7009}
	expectCohen := 5.1712
	expectMeanDiff := 2.325

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
	if !floatEquals(*result.MeanDiff, expectMeanDiff, 0.01) {
		t.Errorf("Paired Mean difference mismatch, got %f, want %f", *result.MeanDiff, expectMeanDiff)
	}
}

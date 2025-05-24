package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestCorrelation(t *testing.T) {
	x := insyra.NewDataList([]float64{10, 20, 30, 40, 50})
	y := insyra.NewDataList([]float64{15, 22, 29, 41, 48})

	tests := []struct {
		method       stats.CorrelationMethod
		expectedStat float64
		expectedP    float64
		name         string
		tolerance    float64
	}{
		{
			method:       stats.PearsonCorrelation,
			expectedStat: 0.9948497511671097,
			expectedP:    0.000443343538312087,
			name:         "Pearson",
			tolerance:    1e-6,
		},
		{
			method:       stats.KendallCorrelation,
			expectedStat: 1.0,
			expectedP:    0.016666666666666666,
			name:         "Kendall",
			tolerance:    1e-6,
		},
		{
			method:       stats.SpearmanCorrelation,
			expectedStat: 1.0,
			expectedP:    1.4042654220543672e-24,
			name:         "Spearman",
			tolerance:    1e-6,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := stats.Correlation(x, y, tc.method)
			if result == nil {
				t.Errorf("%s: expected non-nil result", tc.name)
				return
			}
			if math.Abs(result.Statistic-tc.expectedStat) > tc.tolerance {
				t.Errorf("%s stat mismatch: got %v, want %v", tc.name, result.Statistic, tc.expectedStat)
			}
			if math.Abs(result.PValue-tc.expectedP) > tc.tolerance {
				t.Errorf("%s p-value mismatch: got %v, want %v", tc.name, result.PValue, tc.expectedP)
			}
		})
	}
}

func TestCorrelation_MoreCases(t *testing.T) {
	type TestCase struct {
		name         string
		x            []float64
		y            []float64
		method       stats.CorrelationMethod
		expectedStat float64
		expectedP    float64
		tolerance    float64
		expectNil    bool
	}

	testCases := []TestCase{
		// 正相關
		{"PerfectPositive_Pearson", []float64{1, 2, 3, 4, 5}, []float64{10, 20, 30, 40, 50}, stats.PearsonCorrelation, 1.0, 0.0, 1e-6, false},
		{"PerfectPositive_Kendall", []float64{1, 2, 3, 4, 5}, []float64{10, 20, 30, 40, 50}, stats.KendallCorrelation, 1.0, 0.016666666666666666, 1e-6, false},
		{"PerfectPositive_Spearman", []float64{1, 2, 3, 4, 5}, []float64{10, 20, 30, 40, 50}, stats.SpearmanCorrelation, 1.0, 1.4042654220543672e-24, 1e-6, false},

		// 負相關
		{"PerfectNegative_Pearson", []float64{1, 2, 3, 4, 5}, []float64{50, 40, 30, 20, 10}, stats.PearsonCorrelation, -1.0, 0.0, 1e-6, false},
		{"PerfectNegative_Kendall", []float64{1, 2, 3, 4, 5}, []float64{50, 40, 30, 20, 10}, stats.KendallCorrelation, -1.0, 0.016666666666666666, 1e-6, false},
		{"PerfectNegative_Spearman", []float64{1, 2, 3, 4, 5}, []float64{50, 40, 30, 20, 10}, stats.SpearmanCorrelation, -1.0, 1.4042654220543672e-24, 1e-6, false},

		// 無變異 → 應該回傳 nil
		{"NoCorrelation_Pearson", []float64{1, 2, 3, 4, 5}, []float64{3, 3, 3, 3, 3}, stats.PearsonCorrelation, 0, 0, 0, true},
		{"NoCorrelation_Kendall", []float64{1, 2, 3, 4, 5}, []float64{3, 3, 3, 3, 3}, stats.KendallCorrelation, 0, 0, 0, true},
		{"NoCorrelation_Spearman", []float64{1, 2, 3, 4, 5}, []float64{3, 3, 3, 3, 3}, stats.SpearmanCorrelation, 0, 0, 0, true},

		// 亂數 → 有值但不是 1/-1
		{"Random_Pearson", []float64{5, 1, 3, 2, 4}, []float64{10, 7, 8, 6, 9}, stats.PearsonCorrelation, 0.9, 0.0373860734684987, 1e-6, false},
		{"Random_Kendall", []float64{5, 1, 3, 2, 4}, []float64{10, 7, 8, 6, 9}, stats.KendallCorrelation, 0.8, 0.0833333333333333, 1e-6, false},
		{"Random_Spearman", []float64{5, 1, 3, 2, 4}, []float64{10, 7, 8, 6, 9}, stats.SpearmanCorrelation, 0.9, 0.0373860734684987, 1e-6, false},

		// 長度不一致 → 應回傳 nil
		{"UnequalLength", []float64{1, 2, 3}, []float64{4, 5}, stats.PearsonCorrelation, 0, 0, 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			x := insyra.NewDataList(tc.x)
			y := insyra.NewDataList(tc.y)
			result := stats.Correlation(x, y, tc.method)

			if tc.expectNil {
				if result != nil {
					t.Errorf("%s: expected nil, got %+v", tc.name, result)
				}
				return
			}

			if result == nil {
				t.Errorf("%s: expected result, got nil", tc.name)
				return
			}
			if math.Abs(result.Statistic-tc.expectedStat) > tc.tolerance {
				t.Errorf("%s stat mismatch: got %v, want %v", tc.name, result.Statistic, tc.expectedStat)
			}
			if math.Abs(result.PValue-tc.expectedP) > tc.tolerance {
				t.Errorf("%s p-value mismatch: got %v, want %v", tc.name, result.PValue, tc.expectedP)
			}
		})
	}
}

package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestLinearRegression_MultipleCases(t *testing.T) {
	cases := []struct {
		name     string
		x, y     []float64
		expected *stats.LinearRegressionResult
	}{
		{
			name: "Case 1",
			x:    []float64{38.08, 95.07, 73.20, 59.87, 15.60, 15.60, 5.81, 86.62, 60.11, 70.81},
			y:    []float64{212.12, 466.84, 359.15, 280.27, 84.92, 77.83, 30.49, 446.84, 286.32, 369.78},
			expected: &stats.LinearRegressionResult{
				Slope:            4.930690301857441,
				Intercept:        4.680441150169841,
				RSquared:         0.9927581012775553,
				AdjustedRSquared: 0.9918528639372497,
				StandardError:    0.148890584666395,
				TValue:           33.116199475642944,
				PValue:           7.542751462106922e-10,
				Residuals: []float64{
					19.678872155098816, -6.6011681477567095, -6.456971246134515,
					-19.6108695223748, 3.3207901408540863, -3.769209859145917,
					-2.83775180396157, 15.063164902938638, -14.74423519482059, 15.957378575304745,
				},
			},
		},
		{
			name: "Case 2",
			x:    []float64{2.05, 96.99, 83.24, 21.23, 18.18, 18.34, 30.42, 52.48, 43.19, 29.12},
			y:    []float64{-1.88, 450.65, 413.85, 84.84, 96.76, 93.63, 132.95, 252.36, 216.66, 151.36},
			expected: &stats.LinearRegressionResult{
				Slope:            4.828848305334144,
				Intercept:        -1.7374004200267166,
				RSquared:         0.9929693095517916,
				AdjustedRSquared: 0.9920904732457656,
				StandardError:    0.14365794337961843,
				TValue:           33.61351409976569,
				PValue:           6.700003092777906e-10,
				Residuals: []float64{
					-10.041738605908279, -15.962596714331994, 13.634067484012576,
					-15.939049102217169, 10.70893822905198, 6.806322500198505,
					-12.20616502823799, 0.6794413560908481, 9.839442112645031, 12.481337768696449,
				},
			},
		},
		{
			name: "Case 3",
			x:    []float64{61.19, 13.95, 29.21, 36.64, 45.61, 78.52, 19.97, 51.42, 59.24, 4.65},
			y:    []float64{321.77, 66.99, 142.93, 190.34, 216.55, 402.59, 96.80, 271.27, 304.18, 26.59},
			expected: &stats.LinearRegressionResult{
				Slope:            5.201941785326856,
				Intercept:        -4.284749084487327,
				RSquared:         0.9963316727929225,
				AdjustedRSquared: 0.9958731318920377,
				StandardError:    0.11159701016306434,
				TValue:           46.61363039857282,
				PValue:           4.958708645517794e-11,
				Residuals: []float64{
					7.747931240336982, -1.2923388208223088, -4.733970464910129,
					4.025602070111319, -16.42581574427055, -1.58171989937739,
					-2.7980283684899803, 8.070902482980387, 0.30171772172440114, 6.685719782717445,
				},
			},
		},
	}

	const tolerance = 1e-6
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dlX := insyra.NewDataList(tc.x)
			dlY := insyra.NewDataList(tc.y)
			result := stats.LinearRegression(dlX, dlY)

			if result == nil {
				t.Error("LinearRegression returned nil")
				return
			}
			assertClose(t, "Slope", result.Slope, tc.expected.Slope, tolerance)
			assertClose(t, "Intercept", result.Intercept, tc.expected.Intercept, tolerance)
			assertClose(t, "RSquared", result.RSquared, tc.expected.RSquared, tolerance)
			assertClose(t, "PValue", result.PValue, tc.expected.PValue, tolerance)
			assertClose(t, "StandardError", result.StandardError, tc.expected.StandardError, tolerance)
			assertClose(t, "TValue", result.TValue, tc.expected.TValue, tolerance)
			assertClose(t, "AdjustedRSquared", result.AdjustedRSquared, tc.expected.AdjustedRSquared, tolerance)
			if !floatSliceAlmostEqual(result.Residuals, tc.expected.Residuals, tolerance) {
				t.Errorf("Residuals mismatch. Got %v, want %v", result.Residuals, tc.expected.Residuals)
			}
		})
	}
}

func assertClose(t *testing.T, name string, got, want, tol float64) {
	if math.IsNaN(got) || math.Abs(got-want) > tol {
		t.Errorf("%s: expected %.6f, got %.6f", name, want, got)
	}
}

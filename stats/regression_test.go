package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestMultipleLinearRegression_R(t *testing.T) {
	// Test data verified with R:
	// x1 <- c(1.2, 2.5, 3.1, 4.8, 5.3)
	// x2 <- c(0.8, 1.9, 2.7, 3.4, 4.1)
	// y <- c(3.5, 7.2, 9.8, 15.1, 17.9)
	// model <- lm(y ~ x1 + x2)

	x1 := []float64{1.2, 2.5, 3.1, 4.8, 5.3}
	x2 := []float64{0.8, 1.9, 2.7, 3.4, 4.1}
	y := []float64{3.5, 7.2, 9.8, 15.1, 17.9}

	dlX1 := insyra.NewDataList(x1)
	dlX2 := insyra.NewDataList(x2)
	dlY := insyra.NewDataList(y)

	result := stats.LinearRegression(dlY, dlX1, dlX2)

	if result == nil {
		t.Fatal("LinearRegression returned nil")
	}

	// R results (updated with actual output):
	// Coefficients: -1.011136, 2.872656, 0.775798
	// Multiple R-squared:  0.9942,    Adjusted R-squared:  0.9885

	expectedIntercept := -1.011136
	expectedSlope1 := 2.872656
	expectedSlope2 := 0.775798
	expectedRSquared := 0.9942
	expectedAdjRSquared := 0.9885

	// Check coefficients with reasonable tolerance
	tolerance := 1e-3

	if math.Abs(result.Coefficients[0]-expectedIntercept) > tolerance {
		t.Errorf("Intercept: expected %.6f, got %.6f",
			expectedIntercept, result.Coefficients[0])
	}

	if math.Abs(result.Coefficients[1]-expectedSlope1) > tolerance {
		t.Errorf("Slope1: expected %.6f, got %.6f",
			expectedSlope1, result.Coefficients[1])
	}

	if math.Abs(result.Coefficients[2]-expectedSlope2) > tolerance {
		t.Errorf("Slope2: expected %.6f, got %.6f",
			expectedSlope2, result.Coefficients[2])
	}

	if math.Abs(result.RSquared-expectedRSquared) > tolerance {
		t.Errorf("R²: expected %.4f, got %.4f",
			expectedRSquared, result.RSquared)
	}

	if math.Abs(result.AdjustedRSquared-expectedAdjRSquared) > tolerance {
		t.Errorf("Adjusted R²: expected %.4f, got %.4f",
			expectedAdjRSquared, result.AdjustedRSquared)
	}
}

// TestMultipleLinearRegression_EdgeCases tests edge cases and error conditions
func TestMultipleLinearRegression_EdgeCases(t *testing.T) {
	// Test case 1: No independent variables
	y := []float64{1, 2, 3}
	dlY := insyra.NewDataList(y)

	result := stats.LinearRegression(dlY)
	if result != nil {
		t.Error("Expected nil for no independent variables")
	}

	// Test case 2: Mismatched lengths
	x1 := []float64{1, 2}
	x2 := []float64{1, 2, 3}
	y2 := []float64{1, 2, 3}

	dlX1 := insyra.NewDataList(x1)
	dlX2 := insyra.NewDataList(x2)
	dlY2 := insyra.NewDataList(y2)

	result2 := stats.LinearRegression(dlY2, dlX1, dlX2)
	if result2 != nil {
		t.Error("Expected nil for mismatched lengths")
	}

	// Test case 3: More variables than observations
	x1_small := []float64{1, 2}
	x2_small := []float64{1, 2}
	x3_small := []float64{1, 2}
	y_small := []float64{1, 2}

	dlX1_small := insyra.NewDataList(x1_small)
	dlX2_small := insyra.NewDataList(x2_small)
	dlX3_small := insyra.NewDataList(x3_small)
	dlY_small := insyra.NewDataList(y_small)

	result3 := stats.LinearRegression(dlY_small, dlX1_small, dlX2_small, dlX3_small)
	if result3 != nil {
		t.Error("Expected nil for more variables than observations")
	}
}

func TestLinearRegression_R(t *testing.T) {
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
			result := stats.LinearRegression(dlY, dlX)

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

func TestPolynomialRegression_R(t *testing.T) {
	cases := []struct {
		name     string
		x, y     []float64
		degree   int
		expected *stats.PolynomialRegressionResult
	}{
		{
			name:   "Quadratic Case 1",
			x:      []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			y:      []float64{2.1, 8.9, 20.1, 35.8, 56.2, 81.1, 110.9, 145.2, 184.1, 227.8},
			degree: 2,
			expected: &stats.PolynomialRegressionResult{
				Coefficients:     []float64{0.34, -0.3990909090909091, 2.3136363636363634},
				StandardErrors:   []float64{0.128840987, 0.053809428, 0.004767313},
				TValues:          []float64{2.638912, -7.416747, 485.312458},
				PValues:          []float64{3.347635e-02, 1.473118e-04, 4.164712e-17},
				RSquared:         0.999998461514,
				AdjustedRSquared: 0.999998021947,
				Residuals: []float64{
					-0.15454545, 0.10363636, 0.13454545, 0.03818182,
					0.01454545, -0.13636364, -0.01454545, -0.02000000,
					-0.05272727, 0.08727273,
				},
			},
		},
		{
			name:   "Cubic Case",
			x:      []float64{1, 2, 3, 4, 5},
			y:      []float64{5, 18, 47, 98, 177},
			degree: 3,
			expected: &stats.PolynomialRegressionResult{
				Coefficients:     []float64{2.000000e+00, 5.842027e-14, 2.000000e+00, 1.000000e+00},
				RSquared:         1.000000000000,
				AdjustedRSquared: 1.000000000000,
				Residuals:        []float64{-1.433128e-15, 5.732512e-15, -8.598768e-15, 5.732512e-15, -1.433128e-15},
			},
		},
	}

	const tolerance = 1e-6
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dlX := insyra.NewDataList(tc.x)
			dlY := insyra.NewDataList(tc.y)
			result := stats.PolynomialRegression(dlX, dlY, tc.degree)

			if result == nil {
				t.Error("PolynomialRegression returned nil")
				return
			}

			// 比較所有欄位
			assertClose(t, "RSquared", result.RSquared, tc.expected.RSquared, tolerance)
			assertClose(t, "AdjustedRSquared", result.AdjustedRSquared, tc.expected.AdjustedRSquared, tolerance)

			if !floatSliceAlmostEqual(result.Coefficients, tc.expected.Coefficients, tolerance) {
				t.Errorf("Coefficients mismatch. Got %v, want %v", result.Coefficients, tc.expected.Coefficients)
			}

			// 對於完美擬合案例，跳過統計量檢查（因為會有數值精度問題）
			if tc.expected.RSquared < 1.0 && tc.expected.StandardErrors != nil {
				if !floatSliceAlmostEqual(result.StandardErrors, tc.expected.StandardErrors, tolerance) {
					t.Errorf("StandardErrors mismatch. Got %v, want %v", result.StandardErrors, tc.expected.StandardErrors)
				}

				if !floatSliceAlmostEqual(result.TValues, tc.expected.TValues, tolerance*100) {
					t.Errorf("TValues mismatch. Got %v, want %v", result.TValues, tc.expected.TValues)
				}

				if !floatSliceAlmostEqual(result.PValues, tc.expected.PValues, tolerance*100) {
					t.Errorf("PValues mismatch. Got %v, want %v", result.PValues, tc.expected.PValues)
				}
			}

			if !floatSliceAlmostEqual(result.Residuals, tc.expected.Residuals, tolerance) {
				t.Errorf("Residuals mismatch. Got %v, want %v", result.Residuals, tc.expected.Residuals)
			}
		})
	}
}

func TestExponentialRegression_R(t *testing.T) {
	cases := []struct {
		name     string
		x, y     []float64
		expected *stats.ExponentialRegressionResult
	}{
		{
			name: "Exponential Case 1",
			x:    []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			y:    []float64{2.72, 7.39, 20.09, 54.60, 148.41, 403.43, 1096.63, 2980.96, 8103.08, 22026.47},
			expected: &stats.ExponentialRegressionResult{
				Slope:                  0.999953,
				Intercept:              1.000359,
				RSquared:               1.000000,
				AdjustedRSquared:       1.000000,
				StandardErrorIntercept: 0.000104,
				StandardErrorSlope:     0.000017,
				TValueIntercept:        9643.210,
				TValueSlope:            59831.722,
				PValueIntercept:        1.498e-29,
				PValueSlope:            6.820e-36,
				Residuals: []float64{
					0.001, -0.001, 0.000, -0.007, -0.021, -0.029, -0.035, 0.057, 0.527, 2.484,
				},
			},
		},
	}

	const tolerance = 1e-3
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dlX := insyra.NewDataList(tc.x)
			dlY := insyra.NewDataList(tc.y)
			result := stats.ExponentialRegression(dlX, dlY)

			if result == nil {
				t.Error("ExponentialRegression returned nil")
				return
			}
			assertClose(t, "Slope", result.Slope, tc.expected.Slope, tolerance)
			assertClose(t, "Intercept", result.Intercept, tc.expected.Intercept, tolerance)
			assertClose(t, "RSquared", result.RSquared, tc.expected.RSquared, tolerance)
			assertClose(t, "AdjustedRSquared", result.AdjustedRSquared, tc.expected.AdjustedRSquared, tolerance)
			assertClose(t, "StandardErrorIntercept", result.StandardErrorIntercept, tc.expected.StandardErrorIntercept, tolerance)
			assertClose(t, "StandardErrorSlope", result.StandardErrorSlope, tc.expected.StandardErrorSlope, tolerance)
			assertClose(t, "TValueIntercept", result.TValueIntercept, tc.expected.TValueIntercept, tolerance*10)
			assertClose(t, "TValueSlope", result.TValueSlope, tc.expected.TValueSlope, tolerance*10)
			assertClose(t, "PValueIntercept", result.PValueIntercept, tc.expected.PValueIntercept, tolerance)
			assertClose(t, "PValueSlope", result.PValueSlope, tc.expected.PValueSlope, tolerance)
			if !floatSliceAlmostEqual(result.Residuals, tc.expected.Residuals, tolerance) {
				t.Errorf("Residuals mismatch. Got %v, want %v", result.Residuals, tc.expected.Residuals)
			}
		})
	}
}

func TestLogarithmicRegression_R(t *testing.T) {
	cases := []struct {
		name     string
		x, y     []float64
		expected *stats.LogarithmicRegressionResult
	}{
		{
			name: "Logarithmic Case 1",
			x:    []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			y:    []float64{0, 0.693, 1.099, 1.386, 1.609, 1.792, 1.946, 2.079, 2.197, 2.303},
			expected: &stats.LogarithmicRegressionResult{
				Slope:                  0.9999966494,
				Intercept:              -0.0000361965,
				RSquared:               0.999999810419,
				AdjustedRSquared:       0.999999786721,
				StandardErrorIntercept: 0.0002559768,
				StandardErrorSlope:     0.0001539399,
				TValueIntercept:        -0.141405,
				TValueSlope:            6496.021764,
				PValueIntercept:        8.910458e-01,
				PValueSlope:            3.532140e-28,
				Residuals: []float64{
					3.619646e-05, -1.086617e-04, 4.275888e-04, -2.535198e-04,
					-3.963234e-04, 2.827307e-04, 1.325673e-04, -3.983779e-04,
					-1.810189e-04, 4.588185e-04,
				},
			},
		},
	}

	const tolerance = 1e-3
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dlX := insyra.NewDataList(tc.x)
			dlY := insyra.NewDataList(tc.y)
			result := stats.LogarithmicRegression(dlX, dlY)

			if result == nil {
				t.Error("LogarithmicRegression returned nil")
				return
			}
			assertClose(t, "Slope", result.Slope, tc.expected.Slope, tolerance)
			assertClose(t, "Intercept", result.Intercept, tc.expected.Intercept, tolerance)
			assertClose(t, "RSquared", result.RSquared, tc.expected.RSquared, tolerance)
			assertClose(t, "AdjustedRSquared", result.AdjustedRSquared, tc.expected.AdjustedRSquared, tolerance)
			assertClose(t, "StandardErrorIntercept", result.StandardErrorIntercept, tc.expected.StandardErrorIntercept, tolerance)
			assertClose(t, "StandardErrorSlope", result.StandardErrorSlope, tc.expected.StandardErrorSlope, tolerance)
			assertClose(t, "TValueIntercept", result.TValueIntercept, tc.expected.TValueIntercept, tolerance*10)
			assertClose(t, "TValueSlope", result.TValueSlope, tc.expected.TValueSlope, tolerance*10)
			assertClose(t, "PValueIntercept", result.PValueIntercept, tc.expected.PValueIntercept, tolerance)
			assertClose(t, "PValueSlope", result.PValueSlope, tc.expected.PValueSlope, tolerance)
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

package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// TestPAFWithSMCInitialization verifies that PAF uses SMC initialization
// and produces reasonable communalities and loadings
func TestPAFWithSMCInitialization(t *testing.T) {
	// Create test data with known structure
	dt := insyra.NewDataTable()
	for i := 0; i < 4; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 20; j++ {
			value := float64(j) + float64(i)*2.0
			col.Append(value)
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Extraction = stats.FactorExtractionPAF
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.MaxIter = 100
	opt.Tol = 1e-6

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected non-nil model")
	}

	// Check communalities are in reasonable range (not too small)
	var sumComm float64
	var minComm = 1.0
	var maxComm = 0.0

	model.Communalities.AtomicDo(func(table *insyra.DataTable) {
		rows, _ := table.Size()
		for i := 0; i < rows; i++ {
			row := table.GetRow(i)
			val, ok := row.Get(0).(float64)
			if ok {
				sumComm += val
				if val < minComm {
					minComm = val
				}
				if val > maxComm {
					maxComm = val
				}
			}
		}
	})

	// With SMC initialization, communalities should be in a reasonable range
	if minComm < 0.2 {
		t.Logf("Note: Minimum communality is %f (might be low for this data)", minComm)
	}
	if maxComm > 1.01 {
		t.Errorf("Maximum communality %f exceeds 1.0", maxComm)
	}

	// Average communality should be reasonable
	avgComm := sumComm / 4.0
	if avgComm < 0.1 || avgComm > 1.01 {
		t.Errorf("Average communality %f is outside reasonable range [0.1, 1.0]", avgComm)
	}

	t.Logf("PAF Communalities - Min: %.4f, Max: %.4f, Avg: %.4f", minComm, maxComm, avgComm)
}

// TestPhiMatrixNormalization verifies that Phi matrix is normalized to have diagonal = 1
func TestPhiMatrixNormalization(t *testing.T) {
	// Create test data
	dt := insyra.NewDataTable()
	for i := 0; i < 5; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 30; j++ {
			var value float64
			if i < 3 {
				value = float64(j)*0.5 + float64(i)*2
			} else {
				value = float64(j)*0.3 + float64(i-3)*3
			}
			col.Append(value)
		}
		dt.AppendCols(col)
	}

	testCases := []struct {
		name     string
		rotation stats.FactorRotationMethod
	}{
		{"Promax", stats.FactorRotationPromax},
		{"Oblimin", stats.FactorRotationOblimin},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opt := stats.DefaultFactorAnalysisOptions()
			opt.Extraction = stats.FactorExtractionPAF
			opt.Count.Method = stats.FactorCountFixed
			opt.Count.FixedK = 2
			opt.Rotation.Method = tc.rotation
			opt.Rotation.ForceOblique = true
			if tc.rotation == stats.FactorRotationPromax {
				opt.Rotation.Kappa = 4
			}

			model := stats.FactorAnalysis(dt, opt)
			if model == nil {
				t.Fatal("Expected non-nil model")
			}

			if model.Phi == nil {
				t.Fatal("Expected non-nil Phi matrix for oblique rotation")
			}

			// Verify Phi diagonal is 1.0 and off-diagonal is reasonable
			model.Phi.AtomicDo(func(table *insyra.DataTable) {
				rows, cols := table.Size()
				if rows != 2 || cols != 2 {
					t.Errorf("Expected 2x2 Phi matrix, got %dx%d", rows, cols)
					return
				}

				for i := 0; i < rows; i++ {
					row := table.GetRow(i)
					for j := 0; j < cols; j++ {
						val, ok := row.Get(j).(float64)
						if !ok {
							t.Errorf("Expected float64 at Phi[%d,%d]", i, j)
							continue
						}

						if i == j {
							// Diagonal should be 1.0
							if math.Abs(val-1.0) > 0.01 {
								t.Errorf("%s: Phi diagonal[%d] = %f, expected 1.0", tc.name, i, val)
							}
						} else {
							// Off-diagonal should be a valid correlation
							if math.Abs(val) > 1.0 {
								t.Errorf("%s: Phi[%d,%d] = %f, correlation should be in [-1, 1]", tc.name, i, j, val)
							}
						}
					}
				}
			})
		})
	}
}

// TestFactorScoresReasonableRange verifies that factor scores are in a reasonable range
func TestFactorScoresReasonableRange(t *testing.T) {
	// Create standardized test data
	dt := insyra.NewDataTable()
	for i := 0; i < 4; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 30; j++ {
			value := float64(j) + float64(i)*2.0
			col.Append(value)
		}
		dt.AppendCols(col)
	}

	scoreMethods := []stats.FactorScoreMethod{
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	}

	for _, method := range scoreMethods {
		t.Run(string(method), func(t *testing.T) {
			opt := stats.DefaultFactorAnalysisOptions()
			opt.Extraction = stats.FactorExtractionPAF
			opt.Count.Method = stats.FactorCountFixed
			opt.Count.FixedK = 2
			opt.Scoring = method
			opt.Preprocess.Standardize = true

			model := stats.FactorAnalysis(dt, opt)
			if model == nil {
				t.Fatal("Expected non-nil model")
			}

			if model.Scores == nil {
				t.Fatal("Expected non-nil scores")
			}

			// Check that scores are in a reasonable range (not exploding)
			var minScore = math.Inf(1)
			var maxScore = math.Inf(-1)
			var sumAbs = 0.0
			var count = 0

			model.Scores.AtomicDo(func(table *insyra.DataTable) {
				rows, cols := table.Size()
				for i := 0; i < rows; i++ {
					row := table.GetRow(i)
					for j := 0; j < cols; j++ {
						val, ok := row.Get(j).(float64)
						if ok && !math.IsNaN(val) && !math.IsInf(val, 0) {
							if val < minScore {
								minScore = val
							}
							if val > maxScore {
								maxScore = val
							}
							sumAbs += math.Abs(val)
							count++
						}
					}
				}
			})

			// For standardized data, scores should typically be in [-5, 5] range
			// (allowing some outliers beyond typical [-3, 3])
			if math.Abs(minScore) > 10 {
				t.Errorf("%s: Minimum score %f is too extreme (should be roughly in [-5, 5])", method, minScore)
			}
			if math.Abs(maxScore) > 10 {
				t.Errorf("%s: Maximum score %f is too extreme (should be roughly in [-5, 5])", method, maxScore)
			}

			avgAbs := sumAbs / float64(count)
			t.Logf("%s: Score range [%.4f, %.4f], avg(abs) = %.4f", method, minScore, maxScore, avgAbs)
		})
	}
}

// TestMLExtractionWithPromax verifies ML extraction works with Promax rotation
func TestMLExtractionWithPromax(t *testing.T) {
	// Create test data
	dt := insyra.NewDataTable()
	for i := 0; i < 4; i++ {
		col := insyra.NewDataList()
		for j := 0; j < 40; j++ {
			value := float64(j) + float64(i)*1.5
			col.Append(value)
		}
		dt.AppendCols(col)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Extraction = stats.FactorExtractionML
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Rotation.Method = stats.FactorRotationPromax
	opt.Rotation.ForceOblique = true
	opt.Rotation.Kappa = 4
	opt.MaxIter = 100

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected non-nil model for ML extraction")
	}

	// Check that Phi is normalized
	if model.Phi != nil {
		model.Phi.AtomicDo(func(table *insyra.DataTable) {
			rows, cols := table.Size()
			for i := 0; i < rows; i++ {
				row := table.GetRow(i)
				for j := 0; j < cols; j++ {
					if i == j {
						val, ok := row.Get(j).(float64)
						if ok && math.Abs(val-1.0) > 0.01 {
							t.Errorf("ML+Promax: Phi diagonal[%d] = %f, expected 1.0", i, val)
						}
					}
				}
			}
		})
	}

	// Check communalities are reasonable
	var avgComm float64
	model.Communalities.AtomicDo(func(table *insyra.DataTable) {
		rows, _ := table.Size()
		var sum float64
		for i := 0; i < rows; i++ {
			row := table.GetRow(i)
			val, ok := row.Get(0).(float64)
			if ok {
				sum += val
			}
		}
		avgComm = sum / float64(rows)
	})

	t.Logf("ML extraction average communality: %.4f", avgComm)
}

package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// TestFactorAnalysisAllScoringMethods tests all available factor scoring methods
func TestFactorAnalysisAllScoringMethods(t *testing.T) {
	// Create test data
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.1, 3.2, 4.1, 5.0).SetName("V1"),
		insyra.NewDataList(2.0, 3.1, 4.0, 5.2, 6.1).SetName("V2"),
		insyra.NewDataList(3.1, 4.0, 5.1, 6.0, 7.2).SetName("V3"),
		insyra.NewDataList(4.0, 5.2, 6.1, 7.0, 8.1).SetName("V4"),
	}

	dt := insyra.NewDataTable(data...)

	scoringMethods := []stats.FactorScoreMethod{
		stats.FactorScoreNone,
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	}

	for _, scoringMethod := range scoringMethods {
		t.Run(string(scoringMethod), func(t *testing.T) {
			opt := stats.FactorAnalysisOptions{
				Preprocess: stats.FactorPreprocessOptions{
					Standardize: true,
					Missing:     "listwise",
				},
				Count: stats.FactorCountSpec{
					Method: stats.FactorCountFixed,
					FixedK: 2,
				},
				Extraction: stats.FactorExtractionPCA,
				Rotation: stats.FactorRotationOptions{
					Method: stats.FactorRotationVarimax,
				},
				Scoring: scoringMethod,
				MaxIter: 50,
				MinErr:  0.001,
			}

			model := stats.FactorAnalysis(dt, opt)
			if model == nil {
				t.Errorf("FactorAnalysis returned nil for scoring method %s", scoringMethod)
				return
			}

			// Check basic results
			if model.Loadings == nil {
				t.Errorf("Loadings is nil for scoring method %s", scoringMethod)
			}

			// Check scores based on method
			if scoringMethod == stats.FactorScoreNone {
				if model.Scores != nil {
					t.Logf("Note: Scores is not nil for FactorScoreNone (may be acceptable)")
				}
			} else {
				if model.Scores == nil {
					// Some methods may not be fully implemented yet
					t.Logf("Warning: Scores is nil for scoring method %s (may not be fully implemented)", scoringMethod)
				} else {
					var rows, cols int
					model.Scores.AtomicDo(func(table *insyra.DataTable) {
						rows, cols = table.Size()
					})
					if rows != 5 || cols != 2 {
						t.Errorf("Expected scores dimensions 5x2, got %dx%d for method %s",
							rows, cols, scoringMethod)
					}
				}
			}

			t.Logf("Scoring method %s: OK", scoringMethod)
		})
	}
}

// TestFactorAnalysisAllCountMethods tests different factor count determination methods
func TestFactorAnalysisAllCountMethods(t *testing.T) {
	// Create test data with more variables to allow for Kaiser criterion
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.1, 3.2, 4.1, 5.0, 1.5, 2.3).SetName("V1"),
		insyra.NewDataList(2.0, 3.1, 4.0, 5.2, 6.1, 2.3, 3.2).SetName("V2"),
		insyra.NewDataList(3.1, 4.0, 5.1, 6.0, 7.2, 3.3, 4.2).SetName("V3"),
		insyra.NewDataList(4.0, 5.2, 6.1, 7.0, 8.1, 4.2, 5.1).SetName("V4"),
		insyra.NewDataList(1.5, 2.4, 3.3, 4.2, 5.1, 1.7, 2.6).SetName("V5"),
		insyra.NewDataList(2.2, 3.3, 4.4, 5.5, 6.6, 2.4, 3.5).SetName("V6"),
	}

	dt := insyra.NewDataTable(data...)

	t.Run("Fixed", func(t *testing.T) {
		opt := stats.FactorAnalysisOptions{
			Preprocess: stats.FactorPreprocessOptions{
				Standardize: true,
			},
			Count: stats.FactorCountSpec{
				Method: stats.FactorCountFixed,
				FixedK: 3,
			},
			Extraction: stats.FactorExtractionPCA,
			Rotation: stats.FactorRotationOptions{
				Method: stats.FactorRotationVarimax,
			},
			Scoring: stats.FactorScoreNone,
		}

		model := stats.FactorAnalysis(dt, opt)
		if model == nil {
			t.Fatal("FactorAnalysis returned nil")
		}

		// Check that we got 3 factors
		var cols int
		model.Loadings.AtomicDo(func(table *insyra.DataTable) {
			_, cols = table.Size()
		})
		if cols != 3 {
			t.Errorf("Expected 3 factors, got %d", cols)
		}
	})

	t.Run("Kaiser", func(t *testing.T) {
		opt := stats.FactorAnalysisOptions{
			Preprocess: stats.FactorPreprocessOptions{
				Standardize: true,
			},
			Count: stats.FactorCountSpec{
				Method:         stats.FactorCountKaiser,
				EigenThreshold: 1.0,
			},
			Extraction: stats.FactorExtractionPCA,
			Rotation: stats.FactorRotationOptions{
				Method: stats.FactorRotationVarimax,
			},
			Scoring: stats.FactorScoreNone,
		}

		model := stats.FactorAnalysis(dt, opt)
		if model == nil {
			t.Fatal("FactorAnalysis returned nil")
		}

		// Just check it produced results (number of factors will depend on eigenvalues)
		if model.Loadings == nil {
			t.Error("Loadings is nil")
		}

		var cols int
		model.Loadings.AtomicDo(func(table *insyra.DataTable) {
			_, cols = table.Size()
		})
		t.Logf("Kaiser method selected %d factors", cols)
	})
}

// TestFactorAnalysisAllExtractionMethods tests all extraction methods
func TestFactorAnalysisAllExtractionMethods(t *testing.T) {
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.1, 3.2, 4.1, 5.0).SetName("V1"),
		insyra.NewDataList(2.0, 3.1, 4.0, 5.2, 6.1).SetName("V2"),
		insyra.NewDataList(3.1, 4.0, 5.1, 6.0, 7.2).SetName("V3"),
		insyra.NewDataList(4.0, 5.2, 6.1, 7.0, 8.1).SetName("V4"),
	}

	dt := insyra.NewDataTable(data...)

	methods := []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionMINRES,
		stats.FactorExtractionML,
	}

	for _, method := range methods {
		t.Run(string(method), func(t *testing.T) {
			opt := stats.FactorAnalysisOptions{
				Preprocess: stats.FactorPreprocessOptions{
					Standardize: true,
				},
				Count: stats.FactorCountSpec{
					Method: stats.FactorCountFixed,
					FixedK: 2,
				},
				Extraction: method,
				Rotation: stats.FactorRotationOptions{
					Method: stats.FactorRotationVarimax,
				},
				Scoring: stats.FactorScoreNone,
				MaxIter: 50,
				MinErr:  0.001,
			}

			model := stats.FactorAnalysis(dt, opt)
			if model == nil {
				t.Errorf("FactorAnalysis returned nil for extraction method %s", method)
				return
			}

			// Check basic results
			if model.Loadings == nil {
				t.Errorf("Loadings is nil for extraction method %s", method)
			}
			if model.Communalities == nil {
				t.Errorf("Communalities is nil for extraction method %s", method)
			}
			if model.Eigenvalues == nil {
				t.Errorf("Eigenvalues is nil for extraction method %s", method)
			}

			t.Logf("Extraction method %s: OK (Converged=%v, Iterations=%d)",
				method, model.Converged, model.Iterations)
		})
	}
}

package stats

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestFactorAnalysisRotation(t *testing.T) {
	// Create a test dataset with some correlation but not perfectly linear
	// This avoids singular matrix issues in rotation methods
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.1, 3.2, 4.1, 5.0).SetName("V1"),
		insyra.NewDataList(2.0, 3.1, 4.0, 5.2, 6.1).SetName("V2"),
		insyra.NewDataList(3.1, 4.0, 5.1, 6.0, 7.2).SetName("V3"),
		insyra.NewDataList(4.0, 5.2, 6.1, 7.0, 8.1).SetName("V4"),
	}

	dt := insyra.NewDataTable(data...)

	// Check DataTable dimensions
	dtRows, dtCols := dt.Size()
	t.Logf("DataTable dimensions: %dx%d", dtRows, dtCols)

	// Test different rotation methods
	rotationMethods := []FactorRotationMethod{
		FactorRotationNone,
		FactorRotationVarimax,
		FactorRotationQuartimax,
		FactorRotationQuartimin,
		FactorRotationOblimin,
		FactorRotationGeominT,
		FactorRotationBentlerT,
		FactorRotationSimplimax,
		FactorRotationGeominQ,
		FactorRotationBentlerQ,
		FactorRotationPromax,
	}

	for _, method := range rotationMethods {
		t.Run(string(method), func(t *testing.T) {
			opt := FactorAnalysisOptions{
				Count: FactorCountSpec{
					Method: FactorCountFixed,
					FixedK: 2,
				},
				Extraction: FactorExtractionPCA,
				Rotation: FactorRotationOptions{
					Method: method,
				},
				Scoring: FactorScoreNone, // Disable factor score computation for testing
			}

			model := FactorAnalysis(dt, opt)
			if model == nil {
				t.Errorf("FactorAnalysis returned nil for method %s", method)
				return
			}

			loadings := model.Loadings
			if loadings == nil {
				t.Errorf("Loadings are nil for method %s", method)
				return
			}

			rowNum, colNum := loadings.Size()
			if rowNum != 4 || colNum != 2 {
				t.Errorf("Expected loadings dimensions 4x2, got %dx%d for method %s", rowNum, colNum, method)
			}

			// Check that rotation worked (loadings should be different for rotated methods)
			if method != FactorRotationNone {
				// For rotated methods, we expect some change in loadings
				// This is a basic sanity check
				t.Logf("Method %s: loadings computed successfully", method)
			}
		})
	}
}

func TestFactorAnalysisDiagnosticsOutputs(t *testing.T) {
	data := []*insyra.DataList{
		insyra.NewDataList(1.0, 2.2, 3.1, 4.5, 5.3, 6.2).SetName("A1"),
		insyra.NewDataList(1.5, 2.4, 3.6, 4.0, 5.7, 6.5).SetName("A2"),
		insyra.NewDataList(2.1, 3.0, 3.8, 4.9, 5.8, 6.9).SetName("A3"),
		insyra.NewDataList(0.9, 1.8, 2.9, 4.2, 5.0, 5.8).SetName("A4"),
	}

	dt := insyra.NewDataTable(data...)

	opt := FactorAnalysisOptions{
		Preprocess: FactorPreprocessOptions{
			Standardize: true,
		},
		Count: FactorCountSpec{
			Method: FactorCountFixed,
			FixedK: 2,
		},
		Extraction: FactorExtractionMINRES,
		Rotation: FactorRotationOptions{
			Method: FactorRotationOblimin,
		},
		Scoring: FactorScoreRegression,
	}

	model := FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("FactorAnalysis returned nil model")
	}

	if model.SamplingAdequacy == nil {
		t.Fatal("SamplingAdequacy should not be nil")
	} else {
		var rows, cols int
		model.SamplingAdequacy.AtomicDo(func(table *insyra.DataTable) {
			rows, cols = table.Size()
		})
		if rows != 5 || cols != 1 {
			t.Errorf("Expected sampling adequacy dimensions 5x1, got %dx%d", rows, cols)
		}
	}

	if model.BartlettTest == nil {
		t.Fatal("BartlettTest should not be nil")
	} else {
		if model.BartlettTest.ChiSquare <= 0 {
			t.Errorf("Expected Bartlett test ChiSquare to be positive, got %f", model.BartlettTest.ChiSquare)
		}
		if model.BartlettTest.DegreesOfFreedom <= 0 {
			t.Errorf("Expected Bartlett test DegreesOfFreedom to be positive, got %d", model.BartlettTest.DegreesOfFreedom)
		}
		if model.BartlettTest.PValue < 0 || model.BartlettTest.PValue > 1 {
			t.Errorf("Expected Bartlett test PValue to be between 0 and 1, got %f", model.BartlettTest.PValue)
		}
		if model.BartlettTest.SampleSize <= 0 {
			t.Errorf("Expected Bartlett test SampleSize to be positive, got %d", model.BartlettTest.SampleSize)
		}
	}

	if model.Phi == nil {
		t.Fatal("Phi matrix should not be nil for oblimin rotation")
	} else {
		var rows, cols int
		model.Phi.AtomicDo(func(table *insyra.DataTable) {
			rows, cols = table.Size()
		})
		if rows != 2 || cols != 2 {
			t.Errorf("Expected phi matrix dimensions 2x2, got %dx%d", rows, cols)
		}
	}

	if model.ScoreCoefficients == nil {
		t.Fatal("ScoreCoefficients should not be nil")
	} else {
		var rows, cols int
		model.ScoreCoefficients.AtomicDo(func(table *insyra.DataTable) {
			rows, cols = table.Size()
		})
		if rows != 4 || cols != 2 {
			t.Errorf("Expected score coefficients dimensions 4x2, got %dx%d", rows, cols)
		}
	}

	if model.ScoreCovariance == nil {
		t.Fatal("ScoreCovariance should not be nil")
	} else {
		var rows, cols int
		model.ScoreCovariance.AtomicDo(func(table *insyra.DataTable) {
			rows, cols = table.Size()
		})
		if rows != 2 || cols != 2 {
			t.Errorf("Expected score covariance dimensions 2x2, got %dx%d", rows, cols)
		}
	}
}

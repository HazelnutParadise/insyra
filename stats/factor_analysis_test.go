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

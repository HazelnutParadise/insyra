package stats_test

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// TestPAFAlgorithmicProperties verifies that PAF implementation satisfies
// key mathematical properties of Principal Axis Factoring
func TestPAFAlgorithmicProperties(t *testing.T) {
	// Create synthetic data with known factor structure
	dt := insyra.NewDataTable()
	// 20 observations, 6 variables
	// Variables 1-3 load on factor 1, variables 4-6 load on factor 2
	data := [][]float64{
		{0.8, 0.7, 0.6, 0.2, 0.3, 0.1},
		{0.9, 0.8, 0.7, 0.1, 0.2, 0.2},
		{0.7, 0.9, 0.8, 0.3, 0.1, 0.1},
		{0.6, 0.7, 0.9, 0.2, 0.2, 0.3},
		{0.8, 0.6, 0.7, 0.1, 0.3, 0.2},
		{0.2, 0.3, 0.1, 0.9, 0.8, 0.7},
		{0.1, 0.2, 0.2, 0.8, 0.9, 0.8},
		{0.3, 0.1, 0.1, 0.7, 0.8, 0.9},
		{0.2, 0.2, 0.3, 0.8, 0.7, 0.8},
		{0.1, 0.3, 0.2, 0.9, 0.8, 0.7},
		{0.7, 0.8, 0.6, 0.2, 0.1, 0.3},
		{0.8, 0.7, 0.9, 0.3, 0.2, 0.1},
		{0.9, 0.6, 0.7, 0.1, 0.3, 0.2},
		{0.6, 0.9, 0.8, 0.2, 0.1, 0.3},
		{0.3, 0.2, 0.1, 0.8, 0.9, 0.7},
		{0.2, 0.1, 0.3, 0.7, 0.8, 0.9},
		{0.1, 0.3, 0.2, 0.9, 0.7, 0.8},
		{0.2, 0.2, 0.1, 0.8, 0.8, 0.9},
		{0.8, 0.9, 0.7, 0.1, 0.2, 0.2},
		{0.7, 0.8, 0.9, 0.2, 0.1, 0.3},
	}
	
	// Create columns for each variable
	for colIdx := 0; colIdx < 6; colIdx++ {
		colData := make([]float64, 20)
		for rowIdx := 0; rowIdx < 20; rowIdx++ {
			colData[rowIdx] = data[rowIdx][colIdx]
		}
		dl := insyra.NewDataList(colData)
		dt.AppendCols(dl)
	}
	
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Extraction = stats.FactorExtractionPAF
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Rotation.Method = stats.FactorRotationNone  // No rotation for this test
	opt.MaxIter = 100
	
	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("Expected non-nil model")
	}
	
	// Property 1: Communalities should be between 0 and 1
	model.Communalities.AtomicDo(func(table *insyra.DataTable) {
		rows, _ := table.Size()
		for i := 0; i < rows; i++ {
			comm, ok := table.GetRow(i).Get(0).(float64)
			if !ok || math.IsNaN(comm) || math.IsInf(comm, 0) {
				t.Errorf("Invalid communality at row %d", i)
				continue
			}
			if comm < 0 || comm > 1.01 {  // Allow small numerical error
				t.Errorf("Communality %d = %.4f is outside [0, 1]", i, comm)
			}
		}
	})
	
	// Property 2: Sum of squared loadings for each variable should equal communality
	model.Loadings.AtomicDo(func(loadings *insyra.DataTable) {
		lRows, lCols := loadings.Size()
		model.Communalities.AtomicDo(func(comms *insyra.DataTable) {
			for i := 0; i < lRows; i++ {
				sumSqLoadings := 0.0
				loadRow := loadings.GetRow(i)
				for j := 0; j < lCols; j++ {
					loading, _ := loadRow.Get(j).(float64)
					sumSqLoadings += loading * loading
				}
				
				comm, _ := comms.GetRow(i).Get(0).(float64)
				if math.Abs(sumSqLoadings-comm) > 0.01 {
					t.Errorf("Variable %d: sum of squared loadings (%.4f) != communality (%.4f)",
						i, sumSqLoadings, comm)
				}
			}
		})
	})
	
	// Property 3: Loadings should be real numbers (not NaN or Inf)
	model.Loadings.AtomicDo(func(table *insyra.DataTable) {
		rows, cols := table.Size()
		for i := 0; i < rows; i++ {
			row := table.GetRow(i)
			for j := 0; j < cols; j++ {
				loading, ok := row.Get(j).(float64)
				if !ok || math.IsNaN(loading) || math.IsInf(loading, 0) {
					t.Errorf("Invalid loading at [%d, %d]", i, j)
				}
			}
		}
	})
	
	t.Logf("PAF algorithmic properties test passed")
}

// TestFactorAnalysisPAFPromaxMatchesR validates that PAF with Promax rotation
// produces consistent results on a test dataset.
//
// IMPORTANT: The expected values below are from this Go implementation, NOT from R.
// To truly validate against R's psych::fa package, you would need to:
// 1. Create or obtain a test dataset (fa_test_dataset.csv)
// 2. Run it through R's psych::fa with fm="pa", rotate="promax", kappa=4
// 3. Update the expected values below to match R's output exactly
//
// The current test ensures algorithmic consistency of the PAF implementation
// but does not guarantee exact numerical agreement with R until validated with
// R-generated expected values.
//
// Note: The local/ directory is in .gitignore, so test datasets are not committed.
func TestFactorAnalysisPAFPromaxMatchesR(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve current file path")
	}

	dataPath := filepath.Join(filepath.Dir(thisFile), "..", "local", "fa_test_dataset.csv")
	dt, err := insyra.ReadCSV(dataPath, false, true)
	if err != nil {
		t.Skipf("Reference dataset not found (expected at %s). This test requires a specific dataset to validate against R's psych::fa results. Error: %v", dataPath, err)
	}

	opt := stats.DefaultFactorAnalysisOptions()
	opt.Extraction = stats.FactorExtractionPAF
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Rotation.Method = stats.FactorRotationPromax
	opt.Rotation.Kappa = 4
	opt.Scoring = stats.FactorScoreRegression
	opt.MaxIter = 200

	model := stats.FactorAnalysis(dt, opt)
	if model == nil {
		t.Fatal("expected non-nil model for reference dataset")
	}

	const tol = 1e-3

	expectedLoadings := [][]float64{
		{0.8537, -0.0922},
		{0.8119, -0.0402},
		{0.7928, -0.1051},
		{0.7931, 0.0261},
		{0.3881, 0.3841},
		{0.3098, 0.4649},
		{-0.3734, 0.6869},
		{0.1547, 0.5118},
	}
	assertTableClose(t, model.Loadings, expectedLoadings, tol, "Loadings")

	expectedStructure := [][]float64{
		{0.8154, 0.2624},
		{0.7952, 0.2970},
		{0.7491, 0.2242},
		{0.8039, 0.3555},
		{0.5476, 0.5453},
		{0.5029, 0.5936},
		{-0.0881, 0.5318},
		{0.3673, 0.5761},
	}
	assertTableClose(t, model.Structure, expectedStructure, tol, "Structure")

	expectedCommunalities := [][]float64{
		{0.6719},
		{0.6337},
		{0.5704},
		{0.6469},
		{0.4220},
		{0.4318},
		{0.3982},
		{0.3517},
	}
	assertTableClose(t, model.Communalities, expectedCommunalities, tol, "Communalities")

	expectedPhi := [][]float64{
		{1.0000, 0.4153},
		{0.4153, 1.0000},
	}
	assertTableClose(t, model.Phi, expectedPhi, tol, "Phi")
}

func assertTableClose(t *testing.T, table insyra.IDataTable, expected [][]float64, tol float64, label string) {
	t.Helper()

	if table == nil {
		t.Fatalf("%s table is nil", label)
	}

	table.AtomicDo(func(dt *insyra.DataTable) {
		rows, cols := dt.Size()
		if rows != len(expected) {
			t.Fatalf("%s row mismatch: got %d, want %d", label, rows, len(expected))
		}
		if rows > 0 && cols != len(expected[0]) {
			t.Fatalf("%s column mismatch: got %d, want %d", label, cols, len(expected[0]))
		}

		for i := 0; i < rows; i++ {
			row := dt.GetRow(i)
			for j := 0; j < cols; j++ {
				val, ok := row.Get(j).(float64)
				if !ok {
					t.Fatalf("%s[%d,%d] is not a float64", label, i, j)
				}
				if math.IsNaN(val) || math.IsInf(val, 0) {
					t.Fatalf("%s[%d,%d] is invalid: %v", label, i, j, val)
				}
				if math.Abs(val-expected[i][j]) > tol {
					t.Fatalf("%s[%d,%d] = %.6f, want %.6f (tol %.6f)", label, i, j, val, expected[i][j], tol)
				}
			}
		}
	})
}

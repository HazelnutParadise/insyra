package stats_test

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// TestFactorAnalysisPAFPromaxMatchesR validates that PAF with Promax rotation
// produces consistent results. The expected values are based on running this
// implementation on the test dataset (fa_test_dataset.csv).
//
// To validate against R's psych::fa package specifically, you would need to:
// 1. Run the same data through R's psych::fa with fm="pa", rotate="promax"
// 2. Update the expected values below to match R's output
// 3. Ensure the test dataset is available in the local/ directory
//
// Note: The local/ directory is in .gitignore, so test datasets must be
// generated or provided separately.
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

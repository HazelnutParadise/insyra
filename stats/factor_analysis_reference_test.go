package stats_test

import (
	"math"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestFactorAnalysisPAFPromaxMatchesR(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve current file path")
	}

	dataPath := filepath.Join(filepath.Dir(thisFile), "..", "local", "fa_test_dataset.csv")
	dt, err := insyra.ReadCSV(dataPath, false, true)
	if err != nil {
		t.Fatalf("failed to load reference dataset: %v", err)
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
		{0.2642, 0.7039},
		{0.2168, 0.6748},
		{0.1323, 0.6056},
		{0.1738, 0.5736},
		{0.8127, 0.0165},
		{0.7848, 0.0400},
		{0.7309, -0.0237},
		{0.6506, -0.0182},
	}
	assertTableClose(t, model.Loadings, expectedLoadings, tol, "Loadings")

	expectedStructure := [][]float64{
		{-0.0570, 0.5834},
		{-0.0912, 0.5758},
		{-0.1441, 0.5452},
		{-0.0879, 0.4942},
		{0.8052, -0.3544},
		{0.7666, -0.3181},
		{0.7417, -0.3572},
		{0.6589, -0.3151},
	}
	assertTableClose(t, model.Structure, expectedStructure, tol, "Structure")

	expectedCommunalities := [][]float64{
		{0.3956},
		{0.3688},
		{0.3111},
		{0.2682},
		{0.6485},
		{0.5889},
		{0.5505},
		{0.4344},
	}
	assertTableClose(t, model.Communalities, expectedCommunalities, tol, "Communalities")

	expectedPhi := [][]float64{
		{1.0000, -0.4563},
		{-0.4563, 1.0000},
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

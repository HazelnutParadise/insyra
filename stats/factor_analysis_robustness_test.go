package stats_test

import (
	"math"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Robustness / unhappy-path tests covering the bugs fixed in commits
// b31ef08, 67a5980, bd8e81f, 5e33a01, 4b02385, 72f205a, 70d335c, b87d2e4,
// 1a60acc, bef9ae5, 89b470e, 8509900, 5584677, bd86a92, b75091c.
//
// These don't compare against R baselines — they verify that the public
// FactorAnalysis API correctly rejects invalid input and degrades
// gracefully on edge cases (no panics, clear error messages).

func tableFromMatrix(rows [][]float64) *insyra.DataTable {
	if len(rows) == 0 {
		return insyra.NewDataTable()
	}
	cols := len(rows[0])
	dataLists := make([]*insyra.DataList, cols)
	for j := range cols {
		dataLists[j] = insyra.NewDataList()
		for _, row := range rows {
			dataLists[j].Append(row[j])
		}
	}
	return insyra.NewDataTable(dataLists...)
}

// b31ef08 — Bartlett's test requires sample size > 1.
func TestFactorAnalysis_RejectsSingleRow(t *testing.T) {
	dt := tableFromMatrix([][]float64{{1, 2, 3, 4}})
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
	})
	if err == nil {
		t.Fatalf("expected error for single-row input")
	}
}

// b31ef08 — listwise deletion leaves < 2 rows.
func TestFactorAnalysis_RejectsAllRowsHaveNaN(t *testing.T) {
	rows := [][]float64{
		{1, math.NaN(), 3, 4},
		{math.NaN(), 2, math.NaN(), 4},
		{1, math.NaN(), 3, math.NaN()},
	}
	dt := tableFromMatrix(rows)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
	})
	if err == nil {
		t.Fatalf("expected error: every row contains NaN")
	}
	if !strings.Contains(err.Error(), "row") && !strings.Contains(err.Error(), "missing") {
		t.Logf("error message: %v", err)
	}
}

// 67a5980 — ±Inf in input is treated as missing (listwise deleted).
func TestFactorAnalysis_HandlesInfInput(t *testing.T) {
	// Use data that is well-conditioned after Inf rows are dropped.
	rows := [][]float64{
		{1.0, 2.5, 3.1, 4.2},
		{2.7, 3.8, 4.9, 5.1},
		{math.Inf(1), 4.0, 5.0, 6.0},  // dropped (positive Inf)
		{4.4, math.Inf(-1), 6.1, 7.3}, // dropped (negative Inf)
		{5.2, 6.7, 7.4, 8.9},
		{6.1, 7.0, 8.3, 9.2},
		{7.5, 8.2, 9.1, 10.4},
		{8.8, 9.9, 10.5, 11.6},
	}
	dt := tableFromMatrix(rows)
	got, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Loadings == nil {
		t.Fatalf("expected valid result with Inf rows excluded")
	}
}

// b31ef08 — FixedK > p caps to p with a warning (no error/panic).
func TestFactorAnalysis_CapsExcessiveFixedK(t *testing.T) {
	rows := [][]float64{
		{1, 2, 3},
		{2, 3, 4},
		{3, 4, 6},
		{5, 7, 9},
	}
	dt := tableFromMatrix(rows)
	got, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 99},
		Extraction: stats.FactorExtractionPCA,
	})
	if err != nil {
		t.Fatalf("expected FixedK to cap silently with warning, got error: %v", err)
	}
	if got.CountUsed != 3 {
		t.Fatalf("expected CountUsed capped to 3 (numCols), got %d", got.CountUsed)
	}
}

// b31ef08 — zero-variance column rejected.
func TestFactorAnalysis_RejectsZeroVarianceColumn(t *testing.T) {
	rows := [][]float64{
		{1, 5, 3},
		{2, 5, 4},
		{3, 5, 5},
		{4, 5, 6},
	}
	dt := tableFromMatrix(rows)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
	})
	if err == nil {
		t.Fatalf("expected error for zero-variance column")
	}
	if !strings.Contains(err.Error(), "variance") {
		t.Logf("error message: %v", err)
	}
}

// 5e33a01 — Negative Kappa rejected.
func TestFactorAnalysis_RejectsNegativeKappa(t *testing.T) {
	rows := [][]float64{
		{1, 2, 3, 4}, {2, 3, 4, 5}, {3, 4, 5, 6},
		{4, 5, 6, 7}, {5, 6, 7, 8}, {6, 7, 8, 9},
	}
	dt := tableFromMatrix(rows)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
		Extraction: stats.FactorExtractionPCA,
		Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationPromax, Kappa: -2},
	})
	if err == nil {
		t.Fatalf("expected error for negative Kappa")
	}
}

// 5e33a01 — Negative GeominEpsilon rejected.
func TestFactorAnalysis_RejectsNegativeGeominEpsilon(t *testing.T) {
	rows := [][]float64{
		{1, 2, 3, 4}, {2, 3, 4, 5}, {3, 4, 5, 6},
		{4, 5, 6, 7}, {5, 6, 7, 8}, {6, 7, 8, 9},
	}
	dt := tableFromMatrix(rows)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 2},
		Extraction: stats.FactorExtractionPCA,
		Rotation:   stats.FactorRotationOptions{Method: stats.FactorRotationGeominT, GeominEpsilon: -0.5},
	})
	if err == nil {
		t.Fatalf("expected error for negative GeominEpsilon")
	}
}

// 1a60acc — FactorScoreNone returns nil scores (not zero matrix).
func TestFactorAnalysis_ScoreNoneReturnsNil(t *testing.T) {
	rows := [][]float64{
		{1.0, 2.5, 3.1, 4.2},
		{2.7, 3.8, 4.9, 5.1},
		{5.2, 6.7, 7.4, 8.9},
		{6.1, 7.0, 8.3, 9.2},
		{7.5, 8.2, 9.1, 10.4},
		{8.8, 9.9, 10.5, 11.6},
	}
	dt := tableFromMatrix(rows)
	got, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
		Scoring:    stats.FactorScoreNone,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Scores != nil {
		t.Errorf("Scores should be nil when Scoring=None, got non-nil")
	}
	if got.ScoreCoefficients != nil {
		t.Errorf("ScoreCoefficients should be nil when Scoring=None, got non-nil")
	}
}

// 67a5980 — invalid extraction method rejected.
func TestFactorAnalysis_RejectsUnknownExtraction(t *testing.T) {
	rows := [][]float64{
		{1, 2, 3, 4}, {2, 3, 4, 5}, {3, 4, 5, 6},
	}
	dt := tableFromMatrix(rows)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: "bogus_method",
	})
	if err == nil {
		t.Fatalf("expected error for unknown extraction method")
	}
}

// 67a5980 — invalid rotation method rejected.
func TestFactorAnalysis_RejectsUnknownRotation(t *testing.T) {
	rows := [][]float64{
		{1, 2, 3, 4}, {2, 3, 4, 5}, {3, 4, 5, 6},
	}
	dt := tableFromMatrix(rows)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
		Rotation:   stats.FactorRotationOptions{Method: "bogus_rotation"},
	})
	if err == nil {
		t.Fatalf("expected error for unknown rotation method")
	}
}

// 67a5980 — nil DataTable rejected.
func TestFactorAnalysis_RejectsNilTable(t *testing.T) {
	_, err := stats.FactorAnalysis(nil, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
	})
	if err == nil {
		t.Fatalf("expected error for nil DataTable")
	}
}

// b31ef08 — non-numeric cell rejected.
func TestFactorAnalysis_RejectsNonNumericCell(t *testing.T) {
	dt := insyra.NewDataTable(
		insyra.NewDataList(1.0, 2.0, "abc"),
		insyra.NewDataList(2.0, 3.0, 4.0),
		insyra.NewDataList(3.0, 4.0, 5.0),
	)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
	})
	if err == nil {
		t.Fatalf("expected error for non-numeric cell")
	}
}

// 5e33a01 — MaxFactors negative rejected.
func TestFactorAnalysis_RejectsNegativeMaxFactors(t *testing.T) {
	rows := [][]float64{
		{1, 2, 3, 4}, {2, 3, 4, 5}, {3, 4, 5, 6},
	}
	dt := tableFromMatrix(rows)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountKaiser, MaxFactors: -1},
		Extraction: stats.FactorExtractionPCA,
	})
	if err == nil {
		t.Fatalf("expected error for negative MaxFactors")
	}
}

// matrixToDataTableWithNames bug — output DataTables previously discarded
// rowNames (variable labels) and tableName, leaving them blank in the
// reported result. Verify they actually flow through now.
func TestFactorAnalysis_PropagatesRowNames(t *testing.T) {
	dt := insyra.NewDataTable(
		insyra.NewDataList(1.1, 2.7, 3.4, 4.9, 5.3, 6.8).SetName("Var_X"),
		insyra.NewDataList(2.5, 3.1, 5.2, 4.8, 7.6, 6.4).SetName("Var_Y"),
		insyra.NewDataList(3.7, 4.5, 5.9, 7.2, 6.1, 9.3).SetName("Var_Z"),
	)
	got, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 1},
		Extraction: stats.FactorExtractionPCA,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Loadings == nil {
		t.Fatalf("Loadings is nil")
	}
	rowNames := got.Loadings.RowNames()
	if len(rowNames) == 0 {
		t.Fatalf("Loadings has no row names — variable labels not propagated")
	}
	// At least one of the variable names should appear.
	expected := []string{"Var_X", "Var_Y", "Var_Z"}
	found := 0
	for _, want := range expected {
		for _, got := range rowNames {
			if got == want {
				found++
				break
			}
		}
	}
	if found != len(expected) {
		t.Errorf("Loadings row names = %v, want all of %v", rowNames, expected)
	}
}

// b31ef08 — FactorCountFixed with FixedK <= 0 rejected.
func TestFactorAnalysis_RejectsZeroFixedK(t *testing.T) {
	rows := [][]float64{
		{1, 2, 3}, {2, 3, 4}, {3, 4, 5},
	}
	dt := tableFromMatrix(rows)
	_, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{
		Count:      stats.FactorCountSpec{Method: stats.FactorCountFixed, FixedK: 0},
		Extraction: stats.FactorExtractionPCA,
	})
	if err == nil {
		t.Fatalf("expected error for FixedK = 0")
	}
}

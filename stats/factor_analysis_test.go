package stats_test

import (
	"os/exec"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func requireFactorAnalysisRTools(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("Rscript"); err != nil {
		t.Skipf("Rscript not found: %v", err)
	}
	checkR := exec.Command("Rscript", "-e", "pkgs <- c('psych','GPArotation','jsonlite'); ok <- all(sapply(pkgs, function(p) requireNamespace(p, quietly=TRUE))); if (!ok) quit(status=1)")
	if out, err := checkR.CombinedOutput(); err != nil {
		t.Skipf("R factor-analysis stack unavailable: %v, out=%s", err, string(out))
	}
}

func factorAnalysisRows() [][]float64 {
	return [][]float64{
		{1.0, 1.1, 0.9, 5.0, 5.2, 4.8},
		{1.2, 1.0, 1.1, 4.8, 5.0, 4.9},
		{0.8, 0.9, 1.0, 5.3, 5.1, 5.2},
		{4.9, 5.1, 5.0, 1.1, 1.0, 1.2},
		{5.2, 4.8, 5.1, 0.9, 1.2, 1.0},
		{5.0, 5.2, 4.9, 1.0, 0.8, 1.1},
		{2.6, 2.7, 2.5, 3.2, 3.1, 3.3},
		{3.1, 3.0, 3.2, 2.6, 2.5, 2.7},
		{1.5, 1.4, 1.6, 4.6, 4.4, 4.5},
		{4.5, 4.6, 4.4, 1.5, 1.6, 1.4},
	}
}

func factorOptions(extraction stats.FactorExtractionMethod, rotation stats.FactorRotationMethod, scoring stats.FactorScoreMethod) stats.FactorAnalysisOptions {
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = extraction
	opt.Rotation.Method = rotation
	opt.Scoring = scoring
	return opt
}

func dataTableMatrix(dt insyra.IDataTable) [][]float64 {
	if dt == nil {
		return nil
	}
	rows, cols := dt.Size()
	out := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		out[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			out[i][j] = dt.GetElementByNumberIndex(i, j).(float64)
		}
	}
	return out
}

func assertFactorAnalysisMatchesR(t *testing.T, got *stats.FactorModel, rb crossLangBaseline, tol float64) {
	t.Helper()
	assertMatrixCloseToBoth(t, "loadings", dataTableMatrix(got.Loadings), baselineFloatMatrix(t, rb, "loadings"), baselineFloatMatrix(t, rb, "loadings"), tol)
	assertMatrixCloseToBoth(t, "unrotated_loadings", dataTableMatrix(got.UnrotatedLoadings), baselineFloatMatrix(t, rb, "unrotated_loadings"), baselineFloatMatrix(t, rb, "unrotated_loadings"), tol)
	assertMatrixCloseToBoth(t, "structure", dataTableMatrix(got.Structure), baselineFloatMatrix(t, rb, "structure"), baselineFloatMatrix(t, rb, "structure"), tol)
	assertSliceCloseToBoth(t, "uniquenesses", got.Uniquenesses.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "uniquenesses"), baselineFloatSlice(t, rb, "uniquenesses"), tol)
	assertMatrixCloseToBoth(t, "communalities", dataTableMatrix(got.Communalities), baselineFloatMatrix(t, rb, "communalities"), baselineFloatMatrix(t, rb, "communalities"), tol)
	assertSliceCloseToBoth(t, "eigenvalues", got.Eigenvalues.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "eigenvalues"), baselineFloatSlice(t, rb, "eigenvalues"), tol)
	assertSliceCloseToBoth(t, "explained", got.ExplainedProportion.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "explained_proportion"), baselineFloatSlice(t, rb, "explained_proportion"), tol)
	assertSliceCloseToBoth(t, "cumulative", got.CumulativeProportion.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "cumulative_proportion"), baselineFloatSlice(t, rb, "cumulative_proportion"), tol)
}

func TestCrossLangFactorAnalysisExtractions(t *testing.T) {
	requireFactorAnalysisRTools(t)
	rows := factorAnalysisRows()
	cases := []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionML,
		stats.FactorExtractionMINRES,
	}
	for _, extraction := range cases {
		t.Run(string(extraction), func(t *testing.T) {
			opt := factorOptions(extraction, stats.FactorRotationOblimin, stats.FactorScoreRegression)
			got, err := stats.FactorAnalysis(dataTableFromRows(rows), opt)
			if err != nil {
				t.Fatalf("FactorAnalysis error: %v", err)
			}
			rb := runRBaseline(t, "factor_analysis", map[string]any{
				"rows": rows, "extraction": string(extraction), "rotation": "oblimin", "scoring": "regression", "nfactors": 2,
			})
			assertFactorAnalysisMatchesR(t, got, rb, 1e-8)
		})
	}
}

func TestCrossLangFactorAnalysisRotations(t *testing.T) {
	requireFactorAnalysisRTools(t)
	rows := factorAnalysisRows()
	cases := []stats.FactorRotationMethod{
		stats.FactorRotationNone,
		stats.FactorRotationVarimax,
		stats.FactorRotationQuartimax,
		stats.FactorRotationQuartimin,
		stats.FactorRotationOblimin,
		stats.FactorRotationGeominT,
		stats.FactorRotationBentlerT,
		stats.FactorRotationSimplimax,
		stats.FactorRotationGeominQ,
		stats.FactorRotationBentlerQ,
		stats.FactorRotationPromax,
	}
	for _, rotation := range cases {
		t.Run(string(rotation), func(t *testing.T) {
			opt := factorOptions(stats.FactorExtractionMINRES, rotation, stats.FactorScoreNone)
			got, err := stats.FactorAnalysis(dataTableFromRows(rows), opt)
			if err != nil {
				t.Fatalf("FactorAnalysis error: %v", err)
			}
			rb := runRBaseline(t, "factor_analysis", map[string]any{
				"rows": rows, "extraction": "minres", "rotation": string(rotation), "scoring": "none", "nfactors": 2,
			})
			assertFactorAnalysisMatchesR(t, got, rb, 1e-8)
		})
	}
}

func TestCrossLangFactorAnalysisScoring(t *testing.T) {
	requireFactorAnalysisRTools(t)
	rows := factorAnalysisRows()
	cases := []stats.FactorScoreMethod{
		stats.FactorScoreNone,
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	}
	for _, scoring := range cases {
		t.Run(string(scoring), func(t *testing.T) {
			opt := factorOptions(stats.FactorExtractionMINRES, stats.FactorRotationOblimin, scoring)
			got, err := stats.FactorAnalysis(dataTableFromRows(rows), opt)
			if err != nil {
				t.Fatalf("FactorAnalysis error: %v", err)
			}
			rb := runRBaseline(t, "factor_analysis", map[string]any{
				"rows": rows, "extraction": "minres", "rotation": "oblimin", "scoring": string(scoring), "nfactors": 2,
			})
			assertFactorAnalysisMatchesR(t, got, rb, 1e-8)
			if scoring != stats.FactorScoreNone {
				assertMatrixCloseToBoth(t, "scores", dataTableMatrix(got.Scores), baselineFloatMatrix(t, rb, "scores"), baselineFloatMatrix(t, rb, "scores"), 1e-8)
				assertMatrixCloseToBoth(t, "score_coefficients", dataTableMatrix(got.ScoreCoefficients), baselineFloatMatrix(t, rb, "score_coefficients"), baselineFloatMatrix(t, rb, "score_coefficients"), 1e-8)
				assertMatrixCloseToBoth(t, "score_covariance", dataTableMatrix(got.ScoreCovariance), baselineFloatMatrix(t, rb, "score_covariance"), baselineFloatMatrix(t, rb, "score_covariance"), 1e-8)
			}
		})
	}
}

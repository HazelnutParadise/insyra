package stats_test

import (
	"math"
	"os"
	"os/exec"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

const factorParityTol = 2e-5

func requireFactorAnalysisRTools(t *testing.T) {
	t.Helper()
	if os.Getenv("INSYRA_STRICT_FACTOR_R_PARITY") != "1" {
		t.Skip("set INSYRA_STRICT_FACTOR_R_PARITY=1 to run strict R factor-analysis parity tests")
	}
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

type factorAnalysisDataset struct {
	name     string
	rows     [][]any
	nFactors int
}

func factorAnalysisRowsAny() [][]any {
	return floatRowsToAny(factorAnalysisRows())
}

func factorAnalysisDatasets() []factorAnalysisDataset {
	return []factorAnalysisDataset{
		{name: "two_blocks", rows: factorAnalysisRowsAny(), nFactors: 2},
		{name: "one_factor", rows: [][]any{
			{1.0, 1.1, 0.9, 1.2},
			{1.4, 1.5, 1.3, 1.6},
			{1.8, 1.7, 1.9, 1.75},
			{2.2, 2.3, 2.1, 2.35},
			{2.7, 2.6, 2.8, 2.65},
			{3.1, 3.2, 3.0, 3.25},
			{3.6, 3.5, 3.7, 3.45},
			{4.0, 4.1, 3.9, 4.2},
			{4.4, 4.3, 4.5, 4.35},
			{4.9, 5.0, 4.8, 5.1},
			{5.3, 5.2, 5.4, 5.25},
			{5.8, 5.9, 5.7, 6.0},
		}, nFactors: 1},
		{name: "three_blocks", rows: [][]any{
			{5.0, 4.8, 1.1, 1.2, 2.5, 2.7},
			{4.7, 5.1, 1.3, 1.0, 2.8, 2.6},
			{5.2, 4.9, 1.0, 1.4, 2.4, 2.9},
			{1.2, 1.0, 5.1, 4.9, 2.7, 2.5},
			{1.4, 1.3, 4.8, 5.2, 2.6, 2.8},
			{1.0, 1.1, 5.3, 5.0, 2.9, 2.4},
			{2.7, 2.5, 1.2, 1.4, 5.1, 4.9},
			{2.5, 2.8, 1.1, 1.0, 4.8, 5.2},
			{2.9, 2.6, 1.3, 1.2, 5.3, 5.0},
			{4.4, 4.2, 2.0, 2.1, 3.3, 3.5},
			{2.1, 2.0, 4.3, 4.5, 3.4, 3.2},
			{3.2, 3.4, 2.2, 2.0, 4.4, 4.6},
		}, nFactors: 3},
		{name: "cross_loading", rows: [][]any{
			{1.0, 1.2, 4.9, 5.1, 3.0},
			{1.3, 1.1, 4.7, 5.0, 3.2},
			{0.9, 1.0, 5.2, 4.8, 3.1},
			{4.8, 5.0, 1.2, 1.1, 3.5},
			{5.1, 4.9, 1.0, 1.3, 3.4},
			{4.9, 5.2, 1.3, 0.9, 3.6},
			{2.4, 2.6, 3.2, 3.3, 2.9},
			{3.2, 3.1, 2.5, 2.7, 3.3},
			{1.5, 1.7, 4.4, 4.6, 3.0},
			{4.5, 4.3, 1.6, 1.4, 3.7},
			{2.0, 2.1, 3.8, 3.7, 3.2},
			{3.7, 3.9, 2.0, 2.2, 3.5},
		}, nFactors: 2},
		{name: "missing_rows", rows: [][]any{
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
			{2.0, 2.2, 2.1, nil, 3.8, 3.9},
			{3.8, 3.7, 3.9, 2.1, 2.0, 2.2},
		}, nFactors: 2},
	}
}

func floatRowsToAny(rows [][]float64) [][]any {
	out := make([][]any, len(rows))
	for i := range rows {
		out[i] = make([]any, len(rows[i]))
		for j := range rows[i] {
			out[i][j] = rows[i][j]
		}
	}
	return out
}

func dataTableFromAnyRows(rows [][]any) *insyra.DataTable {
	if len(rows) == 0 {
		return insyra.NewDataTable()
	}
	nCols := len(rows[0])
	dt := insyra.NewDataTable()
	for c := 0; c < nCols; c++ {
		col := insyra.NewDataList().SetName("C")
		for r := 0; r < len(rows); r++ {
			if rows[r][c] == nil {
				col.Append(math.NaN())
			} else {
				col.Append(rows[r][c])
			}
		}
		dt.AppendCols(col)
	}
	return dt
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
	assertOptionalMatrixCloseToR(t, "phi", dataTableMatrix(got.Phi), rb, "phi", tol)
	assertOptionalMatrixCloseToR(t, "rotation_matrix", dataTableMatrix(got.RotationMatrix), rb, "rotation_matrix", tol)
	assertSliceCloseToBoth(t, "sampling_adequacy", got.SamplingAdequacy.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "sampling_adequacy"), baselineFloatSlice(t, rb, "sampling_adequacy"), tol)
	bart := baselineMap(t, rb, "bartlett")
	assertCloseToBoth(t, "bartlett.chi_square", got.BartlettTest.ChiSquare, baselineFloat(t, bart, "chi_square"), baselineFloat(t, bart, "chi_square"), tol)
	assertCloseToBoth(t, "bartlett.df", float64(got.BartlettTest.DegreesOfFreedom), baselineFloat(t, bart, "df"), baselineFloat(t, bart, "df"), tol)
	assertCloseToBoth(t, "bartlett.p_value", got.BartlettTest.PValue, baselineFloat(t, bart, "p_value"), baselineFloat(t, bart, "p_value"), tol)
	assertCloseToBoth(t, "bartlett.sample_size", float64(got.BartlettTest.SampleSize), baselineFloat(t, bart, "sample_size"), baselineFloat(t, bart, "sample_size"), tol)
}

func baselineMap(t *testing.T, m crossLangBaseline, key string) crossLangBaseline {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	raw, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("baseline key %q has non-object type %T", key, v)
	}
	return crossLangBaseline(raw)
}

func assertOptionalMatrixCloseToR(t *testing.T, label string, got [][]float64, rb crossLangBaseline, key string, tol float64) {
	t.Helper()
	if rb[key] == nil {
		if got != nil {
			t.Fatalf("%s expected nil, got %v", label, got)
		}
		return
	}
	if rawMap, ok := rb[key].(map[string]any); ok && len(rawMap) == 0 {
		if got != nil {
			t.Fatalf("%s expected nil, got %v", label, got)
		}
		return
	}
	want := baselineFloatMatrix(t, rb, key)
	assertMatrixCloseToBoth(t, label, got, want, want, tol)
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
			assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
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
			assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
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
			assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
			if scoring != stats.FactorScoreNone {
				assertMatrixCloseToBoth(t, "scores", dataTableMatrix(got.Scores), baselineFloatMatrix(t, rb, "scores"), baselineFloatMatrix(t, rb, "scores"), factorParityTol)
				assertMatrixCloseToBoth(t, "score_coefficients", dataTableMatrix(got.ScoreCoefficients), baselineFloatMatrix(t, rb, "score_coefficients"), baselineFloatMatrix(t, rb, "score_coefficients"), factorParityTol)
				assertMatrixCloseToBoth(t, "score_covariance", dataTableMatrix(got.ScoreCovariance), baselineFloatMatrix(t, rb, "score_covariance"), baselineFloatMatrix(t, rb, "score_covariance"), factorParityTol)
			}
		})
	}
}

func TestCrossLangFactorAnalysisAllModeCombinations(t *testing.T) {
	requireFactorAnalysisRTools(t)
	ds := factorAnalysisDatasets()[0]
	extractions := []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionML,
		stats.FactorExtractionMINRES,
	}
	rotations := []stats.FactorRotationMethod{
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
	scorings := []stats.FactorScoreMethod{
		stats.FactorScoreNone,
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	}
	for _, extraction := range extractions {
		for _, rotation := range rotations {
			for _, scoring := range scorings {
				t.Run(string(extraction)+"/"+string(rotation)+"/"+string(scoring), func(t *testing.T) {
					opt := factorOptions(extraction, rotation, scoring)
					got, err := stats.FactorAnalysis(dataTableFromAnyRows(ds.rows), opt)
					if err != nil {
						t.Fatalf("FactorAnalysis error: %v", err)
					}
					rb := runRBaseline(t, "factor_analysis", map[string]any{
						"rows": ds.rows, "extraction": string(extraction), "rotation": string(rotation), "scoring": string(scoring), "nfactors": ds.nFactors,
					})
					assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
					if scoring != stats.FactorScoreNone {
						assertMatrixCloseToBoth(t, "scores", dataTableMatrix(got.Scores), baselineFloatMatrix(t, rb, "scores"), baselineFloatMatrix(t, rb, "scores"), factorParityTol)
						assertMatrixCloseToBoth(t, "score_coefficients", dataTableMatrix(got.ScoreCoefficients), baselineFloatMatrix(t, rb, "score_coefficients"), baselineFloatMatrix(t, rb, "score_coefficients"), factorParityTol)
						assertMatrixCloseToBoth(t, "score_covariance", dataTableMatrix(got.ScoreCovariance), baselineFloatMatrix(t, rb, "score_covariance"), baselineFloatMatrix(t, rb, "score_covariance"), factorParityTol)
					}
				})
			}
		}
	}
}

func TestCrossLangFactorAnalysisRepresentativeDatasets(t *testing.T) {
	requireFactorAnalysisRTools(t)
	cases := []struct {
		dataset    factorAnalysisDataset
		extraction stats.FactorExtractionMethod
		rotation   stats.FactorRotationMethod
		scoring    stats.FactorScoreMethod
	}{
		{factorAnalysisDatasets()[1], stats.FactorExtractionMINRES, stats.FactorRotationNone, stats.FactorScoreRegression},
		{factorAnalysisDatasets()[1], stats.FactorExtractionPCA, stats.FactorRotationNone, stats.FactorScoreRegression},
		{factorAnalysisDatasets()[2], stats.FactorExtractionMINRES, stats.FactorRotationOblimin, stats.FactorScoreRegression},
		{factorAnalysisDatasets()[2], stats.FactorExtractionML, stats.FactorRotationVarimax, stats.FactorScoreBartlett},
		{factorAnalysisDatasets()[3], stats.FactorExtractionPAF, stats.FactorRotationPromax, stats.FactorScoreAndersonRubin},
		{factorAnalysisDatasets()[4], stats.FactorExtractionMINRES, stats.FactorRotationOblimin, stats.FactorScoreRegression},
	}
	for _, tc := range cases {
		t.Run(tc.dataset.name+"/"+string(tc.extraction)+"/"+string(tc.rotation)+"/"+string(tc.scoring), func(t *testing.T) {
			opt := factorOptions(tc.extraction, tc.rotation, tc.scoring)
			opt.Count.FixedK = tc.dataset.nFactors
			got, err := stats.FactorAnalysis(dataTableFromAnyRows(tc.dataset.rows), opt)
			if err != nil {
				t.Fatalf("FactorAnalysis error: %v", err)
			}
			rb := runRBaseline(t, "factor_analysis", map[string]any{
				"rows": tc.dataset.rows, "extraction": string(tc.extraction), "rotation": string(tc.rotation), "scoring": string(tc.scoring), "nfactors": tc.dataset.nFactors,
			})
			assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
			if tc.scoring != stats.FactorScoreNone {
				assertMatrixCloseToBoth(t, "scores", dataTableMatrix(got.Scores), baselineFloatMatrix(t, rb, "scores"), baselineFloatMatrix(t, rb, "scores"), factorParityTol)
				assertMatrixCloseToBoth(t, "score_coefficients", dataTableMatrix(got.ScoreCoefficients), baselineFloatMatrix(t, rb, "score_coefficients"), baselineFloatMatrix(t, rb, "score_coefficients"), factorParityTol)
				assertMatrixCloseToBoth(t, "score_covariance", dataTableMatrix(got.ScoreCovariance), baselineFloatMatrix(t, rb, "score_covariance"), baselineFloatMatrix(t, rb, "score_covariance"), factorParityTol)
			}
		})
	}
}

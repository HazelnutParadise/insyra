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

func generatedOneFactorRows() [][]any {
	rows := make([][]any, 18)
	for i := range rows {
		x := float64(i)
		f := -2.2 + 0.27*x + 0.35*math.Sin(0.47*x)
		rows[i] = []any{
			1.0 + 0.92*f + 0.22*math.Sin(1.31*x),
			-0.5 + 0.84*f + 0.25*math.Cos(1.07*x),
			0.7 + 1.05*f + 0.20*math.Sin(0.83*x+0.4),
			2.0 + 0.76*f + 0.24*math.Cos(1.53*x+0.2),
		}
	}
	return rows
}

func factorAnalysisDatasets() []factorAnalysisDataset {
	return []factorAnalysisDataset{
		{name: "two_blocks", rows: factorAnalysisRowsAny(), nFactors: 2},
		{name: "one_factor", rows: generatedOneFactorRows(), nFactors: 1},
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

func generatedFactorAnalysisRows(n int, variant int) [][]any {
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := float64(i%7) - 3.0 + 0.18*float64(i/7)
		f2 := 1.15*math.Sin(0.73*x) + 0.35*math.Cos(0.29*x)
		f3 := 0.75*math.Cos(0.41*x) - 0.20*float64(i%5)
		j1 := 0.55 * math.Sin(1.37*x+float64(variant))
		j2 := 0.50 * math.Cos(0.91*x+float64(variant))
		shift := float64(variant-1) * 1.7
		scale := 1.0 + 0.35*float64(variant)
		rows[i] = []any{
			shift + scale*(0.72*f1+0.18*f2+j1+0.55*math.Sin(2.11*x)),
			-0.6 + scale*(0.64*f1-0.13*f2+0.08*f3+j2+0.58*math.Cos(1.73*x)),
			1.3 + scale*(0.15*f1+0.70*f2-0.06*f3-j1+0.56*math.Sin(1.19*x)),
			-1.1 + scale*(-0.10*f1+0.66*f2+0.12*f3+j2+0.54*math.Cos(2.37*x)),
			2.4 + scale*(0.38*f1+0.34*f2+0.28*f3+j1-j2+0.52*math.Sin(2.71*x)),
			-2.0 + scale*(-0.32*f1+0.22*f2+0.58*f3-j1+0.56*math.Cos(1.53*x)),
		}
	}
	return rows
}

func generatedModerateThreeFactorRows() [][]any {
	const n = 32
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := 1.20*math.Sin(0.39*x) + 0.32*float64(i%5-2)
		f2 := 1.05*math.Cos(0.53*x+0.4) - 0.24*float64(i%7-3)
		f3 := 0.88*math.Sin(0.71*x-0.2) + 0.42*math.Cos(0.17*x)
		n1 := 0.42 * math.Sin(1.37*x+0.2)
		n2 := 0.39 * math.Cos(1.11*x-0.4)
		n3 := 0.36 * math.Sin(1.83*x+0.7)
		rows[i] = []any{
			1.0 + 0.78*f1 + 0.18*f2 + n1,
			-0.7 + 0.72*f1 - 0.12*f3 + n2,
			0.4 + 0.20*f1 + 0.70*f2 + n3,
			-1.2 + 0.75*f2 + 0.14*f3 - n1,
			1.8 + 0.16*f1 + 0.76*f3 - n2,
			-2.1 - 0.18*f2 + 0.68*f3 + n3,
			0.2 + 0.34*f1 + 0.31*f2 + 0.29*f3 + 0.33*math.Cos(2.07*x),
		}
	}
	return rows
}

func generatedWeakTwoFactorRows() [][]any {
	const n = 34
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := 0.95*math.Sin(0.31*x) + 0.18*float64(i%6-2)
		f2 := 0.90*math.Cos(0.47*x+0.3) - 0.14*float64(i%5-2)
		rows[i] = []any{
			0.2 + 0.54*f1 + 0.10*f2 + 0.64*math.Sin(1.19*x),
			-0.4 + 0.49*f1 - 0.08*f2 + 0.61*math.Cos(1.43*x+0.2),
			1.1 + 0.12*f1 + 0.52*f2 + 0.66*math.Sin(1.71*x-0.3),
			-1.3 - 0.11*f1 + 0.47*f2 + 0.63*math.Cos(1.97*x),
			0.7 + 0.32*f1 + 0.28*f2 + 0.67*math.Sin(2.23*x+0.5),
		}
	}
	return rows
}

func generatedHighCorrelationRows() [][]any {
	const n = 28
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f := 1.10*math.Sin(0.27*x) + 0.72*math.Cos(0.13*x) + 0.10*float64(i%4-1)
		rows[i] = []any{
			1.0 + 0.96*f + 0.11*math.Sin(1.07*x),
			-0.5 + 0.93*f + 0.10*math.Cos(1.29*x+0.1),
			0.8 + 0.98*f + 0.09*math.Sin(1.51*x-0.2),
			-1.4 + 0.91*f + 0.12*math.Cos(1.73*x+0.4),
			0.1 + 0.88*f + 0.13*math.Sin(1.91*x),
		}
	}
	return rows
}

func generatedMixedSignRows() [][]any {
	const n = 30
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := 1.15*math.Sin(0.35*x) + 0.22*float64(i%7-3)
		f2 := 1.00*math.Cos(0.49*x-0.2) - 0.18*float64(i%6-2)
		rows[i] = []any{
			0.3 + 0.80*f1 + 0.15*f2 + 0.38*math.Sin(1.23*x),
			-0.8 - 0.74*f1 + 0.18*f2 + 0.36*math.Cos(1.41*x),
			1.2 + 0.22*f1 + 0.76*f2 + 0.35*math.Sin(1.69*x+0.2),
			-1.5 + 0.16*f1 - 0.72*f2 + 0.37*math.Cos(1.87*x-0.4),
			0.6 + 0.46*f1 - 0.34*f2 + 0.40*math.Sin(2.09*x),
			-0.2 - 0.38*f1 - 0.42*f2 + 0.39*math.Cos(2.31*x+0.1),
		}
	}
	return rows
}

func generatedFactorAnalysisDatasets() []factorAnalysisDataset {
	rowsWithMissing := generatedFactorAnalysisRows(22, 2)
	rowsWithMissing[3][4] = nil
	rowsWithMissing[17][1] = nil

	return []factorAnalysisDataset{
		{name: "generated_oblique", rows: generatedFactorAnalysisRows(24, 0), nFactors: 2},
		{name: "generated_scaled_shifted", rows: generatedFactorAnalysisRows(26, 1), nFactors: 2},
		{name: "generated_complete_case", rows: rowsWithMissing, nFactors: 2},
		{name: "generated_moderate_three_factor", rows: generatedModerateThreeFactorRows(), nFactors: 3},
		{name: "generated_weak_two_factor", rows: generatedWeakTwoFactorRows(), nFactors: 2},
		{name: "generated_high_correlation", rows: generatedHighCorrelationRows(), nFactors: 1},
		{name: "generated_mixed_sign", rows: generatedMixedSignRows(), nFactors: 2},
	}
}

func factorAnalysisFullCombinationDatasets() []factorAnalysisDataset {
	base := factorAnalysisDatasets()
	return []factorAnalysisDataset{
		base[0],
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
	if got.CountUsed <= 0 {
		t.Fatalf("expected positive factor count, got %d", got.CountUsed)
	}
	if !got.Converged {
		t.Fatalf("expected factor extraction to converge")
	}
	if !got.RotationConverged {
		t.Fatalf("expected factor rotation to converge")
	}
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
	for _, ds := range factorAnalysisFullCombinationDatasets() {
		for _, extraction := range extractions {
			for _, rotation := range rotations {
				for _, scoring := range scorings {
					t.Run(ds.name+"/"+string(extraction)+"/"+string(rotation)+"/"+string(scoring), func(t *testing.T) {
						opt := factorOptions(extraction, rotation, scoring)
						opt.Count.FixedK = ds.nFactors
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
}

func TestCrossLangFactorAnalysisRepresentativeDatasets(t *testing.T) {
	requireFactorAnalysisRTools(t)
	base := factorAnalysisDatasets()
	generated := generatedFactorAnalysisDatasets()
	cases := []struct {
		dataset    factorAnalysisDataset
		extraction stats.FactorExtractionMethod
		rotation   stats.FactorRotationMethod
		scoring    stats.FactorScoreMethod
	}{
		{base[1], stats.FactorExtractionPAF, stats.FactorRotationNone, stats.FactorScoreRegression},
		{base[1], stats.FactorExtractionPCA, stats.FactorRotationNone, stats.FactorScoreRegression},
		{base[2], stats.FactorExtractionMINRES, stats.FactorRotationOblimin, stats.FactorScoreRegression},
		{base[2], stats.FactorExtractionML, stats.FactorRotationVarimax, stats.FactorScoreBartlett},
		{base[3], stats.FactorExtractionPAF, stats.FactorRotationPromax, stats.FactorScoreAndersonRubin},
		{base[4], stats.FactorExtractionMINRES, stats.FactorRotationOblimin, stats.FactorScoreRegression},
		{generated[0], stats.FactorExtractionPCA, stats.FactorRotationBentlerQ, stats.FactorScoreAndersonRubin},
		{generated[0], stats.FactorExtractionPAF, stats.FactorRotationGeominQ, stats.FactorScoreBartlett},
		{generated[1], stats.FactorExtractionML, stats.FactorRotationGeominT, stats.FactorScoreBartlett},
		{generated[2], stats.FactorExtractionPAF, stats.FactorRotationQuartimin, stats.FactorScoreRegression},
		{generated[3], stats.FactorExtractionPCA, stats.FactorRotationOblimin, stats.FactorScoreRegression},
		{generated[3], stats.FactorExtractionPAF, stats.FactorRotationGeominQ, stats.FactorScoreAndersonRubin},
		{generated[4], stats.FactorExtractionPCA, stats.FactorRotationQuartimax, stats.FactorScoreRegression},
		{generated[4], stats.FactorExtractionPAF, stats.FactorRotationNone, stats.FactorScoreBartlett},
		{generated[5], stats.FactorExtractionPCA, stats.FactorRotationNone, stats.FactorScoreRegression},
		{generated[5], stats.FactorExtractionPAF, stats.FactorRotationNone, stats.FactorScoreBartlett},
		{generated[6], stats.FactorExtractionML, stats.FactorRotationQuartimin, stats.FactorScoreRegression},
		{generated[6], stats.FactorExtractionPAF, stats.FactorRotationPromax, stats.FactorScoreBartlett},
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

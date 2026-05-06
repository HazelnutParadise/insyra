package stats_test

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/stats"
)

// TestFieldDiffAllFailingCombos runs every failing factor-analysis test
// combination, computes Go vs R per-field max abs delta across loadings,
// uniquenesses, communalities, structure, scores, etc, and prints a
// per-combo table. Run with INSYRA_FIELD_DIFF=1.
func TestFieldDiffAllFailingCombos(t *testing.T) {
	if os.Getenv("INSYRA_FIELD_DIFF") != "1" {
		t.Skip("set INSYRA_FIELD_DIFF=1 to run field diff")
	}
	requireFactorAnalysisRTools(t)

	type combo struct {
		dataset    string
		rows       [][]any
		nFactors   int
		extraction stats.FactorExtractionMethod
		rotation   stats.FactorRotationMethod
		scoring    stats.FactorScoreMethod
	}

	// Replicate dataset map from factor_analysis_test.go
	datasetMap := map[string][][]any{
		"generated_near_collinear":      generatedNearCollinearRows(),
		"generated_mixed_scale":         generatedMixedScaleRows(),
		"generated_moderate_three":      generatedModerateThreeFactorRows(),
		"generated_high_correlation":    generatedHighCorrelationRows(),
		"generated_negative_dominant":   generatedNegativeDominantRows(),
		"generated_wide_two_factor":     generatedWideTwoFactorRows(),
	}
	datasetNFactors := map[string]int{
		"generated_near_collinear":    2,
		"generated_mixed_scale":       2,
		"generated_moderate_three":    3,
		"generated_high_correlation":  2,
		"generated_negative_dominant": 2,
		"generated_wide_two_factor":   2,
	}

	failingCombos := []struct {
		dataset, extraction, rotation, scoring string
	}{
		// Pathological (currently failing — for reference):
		{"generated_near_collinear", "ml", "varimax", "regression"},
		{"generated_mixed_scale", "minres", "bentlerQ", "regression"},
		// Well-conditioned (currently passing — what's their drift floor?):
		{"generated_moderate_three", "ml", "varimax", "regression"},
		{"generated_moderate_three", "minres", "oblimin", "regression"},
		{"generated_moderate_three", "paf", "none", "regression"},
		{"generated_moderate_three", "pca", "varimax", "regression"},
		{"generated_high_correlation", "ml", "varimax", "regression"},
		{"generated_high_correlation", "minres", "promax", "bartlett"},
		{"generated_negative_dominant", "ml", "none", "regression"},
		{"generated_wide_two_factor", "minres", "varimax", "regression"},
	}

	type fieldDelta struct {
		field    string
		maxDelta float64
		example  string // "got=..., r=..." for the worst entry
	}

	for _, fc := range failingCombos {
		rows, ok := datasetMap[fc.dataset]
		if !ok {
			continue
		}
		nFactors := datasetNFactors[fc.dataset]
		opt := factorOptions(stats.FactorExtractionMethod(fc.extraction),
			stats.FactorRotationMethod(fc.rotation),
			stats.FactorScoreMethod(fc.scoring))
		opt.Count.FixedK = nFactors
		got, err := stats.FactorAnalysis(dataTableFromAnyRows(rows), opt)
		if err != nil {
			t.Logf("[%s/%s/%s/%s] FactorAnalysis error: %v", fc.dataset, fc.extraction, fc.rotation, fc.scoring, err)
			continue
		}
		rb := runRBaseline(t, "factor_analysis", map[string]any{
			"rows":       rows,
			"extraction": fc.extraction,
			"rotation":   fc.rotation,
			"scoring":    fc.scoring,
			"nfactors":   nFactors,
		})

		var deltas []fieldDelta

		matrixFields := map[string]func() [][]float64{
			"loadings":           func() [][]float64 { return dataTableMatrix(got.Loadings) },
			"unrotated_loadings": func() [][]float64 { return dataTableMatrix(got.UnrotatedLoadings) },
			"structure":          func() [][]float64 { return dataTableMatrix(got.Structure) },
			"communalities":      func() [][]float64 { return dataTableMatrix(got.Communalities) },
			"phi":                func() [][]float64 { return dataTableMatrix(got.Phi) },
			"rotation_matrix":    func() [][]float64 { return dataTableMatrix(got.RotationMatrix) },
			"score_coefficients": func() [][]float64 { return dataTableMatrix(got.ScoreCoefficients) },
			"score_covariance":   func() [][]float64 { return dataTableMatrix(got.ScoreCovariance) },
			"scores":             func() [][]float64 { return dataTableMatrix(got.Scores) },
		}
		for name, getter := range matrixFields {
			goM := getter()
			rM := safeMatrix(rb, name)
			d, ex := matrixMaxDelta(goM, rM)
			deltas = append(deltas, fieldDelta{name, d, ex})
		}
		uniGo := got.Uniquenesses.GetColByNumber(0).ToF64Slice()
		uniR := safeFloatSlice(rb, "uniquenesses")
		d, ex := sliceMaxDelta(uniGo, uniR)
		deltas = append(deltas, fieldDelta{"uniquenesses", d, ex})

		evGo := got.Eigenvalues.GetColByNumber(0).ToF64Slice()
		evR := safeFloatSlice(rb, "eigenvalues")
		d, ex = sliceMaxDelta(evGo, evR)
		deltas = append(deltas, fieldDelta{"eigenvalues", d, ex})

		expGo := got.ExplainedProportion.GetColByNumber(0).ToF64Slice()
		expR := safeFloatSlice(rb, "explained_proportion")
		d, ex = sliceMaxDelta(expGo, expR)
		deltas = append(deltas, fieldDelta{"explained_proportion", d, ex})

		cumGo := got.CumulativeProportion.GetColByNumber(0).ToF64Slice()
		cumR := safeFloatSlice(rb, "cumulative_proportion")
		d, ex = sliceMaxDelta(cumGo, cumR)
		deltas = append(deltas, fieldDelta{"cumulative_proportion", d, ex})

		samGo := got.SamplingAdequacy.GetColByNumber(0).ToF64Slice()
		samR := safeFloatSlice(rb, "sampling_adequacy")
		d, ex = sliceMaxDelta(samGo, samR)
		deltas = append(deltas, fieldDelta{"sampling_adequacy", d, ex})

		sort.Slice(deltas, func(i, j int) bool {
			return deltas[i].maxDelta > deltas[j].maxDelta
		})
		fmt.Fprintf(os.Stderr, "\n=== %s/%s/%s/%s ===\n",
			fc.dataset, fc.extraction, fc.rotation, fc.scoring)
		for _, d := range deltas {
			if d.maxDelta == 0 {
				continue
			}
			tag := "OK"
			if d.maxDelta > 2e-5 {
				tag = "** FAIL"
			}
			fmt.Fprintf(os.Stderr, "  %-25s max delta = %-12g %s   %s\n",
				d.field, d.maxDelta, tag, d.example)
		}
	}
}

func safeMatrix(rb crossLangBaseline, key string) [][]float64 {
	v, ok := rb[key]
	if !ok || v == nil {
		return nil
	}
	rows, ok := v.([]any)
	if !ok {
		return nil
	}
	if len(rows) == 0 {
		return nil
	}
	if _, isFlat := rows[0].(float64); isFlat {
		out := make([][]float64, len(rows))
		for i, x := range rows {
			f, _ := x.(float64)
			out[i] = []float64{f}
		}
		return out
	}
	mat := make([][]float64, len(rows))
	for i, r := range rows {
		ra, _ := r.([]any)
		row := make([]float64, len(ra))
		for j, c := range ra {
			f, _ := c.(float64)
			row[j] = f
		}
		mat[i] = row
	}
	return mat
}

func safeFloatSlice(rb crossLangBaseline, key string) []float64 {
	v, ok := rb[key]
	if !ok || v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		if f, ok := v.(float64); ok {
			return []float64{f}
		}
		return nil
	}
	out := make([]float64, len(arr))
	for i, x := range arr {
		f, _ := x.(float64)
		out[i] = f
	}
	return out
}

func sliceMaxDelta(go_, r []float64) (float64, string) {
	if len(go_) == 0 || len(r) == 0 {
		return 0, ""
	}
	n := len(go_)
	if len(r) < n {
		n = len(r)
	}
	maxD := 0.0
	exi := -1
	for i := 0; i < n; i++ {
		if math.IsNaN(go_[i]) || math.IsNaN(r[i]) {
			continue
		}
		d := math.Abs(go_[i] - r[i])
		if d > maxD {
			maxD = d
			exi = i
		}
	}
	if exi < 0 {
		return 0, ""
	}
	return maxD, fmt.Sprintf("at [%d] go=%.10g r=%.10g", exi, go_[exi], r[exi])
}

func matrixMaxDelta(goM, rM [][]float64) (float64, string) {
	if len(goM) == 0 || len(rM) == 0 {
		return 0, ""
	}
	maxD := 0.0
	exi, exj := -1, -1
	rows := len(goM)
	if len(rM) < rows {
		rows = len(rM)
	}
	for i := 0; i < rows; i++ {
		cols := len(goM[i])
		if len(rM[i]) < cols {
			cols = len(rM[i])
		}
		for j := 0; j < cols; j++ {
			if math.IsNaN(goM[i][j]) || math.IsNaN(rM[i][j]) {
				continue
			}
			d := math.Abs(goM[i][j] - rM[i][j])
			if d > maxD {
				maxD = d
				exi, exj = i, j
			}
		}
	}
	if exi < 0 {
		return 0, ""
	}
	return maxD, fmt.Sprintf("at [%d,%d] go=%.10g r=%.10g",
		exi, exj, goM[exi][exj], rM[exi][exj])
}

// silence "unused" if go_ var name shadows builtin
var _ = strings.HasPrefix
